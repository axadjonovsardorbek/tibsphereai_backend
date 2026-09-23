// Package ai - Gemini AI integratsiya qatlami (BACKEND_STRUCTURE.md 4,5,7-bo'lim):
// bemor personaji bilan muloqot, debriefing hisoboti va admin uchun avtomatik
// keys generatsiyasi. GEMINI_API_KEY bo'sh bo'lganda mock implementatsiya
// ishlatiladi - shu tufayli service/handler qatlamlari haqiqiy kalit kelguncha
// ham to'liq ishlaydi (deterministik, lekin real bo'lmagan javoblar bilan).
package ai

import (
	"context"
	"tibsphereai/internal/config"
	"tibsphereai/pkg/models"
	"tibsphereai/storage"
)

// Usage - bitta Gemini chaqiruvi uchun sarflangan token/xarajat (dashboard va
// ai_usage_logs uchun, BACKEND_STRUCTURE.md 7-bo'lim: "Gemini AI xarajatlari
// va sarflangan tokenlar").
type Usage struct {
	PromptTokens int
	OutputTokens int
	TotalTokens  int
	CostUSD      float64
}

// PatientAI - Anamnez (BACKEND_STRUCTURE.md 4-bo'lim): shifokor savoliga
// bemor personajida klinik to'g'ri javob qaytaradi.
type PatientAI interface {
	Reply(ctx context.Context, caseScenario map[string]any, question string) (string, *Usage, error)
}

// DebriefAI - Yakuniy hisobot (BACKEND_STRUCTURE.md 5-bo'lim): qadamlar logi
// asosida tahliliy debriefing tayyorlaydi.
type DebriefAI interface {
	Generate(ctx context.Context, caseScenario map[string]any, events []models.SimulationEventReq) (*models.DebriefReportCreateReq, *Usage, error)
}

// CaseGeneratorAI - Admin uchun avtomatik keys yaratish (BACKEND_STRUCTURE.md 7-bo'lim).
type CaseGeneratorAI interface {
	Generate(ctx context.Context, req *models.CaseGenReq) (*models.CaseCreateReq, *Usage, error)
}

// ActionEvalRes - shifokorning bitta klinik harakati (ko'rik/tahlil/instrumental/
// dori/protsedura - "question" bundan mustasno, u alohida PatientAI orqali
// suhbat sifatida ishlaydi) qanchalik to'g'ri bajarilganini bildiradi.
type ActionEvalRes struct {
	IsCorrect   bool
	HealthDelta int // -100..+100 (manfiy - noto'g'ri/kechiktirilgan harakat, ijobiy - to'g'ri)
	Feedback    string
}

// ActionEvaluatorAI - "question" bo'lmagan qadamlarni (BACKEND_STRUCTURE.md
// 4-bo'lim: fizikal ko'rik, diagnostika, muolaja) case scenario/expected_steps
// asosida real vaqtda baholaydi - shu orqali Sog'liq shkalasi va Ball haqiqiy
// klinik to'g'rilikka mos ravishda o'zgaradi (avval har doim health_delta=0 edi).
type ActionEvaluatorAI interface {
	Evaluate(ctx context.Context, caseScenario map[string]any, eventType string, payload map[string]any) (*ActionEvalRes, *Usage, error)
}

// Client - AI xizmatlarini bitta joyda ushlab turadi.
type Client struct {
	Patient PatientAI
	Debrief DebriefAI
	CaseGen CaseGeneratorAI
	Action  ActionEvaluatorAI
	Enabled bool // haqiqiy Gemini API ulanganmi (GEMINI_API_KEY bor)
}

// NewClient - GEMINI_API_KEY bo'sh bo'lsa mock, aks holda haqiqiy Gemini
// klientini qaytaradi. promptRepo - admin panelda tahrirlangan promptlarni
// (ai_prompts jadvali) o'qish uchun, kodda qattiq yozilgan default'larni
// GEMINI_API_KEY bo'lsa ham o'zgartirmasdan ishlatish mumkin bo'lishi uchun.
func NewClient(cfg *config.Config, promptRepo storage.AIPromptI) *Client {
	if cfg.GEMINI_API_KEY == "" {
		m := &mockClient{}
		return &Client{Patient: m, Debrief: m, Action: m, CaseGen: mockCaseGen{}, Enabled: false}
	}

	g := newGeminiClient(cfg, promptRepo)
	return &Client{Patient: g, Debrief: g, Action: g, CaseGen: geminiCaseGen{g}, Enabled: true}
}
