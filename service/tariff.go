package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
	"time"

	"github.com/go-redis/redis"
)

type TariffService struct {
	storage st.Storage
	rdb     *redis.Client
}

func NewTariffService(storage *st.Storage, rdb *redis.Client) *TariffService {
	return &TariffService{storage: *storage, rdb: rdb}
}

// validateTariffKind - kind'ga qarab faqat tegishli maydon tekshiriladi:
// subscription uchun duration (> 0), coin_package uchun coins (> 0).
// Ikkinchisi e'tiborga olinmaydi.
func validateTariffKind(kind string, duration, coins int) error {
	switch kind {
	case "subscription":
		if duration <= 0 {
			return errors.New("duration must be greater than 0 for subscription")
		}
	case "coin_package":
		if coins <= 0 {
			return errors.New("coins must be greater than 0 for coin_package")
		}
	default:
		return fmt.Errorf("invalid kind %q (subscription|coin_package)", kind)
	}
	return nil
}

func (s *TariffService) Create(ctx context.Context, req *m.TariffCreateReq) (*m.ById, error) {
	if req.Kind == "" {
		req.Kind = "subscription"
	}
	if err := validateTariffKind(req.Kind, req.Duration, req.Coins); err != nil {
		return nil, err
	}
	return s.storage.TariffS.Create(req)
}

func (s *TariffService) GetById(ctx context.Context, req *m.ById) (*m.TariffRes, error) {
	return s.storage.TariffS.GetById(req)
}

func (s *TariffService) GetAll(ctx context.Context, req *m.TariffGetListReq) (*m.PaginationResponse, *m.TariffList, error) {
	res := &m.PaginationResponse{}

	key := fmt.Sprintf("duration:%d",
		req.Duration)

	if s.rdb != nil {
		if b, err := s.rdb.Get(key).Bytes(); err == nil && len(b) > 0 {
			_ = json.Unmarshal(b, &res)
		}
	}

	if res.Total == 0 {
		res, _, err := s.storage.TariffS.GetAll(req)
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

func (s *TariffService) Update(ctx context.Context, req *m.TariffUpdateReq) (*m.Void, error) {
	kind := req.Kind
	if kind == "" {
		cur, err := s.storage.TariffS.GetById(&m.ById{Id: req.Id})
		if err != nil {
			return nil, err
		}
		kind = "subscription"
		if cur != nil && cur.Kind != nil {
			kind = *cur.Kind
		}
		req.Kind = kind
	}
	if err := validateTariffKind(kind, req.Duration, req.Coins); err != nil {
		return nil, err
	}
	return s.storage.TariffS.Update(req)
}

func (s *TariffService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.TariffS.Delete(req)
}
