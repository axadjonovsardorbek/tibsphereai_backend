package models

type NotificationCreateBody struct {
	Title   map[string]string `json:"title"`
	Message map[string]string `json:"message"`
	Type    string            `json:"type"` // selected | all
	UserId  string            `json:"user_id"`
}

type NotificationCreateReq struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Type    string `json:"type"`
	UserId  string `json:"user_id"` // Type == "selected" bo'lganda
}

type MasterNotificationRes struct {
	Id        *string           `json:"id"`
	Title     map[string]string `json:"title"`
	Message   map[string]string `json:"message"`
	Type      *string           `json:"type"`
	CreatedAt *string           `json:"created_at"`
	UpdatedAt *string           `json:"updated_at"`
}

type MasterNotificationList struct {
	Notifications []*MasterNotificationRes `json:"notifications"`
	Count         int                      `json:"count"`
}

type UserNotificationRes struct {
	Id        *string `json:"id"`
	Title     *string `json:"title"`
	Message   *string `json:"message"`
	IsRead    *bool   `json:"is_read"`
	CreatedAt *string `json:"created_at"`
	UpdatedAt *string `json:"updated_at"`
}

type UserNotificationList struct {
	Notifications []*UserNotificationRes `json:"notifications"`
	Count         int                    `json:"count"`
}

type NotificationGetListReq struct {
	UserId   string     `json:"user_id"`
	Title    string     `json:"title"`
	IsRead   string     `json:"is_read"`
	Language string     `json:"language"`
	Filter   Pagination `json:"filter"`
}

type NotificationMarkReadReq struct {
	Id     string `json:"id"`
	UserId string `json:"user_id"`
}

type MasterNotificationUpdateBody struct {
	Title   map[string]string `json:"title"`
	Message map[string]string `json:"message"`
}

type MasterNotificationUpdateReq struct {
	Id      string `json:"id"`
	Title   string `json:"title"`
	Message string `json:"message"`
}
