package models

type FavoriteToggleReq struct {
	UserId string `json:"user_id"`
	CaseId string `json:"case_id"`
}

type FavoriteToggleRes struct {
	IsFavorite bool `json:"is_favorite"`
}

type FavoriteGetListReq struct {
	UserId   string     `json:"user_id"`
	Language string     `json:"language"`
	Filter   Pagination `json:"filter"`
}
