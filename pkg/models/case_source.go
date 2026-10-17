package models

// CaseSource - AI case generatsiyasi uchun ishonchli manba: Content - veb-havola
// (http/https) yoki to'g'ridan-to'g'ri matn (klinik qo'llanmadan parcha).

type CaseSourceCreateReq struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type CaseSourceUpdateReq struct {
	Id      string `json:"id"`
	Name    string `json:"name"`
	Content string `json:"content"`
}

type CaseSourceRes struct {
	Id        *string `json:"id"`
	Name      *string `json:"name"`
	Content   *string `json:"content"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
}

type CaseSourceList struct {
	Sources []*CaseSourceRes `json:"sources"`
	Count   int              `json:"count"`
}

type CaseSourceGetListReq struct {
	Name   string     `json:"name"`
	Filter Pagination `json:"filter"`
}
