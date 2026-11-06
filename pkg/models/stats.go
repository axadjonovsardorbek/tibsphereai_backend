package models

// Admin statistikasi (TZ_STATISTICS). Barcha davrlar Toshkent kuni bo'yicha: from..to (ikkalasi ham kiradi).

// StatsPeriodReq - ?from=YYYY-MM-DD&to=YYYY-MM-DD (berilmasa oxirgi 30 kun).
type StatsPeriodReq struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type StatsPeriod struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ---------- 1. Moliya ----------

type FinanceStatsRes struct {
	Period       StatsPeriod     `json:"period"`
	UsdToUzsRate float64         `json:"usd_to_uzs_rate"`
	RevenueUZS   float64         `json:"revenue_uzs"`
	PaidOrders   int             `json:"paid_orders"`
	AICostUSD    float64         `json:"ai_cost_usd"`
	AICostUZS    float64         `json:"ai_cost_uzs"`
	ProfitUZS    float64         `json:"profit_uzs"` // tushum - AI xarajati
	ActiveUsers  int             `json:"active_users"`
	PayingUsers  int             `json:"paying_users"`
	ARPU         float64         `json:"arpu_uzs"`  // tushum / davrdagi faol foydalanuvchilar
	ARPPU        float64         `json:"arppu_uzs"` // tushum / davrda to'lagan foydalanuvchilar
	Daily        []FinanceDay    `json:"daily"`
	ByTariff     []FinanceTariff `json:"by_tariff"`
}

type FinanceDay struct {
	Date       string  `json:"date"`
	RevenueUZS float64 `json:"revenue_uzs"`
	AICostUSD  float64 `json:"ai_cost_usd"`
	AICostUZS  float64 `json:"ai_cost_uzs"`
}

type FinanceTariff struct {
	TariffId   string  `json:"tariff_id"`
	Name       string  `json:"name"`
	Kind       string  `json:"kind"` // subscription | coin_package
	Sold       int     `json:"sold"`
	RevenueUZS float64 `json:"revenue_uzs"`
}

// ---------- 2. AI xarajati ----------

type AICostStatsRes struct {
	Period       StatsPeriod `json:"period"`
	TotalCostUSD float64     `json:"total_cost_usd"`
	TotalTokens  int64       `json:"total_tokens"`
	Calls        int         `json:"calls"`
	// CoinsSpent - davrda yechilgan jami tanga (bepul + balans); CostPerCoinUSD - bepul/tangali
	// foydalanuvchilar AI xarajati / CoinsSpent: 1 tanganing haqiqiy tannarxi.
	CoinsSpent     int             `json:"coins_spent"`
	CostPerCoinUSD float64         `json:"cost_per_coin_usd"`
	ByFeature      []AICostFeature `json:"by_feature"`
	PerAction      []AICostAction  `json:"per_action"`
	ByTier         []AICostTier    `json:"by_tier"`
}

type AICostFeature struct {
	Feature string  `json:"feature"` // practice_generate, practice_evaluate, chat, debrief, flashcard_deck, roadmap, practice_image...
	Calls   int     `json:"calls"`
	Tokens  int64   `json:"tokens"`
	CostUSD float64 `json:"cost_usd"`
}

type AICostAction struct {
	Action     string  `json:"action"` // case_session | test_session | simulation | flashcard_deck | roadmap
	Count      int     `json:"count"`
	AvgCostUSD float64 `json:"avg_cost_usd"`
	AvgCoins   float64 `json:"avg_coins"` // foydalanuvchidan o'rtacha yechilgan tanga
}

type AICostTier struct {
	Tier    string  `json:"tier"` // free | coin | premium | system (admin/case generatsiyasi)
	Calls   int     `json:"calls"`
	Users   int     `json:"users"`
	CostUSD float64 `json:"cost_usd"`
}

// ---------- 3. Premium va eng ko'p sarflaganlar ----------

