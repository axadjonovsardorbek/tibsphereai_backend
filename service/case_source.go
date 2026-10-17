package service

import (
	"context"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

type CaseSourceService struct {
	storage st.Storage
}

func NewCaseSourceService(storage *st.Storage) *CaseSourceService {
	return &CaseSourceService{storage: *storage}
}

func (s *CaseSourceService) Create(ctx context.Context, req *m.CaseSourceCreateReq) (*m.ById, error) {
	return s.storage.CaseSourceS.Create(req)
}
func (s *CaseSourceService) GetById(ctx context.Context, req *m.ById) (*m.CaseSourceRes, error) {
	return s.storage.CaseSourceS.GetById(req)
}
func (s *CaseSourceService) GetAll(ctx context.Context, req *m.CaseSourceGetListReq) (*m.PaginationResponse, *m.CaseSourceList, error) {
	return s.storage.CaseSourceS.GetAll(req)
}
func (s *CaseSourceService) Update(ctx context.Context, req *m.CaseSourceUpdateReq) (*m.Void, error) {
	return s.storage.CaseSourceS.Update(req)
}
func (s *CaseSourceService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.CaseSourceS.Delete(req)
}
