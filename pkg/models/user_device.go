package models

// UserDevice - ko'p qurilma push support (users.fcm_token o'rnini bosadi, bir user
// bir nechta qurilmadan kirsa hammasiga push boradi).

type UserDeviceRegisterReq struct {
	UserId   string `json:"user_id"`
	FCMToken string `json:"fcm_token"`
	Platform string `json:"platform"` // ios|android|web
}

type UserDeviceUnregisterReq struct {
	FCMToken string `json:"fcm_token"`
}
