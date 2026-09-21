package service

import (
	"context"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

type FavoriteService struct {
	storage st.Storage
}

func NewFavoriteService(storage *st.Storage) *FavoriteService {
	return &FavoriteService{storage: *storage}
}

func (s *FavoriteService) Toggle(ctx context.Context, req *m.FavoriteToggleReq) (*m.FavoriteToggleRes, error) {
	return s.storage.FavoriteS.Toggle(req)
}

func (s *FavoriteService) GetAll(ctx context.Context, req *m.FavoriteGetListReq) (*m.PaginationResponse, *m.CaseMobileList, error) {
	return s.storage.FavoriteS.GetAll(req)
}
