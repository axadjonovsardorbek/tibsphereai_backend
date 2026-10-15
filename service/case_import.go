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
	if req.MaxAttempts, err = atoiOr(row["max_attempts"], m.DefaultCaseMaxAttempts); err != nil {
		return nil, fmt.Errorf("max_attempts: %w", err)
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

// caseImportColumns - shablon ustunlari: sarlavha, tavsif, namuna qiymat.
var caseImportColumns = [][3]string{
	{"topic", "Mavzu nomi (uz/ru/en) yoki mavzu ID. Bo'sh bo'lsa mavzusiz qoladi", "Kardiologiya"},
	{"title_uz", "MAJBURIY (yoki title_ru / title_en). Case nomi", "Ko'krak qafasidagi og'riq"},
	{"title_ru", "Case nomi (ru)", "Боль в грудной клетке"},
	{"title_en", "Case nomi (en)", "Chest pain"},
	{"subtitle_uz", "Qisqa izoh (uz)", "55 yoshli erkak"},
	{"subtitle_ru", "Qisqa izoh (ru)", ""},
	{"subtitle_en", "Qisqa izoh (en)", ""},
	{"chief_complaint_uz", "Bemor shikoyati (uz)", "Ko'krak qafasida siquvchi og'riq, chap qo'lga tarqaladi"},
	{"chief_complaint_ru", "Bemor shikoyati (ru)", ""},
	{"chief_complaint_en", "Bemor shikoyati (en)", ""},
	{"difficulty", "easy | medium | hard (bo'sh = medium)", "medium"},
	{"patient_age", "Bemor yoshi (butun son)", "55"},
	{"patient_gender", "male | female", "male"},
	{"expected_duration_minutes", "Kutilgan davomiylik, daqiqa (bo'sh = 10)", "10"},
	{"max_attempts", "Sessiyada yuborish mumkin bo'lgan xabarlar soni (bo'sh = 10)", "10"},
	{"hr", "Yurak urishi (bpm)", "110"},
	{"bp", "Qon bosimi", "140/90"},
	{"spo2", "SpO2 (%)", "96"},
	{"rr", "Nafas olish soni", "20"},
	{"temp", "Harorat (°C), vergul ham bo'ladi", "36.8"},
	{"gcs", "Glazgo shkalasi (3-15)", "15"},
	{"visual_state", "Vizual holat (ixtiyoriy)", ""},
	{"expected_answer", "To'g'ri javob / tashxis (AI baholash uchun asos)", "O'tkir koronar sindrom"},
	{"cover_image_url", "Muqova rasmi URL (ixtiyoriy)", ""},
	{"order_num", "Tartib raqami (bo'sh = 1)", "1"},
	{"scenario", "Ixtiyoriy JSON obyekt (murakkab ssenariy)", ""},
}

// CaseImportTemplateXLSX - "Case'lar" varag'i (sarlavha + 1 ta namuna qator) va
// "Izoh" varag'i (har ustun tavsifi) bilan tayyor shablon.
func CaseImportTemplateXLSX() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	const cases = "Cases"
	const help = "Izoh"
	if err := f.SetSheetName(f.GetSheetName(0), cases); err != nil {
		return nil, err
	}
	if _, err := f.NewSheet(help); err != nil {
		return nil, err
	}

	headStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2563EB"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	reqStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"DC2626"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	f.SetSheetRow(help, "A1", &[]string{"Ustun", "Tavsif", "Namuna"})
	for i, col := range caseImportColumns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(cases, cell, col[0])
		style := headStyle
		if col[0] == "title_uz" {
			style = reqStyle
		}
		f.SetCellStyle(cases, cell, cell, style)
		sample, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(cases, sample, col[2])
		width := float64(len(col[0])) + 6
		if col[0] == "chief_complaint_uz" || col[0] == "expected_answer" || col[0] == "title_uz" {
			width = 36
		}
		colName, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(cases, colName, colName, width)

		f.SetSheetRow(help, fmt.Sprintf("A%d", i+2), &[]string{col[0], col[1], col[2]})
	}
	f.SetColWidth(help, "A", "A", 28)
	f.SetColWidth(help, "B", "B", 70)
	f.SetColWidth(help, "C", "C", 40)
	f.SetCellStyle(help, "A1", "C1", headStyle)
	f.SetPanes(cases, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	f.SetActiveSheet(0)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
