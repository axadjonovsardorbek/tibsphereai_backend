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
