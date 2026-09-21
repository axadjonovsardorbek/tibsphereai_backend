package service

import (
	"context"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// DebriefService - hisobot SimulationService.Finish ichida AI (internal/ai.DebriefAI)
// yordamida darhol generatsiya qilinadi va saqlanadi; bu yerda faqat o'qiladi.
type DebriefService struct {
	storage st.Storage
}

func NewDebriefService(storage *st.Storage) *DebriefService {
	return &DebriefService{storage: *storage}
}

func (s *DebriefService) GetBySessionId(ctx context.Context, sessionId string) (*m.DebriefReportRes, error) {
	return s.storage.DebriefS.GetBySessionId(sessionId)
}
