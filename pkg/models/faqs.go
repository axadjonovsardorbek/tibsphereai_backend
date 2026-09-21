package models

type FaqCreateReq struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
	OrderNum int    `json:"order_num"`
}

type FaqCreateBody struct {
	Question map[string]string `json:"question"`
	Answer   map[string]string `json:"answer"`
	OrderNum int               `json:"order_num"`
}

type FaqRes struct {
	Id        string            `json:"id"`
	Question  map[string]string `json:"question"`
	Answer    map[string]string `json:"answer"`
	OrderNum  *int              `json:"order_num"`
	CreatedAt *string           `json:"created_at"`
	UpdatedAt *string           `json:"updated_at"`
}

type FaqList struct {
	Faqs  []*FaqRes `json:"faqs"`
	Count int       `json:"count"`
}

type FaqGetListReq struct {
	Question string     `json:"question"`
	Filter   Pagination `json:"filter"`
}

type FaqUpdateBody struct {
	Question map[string]string `json:"question"`
	Answer   map[string]string `json:"answer"`
	OrderNum int               `json:"order_num"`
}

type FaqUpdateReq struct {
	Id       string `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
	OrderNum int    `json:"order_num"`
}

type FaqMobileRes struct {
	Id        string  `json:"id"`
	Question  *string `json:"question"`
	Answer    *string `json:"answer"`
	OrderNum  *int    `json:"order_num"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
}

type FaqMobileList struct {
	Faqs  []*FaqMobileRes `json:"faqs"`
	Count int             `json:"count"`
}

type FaqGetListMobileReq struct {
	Question string `json:"question"`
	Language string `json:"language"`
}
