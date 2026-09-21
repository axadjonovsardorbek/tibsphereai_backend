package models

// Partner - Hamkor o'quv markazlari (TZ 5-6-bo'lim). Mobil ilovada tavsiya banneri
// sifatida ko'rsatiladi VA o'z promo_codes'larini yaratadi (Hamkor Admin roli orqali).

type PartnerCreateReq struct {
	Name        string            `json:"name"`
	ImageUrl    string            `json:"image_url"`
	LinkUrl     string            `json:"link_url"`
	Description map[string]string `json:"description"`
}

type PartnerUpdateReq struct {
	Id          string            `json:"id"`
	Name        string            `json:"name"`
	ImageUrl    string            `json:"image_url"`
	LinkUrl     string            `json:"link_url"`
	Description map[string]string `json:"description"`
}

type PartnerRes struct {
	Id          *string           `json:"id"`
	Name        *string           `json:"name"`
	ImageUrl    *string           `json:"image_url"`
	LinkUrl     *string           `json:"link_url"`
	Description map[string]string `json:"description"`
	CreatedAt   *string           `json:"created_at"`
	UpdatedAt   *string           `json:"updated_at"`
}

type PartnerList struct {
	Partners []*PartnerRes `json:"partners"`
	Count    int           `json:"count"`
}

type PartnerGetListReq struct {
	Name   string     `json:"name"`
	Filter Pagination `json:"filter"`
}

// PartnerMobileRes - Home ekranidagi "Hamkor kurslari" banneri uchun qisqartirilgan ko'rinish.
type PartnerMobileRes struct {
	Id       *string `json:"id"`
	Name     *string `json:"name"`
	ImageUrl *string `json:"image_url"`
	LinkUrl  *string `json:"link_url"`
}

type PartnerMobileList struct {
	Partners []*PartnerMobileRes `json:"partners"`
	Count    int                 `json:"count"`
}
