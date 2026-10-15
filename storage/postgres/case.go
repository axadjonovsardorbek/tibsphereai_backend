package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CaseRepo struct {
	db *pgxpool.Pool
}

func NewCaseRepo(db *pgxpool.Pool) *CaseRepo {
	return &CaseRepo{db: db}
}

func (r *CaseRepo) Create(req *m.CaseCreateReq) (*m.ById, error) {
	id := uuid.NewString()
	vitals, _ := json.Marshal(req.InitialVitals)
	scenario, _ := json.Marshal(req.Scenario)

	// topic_id - AI-generatsiya qilingan qoralamada bo'sh bo'lishi mumkin
	// (admin publish qilishdan oldin o'zi biriktiradi); "" ni uuid ustuniga
	// yuborish SQLSTATE 22P02 beradi, shu sabab NULL'ga aylantiramiz.
	var topicId interface{}
	if req.TopicId != "" {
		topicId = req.TopicId
	}

	maxAttempts := req.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = m.DefaultCaseMaxAttempts
	}

	_, err := r.db.Exec(context.Background(), `
		INSERT INTO cases (
			id, topic_id, title, subtitle, expected_answer, cover_image_url, difficulty, chief_complaint,
			patient_age, patient_gender, expected_duration_minutes, initial_vitals,
			visual_state, scenario, status, is_ai_generated, order_num, max_attempts
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,'draft',$15,$16,$17)
	`, id, topicId, req.Title, req.Subtitle, req.ExpectedAnswer, req.CoverImageUrl, req.Difficulty, req.ChiefComplaint,
		req.PatientAge, req.PatientGender, req.ExpectedDurationMinutes, string(vitals),
		req.VisualState, string(scenario), req.IsAIGenerated, req.OrderNum, maxAttempts)
	if err != nil {
		return nil, fmt.Errorf("error while creating case: %w", err)
	}
	return &m.ById{Id: id}, nil
}

// caseColumns / caseJoins - admin panel uchun case + tegishli topic/category
// nomlarini birga qaytaradi (BACKEND_STRUCTURE.md: Bo'limlar -> Mavzular -> Case'lar).
const caseColumns = `
	c.id, c.topic_id, t.name, cat.id, cat.name,
	c.title, c.subtitle, c.expected_answer, c.cover_image_url, c.difficulty, c.chief_complaint,
	c.patient_age, c.patient_gender, c.expected_duration_minutes, c.initial_vitals,
	c.visual_state, c.scenario, c.status, c.is_ai_generated, c.order_num, c.created_at::text, c.updated_at::text, c.max_attempts
`

const caseJoins = `
	LEFT JOIN topics t ON t.id = c.topic_id AND t.deleted_at = 0
	LEFT JOIN categories cat ON cat.id = t.category_id AND cat.deleted_at = 0
`

func scanCase(row interface{ Scan(...interface{}) error }) (*m.CaseRes, error) {
	var titleJSON, subtitleJSON, chiefJSON, vitalsJSON, scenarioJSON, topicNameJSON, categoryNameJSON []byte
	c := m.CaseRes{}
	err := row.Scan(
		&c.Id, &c.TopicId, &topicNameJSON, &c.CategoryId, &categoryNameJSON,
		&titleJSON, &subtitleJSON, &c.ExpectedAnswer, &c.CoverImageUrl, &c.Difficulty, &chiefJSON,
		&c.PatientAge, &c.PatientGender, &c.ExpectedDurationMinutes, &vitalsJSON,
		&c.VisualState, &scenarioJSON, &c.Status, &c.IsAIGenerated, &c.OrderNum, &c.CreatedAt, &c.UpdatedAt, &c.MaxAttempts,
	)
	if err != nil {
		return nil, err
	}

	json.Unmarshal(titleJSON, &c.Title)
	json.Unmarshal(subtitleJSON, &c.Subtitle)
	json.Unmarshal(chiefJSON, &c.ChiefComplaint)
	json.Unmarshal(topicNameJSON, &c.TopicName)
	json.Unmarshal(categoryNameJSON, &c.CategoryName)
	var vitals m.CaseVitals
	json.Unmarshal(vitalsJSON, &vitals)
	c.InitialVitals = &vitals
	json.Unmarshal(scenarioJSON, &c.Scenario)

	return &c, nil
}

func (r *CaseRepo) GetById(req *m.ById) (*m.CaseRes, error) {
	row := r.db.QueryRow(context.Background(), `
		SELECT `+caseColumns+`
		FROM cases c `+caseJoins+`
		WHERE c.id = $1 AND c.deleted_at = 0
	`, req.Id)

	c, err := scanCase(row)
	if err == pgx.ErrNoRows {
		return nil, errs.ErrCaseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting case by id: %w", err)
	}
	return c, nil
}

