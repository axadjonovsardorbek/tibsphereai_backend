package service

import (
	"fmt"
	"tibsphereai/internal/ai"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// logAIUsage - Gemini chaqiruvidan qaytgan Usage'ni ai_usage_logs jadvaliga
// yozadi (BACKEND_STRUCTURE.md 7-bo'lim: "Gemini AI xarajatlari va sarflangan
// tokenlar" dashboard'i shu jadvaldan o'qiydi, [[storage/postgres/dashboard.go]]).
// Yozish xatosi butun so'rovni buzmasin uchun faqat log qilinadi.
func logAIUsage(storage *st.Storage, feature string, usage *ai.Usage, userId, caseId string) {
	if usage == nil {
		return
	}
	err := storage.AIUsageLogS.Log(&m.AIUsageLogCreateReq{
		Feature: feature,
		Tokens:  usage.TotalTokens,
		CostUSD: usage.CostUSD,
		UserId:  userId,
		CaseId:  caseId,
	})
	if err != nil {
		fmt.Println("failed to log AI usage:", err)
	}
}
