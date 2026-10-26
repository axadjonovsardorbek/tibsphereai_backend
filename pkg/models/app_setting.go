package models

// AppSetting - admin panel orqali o'zgartiriladigan dinamik parametrlar
// (BACKEND_STRUCTURE.md 6,7-bo'lim: kunlik bepul limitni admin o'zgartira oladi).

type AppSettingSetReq struct {
	Key   string `json:"key"`
	Value string `json:"value"` // JSON string sifatida saqlanadi
}

type AppSettingRes struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt string `json:"updated_at"`
}

const AppSettingDailyFreeLimit = "daily_free_limit"

// AppSettingCoinsPer1kTokens - token-to-coin tarifi: 1 000 AI token (prompt+completion)
// necha tangaga teng (kasr son mumkin, masalan 0.5). Admin o'zgartiradi, keyingi
// hisob-kitoblarda darhol kuchga kiradi. 0 - tanga yechilmaydi.
const AppSettingCoinsPer1kTokens = "coins_per_1k_tokens"

// CoinRateRes - admin panelning "Token narxi" formasi uchun.
type CoinRateRes struct {
	CoinsPer1kTokens float64 `json:"coins_per_1k_tokens"`
}

// AppSettingVoiceEnabled - "true"/"false", platforma ovozli funksiyasi (shikoyat/chat
// matnini frontend TTS orqali o'qib berish) global yoqilgan/o'chirilganini bildiradi.
const AppSettingVoiceEnabled = "voice_enabled"

// VoiceSettingRes - mobil ilova ovoz tugmasini ko'rsatish/yashirish uchun.
type VoiceSettingRes struct {
	Enabled bool `json:"enabled"`
}

// UserLimitStatusRes - home ekranidagi "35/50 Limit" belgisi uchun
// (BACKEND_STRUCTURE.md 6-bo'lim: kunlik bepul urinishlar).
type UserLimitStatusRes struct {
	Used            int  `json:"used"`
	Total           int  `json:"total"`
	Remaining       int  `json:"remaining"`
	HasSubscription bool `json:"has_subscription"` // true - limit cheksiz (Premium)
	// CoinsPer1kTokens - limit tugagach case tanga hisobidan boshlanadi: har 1 000 AI token
	// shu miqdor tangaga teng (haqiqiy sarf sessiya davomida yechiladi).
	CoinsPer1kTokens float64 `json:"coins_per_1k_tokens"`
}

// --- Case qiyinlik darajasi (difficulty) koeffitsienti (admin sozlaydi) ---
//
// Avval xp=health_percent, coins=health_percent/10 edi (faqat sog'liqqa qarab,
// case qanchalik qiyin bo'lishidan qat'i nazar bir xil). Endi bazaviy XP/Coin
// foydalanuvchining O'Z DARAJASI (levels.xp_reward/coin_reward, [[LevelRes]]) bo'yicha
// olinadi, case "medium"/"hard" bo'lsa esa shu bazaga koeffitsient bilan ko'paytiriladi
// (masalan medium=1.5x, hard=2x) - admin panelda "Sozlamalar" -> "Mukofot sozlamalari"
// orqali boshqariladi. Yakuniy mukofot bundan tashqari health_percent'ga (qobiliyat/
// to'g'ri bajarish darajasi) mutanosib beriladi. Ko'ring [[SimulationService.Finish]].
const (
	AppSettingMediumRewardMultiplier = "medium_reward_multiplier" // masalan "1.5"
	AppSettingHardRewardMultiplier   = "hard_reward_multiplier"   // masalan "2"
)

// DifficultyRewardRes - admin panelning "Mukofot sozlamalari" formasi uchun,
// joriy koeffitsientlar bir so'rovda ("easy" har doim 1x, bazaviy qiymatning o'zi).
type DifficultyRewardRes struct {
	MediumMultiplier float64 `json:"medium_multiplier"`
	HardMultiplier   float64 `json:"hard_multiplier"`
}

// AppSettingDailyFreeCoins - har 24 soatda har bir foydalanuvchiga beriladigan bepul tangalar
// (to'planmaydi, har kuni yangidan). Eski daily_free_limit (keys soni) faqat eski simulyatsiya
// moduli uchun qoladi.
const AppSettingDailyFreeCoins = "daily_free_coins"

// DailyFreeCoinsRes - admin panel "Kunlik bepul tangalar" formasi uchun.
type DailyFreeCoinsRes struct {
	DailyFreeCoins int `json:"daily_free_coins"`
}
