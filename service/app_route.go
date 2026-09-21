package service

import (
	"context"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"

	"github.com/go-redis/redis"
)

type AppRouteService struct {
	storage st.Storage
	rdb     *redis.Client
}

func NewAppRouteService(storage *st.Storage, rdb *redis.Client) *AppRouteService {
	return &AppRouteService{storage: *storage, rdb: rdb}
}

func (s *AppRouteService) Create(ctx context.Context, req *m.AppRouteCreateReq) (*m.ById, error) {
	return s.storage.AppRouteS.Create(req)
}
func (s *AppRouteService) GetById(ctx context.Context, req *m.ById) (*m.AppRouteRes, error) {
	return s.storage.AppRouteS.GetById(req)
}
func (s *AppRouteService) GetAll(ctx context.Context, req *m.AppRouteGetListReq) (*m.PaginationResponse, *m.AppRouteList, error) {
	return s.storage.AppRouteS.GetAll(req)
}
func (s *AppRouteService) Update(ctx context.Context, req *m.AppRouteUpdateReq) (*m.Void, error) {
	return s.storage.AppRouteS.Update(req)
}
func (s *AppRouteService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.AppRouteS.Delete(req)
}