func (r *CaseRepo) GetAll(req *m.CaseGetListReq) (*m.PaginationResponse, *m.CaseList, error) {
	query := `
	SELECT COUNT(c.id) OVER() AS total_count, ` + caseColumns + `
	FROM cases c ` + caseJoins + `
	WHERE c.deleted_at = 0
	`
	var args []interface{}
	if req.TopicId != "" && req.TopicId != "string" {
		args = append(args, req.TopicId)
		query += fmt.Sprintf(" AND c.topic_id = $%d", len(args))
	}
	if req.CategoryId != "" && req.CategoryId != "string" {
		args = append(args, req.CategoryId)
		query += fmt.Sprintf(" AND cat.id = $%d", len(args))
	}
	if req.Difficulty != "" && req.Difficulty != "string" {
		args = append(args, req.Difficulty)
		query += fmt.Sprintf(" AND c.difficulty = $%d", len(args))
	}
	if req.Status != "" && req.Status != "string" {
		args = append(args, req.Status)
		query += fmt.Sprintf(" AND c.status = $%d", len(args))
	}
	if req.PatientGender != "" && req.PatientGender != "string" {
		args = append(args, req.PatientGender)
		query += fmt.Sprintf(" AND c.patient_gender = $%d", len(args))
	}
	if req.IsAIGenerated == "true" || req.IsAIGenerated == "false" {
		args = append(args, req.IsAIGenerated == "true")
		query += fmt.Sprintf(" AND c.is_ai_generated = $%d", len(args))
	}
	if req.Search != "" && req.Search != "string" {
		args = append(args, "%"+req.Search+"%")
		n := len(args)
		query += fmt.Sprintf(" AND (c.title::text ILIKE $%d OR c.subtitle::text ILIKE $%d OR c.chief_complaint::text ILIKE $%d)", n, n, n)
	}
	query += " ORDER BY c.order_num ASC"
	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting cases: %w", err)
	}
	defer rows.Close()

	list := m.CaseList{}
	for rows.Next() {
		var total int
		c := m.CaseRes{}
		var titleJSON, subtitleJSON, chiefJSON, vitalsJSON, scenarioJSON, topicNameJSON, categoryNameJSON []byte
		if err := rows.Scan(
			&total, &c.Id, &c.TopicId, &topicNameJSON, &c.CategoryId, &categoryNameJSON,
			&titleJSON, &subtitleJSON, &c.ExpectedAnswer, &c.CoverImageUrl, &c.Difficulty, &chiefJSON,
			&c.PatientAge, &c.PatientGender, &c.ExpectedDurationMinutes, &vitalsJSON,
			&c.VisualState, &scenarioJSON, &c.Status, &c.IsAIGenerated, &c.OrderNum, &c.CreatedAt, &c.UpdatedAt, &c.MaxAttempts,
		); err != nil {
			return nil, nil, fmt.Errorf("error while scanning case: %w", err)
		}
		json.Unmarshal(titleJSON, &c.Title)
		json.Unmarshal(subtitleJSON, &c.Subtitle)
		json.Unmarshal(chiefJSON, &c.ChiefComplaint)
		json.Unmarshal(topicNameJSON, &c.TopicName)
		json.Unmarshal(categoryNameJSON, &c.CategoryName)
		var vitals m.CaseVitals
		json.Unmarshal(vitalsJSON, &vitals)
		c.InitialVitals = &vitals
		json.Unmarshal(scenarioJSON, &c.Scenario)

		list.Cases = append(list.Cases, &c)
		list.Count = total
	}

	response := m.NewPaginationResponse(req.Filter, list.Count, list.Cases)
	return &response, &list, nil
}

