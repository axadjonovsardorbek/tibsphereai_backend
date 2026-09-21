package models

type TelegramLoginReq struct {
	Phone string `json:"phone"`
	Name  string `json:"name"`
}

type SendConfirmationCodeReq struct {
	Identifier string `json:"identifier"`
	Type       string `json:"type"` // "email" or "telegram"
}

type ConfirmationReq struct {
	Identifier       string `json:"identifier"`
	ConfirmationCode string `json:"confirmation_code"`
	Type             string `json:"type"` // "email" or "telegram"
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
	IdToken string `json:"id_token"`
}

// GoogleAuthReq - Google tomonidan allaqachon tasdiqlangan claim'lar
// (storage qatlami hech qachon tashqi HTTP chaqirmaydi).
type GoogleAuthReq struct {
	GoogleId string `json:"google_id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
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
