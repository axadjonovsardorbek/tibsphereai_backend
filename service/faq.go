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

type FaqService struct {
	storage st.Storage
	rdb     *redis.Client
}

func NewFaqService(storage *st.Storage, rdb *redis.Client) *FaqService {
	return &FaqService{storage: *storage, rdb: rdb}
}

func (s *FaqService) Create(ctx context.Context, req *m.FaqCreateReq) (*m.ById, error) {
	return s.storage.FaqS.Create(req)
}
func (s *FaqService) GetById(ctx context.Context, req *m.ById) (*m.FaqRes, error) {
	return s.storage.FaqS.GetById(req)
}
func (s *FaqService) GetAll(ctx context.Context, req *m.FaqGetListReq) (*m.PaginationResponse, *m.FaqList, error) {
	res := &m.PaginationResponse{}

	key := fmt.Sprintf("question:%s_page:%d_limit:%d",
		req.Question, req.Filter.Page, req.Filter.Limit)

	if s.rdb != nil {
		if b, err := s.rdb.Get(key).Bytes(); err == nil && len(b) > 0 {
			_ = json.Unmarshal(b, &res)
		}
	}

	if res.Total == 0 {
		res, _, err := s.storage.FaqS.GetAll(req)
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
func (s *FaqService) Update(ctx context.Context, req *m.FaqUpdateReq) (*m.Void, error) {
	return s.storage.FaqS.Update(req)
}
func (s *FaqService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.FaqS.Delete(req)
}
func (s *FaqService) GetByIdMobile(ctx context.Context, req *m.ByIdMobile) (*m.FaqMobileRes, error) {
	return s.storage.FaqS.GetByIdMobile(req)
}
func (s *FaqService) GetAllMobile(ctx context.Context, req *m.FaqGetListMobileReq) (*m.PaginationResponse, *m.FaqMobileList, error) {
	res := &m.PaginationResponse{}

	key := fmt.Sprintf("question:%s_lang:%s",
		req.Question, req.Language)

	if s.rdb != nil {
		if b, err := s.rdb.Get(key).Bytes(); err == nil && len(b) > 0 {
			_ = json.Unmarshal(b, &res)
		}
	}

	if res.Total == 0 {
		res, _, err := s.storage.FaqS.GetAllMobile(req)
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
