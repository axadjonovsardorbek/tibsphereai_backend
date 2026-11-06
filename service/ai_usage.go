package service

import (
	"fmt"
	"tibsphereai/internal/ai"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// aiRef - AI chaqiruvi kimga va qaysi sessiyaga tegishli (statistika uchun).
type aiRef struct {
	UserId, CaseId string
	SessionKind    string // m.AISessionSimulation | m.AISessionPractice | ""
	SessionId      string
}

// logAIUsage - Gemini chaqiruvidan qaytgan Usage'ni ai_usage_logs jadvaliga
// yozadi (BACKEND_STRUCTURE.md 7-bo'lim: "Gemini AI xarajatlari va sarflangan
// tokenlar" dashboard'i shu jadvaldan o'qiydi, [[storage/postgres/dashboard.go]]).
// Yozish xatosi butun so'rovni buzmasin uchun faqat log qilinadi.
func logAIUsage(storage *st.Storage, feature string, usage *ai.Usage, ref aiRef) {
	if usage == nil {
		return
	}
	err := storage.AIUsageLogS.Log(&m.AIUsageLogCreateReq{
		Feature:          feature,
		Tokens:           usage.TotalTokens,
		CostUSD:          usage.CostUSD,
		UserId:           ref.UserId,
		CaseId:           ref.CaseId,
		SessionKind:      ref.SessionKind,
		SessionId:        ref.SessionId,
		Model:            usage.Model,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.OutputTokens,
	})
	if err != nil {
		fmt.Println("failed to log AI usage:", err)
	}
}
