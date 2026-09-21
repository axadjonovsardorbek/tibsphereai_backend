package models

// PromoCode - TZ 5-6-bo'lim: Hamkor (yoki Super Admin) OLDINDAN tanga qiymati bilan
// promokod yaratadi, o'z talabasiga taqdim etadi. Talaba mobil ilovada kodni kiritganda
// tizim avtomatik tangalarni balansga qo'shadi va statusni "used" qiladi (qo'lda emas).

type PromoCodeCreateReq struct {
	PartnerId string `json:"partner_id"`
	Value     int    `json:"value"` // tanga miqdori
	CreatedBy string `json:"created_by"`
	ExpiresAt string `json:"expires_at"`
}

type PromoCodeRes struct {
	Id          *string `json:"id"`
	Code        *string `json:"code"`
	PartnerId   *string `json:"partner_id"`
	PartnerName *string `json:"partner_name"`
	Value       *int    `json:"value"`
	CreatedBy   *string `json:"created_by"`
	UsedBy      *string `json:"used_by"`
	Status      *string `json:"status"` // active | used | expired
	IssuedAt    *string `json:"issued_at"`
	ExpiresAt   *string `json:"expires_at"`
	UsedAt      *string `json:"used_at"`
}

type PromoCodeList struct {
	PromoCodes []*PromoCodeRes `json:"promocodes"`
	Count      int             `json:"count"`
}

type PromoCodeGetListReq struct {
	PartnerId string     `json:"partner_id"`
	UserId    string     `json:"user_id"` // "mening promokodlarim" (used_by)
	Status    string     `json:"status"`
	Filter    Pagination `json:"filter"`
}

// PromoCodeRedeemReq - foydalanuvchi mobil ilovada kodni kiritganda (TZ 2.3-bo'lim).
type PromoCodeRedeemReq struct {
	Code   string `json:"code"`
	UserId string `json:"user_id"`
}

// PromoCodeRedeemRes - qo'shilgan tanga miqdori bilan javob.
type PromoCodeRedeemRes struct {
	CoinsAdded int `json:"coins_added"`
}
