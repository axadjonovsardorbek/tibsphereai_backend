package models

type TelegramLoginReq struct {
	Phone string `json:"phone"`
	Name  string `json:"name"`
	// ReferralCode - ixtiyoriy, faqat yangi user yaratilayotganda ishlatiladi
	// (mavjud user login qilayotganda e'tiborga olinmaydi). Ko'ring [[ReferralInfoRes]].
	ReferralCode string `json:"referral_code"`
}

type SendConfirmationCodeReq struct {
	Identifier string `json:"identifier"`
	Type       string `json:"type"` // "email" or "telegram"
}

type ConfirmationReq struct {
	Identifier       string `json:"identifier"`
	ConfirmationCode string `json:"confirmation_code"`
	Type             string `json:"type"` // "email" or "telegram"
	// ReferralCode - ixtiyoriy, faqat yangi user yaratilayotganda ishlatiladi.
	ReferralCode string `json:"referral_code"`
}
type UserCheckReq struct {
	Identifier string `json:"identifier"`
}

type UserCheckRes struct {
	HasAccount bool `json:"has_account"`
}

// GoogleLoginReq - "1 bosqichli kirish" (BACKEND_STRUCTURE.md 1-bo'lim).
// Handler shu raw id_token'ni service qatlamiga uzatadi, service Google
// tokeninfo/idtoken orqali tekshirib GoogleAuthReq'ga aylantiradi.
type GoogleLoginReq struct {
	IdToken      string `json:"id_token"`
	ReferralCode string `json:"referral_code"`
}

// GoogleAuthReq - Google tomonidan allaqachon tasdiqlangan claim'lar
// (storage qatlami hech qachon tashqi HTTP chaqirmaydi).
type GoogleAuthReq struct {
	GoogleId     string `json:"google_id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	ReferralCode string `json:"referral_code"`
}

// ReferralInfoRes - "do'stlarni taklif qilish" ekrani uchun (mobile, profil bo'limi).
type ReferralInfoRes struct {
	ReferralCode     string                `json:"referral_code"`
	InvitedCount     int                   `json:"invited_count"`
	TotalCoinsEarned int                   `json:"total_coins_earned"`
	ReferrerReward   int                   `json:"referrer_reward"` // 1ta taklif uchun necha tanga beriladi (app_settings)
	ReferredReward   int                   `json:"referred_reward"` // taklif qilinganga necha tanga beriladi
	Invited          []ReferralInvitedItem `json:"invited"`
}

type ReferralInvitedItem struct {
	Name       *string `json:"name"`
	JoinedAt   string  `json:"joined_at"`
	CoinsGiven int     `json:"coins_given"`
}

type UserRes struct {
	Id             string  `json:"id"`
	Name           *string `json:"name"`
	PhoneNumber    *string `json:"phone_number"`
	Email          *string `json:"email"`
	ImageUrl       *string `json:"image_url"`
	Language       *string `json:"language"`
	Specialization *string `json:"specialization"` // student|resident|doctor
	Level          *int    `json:"level"`
	XP             *int    `json:"xp"`
	Coins          *int    `json:"coins"`
	StreakCount    *int    `json:"streak_count"`
	CreatedAt      *string `json:"created_at"`
	UpdatedAt      *string `json:"updated_at"`
}

type UserUpdateReq struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	PhoneNumber    string `json:"phone_number"`
	Email          string `json:"email"`
	ImageUrl       string `json:"image_url"`
	Language       string `json:"language"`
	Specialization string `json:"specialization"`
}

type UserUpdateBody struct {
	Name           string `json:"name"`
	PhoneNumber    string `json:"phone_number"`
	Email          string `json:"email"`
	ImageUrl       string `json:"image_url"`
	Language       string `json:"language"`
	Specialization string `json:"specialization"`
}

type UserGetListReq struct {
	Filter      Pagination `json:"filter"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	PhoneNumber string     `json:"phone_number"`
}

type UserList struct {
	Users []UserRes `json:"users"`
	Count int       `json:"count"`
}
