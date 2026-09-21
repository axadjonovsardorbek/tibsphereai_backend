package models

// Audience - Category qaysi rejim uchun ekanini bildiradi (frontend home
// ekranidagi "Shifokorlar rejimi" / "Birinchi yordam (Aholi)" toggle).
const (
	AudienceDoctor  = "doctor"
	AudienceCitizen = "citizen"
)

type CategoryCreateReq struct {
	Name         map[string]string `json:"name"`
	IconUrl      string            `json:"icon_url"`
	OrderNum     int               `json:"order_num"`
	CoinsPerCase int               `json:"coins_per_case"`
	Audience     string            `json:"audience"` // doctor|citizen, bo'sh - doctor
}

type CategoryRes struct {
	Id           *string           `json:"id"`
	Name         map[string]string `json:"name"`
	IconUrl      *string           `json:"icon_url"`
	OrderNum     *int              `json:"order_num"`
	CoinsPerCase *int              `json:"coins_per_case"`
	Audience     *string           `json:"audience"`
	CreatedAt    *string           `json:"created_at"`
	UpdatedAt    *string           `json:"updated_at"`
}

type CategoryList struct {
	Categories []*CategoryRes `json:"categories"`
	Count      int            `json:"count"`
}

type CategoryGetListReq struct {
	Name     string     `json:"name"`
	Audience string     `json:"audience"`
	Filter   Pagination `json:"filter"`
}

type CategoryUpdateReq struct {
	Id           string            `json:"id"`
	Name         map[string]string `json:"name"`
	IconUrl      string            `json:"icon_url"`
	OrderNum     int               `json:"order_num"`
	CoinsPerCase int               `json:"coins_per_case"`
	Audience     string            `json:"audience"`
}

type CategoryMobileRes struct {
	Id           *string `json:"id"`
	Name         *string `json:"name"`
	IconUrl      *string `json:"icon_url"`
	OrderNum     *int    `json:"order_num"`
	CasesCount   *int    `json:"cases_count"`
	CoinsPerCase *int    `json:"coins_per_case"`
	Audience     *string `json:"audience"`
	// CasesRemaining - shu foydalanuvchi ushbu bo'limda tanga evaziga sotib olgan va
	// hali sarflamagan case-huquqlari soni (Premium obunasiz kirish uchun).
	CasesRemaining *int `json:"cases_remaining"`
}

type CategoryMobileList struct {
	Categories []*CategoryMobileRes `json:"categories"`
	Count      int                  `json:"count"`
}

type CategoryGetListMobileReq struct {
	Language string     `json:"language"`
	UserId   string     `json:"user_id"`
	Audience string     `json:"audience"` // doctor|citizen, bo'sh - hammasi
	Filter   Pagination `json:"filter"`
}

// --- Category unlock (tanga evaziga bo'lim ochish, Premium obunaga alternativ) ---

type CategoryUnlockPurchaseReq struct {
	UserId     string `json:"user_id"`
	CategoryId string `json:"category_id"`
	// CaseCount - nechta case-huquqi sotib olinayotgani (masalan 10 ta case uchun 10*coins_per_case tanga).
	CaseCount int `json:"case_count"`
}

type CategoryUnlockRes struct {
	CategoryId     string `json:"category_id"`
	CasesRemaining int    `json:"cases_remaining"`
	CoinsSpent     int    `json:"coins_spent"`
}

type CategoryUnlockStatusRes struct {
	CategoryId     string `json:"category_id"`
	CoinsPerCase   int    `json:"coins_per_case"`
	CasesRemaining int    `json:"cases_remaining"`
}
