package models

type DashboardRes struct {
	Categories          int     `json:"categories"`
	Cases               int     `json:"cases"`
	Users               int     `json:"users"`
	ActiveUsers         int     `json:"active_users"`
	ActiveSubscriptions int     `json:"active_subscriptions"`
	CompletedSessions   int     `json:"completed_sessions"`
	AITotalCostUSD      float64 `json:"ai_total_cost_usd"`
}

type GetDashboardReq struct {
	Type string `json:"type"` //range, day, week, month, year
	Day  string `json:"day"`  //2024-06-01
	From string `json:"from"` //2024-06-01
	To   string `json:"to"`   //2024-06-30
}
