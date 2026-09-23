package models

// Tariff.Kind - "subscription" (Premium obuna, Duration oy hisobida ishlatiladi)
// yoki "coin_package" (bir martalik tanga sotib olish, Coins ishlatiladi).
type TariffCreateReq struct {
	Name        string  `json:"name"`
	Kind        string  `json:"kind"` // "subscription" | "coin_package", bo'sh bo'lsa "subscription"
	Duration    int     `json:"duration"`
	Coins       int     `json:"coins"` // faqat kind="coin_package" uchun
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

type TariffRes struct {
	Id          *string  `json:"id"`
	Name        *string  `json:"name"`
	Kind        *string  `json:"kind"`
	Duration    *int     `json:"duration"`
	Coins       *int     `json:"coins"`
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
	Duration int    `json:"duration"`
	Kind     string `json:"kind"` // filter: "subscription" | "coin_package"
}

type TariffUpdateBody struct {
	Name        string  `json:"name"`
	Kind        string  `json:"kind"`
	Duration    int     `json:"duration"`
	Coins       int     `json:"coins"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}

type TariffUpdateReq struct {
	Id          string  `json:"id"`
	Name        string  `json:"name"`
	Kind        string  `json:"kind"`
	Duration    int     `json:"duration"`
	Coins       int     `json:"coins"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
}
