package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	m "tibsphereai/pkg/models"

	"github.com/xuri/excelize/v2"
)

const maxCaseImportRows = 500

// ParseCaseImportFile - .csv yoki .xlsx faylni "sarlavha -> qiymat" qatorlariga aylantiradi.
// Sarlavhalar kichik harfga o'tkaziladi, bo'sh joylar "_" ga almashtiriladi. Birinchi
// bo'sh bo'lmagan qator sarlavha hisoblanadi.
func ParseCaseImportFile(filename string, data []byte) ([]map[string]string, error) {
	var grid [][]string
	var err error

	switch {
	case strings.HasSuffix(strings.ToLower(filename), ".xlsx"):
		grid, err = readXLSX(data)
	case strings.HasSuffix(strings.ToLower(filename), ".csv"):
		grid, err = readCSV(data)
	default:
		return nil, errors.New("faqat .csv yoki .xlsx fayl qabul qilinadi")
	}
	if err != nil {
		return nil, err
	}

	headerIdx := -1
	for i, row := range grid {
		if !rowEmpty(row) {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return nil, errors.New("fayl bo'sh")
	}

	headers := make([]string, len(grid[headerIdx]))
	for i, h := range grid[headerIdx] {
		headers[i] = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(h)), " ", "_")
	}

	var rows []map[string]string
	for i := headerIdx + 1; i < len(grid); i++ {
		if rowEmpty(grid[i]) {
			continue
		}
		row := map[string]string{"_row": strconv.Itoa(i + 1)}
		for j, v := range grid[i] {
			if j < len(headers) && headers[j] != "" {
				row[headers[j]] = strings.TrimSpace(v)
			}
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, errors.New("faylda case qatorlari yo'q")
	}
	if len(rows) > maxCaseImportRows {
		return nil, fmt.Errorf("bir martada ko'pi bilan %d ta case yuklash mumkin", maxCaseImportRows)
	}
	return rows, nil
}

func rowEmpty(row []string) bool {
	for _, v := range row {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

func readXLSX(data []byte) ([][]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("xlsx faylni o'qib bo'lmadi: %w", err)
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("xlsx faylda varaq yo'q")
	}
	return f.GetRows(sheets[0])
}

func readCSV(data []byte) ([][]string, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")) // Excel UTF-8 BOM
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	// Excel (mintaqaga qarab) ";" bilan saqlashi mumkin - sarlavha qatoriga qarab aniqlaymiz.
	firstLine := string(data)
	if i := strings.IndexByte(firstLine, '\n'); i >= 0 {
		firstLine = firstLine[:i]
	}
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		r.Comma = ';'
	}
	var grid [][]string
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("csv faylni o'qib bo'lmadi: %w", err)
		}
		grid = append(grid, rec)
	}
	return grid, nil
}

// ImportCases - ParseCaseImportFile natijasidagi har qatordan case yaratadi (draft).
// publish=true bo'lsa yaratilgan case'lar darhol e'lon qilinadi (deploy).
func (s *CaseService) ImportCases(ctx context.Context, rows []map[string]string, publish bool) *m.CaseImportRes {
	res := &m.CaseImportRes{Total: len(rows), Ids: []string{}, Errors: []m.CaseImportRowError{}}

	for _, row := range rows {
		rowNum, _ := strconv.Atoi(row["_row"])
		req, err := s.caseFromRow(ctx, row)
		if err != nil {
			res.Failed++
			res.Errors = append(res.Errors, m.CaseImportRowError{Row: rowNum, Message: err.Error()})
			continue
		}
		created, err := s.storage.CaseS.Create(req)
		if err != nil {
			res.Failed++
			res.Errors = append(res.Errors, m.CaseImportRowError{Row: rowNum, Message: err.Error()})
			continue
		}
		res.Created++
		res.Ids = append(res.Ids, created.Id)
		if publish {
			if _, err := s.storage.CaseS.Publish(&m.ById{Id: created.Id}); err != nil {
				res.Errors = append(res.Errors, m.CaseImportRowError{Row: rowNum, Message: "yaratildi, lekin e'lon qilinmadi: " + err.Error()})
				continue
			}
			res.Published++
		}
	}
	return res
}

