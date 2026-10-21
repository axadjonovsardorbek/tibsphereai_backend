package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"tibsphereai/internal/ai"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// DebriefService - hisobot odatda SimulationService.Finish ichida AI (internal/ai.DebriefAI)
// yordamida darhol generatsiya qilinadi va saqlanadi. Agar sessiya yopilgan bo'lsa-yu
// hisobot yo'q bo'lsa (Finish paytida Gemini xato bergan, eski sessiya va h.k.),
// o'qish paytida bir marta qayta generatsiya qilinadi (self-heal).
type DebriefService struct {
	storage  st.Storage
	ai       *ai.Client
	inflight sync.Map // sessionId -> struct{}: bir vaqtda bitta generatsiya
}

func NewDebriefService(storage *st.Storage, aiClient *ai.Client) *DebriefService {
	return &DebriefService{storage: *storage, ai: aiClient}
}

func (s *DebriefService) GetBySessionId(ctx context.Context, sessionId string) (*m.DebriefReportRes, error) {
	res, err := s.storage.DebriefS.GetBySessionId(sessionId)
	if err == nil || !errors.Is(err, st.ErrDebriefNotFound) {
		return res, err
	}

	// Hisobot yo'q: sabab sessiya holatiga bog'liq.
	session, sErr := s.storage.SimulationS.GetById(&m.ById{Id: sessionId})
	if sErr != nil || session == nil {
		return nil, errs.ErrSessionNotFound
	}
	if session.Status == nil || *session.Status == "active" {
		return nil, errs.ErrDebriefNotReady
	}

	// Sessiya yopilgan, hisobot yo'q - qayta generatsiya qilib ko'ramiz.
	if _, loaded := s.inflight.LoadOrStore(sessionId, struct{}{}); loaded {
		return nil, errs.ErrDebriefNotReady // boshqa so'rov hozir generatsiya qilmoqda
	}
	defer s.inflight.Delete(sessionId)

	if gErr := s.regenerate(ctx, sessionId, session); gErr != nil {
		fmt.Println("debrief regeneration failed:", sessionId, gErr)
		return nil, errs.ErrDebriefFailed
	}
	return s.storage.DebriefS.GetBySessionId(sessionId)
}

func (s *DebriefService) regenerate(ctx context.Context, sessionId string, session *m.SimulationSessionRes) error {
	if s.ai == nil || s.ai.Debrief == nil || session.CaseId == nil {
		return errors.New("AI debrief is not configured")
	}
	caseRes, err := s.storage.CaseS.GetById(&m.ById{Id: *session.CaseId})
	if err != nil || caseRes == nil {
		return fmt.Errorf("case not found: %v", err)
	}
	events, err := s.storage.SimulationS.GetEvents(sessionId)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return errors.New("session has no events")
	}
	d, usage, err := s.ai.Debrief.Generate(ctx, caseRes.Scenario, events)
	userId := ""
	if session.UserId != nil {
		userId = *session.UserId
	}
	logAIUsage(&s.storage, "debrief", usage, userId, *session.CaseId)
	if err != nil {
		return err
	}
	d.SessionId = sessionId
	_, err = s.storage.DebriefS.Create(d)
	return err
}
