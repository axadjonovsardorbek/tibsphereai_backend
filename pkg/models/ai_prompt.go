package models

// AIPrompt - "AI Promptlari va Model parametrlari" (BACKEND_STRUCTURE.md 7-bo'lim):
// bemor personaji va debriefing promptlarini kodni o'zgartirmasdan admin paneldan sozlash.

const (
	AIPromptPatientPersona = "patient_persona"
	AIPromptDebrief        = "debrief"
	AIPromptCaseGeneration = "case_generation"
)

type AIPromptUpsertReq struct {
	Key         string         `json:"key"`
	Template    string         `json:"template"`
	ModelParams map[string]any `json:"model_params"`
}

type AIPromptRes struct {
	Key         string         `json:"key"`
	Template    string         `json:"template"`
	ModelParams map[string]any `json:"model_params"`
	UpdatedAt   string         `json:"updated_at"`
}

type AIUsageLogCreateReq struct {
	Feature string  `json:"feature"` // chat|debrief|case_generation
	Tokens  int     `json:"tokens"`
	CostUSD float64 `json:"cost_usd"`
	UserId  string  `json:"user_id"`
	CaseId  string  `json:"case_id"`
}

type AIUsageSummaryRes struct {
	TotalTokens int64   `json:"total_tokens"`
	TotalCost   float64 `json:"total_cost_usd"`
}

// AITest* - admin panelda "case yechish"ni sinash: Gemini haqiqiy javob
// berayotganini va har bir chaqiruv qancha token/xarajat sarflayotganini
// darrov ko'rish uchun (real sessiya/foydalanuvchisiz, faqat case bo'yicha).

type AITestPatientReq struct {
	CaseId   string `json:"case_id"`
	Question string `json:"question"`
}

type AITestPatientRes struct {
	// IsAction - true bo'lsa, xabar bemor savoli emas, klinik qaror/harakat
	// bayonoti deb tanildi - shunda Reply o'rniga IsCorrect/HealthDelta/Feedback mazmunli.
	IsAction    bool    `json:"is_action"`
	Reply       string  `json:"reply"`
	IsCorrect   bool    `json:"is_correct"`
	HealthDelta int     `json:"health_delta"`
	Feedback    string  `json:"feedback"`
	Tokens      int     `json:"tokens"`
	CostUSD     float64 `json:"cost_usd"`
}

type AITestDebriefReq struct {
	CaseId string               `json:"case_id"`
	Events []SimulationEventReq `json:"events"`
}

type AITestDebriefRes struct {
	Report  *DebriefReportCreateReq `json:"report"`
	Tokens  int                     `json:"tokens"`
	CostUSD float64                 `json:"cost_usd"`
}
