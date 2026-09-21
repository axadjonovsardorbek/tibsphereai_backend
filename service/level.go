package service

import (
	"context"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

type LevelService struct {
	storage st.Storage
}

func NewLevelService(storage *st.Storage) *LevelService {
	return &LevelService{storage: *storage}
}

func (s *LevelService) Create(ctx context.Context, req *m.LevelCreateReq) (*m.ById, error) {
	return s.storage.LevelS.Create(req)
}
func (s *LevelService) GetAll(ctx context.Context) ([]m.LevelRes, error) {
	return s.storage.LevelS.GetAll()
}
func (s *LevelService) Update(ctx context.Context, req *m.LevelUpdateReq) (*m.Void, error) {
	return s.storage.LevelS.Update(req)
}
func (s *LevelService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.LevelS.Delete(req)
}