type PremiumCostStatsRes struct {
	Period       StatsPeriod `json:"period"`
	PremiumUsers int         `json:"premium_users"` // davrda obunasi faol bo'lganlar
	AvgAICostUSD float64     `json:"avg_ai_cost_usd"`
	// LossMakingUsers - premiumlar ichida umrbod AI xarajati (so'mda) umrbod to'lagan pulidan ko'p bo'lganlar.
	LossMakingUsers int          `json:"loss_making_users"`
	TopSpenders     []TopSpender `json:"top_spenders"`
}

type TopSpender struct {
	UserId   string  `json:"user_id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"` // premium | paying | free
	CostUSD  float64 `json:"cost_usd"`
	Tokens   int64   `json:"tokens"`
	Sessions int     `json:"sessions"`
	PaidUZS  float64 `json:"paid_uzs"` // umrbod to'lagan puli
}

// ---------- 5. Faollik ----------

type EngagementStatsRes struct {
	Period   StatsPeriod      `json:"period"`
	DAU      int              `json:"dau"` // bugun
	WAU      int              `json:"wau"` // oxirgi 7 kun
	MAU      int              `json:"mau"` // oxirgi 30 kun
	DAUMAU   float64          `json:"dau_mau"`
	NewUsers int              `json:"new_users"` // davrda
	Daily    []EngagementDay  `json:"daily"`
	Funnel   EngagementFunnel `json:"funnel"`
}

type EngagementDay struct {
	Date             string `json:"date"`
	DAU              int    `json:"dau"`
	NewUsers         int    `json:"new_users"`
	Sessions         int    `json:"sessions"`
	FlashcardReviews int    `json:"flashcard_reviews"`
}

// EngagementFunnel - davrda ro'yxatdan o'tganlar bo'yicha.
type EngagementFunnel struct {
	Registered      int `json:"registered"`
	StartedSession  int `json:"started_session"`
	FinishedSession int `json:"finished_session"`
	ReturnedNextDay int `json:"returned_next_day"`
	Paid            int `json:"paid"`
}

// ---------- 9. Foydalanuvchilar ----------

// UserAdminStats - admin ro'yxatida har foydalanuvchiga qo'shiladigan ko'rsatkichlar.
type UserAdminStats struct {
	LastActiveAt      *string `json:"last_active_at"`
	SessionsCount     int     `json:"sessions_count"`
	AICostUSD         float64 `json:"ai_cost_usd"`
	AITokens          int64   `json:"ai_tokens"`
	CoinsSpent        int     `json:"coins_spent"`
	PaidUZS           float64 `json:"paid_uzs"`
	OrdersCount       int     `json:"orders_count"`
	HasSubscription   bool    `json:"has_subscription"`
	SubscriptionUntil *string `json:"subscription_until"`
	SignupMethod      string  `json:"signup_method"` // telegram | email | google
	Platform          string  `json:"platform"`      // ios | android | web
	AppVersion        string  `json:"app_version"`
	FeedbackCount     int     `json:"feedback_count"`
}

type UserStatsRes struct {
	UserId        string              `json:"user_id"`
	Daily         []UserStatsDay      `json:"daily"` // oxirgi 30 kun
	TopCategories []UserStatsCategory `json:"top_categories"`
	Transactions  []UserStatsCoinTx   `json:"transactions"`
	Orders        []UserStatsOrder    `json:"orders"`
}

type UserStatsDay struct {
	Date       string `json:"date"`
	Sessions   int    `json:"sessions"`
	Tokens     int64  `json:"tokens"`
	CoinsSpent int    `json:"coins_spent"`
}

type UserStatsCategory struct {
	CategoryId string `json:"category_id"`
	Name       string `json:"name"`
	Sessions   int    `json:"sessions"`
}

type UserStatsCoinTx struct {
	Delta     int    `json:"delta"`
	Reason    string `json:"reason"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

type UserStatsOrder struct {
	Id          string  `json:"id"`
	TariffName  string  `json:"tariff_name"`
	Amount      float64 `json:"amount"`
	Status      string  `json:"status"`
	PaymentType string  `json:"payment_type"`
	CreatedAt   string  `json:"created_at"`
	PaidAt      *string `json:"paid_at"`
}
