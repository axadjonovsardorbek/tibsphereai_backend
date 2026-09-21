package service

import (
	"context"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

type PartnerService struct {
	storage st.Storage
}

func NewPartnerService(storage *st.Storage) *PartnerService {
	return &PartnerService{storage: *storage}
}

func (s *PartnerService) Create(ctx context.Context, req *m.PartnerCreateReq) (*m.ById, error) {
	return s.storage.PartnerS.Create(req)
}
func (s *PartnerService) GetById(ctx context.Context, req *m.ById) (*m.PartnerRes, error) {
	return s.storage.PartnerS.GetById(req)
}
func (s *PartnerService) GetAll(ctx context.Context, req *m.PartnerGetListReq) (*m.PaginationResponse, *m.PartnerList, error) {
	return s.storage.PartnerS.GetAll(req)
}
func (s *PartnerService) Update(ctx context.Context, req *m.PartnerUpdateReq) (*m.Void, error) {
	return s.storage.PartnerS.Update(req)
}
func (s *PartnerService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.PartnerS.Delete(req)
}
func (s *PartnerService) GetAllMobile(ctx context.Context) (*m.PartnerMobileList, error) {
	return s.storage.PartnerS.GetAllMobile()
}
