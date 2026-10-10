package models

// Simulation Engine (BACKEND_STRUCTURE.md 4-bo'lim)

type SimulationStartReq struct {
	UserId string `json:"user_id"`
	CaseId string `json:"case_id"`
	// DurationMinutes - home ekranda tanlangan davomiylik (masalan 5/10/15),
	// berilsa case'ning o'z expected_duration_minutes'ini bosib o'tadi.
	DurationMinutes int `json:"duration_minutes"`
	// UseCoins - bepul limit tugagan bo'lsa case'ni tanga evaziga boshlashga rozilik
	// (narx app_settings.case_coin_cost). false bo'lsa tanga hech qachon yechilmaydi.
	UseCoins bool `json:"use_coins"`
}

type SimulationStartRes struct {
	SessionId       string     `json:"session_id"`
	CaseId          string     `json:"case_id"`
	HealthPercent   int        `json:"health_percent"`
	TimeLimitSecond int        `json:"time_limit_seconds"`
	InitialVitals   CaseVitals `json:"initial_vitals"`
	VisualState     string     `json:"visual_state"`
	WSUrl           string     `json:"ws_url"`
}

// SimulationEventReq - shifokorning bitta harakati: savol/ko'rik/tahlil/dori/protsedura.
// TODO: PatientAI/DebriefAI (internal/ai) orqali Gemini javobini shu yerga bog'lash.
type SimulationEventReq struct {
	SessionId string         `json:"session_id"`
	Type      string         `json:"type"` // question|exam|lab|imaging|medication|procedure
	Payload   map[string]any `json:"payload"`
}

type SimulationEventRes struct {
	IsCorrect     bool           `json:"is_correct"`
	HealthDelta   int            `json:"health_delta"`
	HealthPercent int            `json:"health_percent"`
	Response      map[string]any `json:"response"` // AI javobi / lab natijasi va h.k.
	// SessionEnded - health_percent shu harakatdan keyin 0'ga tushib, sessiya
	// server tomonidan avtomatik yakunlangan bo'lsa true (bemor "o'ldi").
	// Frontend shu bayroqni ko'rsa, keyingi buyruq/xabar yuborishni to'xtatib,
	// FinishResult'dagi debrief/XP/coin natijasini ko'rsatishi kerak.
	SessionEnded bool                 `json:"session_ended"`
	FinishResult *SimulationFinishRes `json:"finish_result,omitempty"`
}

type SimulationFinishReq struct {
	SessionId string `json:"session_id"`
	Reason    string `json:"reason"` // manual|health_zero|timeout
}

type SimulationFinishRes struct {
	SessionId    string `json:"session_id"`
	FinalScore   int    `json:"final_score"`
	XPEarned     int    `json:"xp_earned"`
	CoinsEarned  int    `json:"coins_earned"`
	DebriefReady bool   `json:"debrief_ready"`
}

type SimulationSessionRes struct {
	Id            *string `json:"id"`
	UserId        *string `json:"user_id"`
	CaseId        *string `json:"case_id"`
	CaseTitle     *string `json:"case_title"`
	Status        *string `json:"status"`
	HealthPercent *int    `json:"health_percent"`
	FinalScore    *int    `json:"final_score"`
	XPEarned      *int    `json:"xp_earned"`
	CoinsEarned   *int    `json:"coins_earned"`
	StartedAt     *string `json:"started_at"`
	EndedAt       *string `json:"ended_at"`
}

type SimulationSessionList struct {
	Sessions []*SimulationSessionRes `json:"sessions"`
	Count    int                     `json:"count"`
}