func langMap(row map[string]string, prefix string) map[string]string {
	out := map[string]string{}
	for _, l := range []string{"uz", "ru", "en"} {
		if v := row[prefix+"_"+l]; v != "" {
			out[l] = v
		}
	}
	return out
}

func atoiOr(v string, def int) (int, error) {
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%q butun son emas", v)
	}
	return n, nil
}

func (s *CaseService) caseFromRow(ctx context.Context, row map[string]string) (*m.CaseCreateReq, error) {
	title := langMap(row, "title")
	if len(title) == 0 {
		return nil, errors.New("title_uz (yoki title_ru/title_en) majburiy")
	}

	difficulty := strings.ToLower(row["difficulty"])
	if difficulty == "" {
		difficulty = "medium"
	}
	if difficulty != "easy" && difficulty != "medium" && difficulty != "hard" {
		return nil, fmt.Errorf("difficulty easy|medium|hard bo'lishi kerak, berilgan: %q", row["difficulty"])
	}

	req := &m.CaseCreateReq{
		Title:          title,
		Subtitle:       langMap(row, "subtitle"),
		ChiefComplaint: langMap(row, "chief_complaint"),
		ExpectedAnswer: row["expected_answer"],
		CoverImageUrl:  row["cover_image_url"],
		Difficulty:     difficulty,
		PatientGender:  strings.ToLower(row["patient_gender"]),
		VisualState:    row["visual_state"],
	}

	var err error
	if req.PatientAge, err = atoiOr(row["patient_age"], 0); err != nil {
		return nil, fmt.Errorf("patient_age: %w", err)
	}
	if req.ExpectedDurationMinutes, err = atoiOr(row["expected_duration_minutes"], 10); err != nil {
		return nil, fmt.Errorf("expected_duration_minutes: %w", err)
	}
	if req.OrderNum, err = atoiOr(row["order_num"], 1); err != nil {
		return nil, fmt.Errorf("order_num: %w", err)
	}
	if req.InitialVitals.HR, err = atoiOr(row["hr"], 0); err != nil {
		return nil, fmt.Errorf("hr: %w", err)
	}
	if req.InitialVitals.SpO2, err = atoiOr(row["spo2"], 0); err != nil {
		return nil, fmt.Errorf("spo2: %w", err)
	}
	if req.InitialVitals.RR, err = atoiOr(row["rr"], 0); err != nil {
		return nil, fmt.Errorf("rr: %w", err)
	}
	if req.InitialVitals.GCS, err = atoiOr(row["gcs"], 0); err != nil {
		return nil, fmt.Errorf("gcs: %w", err)
	}
	req.InitialVitals.BP = row["bp"]
	if v := strings.ReplaceAll(row["temp"], ",", "."); v != "" {
		if req.InitialVitals.Temp, err = strconv.ParseFloat(v, 64); err != nil {
			return nil, fmt.Errorf("temp: %q son emas", row["temp"])
		}
	}

	if v := row["scenario"]; v != "" {
		var sc map[string]any
		if err := json.Unmarshal([]byte(v), &sc); err != nil {
			return nil, fmt.Errorf("scenario to'g'ri JSON obyekt emas: %w", err)
		}
		req.Scenario = sc
	} else {
		req.Scenario = map[string]any{}
	}

	if t := row["topic"]; t != "" {
		id, err := s.resolveTopic(ctx, t)
		if err != nil {
			return nil, err
		}
		req.TopicId = id
	}
	return req, nil
}

// resolveTopic - "topic" ustuni: mavzu UUID'si yoki nomi (uz/ru/en, katta-kichik harfga befarq).
func (s *CaseService) resolveTopic(ctx context.Context, v string) (string, error) {
	var id string
	err := s.storage.Db.QueryRow(ctx, `
		SELECT id::text FROM topics
		WHERE deleted_at = 0 AND (
			id::text = $1
			OR lower(name->>'uz') = lower($1) OR lower(name->>'ru') = lower($1) OR lower(name->>'en') = lower($1)
		) LIMIT 1
	`, v).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("mavzu topilmadi: %q", v)
	}
	return id, nil
}
