package service

import (
	"context"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// StudyPlanService - O'quv rejasi va push eslatma vaqti (BACKEND_STRUCTURE.md 2-bo'lim).
// TODO: internal/push + cron orqali belgilangan remind_time'da EventStudyReminder yuborish.
type StudyPlanService struct {
	storage st.Storage
}

func NewStudyPlanService(storage *st.Storage) *StudyPlanService {
	return &StudyPlanService{storage: *storage}
}

func (s *StudyPlanService) Upsert(ctx context.Context, req *m.StudyPlanUpsertReq) (*m.Void, error) {
	return s.storage.StudyPlanS.Upsert(req)
}

func (s *StudyPlanService) GetByUserId(ctx context.Context, userId string) (*m.StudyPlanRes, error) {
	return s.storage.StudyPlanS.GetByUserId(userId)
}
