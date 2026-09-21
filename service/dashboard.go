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

type DashboardService struct {
	storage st.Storage
	rdb     *redis.Client
}

func NewDashboardService(storage *st.Storage, rdb *redis.Client) *DashboardService {
	return &DashboardService{storage: *storage, rdb: rdb}
}

func (s *DashboardService) GetDashboard(ctx context.Context, req *m.GetDashboardReq) (*m.DashboardRes, error) {
	key := fmt.Sprintf("dashboard:type=%s:day=%s:from=%s:to=%s", req.Type, req.Day, req.From, req.To)

	if s.rdb != nil {
		b, err := s.rdb.Get(key).Bytes()
		if err == nil && len(b) > 0 {
			var cached m.DashboardRes
			if err := json.Unmarshal(b, &cached); err == nil {
				return &cached, nil
			}
		}
	}

	res, err := s.storage.DashboardS.GetDashboard(req)
	if err != nil {
		return nil, err
	}

	if s.rdb != nil {
		if b, err := json.Marshal(res); err == nil {
			_ = s.rdb.Set(key, b, 300*time.Second).Err()
		}
	}

	return res, nil
}
