package models

// AI real-time practice (new-feature.md): foydalanuvchi bo'lim/mavzu/qiyinlik/rejim
// tanlaydi, AI har safar yangi case yoki test generatsiya qiladi.

const (
	PracticeModeCase = "case"
	PracticeModeTest = "test"

	PracticeStatusActive    = "active"
	PracticeStatusFinished  = "finished"
	PracticeStatusAbandoned = "abandoned"

	// PracticeXPPerCorrect - to'g'ri javob uchun XP (case: score/100 ga proporsional).
	PracticeXPPerCorrect = 5
	// PracticeMaxItems - bir sessiyadagi maksimal elementlar soni (cheksiz case rejimida ham).
	PracticeMaxItems = 50
	// PracticeDefaultTestCount - test rejimida savollar soni ko'rsatilmasa.
	PracticeDefaultTestCount = 10
)

type PracticeStartReq struct {
	UserId     string   `json:"-"`
	Language   string   `json:"-"`
	Mode       string   `json:"mode"`        // case | test
	CategoryId string   `json:"category_id"` // majburiy
	TopicIds   []string `json:"topic_ids"`   // bo'sh = butun bo'lim (aralash)
	Difficulty string   `json:"difficulty"`  // easy | medium | hard
	Count      int      `json:"count"`       // test: savollar soni (5..50), default 10
}

type PracticeSessionRes struct {
	Id         string             `json:"id"`
	Mode       string             `json:"mode"`
	CategoryId string             `json:"category_id"`
	TopicIds   []string           `json:"topic_ids"`
	Difficulty string             `json:"difficulty"`
	TotalItems int                `json:"total_items"`
	Generated  int                `json:"generated"` // hozirgacha generatsiya qilingan elementlar
	Answered   int                `json:"answered"`
	Correct    int                `json:"correct"`
	Status     string             `json:"status"`
	Score      int                `json:"score"`
	XPEarned   int                `json:"xp_earned"`
	StartedAt  string             `json:"started_at"`
	FinishedAt *string            `json:"finished_at"`
	Items      []*PracticeItemRes `json:"items,omitempty"` // faqat detal so'rovida
}

// PracticeOption - test varianti (to'g'ri javob alohida maydonda, javobdan keyin ochiladi).
type PracticeOption struct {
	Key  string `json:"key"` // A..D
	Text string `json:"text"`
}

// PracticeItemRes - userga ko'rinadigan element. Javob berilmaguncha to'g'ri javob/izoh
// YASHIRIN (Correct* va Explanation bo'sh).
type PracticeItemRes struct {
	Id        string `json:"id"`
	SessionId string `json:"session_id"`
	Idx       int    `json:"idx"`
	Kind      string `json:"kind"` // case | test

	// case
	ChiefComplaint string         `json:"chief_complaint,omitempty"`
	Patient        map[string]any `json:"patient,omitempty"` // yosh, jins, anamnez, ko'rik, analizlar
	// test
	Question string           `json:"question,omitempty"`
	Options  []PracticeOption `json:"options,omitempty"`

	Answered bool `json:"answered"`
	// Javobdan keyin:
	UserAnswer    any         `json:"user_answer,omitempty"`
	IsCorrect     *bool       `json:"is_correct,omitempty"`
	Score         *int        `json:"score,omitempty"`   // 0..100
	Verdict       string      `json:"verdict,omitempty"` // correct | partial | wrong
	Feedback      string      `json:"feedback,omitempty"`
	CorrectAnswer string      `json:"correct_answer,omitempty"` // case: expected_answer; test: variant matni
	CorrectKey    string      `json:"correct_key,omitempty"`    // test: A..D
	Explanation   string      `json:"explanation,omitempty"`
	Sources       []SourceRef `json:"sources"`
	SourcesNote   string      `json:"sources_note,omitempty"` // manba bo'lmasa: "Manba topilmadi ..."
	Reported      bool        `json:"reported"`
}

