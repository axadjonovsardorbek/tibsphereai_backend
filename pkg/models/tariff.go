package models

type TariffCreateReq struct {
	Name        string  `json:"name"`
	Duration    int     `json:"duration"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

type TariffRes struct {
	Id          *string  `json:"id"`
	Name        *string  `json:"name"`
	Duration    *int     `json:"duration"`
	Price       *float64 `json:"price"`
	Description *string  `json:"description"`
	CreatedAt   *string  `json:"created_at"`
	UpdatedAt   *string  `json:"updated_at"`
}

type TariffList struct {
	Tariffs []*TariffRes `json:"tariffs"`
	Count   int          `json:"count"`
}

type TariffGetListReq struct {
	Duration int `json:"duration"`
}

type TariffUpdateBody struct {
	Name        string  `json:"name"`
	Duration    int     `json:"duration"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

type TariffUpdateReq struct {
	Id          string  `json:"id"`
	Name        string  `json:"name"`
	Duration    int     `json:"duration"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}
