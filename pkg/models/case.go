package models

// Case - klinik simulyatsiya keysi (BACKEND_STRUCTURE.md 3,4-bo'lim, TZ 6-bo'lim).
// `Scenario` ichida bemor personaji prompti, kutilgan qadamlar/labs/imaging/dorilar
// erkin JSON tuzilma sifatida saqlanadi - Gemini AI shu maydonni to'ldiradi/o'qiydi.
// `ExpectedAnswer` - admin kiritadigan taxminiy javob (AI debrief/scoring uchun asos).

type CaseCreateReq struct {
	TopicId                 string            `json:"topic_id"`
	Title                   map[string]string `json:"title"`
	Subtitle                map[string]string `json:"subtitle"`
	ExpectedAnswer          string            `json:"expected_answer"`
	CoverImageUrl           string            `json:"cover_image_url"`
	Difficulty              string            `json:"difficulty"` // easy|medium|hard
	ChiefComplaint          map[string]string `json:"chief_complaint"`
	PatientAge              int               `json:"patient_age"`
	PatientGender           string            `json:"patient_gender"`
	ExpectedDurationMinutes int               `json:"expected_duration_minutes"`
	InitialVitals           CaseVitals        `json:"initial_vitals"`
	VisualState             string            `json:"visual_state"`
	Scenario                map[string]any    `json:"scenario"`
	IsAIGenerated           bool              `json:"is_ai_generated"`
	OrderNum                int               `json:"order_num"`
}

type CaseVitals struct {
	HR   int     `json:"hr"`
	BP   string  `json:"bp"`
	SpO2 int     `json:"spo2"`
	RR   int     `json:"rr"`
	Temp float64 `json:"temp"`
	GCS  int     `json:"gcs"`
}

type CaseUpdateReq struct {
	Id                      string            `json:"id"`
	TopicId                 string            `json:"topic_id"`
	Title                   map[string]string `json:"title"`
	Subtitle                map[string]string `json:"subtitle"`
	ExpectedAnswer          string            `json:"expected_answer"`
	CoverImageUrl           string            `json:"cover_image_url"`
	Difficulty              string            `json:"difficulty"`
	ChiefComplaint          map[string]string `json:"chief_complaint"`
	PatientAge              int               `json:"patient_age"`
	PatientGender           string            `json:"patient_gender"`
	ExpectedDurationMinutes int               `json:"expected_duration_minutes"`
	InitialVitals           CaseVitals        `json:"initial_vitals"`
	VisualState             string            `json:"visual_state"`
	Scenario                map[string]any    `json:"scenario"`
	OrderNum                int               `json:"order_num"`
}

type CaseRes struct {
	Id                      *string           `json:"id"`
	TopicId                 *string           `json:"topic_id"`
	TopicName               map[string]string `json:"topic_name"`
	CategoryId              *string           `json:"category_id"`
	CategoryName            map[string]string `json:"category_name"`
	Title                   map[string]string `json:"title"`
	Subtitle                map[string]string `json:"subtitle"`
	ExpectedAnswer          *string           `json:"expected_answer"`
	CoverImageUrl           *string           `json:"cover_image_url"`
	Difficulty              *string           `json:"difficulty"`
	ChiefComplaint          map[string]string `json:"chief_complaint"`
	PatientAge              *int              `json:"patient_age"`
	PatientGender           *string           `json:"patient_gender"`
	ExpectedDurationMinutes *int              `json:"expected_duration_minutes"`
	InitialVitals           *CaseVitals       `json:"initial_vitals"`
	VisualState             *string           `json:"visual_state"`
	Scenario                map[string]any    `json:"scenario"`
	Status                  *string           `json:"status"` // draft|published
	IsAIGenerated           *bool             `json:"is_ai_generated"`
	OrderNum                *int              `json:"order_num"`
	CreatedAt               *string           `json:"created_at"`
	UpdatedAt               *string           `json:"updated_at"`
}

type CaseList struct {
	Cases []*CaseRes `json:"cases"`
	Count int        `json:"count"`
}

type CaseGetListReq struct {
	TopicId       string     `json:"topic_id"`
	CategoryId    string     `json:"category_id"`
	Difficulty    string     `json:"difficulty"`
	Status        string     `json:"status"`
	PatientGender string     `json:"patient_gender"`
	IsAIGenerated string     `json:"is_ai_generated"` // "true"|"false"|"" (any)
	Search        string     `json:"search"`          // title/subtitle/chief_complaint bo'yicha qidiradi
	Filter        Pagination `json:"filter"`
}

// CaseMobileRes - foydalanuvchiga ko'rsatiladigan qisqartirilgan / tilga mos ko'rinish
type CaseMobileRes struct {
	Id                      *string `json:"id"`
	TopicId                 *string `json:"topic_id"`
	TopicName               *string `json:"topic_name"`
	CategoryId              *string `json:"category_id"`
	CategoryName            *string `json:"category_name"`
	Title                   *string `json:"title"`
	Subtitle                *string `json:"subtitle"`
	CoverImageUrl           *string `json:"cover_image_url"`
	Difficulty              *string `json:"difficulty"`
	ChiefComplaint          *string `json:"chief_complaint"`
	PatientAge              *int    `json:"patient_age"`
	PatientGender           *string `json:"patient_gender"`
	ExpectedDurationMinutes *int    `json:"expected_duration_minutes"`
	IsFavorite              *bool   `json:"is_favorite"`
	Status                  *string `json:"status"` // new|active|completed|failed|abandoned (foydalanuvchi holati)
}

type CaseMobileList struct {
	Cases []*CaseMobileRes `json:"cases"`
	Count int              `json:"count"`
}

type CaseGetListMobileReq struct {
	UserId     string     `json:"user_id"`
	Language   string     `json:"language"`
	TopicId    string     `json:"topic_id"`
	CategoryId string     `json:"category_id"`
	Difficulty string     `json:"difficulty"`
	Status     string     `json:"status"` // new|in_progress|completed (foydalanuvchi holati bo'yicha)
	Search     string     `json:"search"`
	Filter     Pagination `json:"filter"`
}

type CaseRandomReq struct {
	UserId     string `json:"user_id"`
	Language   string `json:"language"`
	CategoryId string `json:"category_id"` // bo'sh - filtersiz (istalgan bo'lim)
	Difficulty string `json:"difficulty"`  // easy|medium|hard, bo'sh - filtersiz
}
