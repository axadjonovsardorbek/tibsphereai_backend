package service

import (
	"context"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

type CategoryService struct {
	storage st.Storage
}

func NewCategoryService(storage *st.Storage) *CategoryService {
	return &CategoryService{storage: *storage}
}

func (s *CategoryService) Create(ctx context.Context, req *m.CategoryCreateReq) (*m.ById, error) {
	return s.storage.CategoryS.Create(req)
}
func (s *CategoryService) GetById(ctx context.Context, req *m.ById) (*m.CategoryRes, error) {
	return s.storage.CategoryS.GetById(req)
}
func (s *CategoryService) GetAll(ctx context.Context, req *m.CategoryGetListReq) (*m.PaginationResponse, *m.CategoryList, error) {
	return s.storage.CategoryS.GetAll(req)
}
func (s *CategoryService) Update(ctx context.Context, req *m.CategoryUpdateReq) (*m.Void, error) {
	return s.storage.CategoryS.Update(req)
}
func (s *CategoryService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.CategoryS.Delete(req)
}
func (s *CategoryService) GetByIdMobile(ctx context.Context, req *m.ByIdMobile) (*m.CategoryMobileRes, error) {
	return s.storage.CategoryS.GetByIdMobile(req)
}
func (s *CategoryService) GetAllMobile(ctx context.Context, req *m.CategoryGetListMobileReq) (*m.PaginationResponse, *m.CategoryMobileList, error) {
	return s.storage.CategoryS.GetAllMobile(req)
}
