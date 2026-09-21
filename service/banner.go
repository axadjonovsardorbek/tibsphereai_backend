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

type BannerService struct {
	storage st.Storage
	rdb     *redis.Client
}

func NewBannerService(storage *st.Storage, rdb *redis.Client) *BannerService {
	return &BannerService{storage: *storage, rdb: rdb}
}

func (s *BannerService) Create(ctx context.Context, req *m.BannerCreateReq) (*m.ById, error) {
	return s.storage.BannerS.Create(req)
}

func (s *BannerService) GetById(ctx context.Context, req *m.ById) (*m.BannerRes, error) {
	return s.storage.BannerS.GetById(req)
}

func (s *BannerService) GetAll(ctx context.Context, req *m.BannerGetListReq) (*m.PaginationResponse, *m.BannerList, error) {

	res := &m.PaginationResponse{}

	key := fmt.Sprintf("title:%s_page:%d_limit:%d",
		req.Title, req.Filter.Page, req.Filter.Limit)

	if s.rdb != nil {
		if b, err := s.rdb.Get(key).Bytes(); err == nil && len(b) > 0 {
			_ = json.Unmarshal(b, &res)
		}
	}

	if res.Total == 0 {
		res, _, err := s.storage.BannerS.GetAll(req)
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

func (s *BannerService) Update(ctx context.Context, req *m.BannerUpdateReq) (*m.Void, error) {
	return s.storage.BannerS.Update(req)
}

func (s *BannerService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.BannerS.Delete(req)
}

func (s *BannerService) GetByIdMobile(ctx context.Context, req *m.ByIdMobile) (*m.BannerMobileRes, error) {
	return s.storage.BannerS.GetByIdMobile(req)
}

func (s *BannerService) GetAllMobile(ctx context.Context, req *m.BannerGetListMobileReq) (*m.PaginationResponse, *m.BannerMobileList, error) {
	res := &m.PaginationResponse{}

	key := fmt.Sprintf("banners_mobile_lang:%s",
		req.Language)

	if s.rdb != nil {
		if b, err := s.rdb.Get(key).Bytes(); err == nil && len(b) > 0 {
			_ = json.Unmarshal(b, &res)
		}
	}

	if res.Total == 0 {
		res, _, err := s.storage.BannerS.GetAllMobile(req)
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
