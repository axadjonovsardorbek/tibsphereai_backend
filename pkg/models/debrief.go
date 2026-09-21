package models

// Debriefing (BACKEND_STRUCTURE.md 5-bo'lim) - AI orqali generatsiya qilinadi (internal/ai.DebriefAI, hozircha stub).

type DebriefReportCreateReq struct {
	SessionId      string   `json:"session_id"`
	CorrectSteps   []string `json:"correct_steps"`
	IncorrectSteps []string `json:"incorrect_steps"`
	GuidelineNotes string   `json:"guideline_notes"`
	WeakTopics     []string `json:"weak_topics"`
}

type DebriefReportRes struct {
	SessionId      *string  `json:"session_id"`
	CorrectSteps   []string `json:"correct_steps"`
	IncorrectSteps []string `json:"incorrect_steps"`
	FinalScore     *int     `json:"final_score"`
	XPEarned       *int     `json:"xp_earned"`
	CoinsEarned    *int     `json:"coins_earned"`
	GuidelineNotes *string  `json:"guideline_notes"`
	WeakTopics     []string `json:"weak_topics"`
	CreatedAt      *string  `json:"created_at"`
}
