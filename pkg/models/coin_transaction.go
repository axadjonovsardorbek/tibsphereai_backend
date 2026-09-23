package models

// CoinTransactionRes - point_transactions jadvalidagi (kind='coin') bitta yozuv,
// admin "Tangalar savdosi monitoringi" ekrani uchun. reason: simulation_finish,
// category_unlock, coin_purchase, referral_reward, promo_redeem va h.k.
type CoinTransactionRes struct {
	Id            string  `json:"id"`
	UserId        string  `json:"user_id"`
	UserName      *string `json:"user_name"`
	Delta         int     `json:"delta"` // musbat - qo'shilgan, manfiy - sarflangan
	Reason        *string `json:"reason"`
	ReferenceType *string `json:"reference_type"`
	ReferenceId   *string `json:"reference_id"`
	CreatedAt     string  `json:"created_at"`
}

type CoinTransactionList struct {
	Count        int                   `json:"count"`
	Transactions []*CoinTransactionRes `json:"transactions"`
}

type CoinTransactionGetListReq struct {
	UserId string     `json:"user_id"`
	Reason string     `json:"reason"`
	From   string     `json:"from"` // 2024-06-01
	To     string     `json:"to"`   // 2024-06-30
	Filter Pagination `json:"filter"`
}

// CoinMonitoringSummaryRes - umumiy tanga iqtisodiyoti ko'rinishi (admin dashboard).
type CoinMonitoringSummaryRes struct {
	TotalPurchased     int `json:"total_purchased"`      // coin_purchase orqali sotib olingan (pul evaziga)
	TotalEarned        int `json:"total_earned"`         // referral_reward + promo_redeem + simulation_finish (musbat)
	TotalSpent         int `json:"total_spent"`          // category_unlock va h.k. (manfiy tranzaksiyalar, musbat songa aylantirilgan)
	CoinsInCirculation int `json:"coins_in_circulation"` // barcha userlar coins yig'indisi (hozirgi balans)
}
