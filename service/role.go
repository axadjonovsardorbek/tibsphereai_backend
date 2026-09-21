package service

import (
	"context"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// RoleService - admin panel rollari (Roles boshqaruvi, API_DOCUMENTATION_EN.md 1-bo'lim).
type RoleService struct {
	storage st.Storage
}

func NewRoleService(storage *st.Storage) *RoleService {
	return &RoleService{storage: *storage}
}

func (s *RoleService) Create(ctx context.Context, req *m.RoleCreateReq) (*m.ById, error) {
	return s.storage.RoleS.Create(req)
}
func (s *RoleService) GetById(ctx context.Context, req *m.ById) (*m.RoleRes, error) {
	return s.storage.RoleS.GetById(req)
}
func (s *RoleService) GetAll(ctx context.Context, req *m.RoleGetListReq) (*m.PaginationResponse, *m.RoleList, error) {
	return s.storage.RoleS.GetAll(req)
}
func (s *RoleService) Update(ctx context.Context, req *m.RoleUpdateReq) (*m.Void, error) {
	return s.storage.RoleS.Update(req)
}
func (s *RoleService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.RoleS.Delete(req)
}
