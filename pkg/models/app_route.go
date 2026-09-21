package models

import (
	"database/sql/driver"
	"fmt"
)

type AppVersionRes struct {
	Android string `json:"android" db:"android" example:"1.0.5"`
	IOS     string `json:"ios" db:"ios" example:"1.0.5"`
}

type AppLinksRes struct {
	Google string `json:"google" db:"google" example:"https://play.google.com/store/apps/details?id=app"`
	Apple  string `json:"apple" db:"apple" example:"https://apps.apple.com/app/id123456"`
}

type AppRouteRes struct {
	Id                string        `json:"id" db:"id" example:"uuid-string"`
	CallCenter        string        `json:"call_center" db:"call_center" example:"+998901234567"`
	SupportUrl        string        `json:"support_url" db:"support_url" example:"https://t.me/support"`
	AppVersion        AppVersionRes `json:"app_version" swaggertype:"object"`
	AppLinks          AppLinksRes   `json:"app_links" swaggertype:"object"`
	PaymentMinVersion string        `json:"payment_min_version" example:"1.0.5"`
	BuyCourse         bool          `json:"buy_course" db:"buy_course" example:"true"`
	CreatedAt         string        `json:"created_at" example:"2026-02-10T10:00:00Z"`
	UpdatedAt         string        `json:"updated_at" example:"2026-02-10T10:00:00Z"`
}

type AppRouteCreateReq struct {
	CallCenter        string        `json:"call_center" db:"call_center" example:"+998901234567"`
	SupportUrl        string        `json:"support_url" db:"support_url" example:"https://t.me/support"`
	AppVersion        AppVersionRes `json:"app_version" swaggertype:"object"`
	AppLinks          AppLinksRes   `json:"app_links" swaggertype:"object"`
	PaymentMinVersion string        `json:"payment_min_version" example:"1.0.5"`
}

type AppRouteUpdateReq struct {
	Id                string        `json:"id" binding:"required"`
	CallCenter        string        `json:"call_center" db:"call_center" example:"+998901234567"`
	SupportUrl        string        `json:"support_url" db:"support_url" example:"https://t.me/support"`
	AppVersion        AppVersionRes `json:"app_version" swaggertype:"object"`
	AppLinks          AppLinksRes   `json:"app_links" swaggertype:"object"`
	PaymentMinVersion string        `json:"payment_min_version" example:"1.0.5"`
	BuyCourse         bool          `json:"buy_course" db:"buy_course" example:"true"`
}

type AppRouteUpdateBody struct {
	CallCenter        string        `json:"call_center" db:"call_center" example:"+998901234567"`
	SupportUrl        string        `json:"support_url" db:"support_url" example:"https://t.me/support"`
	AppVersion        AppVersionRes `json:"app_version" swaggertype:"object"`
	AppLinks          AppLinksRes   `json:"app_links" swaggertype:"object"`
	PaymentMinVersion string        `json:"payment_min_version" example:"1.0.5"`
	BuyCourse         bool          `json:"buy_course" db:"buy_course" example:"true"`
}

type AppRouteGetListReq struct {
}

type AppRouteList struct {
	AppRoutes []*AppRouteRes `json:"app_routes"`
	Count     int            `json:"count"`
}

type JSONRaw []byte

func (j JSONRaw) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte(`{}`), nil
	}
	return j, nil
}

func (j *JSONRaw) UnmarshalJSON(b []byte) error {
	*j = append((*j)[:0], b...)
	return nil
}

// DB -> Go
func (j *JSONRaw) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*j = nil
		return nil
	case []byte:
		*j = append((*j)[:0], v...)
		return nil
	case string:
		*j = append((*j)[:0], v...)
		return nil
	default:
		return fmt.Errorf("JSONRaw: cannot scan %T", src)
	}
}

// Go -> DB
func (j JSONRaw) Value() (driver.Value, error) {
	if len(j) == 0 {
		return []byte(`{}`), nil
	}
	return []byte(j), nil
}
