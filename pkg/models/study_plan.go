package models

// StudyPlan - O'quv rejasi va push bildirishnoma vaqti (BACKEND_STRUCTURE.md 2-bo'lim).

type StudyPlanUpsertReq struct {
	UserId              string `json:"user_id"`
	RemindTime          string `json:"remind_time"` // "20:00"
	RemindMinutesBefore int    `json:"remind_minutes_before"`
	IsEnabled           bool   `json:"is_enabled"`
}

type StudyPlanRes struct {
	UserId              *string `json:"user_id"`
	RemindTime          *string `json:"remind_time"`
	RemindMinutesBefore *int    `json:"remind_minutes_before"`
	IsEnabled           *bool   `json:"is_enabled"`
}
