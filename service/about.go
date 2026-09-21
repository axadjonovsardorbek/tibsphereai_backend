package service

import (
	"context"
	"encoding/json"
	"fmt"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
	"time"

	"github.com/go-redis/redis"
)

type AboutService struct {
	storage st.Storage
	rdb     *redis.Client
}

func NewAboutService(storage *st.Storage, rdb *redis.Client) *AboutService {
	return &AboutService{storage: *storage, rdb: rdb}
}

func (s *AboutService) Create(ctx context.Context, req *m.AboutCreateReq) (*m.ById, error) {
	return s.storage.AboutS.Create(req)
}
func (s *AboutService) GetById(ctx context.Context, req *m.ById) (*m.AboutRes, error) {
	return s.storage.AboutS.GetById(req)
}
func (s *AboutService) GetAll(ctx context.Context, req *m.AboutGetListReq) (*m.PaginationResponse, *m.AboutList, error) {
	res := &m.PaginationResponse{}

	key := fmt.Sprintf("about_title:%s_page:%d_limit:%d",
		req.Title, req.Filter.Page, req.Filter.Limit)

	if s.rdb != nil {
		if b, err := s.rdb.Get(key).Bytes(); err == nil && len(b) > 0 {
			_ = json.Unmarshal(b, &res)
		}
	}

	if res.Total == 0 {
		res, _, err := s.storage.AboutS.GetAll(req)
		if err != nil {
			return nil, nil, err
		}

		if s.rdb != nil {
			if b, err := json.Marshal(res); err == nil {
				_ = s.rdb.Set(key, b, 60*time.Second).Err()
			}
		}

		return res, nil, nil
	}

	return res, nil, nil
}
func (s *AboutService) Update(ctx context.Context, req *m.AboutUpdateReq) (*m.Void, error) {
	return s.storage.AboutS.Update(req)
}
func (s *AboutService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.AboutS.Delete(req)
}
func (s *AboutService) GetByIdMobile(ctx context.Context, req *m.ByIdMobile) (*m.AboutMobileRes, error) {
	return s.storage.AboutS.GetByIdMobile(req)
}
func (s *AboutService) GetAllMobile(ctx context.Context, req *m.AboutGetListMobileReq) (*m.PaginationResponse, *m.AboutMobileList, error) {
	res := &m.PaginationResponse{}

	key := fmt.Sprintf("about_title:%s_lang:%s",
		req.Title, req.Language)

	if s.rdb != nil {
		if b, err := s.rdb.Get(key).Bytes(); err == nil && len(b) > 0 {
			_ = json.Unmarshal(b, &res)
		}
	}

	if res.Total == 0 {
		res, _, err := s.storage.AboutS.GetAllMobile(req)
		if err != nil {
			return nil, nil, err
		}

		if s.rdb != nil {
			if b, err := json.Marshal(res); err == nil {
				_ = s.rdb.Set(key, b, 60*time.Second).Err()
			}
		}

		return res, nil, nil
	}

	return res, nil, nil
}
