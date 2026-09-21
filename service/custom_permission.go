package service

import (
	"context"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// CustomPermissionService - admin panel modullari + rol-huquq matritsasi (U-Code
// custom-permission patterni, API_DOCUMENTATION_EN.md 2-3-bo'lim). Faqat boshqaruv
// qatlami - haqiqiy ruxsat tekshiruvi hamon Casbin (policy.csv) orqali.
type CustomPermissionService struct {
	storage st.Storage
}

func NewCustomPermissionService(storage *st.Storage) *CustomPermissionService {
	return &CustomPermissionService{storage: *storage}
}

func (s *CustomPermissionService) Create(ctx context.Context, req *m.CustomPermissionCreateReq) (*m.ById, error) {
	return s.storage.CustomPermissionS.Create(req)
}
func (s *CustomPermissionService) Update(ctx context.Context, req *m.CustomPermissionUpdateReq) (*m.Void, error) {
	return s.storage.CustomPermissionS.Update(req)
}
func (s *CustomPermissionService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.CustomPermissionS.Delete(req)
}
func (s *CustomPermissionService) GetRoots(ctx context.Context) (*m.CustomPermissionList, error) {
	return s.storage.CustomPermissionS.GetRoots()
}
func (s *CustomPermissionService) GetChildren(ctx context.Context, parentId string) (*m.CustomPermissionList, error) {
	return s.storage.CustomPermissionS.GetChildren(parentId)
}

// GetAccesses - req.ParentId bo'sh bo'lsa root modullarni, aks holda shu modulning
// bo'lim-modullarini, ikkalasida ham shu rol uchun hozirgi ruxsatlar bilan qaytaradi.
func (s *CustomPermissionService) GetAccesses(ctx context.Context, req *m.RoleAccessGetReq) (*m.RoleAccessList, error) {
	return s.storage.CustomPermissionS.GetAccesses(req.RoleId, req.ParentId)
}

func (s *CustomPermissionService) UpsertAccess(ctx context.Context, req *m.RoleAccessUpsertReq) (*m.Void, error) {
	return s.storage.CustomPermissionS.UpsertAccess(req)
}
