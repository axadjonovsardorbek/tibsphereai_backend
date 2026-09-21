package service

import (
	"context"
	"tibsphereai/internal/ai"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

type CaseService struct {
	storage st.Storage
	ai      *ai.Client
}

func NewCaseService(storage *st.Storage, aiClient *ai.Client) *CaseService {
	return &CaseService{storage: *storage, ai: aiClient}
}

func (s *CaseService) Create(ctx context.Context, req *m.CaseCreateReq) (*m.ById, error) {
	return s.storage.CaseS.Create(req)
}

// GenerateWithAI - "AI yordamida avtomatik keys yaratish" (BACKEND_STRUCTURE.md 7-bo'lim):
// admin mavzu va darajani kiritadi, Gemini AI (yoki GEMINI_API_KEY yo'q bo'lsa xatolik)
// to'liq ssenariyni tayyorlaydi, admin ko'rib chiqib alohida Publish bilan tasdiqlaydi.
func (s *CaseService) GenerateWithAI(ctx context.Context, topic, difficulty string) (*m.ById, error) {
	req, usage, err := s.ai.CaseGen.Generate(ctx, topic, difficulty)
	logAIUsage(&s.storage, "case_generation", usage, "", "")
	if err != nil {
		return nil, err
	}
	req.IsAIGenerated = true
	return s.storage.CaseS.Create(req)
}

func (s *CaseService) GetById(ctx context.Context, req *m.ById) (*m.CaseRes, error) {
	return s.storage.CaseS.GetById(req)
}
func (s *CaseService) GetAll(ctx context.Context, req *m.CaseGetListReq) (*m.PaginationResponse, *m.CaseList, error) {
	return s.storage.CaseS.GetAll(req)
}
func (s *CaseService) Update(ctx context.Context, req *m.CaseUpdateReq) (*m.Void, error) {
	return s.storage.CaseS.Update(req)
}
func (s *CaseService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.CaseS.Delete(req)
}
func (s *CaseService) Publish(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.CaseS.Publish(req)
}
func (s *CaseService) GetByIdMobile(ctx context.Context, req *m.ByIdMobile) (*m.CaseMobileRes, error) {
	return s.storage.CaseS.GetByIdMobile(req)
}
func (s *CaseService) GetAllMobile(ctx context.Context, req *m.CaseGetListMobileReq) (*m.PaginationResponse, *m.CaseMobileList, error) {
	return s.storage.CaseS.GetAllMobile(req)
}
func (s *CaseService) GetRandom(ctx context.Context, req *m.CaseRandomReq) (*m.CaseMobileRes, error) {
	return s.storage.CaseS.GetRandom(req)
}
