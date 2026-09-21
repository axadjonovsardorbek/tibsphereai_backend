package models

type UserActivityCreateReq struct {
	UserId   string `json:"user_id"`
	Activity int64  `json:"activity"`
}

type UserActivityCreateBody struct {
	Activity int64 `json:"activity"`
}

type ActivityStatItem struct {
	Date  string `json:"date"` // day/week/month/range: YYYY-MM-DD, year: YYYY-MM-01
	Value int64  `json:"value"`
}

type ActivityStatsResponse struct {
	UserID string             `json:"user_id"`
	Type   string             `json:"type"` // day|week|month|year|range
	Total  int64              `json:"total"`
	Items  []ActivityStatItem `json:"items"`
}

type UserActivityGetListReq struct {
	UserId   string `json:"user_id"`
	Type     string `json:"type"`
	Date     string `json:"date"`
	FromDate string `json:"from_date"`
	ToDate   string `json:"to_date"`
}

type RatingType string

const (
	RatingTotal RatingType = "total"
	RatingDay   RatingType = "day"
	RatingWeek  RatingType = "week"
	RatingMonth RatingType = "month"
	RatingYear  RatingType = "year"
)

type RatingUserItem struct {
	UserID   string  `json:"user_id"`
	Name     string  `json:"name"`
	ImageUrl *string `json:"image_url"`
	Activity int64   `json:"activity"`
	Rank     int64   `json:"rank"`
	IsMe     bool    `json:"is_me"`
}

type RatingResponse struct {
	Type  RatingType       `json:"type"`
	Limit int              `json:"limit"`
	Items []RatingUserItem `json:"items"`
	Me    *RatingUserItem  `json:"me,omitempty"`
}

type RatingRequest struct {
	Type   RatingType `json:"type"`
	Limit  int        `json:"limit"`
	UserId string     `json:"user_id"`
}