func (r *CaseRepo) Update(req *m.CaseUpdateReq) (*m.Void, error) {
	var conditions []string
	var args []interface{}

	add := func(col string, val interface{}) {
		conditions = append(conditions, col+" = $"+strconv.Itoa(len(args)+1))
		args = append(args, val)
	}

	if req.TopicId != "" && req.TopicId != "string" {
		add("topic_id", req.TopicId)
	}
	if req.Title != nil {
		add("title", req.Title)
	}
	if req.Subtitle != nil {
		add("subtitle", req.Subtitle)
	}
	if req.ExpectedAnswer != "" && req.ExpectedAnswer != "string" {
		add("expected_answer", req.ExpectedAnswer)
	}
	if req.CoverImageUrl != "" && req.CoverImageUrl != "string" {
		add("cover_image_url", req.CoverImageUrl)
	}
	if req.Difficulty != "" && req.Difficulty != "string" {
		add("difficulty", req.Difficulty)
	}
	if req.ChiefComplaint != nil {
		add("chief_complaint", req.ChiefComplaint)
	}
	add("patient_age", req.PatientAge)
	add("patient_gender", req.PatientGender)
	add("expected_duration_minutes", req.ExpectedDurationMinutes)
	if req.MaxAttempts > 0 {
		add("max_attempts", req.MaxAttempts)
	}
	vitals, _ := json.Marshal(req.InitialVitals)
	add("initial_vitals", string(vitals))
	if req.VisualState != "" && req.VisualState != "string" {
		add("visual_state", req.VisualState)
	}
	if req.Scenario != nil {
		scenario, _ := json.Marshal(req.Scenario)
		add("scenario", string(scenario))
	}
	add("order_num", req.OrderNum)

	conditions = append(conditions, "updated_at = now()")
	query := "UPDATE cases SET " + strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0"
	args = append(args, req.Id)

	res, err := r.db.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating case: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("case with id %s couldnt update", req.Id)
	}
	return &m.Void{}, nil
}

func (r *CaseRepo) Delete(req *m.ById) (*m.Void, error) {
	res, err := r.db.Exec(context.Background(), `
		UPDATE cases SET deleted_at = EXTRACT(EPOCH FROM NOW()) WHERE id = $1 AND deleted_at = 0
	`, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting case: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("case with id %s couldnt delete", req.Id)
	}
	return &m.Void{}, nil
}

func (r *CaseRepo) Publish(req *m.ById) (*m.Void, error) {
	res, err := r.db.Exec(context.Background(), `
		UPDATE cases SET status = 'published', updated_at = now() WHERE id = $1 AND deleted_at = 0
	`, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while publishing case: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("case with id %s couldnt publish", req.Id)
	}
	return &m.Void{}, nil
}

// mobileCaseColumns/mobileCaseJoins - mobil ilova uchun tilga mos case ustunlari,
// topic/category nomi bilan birga (BACKEND_STRUCTURE.md: Bo'limlar -> Mavzular -> Case'lar).
const mobileCaseColumns = `
		c.id, c.topic_id, t.name ->> $1, cat.id, cat.name ->> $1,
		c.title ->> $1, c.subtitle ->> $1, c.cover_image_url, c.difficulty,
		c.chief_complaint ->> $1, c.patient_age, c.patient_gender, c.expected_duration_minutes, c.max_attempts,
		EXISTS(SELECT 1 FROM favorites f WHERE f.case_id = c.id AND f.user_id = $2) AS is_favorite,
		COALESCE((
			SELECT s.status FROM simulation_sessions s
			WHERE s.case_id = c.id AND s.user_id = $2
			ORDER BY s.started_at DESC LIMIT 1
		), 'new') AS user_status
`

const mobileCaseJoins = `
	LEFT JOIN topics t ON t.id = c.topic_id AND t.deleted_at = 0
	LEFT JOIN categories cat ON cat.id = t.category_id AND cat.deleted_at = 0
`

