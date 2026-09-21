package service

import (
	"context"

	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

type AdminService struct {
	storage st.Storage
}

func NewAdminService(storage *st.Storage) *AdminService {
	return &AdminService{storage: *storage}
}

func (s *AdminService) Login(ctx context.Context, req *m.LoginReq) (*m.TokenRes, error) {
	return s.storage.AdminS.Login(req)
}
func (s *AdminService) GetById(ctx context.Context, req *m.ById) (*m.AdminRes, error) {
	return s.storage.AdminS.GetById(req)
}
func (s *AdminService) Create(ctx context.Context, req *m.AdminCreateReq) (*m.ById, error) {
	return s.storage.AdminS.Create(req)
}
func (s *AdminService) GetAll(ctx context.Context, req *m.AdminGetListReq) (*m.PaginationResponse, *m.AdminList, error) {
	return s.storage.AdminS.GetAll(req)
}
func (s *AdminService) Update(ctx context.Context, req *m.AdminUpdateReq) (*m.Void, error) {
	return s.storage.AdminS.Update(req)
}
func (s *AdminService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.AdminS.Delete(req)
}
