package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type Params struct {
	From     string
	To       string
	Message  string
	HTMLBody string
	Subject  string
}

type LoginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Filter struct {
	Limit int `json:"limit"`
	Page  int `json:"page"`
}

type TokenRes struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Id           string `json:"id"`
	Role         string `json:"role"`
}

type ChangePasswordReq struct {
	Id          string `json:"id"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
	Role        string `json:"role"`
}

type ChangePasswordBody struct {
	OldPassword     string `json:"old_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

type RefreshPasswordReq struct {
	UserId string `json:"user_id"`
	Role   string `json:"role"`
}

type RefreshPassword struct {
	UserId   string `json:"user_id"`
	Password string `json:"password"`
}

type Void struct{}

type ById struct {
	Id string `json:"id"`
}

type ByIdMobile struct {
	Id       string `json:"id"`
	UserId   string `json:"user_id"`
	Language string `json:"language"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name"`
}

func (l Location) Value() (driver.Value, error) {
	return json.Marshal(l)
}

func (l *Location) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to type assert Location to []byte, got %T", value)
	}
	return json.Unmarshal(bytes, l)
}
