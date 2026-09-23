package service

import (
	"context"
	"fmt"
	"tibsphereai/internal/ai"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// AIPromptService - "AI Promptlari va Model parametrlari" (BACKEND_STRUCTURE.md 7-bo'lim).
type AIPromptService struct {
	storage st.Storage
	ai      *ai.Client
}

func NewAIPromptService(storage *st.Storage, aiClient *ai.Client) *AIPromptService {
	return &AIPromptService{storage: *storage, ai: aiClient}
}

func (s *AIPromptService) GetAll(ctx context.Context) ([]m.AIPromptRes, error) {
	return s.storage.AIPromptS.GetAll()
}
func (s *AIPromptService) Upsert(ctx context.Context, req *m.AIPromptUpsertReq) (*m.Void, error) {
	return s.storage.AIPromptS.Upsert(req)
}
func (s *AIPromptService) GetUsageSummary(ctx context.Context) (*m.AIUsageSummaryRes, error) {
	return s.storage.AIUsageLogS.GetSummary()
}

// TestPatient - admin panel uchun sinov: haqiqiy (yoki mock, GEMINI_API_KEY
// bo'lmasa) bemor AI javobini bir martalik so'rov bilan tekshirish, natija
// bilan birga sarflangan token/xarajatni qaytaradi (ai_usage_logs'ga "chat_test"
// feature bilan yoziladi - dashboard umumiy xarajatga qo'shiladi).
func (s *AIPromptService) TestPatient(ctx context.Context, req *m.AITestPatientReq) (*m.AITestPatientRes, error) {
	caseRes, err := s.storage.CaseS.GetById(&m.ById{Id: req.CaseId})
	if err != nil {
		return nil, err
	}
	if caseRes == nil {
		return nil, fmt.Errorf("case topilmadi: %s", req.CaseId)
	}

	expectedAnswer := ""
	if caseRes.ExpectedAnswer != nil {
		expectedAnswer = *caseRes.ExpectedAnswer
	}

	reply, usage, err := s.ai.Patient.Reply(ctx, caseRes.Scenario, expectedAnswer, req.Question)
	logAIUsage(&s.storage, "chat_test", usage, "", req.CaseId)
	if err != nil {
		return nil, err
	}

	res := &m.AITestPatientRes{
		IsAction:    reply.IsAction,
		Reply:       reply.PatientReply,
		IsCorrect:   reply.IsCorrect,
		HealthDelta: reply.HealthDelta,
		Feedback:    reply.Feedback,
	}
	if usage != nil {
		res.Tokens = usage.TotalTokens
		res.CostUSD = usage.CostUSD
	}
	return res, nil
}

// TestDebrief - admin panel uchun sinov: berilgan case + qadamlar (events)
// bo'yicha debriefing hisobotini bir martalik generatsiya qilib ko'rish.
func (s *AIPromptService) TestDebrief(ctx context.Context, req *m.AITestDebriefReq) (*m.AITestDebriefRes, error) {
	caseRes, err := s.storage.CaseS.GetById(&m.ById{Id: req.CaseId})
	if err != nil {
		return nil, err
	}
	if caseRes == nil {
		return nil, fmt.Errorf("case topilmadi: %s", req.CaseId)
	}

	report, usage, err := s.ai.Debrief.Generate(ctx, caseRes.Scenario, req.Events)
	logAIUsage(&s.storage, "debrief_test", usage, "", req.CaseId)
	if err != nil {
		return nil, err
	}

	res := &m.AITestDebriefRes{Report: report}
	if usage != nil {
		res.Tokens = usage.TotalTokens
		res.CostUSD = usage.CostUSD
	}
	return res, nil
}
