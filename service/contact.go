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

type ContactService struct {
	storage st.Storage
	rdb     *redis.Client
}

func NewContactService(storage *st.Storage, rdb *redis.Client) *ContactService {
	return &ContactService{storage: *storage, rdb: rdb}
}

func (s *ContactService) Create(ctx context.Context, req *m.ContactCreateReq) (*m.ById, error) {
	return s.storage.ContactS.Create(req)
}

func (s *ContactService) GetById(ctx context.Context, req *m.ById) (*m.ContactRes, error) {
	return s.storage.ContactS.GetById(req)
}

func (s *ContactService) GetAll(ctx context.Context, req *m.ContactGetListReq) (*m.PaginationResponse, *m.ContactList, error) {
	res := &m.PaginationResponse{}

	key := fmt.Sprintf("name:%s_phone:%s_page:%d_limit:%d",
		req.Name, req.PhoneNumber, req.Filter.Page, req.Filter.Limit)

	if s.rdb != nil {
		if b, err := s.rdb.Get(key).Bytes(); err == nil && len(b) > 0 {
			_ = json.Unmarshal(b, &res)
		}
	}

	if res.Total == 0 {
		res, _, err := s.storage.ContactS.GetAll(req)
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

func (s *ContactService) Update(ctx context.Context, req *m.ContactUpdateReq) (*m.Void, error) {
	return s.storage.ContactS.Update(req)
}

func (s *ContactService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.ContactS.Delete(req)
}
