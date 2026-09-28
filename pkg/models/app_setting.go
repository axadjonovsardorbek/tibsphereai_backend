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
}

// --- Case qiyinlik darajasi (difficulty) bo'yicha XP/Coin mukofoti (admin sozlaydi) ---
//
// Avval xp=health_percent, coins=health_percent/10 edi (faqat sog'liqqa qarab,
// case qanchalik qiyin bo'lishidan qat'i nazar bir xil). Endi har bir qiyinlik
// darajasi uchun admin panelda ("Sozlamalar" -> "Mukofot sozlamalari") o'zining
// bazaviy XP/Coin qiymati belgilanadi, yakuniy mukofot esa health_percent'ga
// mutanosib beriladi: xp = round(baseXP * health_percent / 100) va h.k.
// Ko'ring [[AppSettingService.DifficultyReward]].
const (
	AppSettingXPRewardEasy   = "xp_reward_easy"
	AppSettingXPRewardMedium = "xp_reward_medium"
	AppSettingXPRewardHard   = "xp_reward_hard"

	AppSettingCoinRewardEasy   = "coin_reward_easy"
	AppSettingCoinRewardMedium = "coin_reward_medium"
	AppSettingCoinRewardHard   = "coin_reward_hard"
)

// DifficultyRewardRes - admin panelning "Mukofot sozlamalari" formasi uchun,
// uchala qiyinlik darajasining joriy bazaviy XP/Coin qiymatlari bir so'rovda.
type DifficultyRewardRes struct {
	XPEasy     int `json:"xp_easy"`
	CoinEasy   int `json:"coin_easy"`
	XPMedium   int `json:"xp_medium"`
	CoinMedium int `json:"coin_medium"`
	XPHard     int `json:"xp_hard"`
	CoinHard   int `json:"coin_hard"`
}
