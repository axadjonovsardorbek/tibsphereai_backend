package models

// Audience - Category qaysi rejim uchun ekanini bildiradi (frontend home
// ekranidagi "Shifokorlar rejimi" / "Birinchi yordam (Aholi)" toggle).
const (
	AudienceDoctor  = "doctor"
	AudienceCitizen = "citizen"
)

type CategoryCreateReq struct {
	Name     map[string]string `json:"name"`
	IconUrl  string            `json:"icon_url"`
	OrderNum int               `json:"order_num"`
	Audience string            `json:"audience"` // doctor|citizen, bo'sh - doctor
}

type CategoryRes struct {
	Id        *string           `json:"id"`
	Name      map[string]string `json:"name"`
	IconUrl   *string           `json:"icon_url"`
	OrderNum  *int              `json:"order_num"`
	Audience  *string           `json:"audience"`
	CreatedAt *string           `json:"created_at"`
	UpdatedAt *string           `json:"updated_at"`
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
	Id       string            `json:"id"`
	Name     map[string]string `json:"name"`
	IconUrl  string            `json:"icon_url"`
	OrderNum int               `json:"order_num"`
	Audience string            `json:"audience"`
}

type CategoryMobileRes struct {
	Id         *string `json:"id"`
	Name       *string `json:"name"`
	IconUrl    *string `json:"icon_url"`
	OrderNum   *int    `json:"order_num"`
	CasesCount *int    `json:"cases_count"`
	Audience   *string `json:"audience"`
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
