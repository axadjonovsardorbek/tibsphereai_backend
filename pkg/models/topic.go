package models

// Topic - Bo'lim (Category) ichidagi Mavzu (TZ 6-bo'lim: Bo'limlar -> Mavzular -> Case'lar).

type TopicCreateReq struct {
	CategoryId string            `json:"category_id"`
	Name       map[string]string `json:"name"`
	OrderNum   int               `json:"order_num"`
	AiHint     string            `json:"ai_hint"` // AI practice uchun yo'nalish (masalan "faqat pediatrik holatlar")
}

type TopicUpdateReq struct {
	Id         string            `json:"id"`
	CategoryId string            `json:"category_id"`
	Name       map[string]string `json:"name"`
	OrderNum   int               `json:"order_num"`
	AiHint     *string           `json:"ai_hint"` // nil = o'zgartirilmaydi
}

type TopicRes struct {
	Id         *string           `json:"id"`
	CategoryId *string           `json:"category_id"`
	Name       map[string]string `json:"name"`
	OrderNum   *int              `json:"order_num"`
	AiHint     *string           `json:"ai_hint"`
	CreatedAt  *string           `json:"created_at"`
	UpdatedAt  *string           `json:"updated_at"`
}

type TopicList struct {
	Topics []*TopicRes `json:"topics"`
	Count  int         `json:"count"`
}

type TopicGetListReq struct {
	CategoryId string     `json:"category_id"`
	Filter     Pagination `json:"filter"`
}

type TopicMobileRes struct {
	Id         *string `json:"id"`
	CategoryId *string `json:"category_id"`
	Name       *string `json:"name"`
	OrderNum   *int    `json:"order_num"`
}

type TopicMobileList struct {
	Topics []*TopicMobileRes `json:"topics"`
	Count  int               `json:"count"`
}

type TopicGetListMobileReq struct {
	CategoryId string     `json:"category_id"`
	Language   string     `json:"language"`
	Filter     Pagination `json:"filter"`
}
