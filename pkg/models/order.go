package models

type OrderStatisticRes struct {
	Id           string  `json:"id"`
	UserId       string  `json:"user_id"`
	UserName     string  `json:"user_name"`
	UserPhone    string  `json:"user_phone"`
	TariffId     string  `json:"tariff_id"`
	TariffName   string  `json:"tariff_name"`
	Duration     int     `json:"duration"`
	Amount       float64 `json:"amount"`
	CoinsUsed    int     `json:"coins_used"`
	TariffAmount float64 `json:"tariff_amount"`
	PaymentType  string  `json:"payment_type"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"created_at"`
	PaidAt       *string `json:"paid_at,omitempty"`
}

type OrderList struct {
	Count  int                  `json:"count"`
	Orders []*OrderStatisticRes `json:"orders"`
}

type OrderGetListReq struct {
	UserId      string     `json:"user_id"`
	TariffId    string     `json:"tariff_id"`
	Status      string     `json:"status"`
	PaymentType string     `json:"payment_type"`
	Type        string     `json:"type"` //range, day, week, month, year
	Day         string     `json:"day"`  //2024-06-01
	From        string     `json:"from"` //2024-06-01
	To          string     `json:"to"`   //2024-06-30
	Filter      Pagination `json:"filter"`
}

// OrderCreateReq - TZ 2.3-bo'lim: obuna pulga (Click/Payme) YOKI tangaga YOKI aralash
// to'lanadi. CoinsUsed - foydalanuvchi ishlatmoqchi bo'lgan tanga soni (ixtiyoriy,
// balansidan oshmasligi kerak). Qolgan summa app_settings.coin_to_uzs_rate bo'yicha
// so'mga aylantirib ayiriladi; amount<=0 bo'lsa gateway'siz to'g'ridan-to'g'ri PAID bo'ladi.
type OrderCreateReq struct {
	UserId    string `json:"user_id"`
	TariffId  string `json:"tariff_id"`
	CoinsUsed int    `json:"coins_used"`
}

type OrderCreateBody struct {
	TariffId  string `json:"tariff_id"`
	CoinsUsed int    `json:"coins_used"`
}

type OrderCreateRes struct {
	Id  string `json:"id"`
	Url string `json:"url"` // to'liq tangaga yopilgan bo'lsa bo'sh qoladi (gateway shart emas)
	// JustPaid - to'liq tangaga yopilib, gateway'siz darhol PAID bo'lgan holat
	// (service qatlami shu orqali promokod/push logikasini ishga tushiradi).
	JustPaid bool `json:"just_paid"`
}

type OrderRes struct {
	Id                string  `json:"id"`
	UserId            string  `json:"user_id"`
	TariffId          string  `json:"tariff_id"`
	Duration          int     `json:"duration"`
	Amount            float64 `json:"amount"`
	Status            string  `json:"status"`
	MerchantPrepareId *int64  `json:"merchant_prepare_id,omitempty"`
	PaidAt            *string `json:"paid_at,omitempty"`
	Meta              *string `json:"meta,omitempty"` // json string ko'rinishida qaytarsang ham bo'ladi
}

type PayURLRes struct {
	OrderId string `json:"order_id"`
	PayURL  string `json:"pay_url"`
}

type ApplyPrepareParams struct {
	OrderId           string
	ClickTransId      int64
	ClickPaydocId     int64
	MerchantPrepareId int64
	Amount            float64
	MetaPatchJSON     []byte // jsonb patch (raw payload)
}

type PrepareResult struct {
	Error             int
	ErrorNote         string
	MerchantPrepareId int64
}

type ApplyCompleteParams struct {
	OrderId           string
	ClickTransId      int64
	MerchantPrepareId int64
	ErrorCode         int
	MetaPatchJSON     []byte
}

type CompleteResult struct {
	Error             int
	ErrorNote         string
	MerchantConfirmId int64
	// Muvaffaqiyatli PAID bo'lgandagina to'ldiriladi - service shu orqali push/bildirishnoma yuboradi.
	JustPaid    bool
	PaidUserId  string
	PaidOrderId string
}
