package models

import "time"

// AI yo'l xaritasi (TZ_MVP): foydalanuvchi natijalaridan bilim xaritasi (qaysi mavzu kuchli/zaif)
// va shaxsiy o'qish yo'li.

const (
	MasteryUnexplored = "unexplored" // hech qachon mashq qilinmagan
	MasteryLearning   = "learning"   // ma'lumot kam (< 3 javob)
	MasteryWeak       = "weak"       // < 50
	MasteryMedium     = "medium"     // 50..74
	MasteryStrong     = "strong"     // >= 75

	RoadmapActionFlashcardReview = "flashcard_review"
	RoadmapActionTest            = "test"
	RoadmapActionCase            = "case"
	RoadmapActionDeck            = "deck" // mavzu bo'yicha yangi fleshkarta to'plami
)

// TopicMastery - bilim xaritasidagi bitta mavzu.
type TopicMastery struct {
	CategoryId    string            `json:"category_id"`
	CategoryName  map[string]string `json:"category_name"`
	TopicId       string            `json:"topic_id"`
	TopicName     map[string]string `json:"topic_name"`
	Answered      int               `json:"answered"`
	Accuracy      float64           `json:"accuracy"` // to'g'ri javoblar ulushi 0..1
	Mastery       int               `json:"mastery"`  // 0..100 (so'nggi natijalar vaznliroq); unexplored bo'lsa 0
	Level         string            `json:"level"`
	LastPracticed *string           `json:"last_practiced"`
	Cards         int               `json:"cards"`     // shu mavzudagi fleshkartalar
	CardsDue      int               `json:"cards_due"` // hozir takrorlashga tayyor
}

// RoadmapStep - o'qish yo'lidagi bitta qadam; frontend shu qadamdan to'g'ridan-to'g'ri mashq boshlay oladi.
type RoadmapStep struct {
	Order      int    `json:"order"`
	Action     string `json:"action"` // flashcard_review | test | case | deck
	CategoryId string `json:"category_id,omitempty"`
	TopicId    string `json:"topic_id,omitempty"`
	TopicName  string `json:"topic_name,omitempty"`
	Difficulty string `json:"difficulty,omitempty"`
	Count      int    `json:"count,omitempty"`
	Reason     string `json:"reason"`
}

type RoadmapRes struct {
	Map         []TopicMastery `json:"map"`
	Path        []RoadmapStep  `json:"path"`
	Summary     string         `json:"summary"`      // AI umumiy xulosa (bo'sh bo'lishi mumkin)
	IsAI        bool           `json:"is_ai"`        // qadam izohlari AI tomonidan yozilganmi
	GeneratedAt *string        `json:"generated_at"` // AI versiyasi yaratilgan vaqt
	Strong      int            `json:"strong_count"`
	Weak        int            `json:"weak_count"`
	Unexplored  int            `json:"unexplored_count"`
	CoinsSpent  int            `json:"coins_spent,omitempty"` // faqat refresh javobida
}

// RoadmapCache - saqlanadigan AI qismi.
type RoadmapCache struct {
	Summary     string    `json:"summary"`
	Reasons     []string  `json:"reasons"` // path qadamlari tartibida
	Language    string    `json:"-"`
	GeneratedAt time.Time `json:"-"`
}

// PerfRow - bitta javob berilgan practice elementi (bilim xaritasi uchun xom ma'lumot).
type PerfRow struct {
	TopicId    string
	Score      int
	IsCorrect  bool
	AnsweredAt time.Time
}

// CatalogTopic - faol bo'lim/mavzu katalogi.
type CatalogTopic struct {
	CategoryId    string
	CategoryName  map[string]string
	CategoryOrder int
	TopicId       string
	TopicName     map[string]string
	TopicOrder    int
}

// TopicCards - mavzu bo'yicha fleshkarta soni.
type TopicCards struct {
	Total int
	Due   int
}

// FeedbackCreateReq - foydalanuvchi fikri.
type FeedbackCreateReq struct {
	UserId     string `json:"-"`
	Rating     int    `json:"rating"` // 1..5 (ixtiyoriy, 0 = ko'rsatilmagan)
	Kind       string `json:"kind"`   // general | bug | idea | content | nps
	Message    string `json:"message"`
	Context    string `json:"context"`
	AppVersion string `json:"app_version"`
}

type FeedbackRes struct {
	Id         string `json:"id"`
	UserId     string `json:"user_id"`
	UserName   string `json:"user_name"`
	Rating     *int   `json:"rating"`
	Kind       string `json:"kind"`
	Message    string `json:"message"`
	Context    string `json:"context"`
	AppVersion string `json:"app_version"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

type FeedbackListReq struct {
	Kind   string     `json:"kind"`
	Status string     `json:"status"`
	Filter Pagination `json:"filter"`
}

type FeedbackSummary struct {
	Total     int     `json:"total"`
	AvgRating float64 `json:"avg_rating"`
	New       int     `json:"new"`
}
