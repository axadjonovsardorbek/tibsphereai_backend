package models

import "time"

// AI fleshkartalar (TZ_MVP): 2 tomonlama (old/orqa, flip), intervalli takrorlash (Spaced Repetition).

const (
	FlashcardRatingAgain = "again" // eslay olmadi - 10 daqiqadan keyin
	FlashcardRatingHard  = "hard"  // zo'rg'a - ~1 kun
	FlashcardRatingGood  = "good"  // esladi - 3 kun, keyin 7 -> 21 -> ...
	FlashcardRatingEasy  = "easy"  // juda oson - 7-14 kun

	FlashcardKindMistakes = "mistakes"
	FlashcardKindTopic    = "topic"
	FlashcardKindManual   = "manual"

	FlashcardMinDeckCards = 5
	FlashcardMaxDeckCards = 30
	// FlashcardMaxPerSession - bitta sessiya xatolaridan yaratiladigan kartalar chegarasi.
	FlashcardMaxPerSession = 10
)

type FlashcardRes struct {
	Id       string      `json:"id"`
	DeckId   string      `json:"deck_id,omitempty"`
	Front    string      `json:"front"`
	Back     string      `json:"back"`
	Sources  []SourceRef `json:"sources"`
	Status   string      `json:"status"`
	Interval int         `json:"interval_days"`
	Reps     int         `json:"repetitions"`
	Lapses   int         `json:"lapses"`
	DueAt    string      `json:"due_at"`
	IsNew    bool        `json:"is_new"` // hali takrorlanmagan
	// NextIntervals - har bir baho tugmasi uchun keyingi interval (kun); again = 0 (~10 daqiqa).
	NextIntervals map[string]int `json:"next_intervals,omitempty"`
	CreatedAt     string         `json:"created_at"`
}

type FlashcardDeckRes struct {
	Id         string `json:"id"`
	Title      string `json:"title"`
	Kind       string `json:"kind"`
	CategoryId string `json:"category_id,omitempty"`
	TopicId    string `json:"topic_id,omitempty"`
	FreeTopic  string `json:"free_topic,omitempty"`
	CardCount  int    `json:"card_count"`
	DueCount   int    `json:"due_count"`
	CreatedAt  string `json:"created_at"`
}

type FlashcardDeckCreateReq struct {
	UserId     string `json:"-"`
	Language   string `json:"-"`
	CategoryId string `json:"category_id"`
	TopicId    string `json:"topic_id"`
	TopicText  string `json:"topic_text"` // erkin mavzu
	Title      string `json:"title"`      // ixtiyoriy
	Count      int    `json:"count"`      // 5..30, default 10
}

type FlashcardCreateReq struct {
	UserId string `json:"-"`
	DeckId string `json:"deck_id"`
	Front  string `json:"front"`
	Back   string `json:"back"`
}

type FlashcardUpdateReq struct {
	UserId string  `json:"-"`
	Id     string  `json:"-"`
	Front  *string `json:"front"`
	Back   *string `json:"back"`
	Status *string `json:"status"` // active | suspended
}

type FlashcardReviewReq struct {
	UserId string `json:"-"`
	CardId string `json:"-"`
	Rating string `json:"rating"` // again | hard | good | easy
}

// FlashcardReviewRes - baholashdan keyingi holat.
type FlashcardReviewRes struct {
	Card         *FlashcardRes `json:"card"`
	NextInterval int           `json:"next_interval_days"`
	DueAt        string        `json:"due_at"`
	StreakCount  int           `json:"streak_count"`
}

type FlashcardDueRes struct {
	Cards    []*FlashcardRes `json:"cards"`
	DueTotal int             `json:"due_total"` // hozir takrorlashga tayyor jami kartalar
}

type FlashcardStatsRes struct {
	Total         int     `json:"total"`
	DueNow        int     `json:"due_now"`
	DueToday      int     `json:"due_today"` // bugun (Toshkent) oxirigacha takrorlanishi kerak
	New           int     `json:"new"`       // hali ko'rilmagan
	Learned       int     `json:"learned"`   // interval >= 21 kun
	ReviewsToday  int     `json:"reviews_today"`
	Retention30d  float64 `json:"retention_30d"` // 0..1: good+easy ulushi
	Decks         int     `json:"decks"`
	StreakCount   int     `json:"streak_count"`
	LongestStreak int     `json:"longest_streak"`
}

// --- storage qatlami ---

type FlashcardRow struct {
	Id              string
	UserId          string
	DeckId          string
	Front           string
	Back            string
	Sources         []SourceRef
	OriginSessionId string
	CategoryId      string
	TopicId         string
	Language        string
	Ease            float64
	IntervalDays    int
	Repetitions     int
	Lapses          int
	DueAt           time.Time
	LastReviewedAt  *time.Time
	Status          string
	CreatedAt       time.Time
}

type FlashcardDeckRow struct {
	Id         string
	UserId     string
	Title      string
	Kind       string
	CategoryId string
	TopicId    string
	FreeTopic  string
	Language   string
	CardCount  int
	DueCount   int
	CreatedAt  time.Time
}

// FlashcardGen - AI yaratgan bitta karta.
type FlashcardGen struct {
	Front       string `json:"front"`
	Back        string `json:"back"`
	UsedSources []int  `json:"used_sources"`
}

// FlashcardGenInput - karta generatsiyasi uchun kirish.
type FlashcardGenInput struct {
	Mode         string // mistakes | topic
	Language     string
	Count        int
	CategoryName string
	TopicNames   []string
	TopicHints   []string
	FreeTopic    bool
	Mistakes     []FlashcardMistake
	Chunks       []SourceChunk
	Existing     []string // foydalanuvchida mavjud kartalar old tomoni (takrorlamaslik uchun)
}

// FlashcardMistake - sessiyadagi xato/qisman javob qilingan element (karta yaratish uchun asos).
type FlashcardMistake struct {
	Kind          string // case | test
	Prompt        string // shikoyat yoki savol
	UserAnswer    string
	CorrectAnswer string
	Explanation   string
}

// FlashcardAdminStats - admin monitoring (TZ_MVP: fleshkartalar monitoringi).
type FlashcardAdminStats struct {
	TotalCards      int                  `json:"total_cards"`
	TotalDecks      int                  `json:"total_decks"`
	UsersWithCards  int                  `json:"users_with_cards"`
	GeneratedToday  int                  `json:"generated_today"`
	ReviewsToday    int                  `json:"reviews_today"`
	ActiveReviewers int                  `json:"active_reviewers_today"`
	Retention30d    float64              `json:"retention_30d"`
	MasteryRate     float64              `json:"mastery_rate"` // interval >= 21 kun bo'lgan kartalar ulushi
	WeakTopics      []FlashcardWeakTopic `json:"weak_topics"`
}

type FlashcardWeakTopic struct {
	TopicId   string            `json:"topic_id"`
	Name      map[string]string `json:"name"`
	CardCount int               `json:"card_count"` // xatolardan yaratilgan kartalar
	WrongRate float64           `json:"wrong_rate"` // practice javoblaridagi xato ulushi (0..1)
	Answered  int               `json:"answered"`
}
