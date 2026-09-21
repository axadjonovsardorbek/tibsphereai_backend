package models

type AboutCreateBody struct {
	Title       map[string]string `json:"title"`
	Description map[string]string `json:"description"`
	LinkUrl     string            `json:"link_url"`
	OrderNum    int               `json:"order_num"`
}

type AboutCreateReq struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	LinkUrl     string `json:"link_url"`
	OrderNum    int    `json:"order_num"`
}

type AboutRes struct {
	Id          string            `json:"id"`
	Title       map[string]string `json:"title"`
	Description map[string]string `json:"description"`
	LinkUrl     *string           `json:"link_url"`
	OrderNum    *int              `json:"order_num"`
	CreatedAt   *string           `json:"created_at"`
	UpdatedAt   *string           `json:"updated_at"`
}

type AboutUpdateBody struct {
	Title       map[string]string `json:"title"`
	Description map[string]string `json:"description"`
	LinkUrl     string            `json:"link_url"`
	OrderNum    int               `json:"order_num"`
}

type AboutUpdateReq struct {
	Id          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	LinkUrl     string `json:"link_url"`
	OrderNum    int    `json:"order_num"`
}

type AboutList struct {
	Abouts []*AboutRes `json:"abouts"`
	Count  int         `json:"count"`
}

type AboutGetListReq struct {
	Title  string     `json:"title"`
	Filter Pagination `json:"filter"`
}

type AboutMobileRes struct {
	Id          string  `json:"id"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	LinkUrl     *string `json:"link_url"`
	OrderNum    *int    `json:"order_num"`
	CreatedAt   *string `json:"created_at"`
	UpdatedAt   *string `json:"updated_at"`
}

type AboutMobileList struct {
	Abouts []*AboutMobileRes `json:"abouts"`
	Count  int               `json:"count"`
}

type AboutGetListMobileReq struct {
	Title    string `json:"title"`
	Language string `json:"language"`
}