type PracticeAnswerReq struct {
	UserId string `json:"-"`
	ItemId string `json:"-"`
	Answer string `json:"answer"` // case: erkin matn; test: variant kaliti (A..D)
}

type PracticeReportReq struct {
	UserId string `json:"-"`
	ItemId string `json:"-"`
	Reason string `json:"reason"`
}

type PracticeListReq struct {
	UserId string     `json:"user_id"`
	Filter Pagination `json:"filter"`
}

type PracticeSessionList struct {
	Sessions []*PracticeSessionRes `json:"sessions"`
	Count    int                   `json:"count"`
}

// PracticeCaseGen - AI generatsiya qilgan case (payload).
type PracticeCaseGen struct {
	ChiefComplaint string         `json:"chief_complaint"`
	Patient        map[string]any `json:"patient"`
	ExpectedAnswer string         `json:"expected_answer"`
	Explanation    string         `json:"explanation"`
	Difficulty     string         `json:"difficulty"`
	Summary        string         `json:"summary"` // takrorlanmaslik uchun qisqa xulosa
}

// PracticeTestGen - AI generatsiya qilgan test savoli (payload).
type PracticeTestGen struct {
	Question     string   `json:"question"`
	Options      []string `json:"options"` // aniq 4 ta
	CorrectIndex int      `json:"correct_index"`
	Explanation  string   `json:"explanation"`
	Difficulty   string   `json:"difficulty"`
	Summary      string   `json:"summary"`
}

// PracticeEval - case javobini AI baholashi.
type PracticeEval struct {
	Score    int    `json:"score"`   // 0..100
	Verdict  string `json:"verdict"` // correct | partial | wrong
	Feedback string `json:"feedback"`
}

// PracticeGenInput - generatsiya uchun kirish.
type PracticeGenInput struct {
	Mode         string
	Language     string
	Difficulty   string
	CategoryName string
	TopicNames   []string
	TopicHints   []string
	Chunks       []SourceChunk // RAG kontekst (bo'sh bo'lishi mumkin)
	Previous     []string      // oldingi elementlar xulosalari
}

// PracticeSessionRow - storage qatlami uchun sessiya qatori.
type PracticeSessionRow struct {
	Id         string
	UserId     string
	Mode       string
	CategoryId string
	TopicIds   []string
	Difficulty string
	TotalItems int
	Status     string
	Score      int
	XPEarned   int
	Language   string
	StartedAt  string
	FinishedAt *string
}

// PracticeItemRow - storage qatlami uchun element qatori.
type PracticeItemRow struct {
	Id             string
	SessionId      string
	UserId         string // sessiya egasi (join orqali)
	Idx            int
	Kind           string
	Payload        []byte // PracticeCaseGen yoki PracticeTestGen JSON (test uchun variantlar aralashtirilgan)
	Sources        []SourceRef
	SourceChunkIds []string
	Summary        string
	UserAnswer     []byte // JSON
	Evaluation     []byte // JSON
	IsCorrect      *bool
	Score          *int
	Reported       bool
	PromptVersion  string
	Model          string
	TokensUsed     int
}

// PracticeStats - admin statistika (new-feature.md 4.1.4).
type PracticeStats struct {
	Sessions      int                   `json:"sessions"`
	ActiveSession int                   `json:"active_sessions"`
	Items         int                   `json:"items"`
	Answered      int                   `json:"answered"`
	CorrectRate   float64               `json:"correct_rate"`
	Tokens        int64                 `json:"tokens"`
	Reports       int                   `json:"reports"`
	ByCategory    []PracticeStatByGroup `json:"by_category"`
	ByTopic       []PracticeStatByGroup `json:"by_topic"`
	RecentReports []PracticeReportRow   `json:"recent_reports"`
}

type PracticeStatByGroup struct {
	Id       string            `json:"id"`
	Name     map[string]string `json:"name"`
	Sessions int               `json:"sessions"`
	Items    int               `json:"items"`
}

type PracticeReportRow struct {
	ItemId    string `json:"item_id"`
	SessionId string `json:"session_id"`
	Kind      string `json:"kind"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
}