func scanCaseMobile(row interface{ Scan(...interface{}) error }) (*m.CaseMobileRes, error) {
	c := m.CaseMobileRes{}
	err := row.Scan(
		&c.Id, &c.TopicId, &c.TopicName, &c.CategoryId, &c.CategoryName, &c.Title, &c.Subtitle, &c.CoverImageUrl, &c.Difficulty,
		&c.ChiefComplaint, &c.PatientAge, &c.PatientGender, &c.ExpectedDurationMinutes, &c.MaxAttempts,
		&c.IsFavorite, &c.Status,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CaseRepo) GetByIdMobile(req *m.ByIdMobile) (*m.CaseMobileRes, error) {
	query := `SELECT ` + mobileCaseColumns + ` FROM cases c ` + mobileCaseJoins + `
	WHERE c.id = $3 AND c.deleted_at = 0 AND c.status = 'published'`

	row := r.db.QueryRow(context.Background(), query, req.Language, req.UserId, req.Id)

	c, err := scanCaseMobile(row)
	if err == pgx.ErrNoRows {
		return nil, errs.ErrCaseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting case by id: %w", err)
	}
	return c, nil
}

// GetAllMobile - user_status/is_favorite hisoblanadigan ustunlar bo'lgani uchun
// status bo'yicha filtr faqat tashqi so'rovda (sub-query ustida) ishlaydi.
func (r *CaseRepo) GetAllMobile(req *m.CaseGetListMobileReq) (*m.PaginationResponse, *m.CaseMobileList, error) {
	inner := `SELECT ` + mobileCaseColumns + `, c.order_num
	FROM cases c ` + mobileCaseJoins + `
	WHERE c.deleted_at = 0 AND c.status = 'published'
	`

	var args []interface{}
	args = append(args, req.Language, req.UserId)

	if req.TopicId != "" && req.TopicId != "string" {
		args = append(args, req.TopicId)
		inner += fmt.Sprintf(" AND c.topic_id = $%d", len(args))
	}
	if req.CategoryId != "" && req.CategoryId != "string" {
		args = append(args, req.CategoryId)
		inner += fmt.Sprintf(" AND cat.id = $%d", len(args))
	}
	if req.Difficulty != "" && req.Difficulty != "string" {
		args = append(args, req.Difficulty)
		inner += fmt.Sprintf(" AND c.difficulty = $%d", len(args))
	}
	if req.Search != "" && req.Search != "string" {
		args = append(args, "%"+req.Search+"%")
		n := len(args)
		inner += fmt.Sprintf(" AND (c.title::text ILIKE $%d OR c.subtitle::text ILIKE $%d OR c.chief_complaint::text ILIKE $%d)", n, n, n)
	}

	query := `SELECT COUNT(*) OVER() AS total_count,
		sub.id, sub.topic_id, sub.topic_name, sub.category_id, sub.category_name, sub.title, sub.subtitle,
		sub.cover_image_url, sub.difficulty, sub.chief_complaint, sub.patient_age, sub.patient_gender,
		sub.expected_duration_minutes, sub.max_attempts, sub.is_favorite, sub.user_status
	FROM (` + inner + `) AS sub(
		id, topic_id, topic_name, category_id, category_name, title, subtitle, cover_image_url, difficulty,
		chief_complaint, patient_age, patient_gender, expected_duration_minutes, is_favorite, user_status, order_num
	) WHERE 1 = 1`

	if req.Status != "" && req.Status != "string" {
		args = append(args, req.Status)
		query += fmt.Sprintf(" AND sub.user_status = $%d", len(args))
	}

	query += " ORDER BY sub.order_num ASC"
	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting cases: %w", err)
	}
	defer rows.Close()

	list := m.CaseMobileList{}
	for rows.Next() {
		c := &m.CaseMobileRes{}
		var total int
		if err := rows.Scan(
			&total, &c.Id, &c.TopicId, &c.TopicName, &c.CategoryId, &c.CategoryName, &c.Title, &c.Subtitle, &c.CoverImageUrl, &c.Difficulty,
			&c.ChiefComplaint, &c.PatientAge, &c.PatientGender, &c.ExpectedDurationMinutes, &c.MaxAttempts,
			&c.IsFavorite, &c.Status,
		); err != nil {
			return nil, nil, fmt.Errorf("error while scanning case: %w", err)
		}
		list.Cases = append(list.Cases, c)
		list.Count = total
	}

	response := m.NewPaginationResponse(req.Filter, list.Count, list.Cases)
	return &response, &list, nil
}

// GetRandom - "Random Case" (BACKEND_STRUCTURE.md 3-bo'lim): foydalanuvchi
// hali yechmagan tasodifiy nashr etilgan keys.
func (r *CaseRepo) GetRandom(req *m.CaseRandomReq) (*m.CaseMobileRes, error) {
	query := `SELECT ` + mobileCaseColumns + `
	FROM cases c ` + mobileCaseJoins + `
	WHERE c.deleted_at = 0 AND c.status = 'published'
	  AND NOT EXISTS (
		SELECT 1 FROM simulation_sessions s
		WHERE s.case_id = c.id AND s.user_id = $2 AND s.status = 'completed'
	  )
	  AND (NULLIF($3, '') IS NULL OR cat.id::text = $3)
	  AND (NULLIF($4, '') IS NULL OR c.difficulty = $4)
	ORDER BY random() LIMIT 1
	`

	row := r.db.QueryRow(context.Background(), query, req.Language, req.UserId, req.CategoryId, req.Difficulty)

	c, err := scanCaseMobile(row)
	if err == pgx.ErrNoRows {
		return nil, errs.ErrCaseNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting random case: %w", err)
	}
	return c, nil
}

func (r *CaseRepo) GetCategoryId(caseId string) (string, error) {
	var categoryId string
	err := r.db.QueryRow(context.Background(), `
		SELECT t.category_id FROM cases c JOIN topics t ON t.id = c.topic_id WHERE c.id = $1
	`, caseId).Scan(&categoryId)
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("case not found: %s", caseId)
	}
	if err != nil {
		return "", fmt.Errorf("error while getting category id for case: %w", err)
	}
	return categoryId, nil
}
