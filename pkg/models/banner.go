package models

type BannerCreateReq struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageUrl    string `json:"image_url"`
	LinkUrl     string `json:"link_url"`
	OrderNum    int    `json:"order_num"`
}

type BannerCreateBody struct {
	Title       map[string]string `json:"title"`
	Description map[string]string `json:"description"`
	ImageUrl    map[string]string `json:"image_url"`
	LinkUrl     string            `json:"link_url"`
	OrderNum    int               `json:"order_num"`
}

type BannerRes struct {
	Id          *string           `json:"id"`
	Title       map[string]string `json:"title"`
	Description map[string]string `json:"description"`
	ImageUrl    map[string]string `json:"image_url"`
	LinkUrl     *string           `json:"link_url"`
	OrderNum    *int              `json:"order_num"`
	CreatedAt   *string           `json:"created_at"`
	UpdatedAt   *string           `json:"updated_at"`
}

type BannerList struct {
	Banners []*BannerRes `json:"banners"`
	Count   int          `json:"count"`
}

type BannerGetListReq struct {
	Title  string     `json:"title"`
	Filter Pagination `json:"filter"`
}

type BannerUpdateBody struct {
	Title       map[string]string `json:"title"`
	Description map[string]string `json:"description"`
	ImageUrl    map[string]string `json:"image_url"`
	LinkUrl     string            `json:"link_url"`
	OrderNum    int               `json:"order_num"`
}

type BannerUpdateReq struct {
	Id          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageUrl    string `json:"image_url"`
	LinkUrl     string `json:"link_url"`
	OrderNum    int    `json:"order_num"`
}

type BannerMobileRes struct {
	Id          *string `json:"id"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	ImageUrl    *string `json:"image_url"`
	LinkUrl     *string `json:"link_url"`
	OrderNum    *int    `json:"order_num"`
	CreatedAt   *string `json:"created_at"`
	UpdatedAt   *string `json:"updated_at"`
}

type BannerMobileList struct {
	Banners []*BannerMobileRes `json:"banners"`
	Count   int                `json:"count"`
}

type BannerGetListMobileReq struct {
	Language string `json:"language"`
}
