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

// StatsService - admin statistikasi (TZ_STATISTICS). Natijalar 60 soniya keshlanadi.
type StatsService struct {
	storage st.Storage
	rdb     *redis.Client
}

func NewStatsService(storage *st.Storage, rdb *redis.Client) *StatsService {
	return &StatsService{storage: *storage, rdb: rdb}
}

const statsMaxDays = 366

var tashkent = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Tashkent"); err == nil {
		return loc
	}
	return time.FixedZone("UTC+5", 5*3600)
}()

// NormalizePeriod - from/to bo'sh bo'lsa oxirgi 30 kun (Toshkent bo'yicha bugun kiradi).
func NormalizePeriod(req *m.StatsPeriodReq) error {
	today := time.Now().In(tashkent).Format(time.DateOnly)
	if req.To == "" {
		req.To = today
	}
	to, err := time.Parse(time.DateOnly, req.To)
	if err != nil {
		return errors.New("to must be YYYY-MM-DD")
	}
	if req.From == "" {
		req.From = to.AddDate(0, 0, -29).Format(time.DateOnly)
	}
	from, err := time.Parse(time.DateOnly, req.From)
	if err != nil {
		return errors.New("from must be YYYY-MM-DD")
	}
	if from.After(to) {
		return errors.New("from must not be after to")
	}
	if to.Sub(from) > statsMaxDays*24*time.Hour {
		return fmt.Errorf("period must not exceed %d days", statsMaxDays)
	}
	return nil
}

func cached[T any](s *StatsService, key string, load func() (*T, error)) (*T, error) {
	if s.rdb != nil {
		if b, err := s.rdb.Get(key).Bytes(); err == nil && len(b) > 0 {
			var v T
			if json.Unmarshal(b, &v) == nil {
				return &v, nil
			}
		}
	}
	v, err := load()
	if err != nil {
		return nil, err
	}
	if s.rdb != nil {
		if b, err := json.Marshal(v); err == nil {
			_ = s.rdb.Set(key, b, 60*time.Second).Err()
		}
	}
	return v, nil
}

func periodKey(name string, req *m.StatsPeriodReq) string {
	return fmt.Sprintf("stats:%s:%s:%s", name, req.From, req.To)
}

func (s *StatsService) Finance(ctx context.Context, req *m.StatsPeriodReq) (*m.FinanceStatsRes, error) {
	if err := NormalizePeriod(req); err != nil {
		return nil, err
	}
	return cached(s, periodKey("finance", req), func() (*m.FinanceStatsRes, error) { return s.storage.StatsS.Finance(req) })
}

func (s *StatsService) AICost(ctx context.Context, req *m.StatsPeriodReq) (*m.AICostStatsRes, error) {
	if err := NormalizePeriod(req); err != nil {
		return nil, err
	}
	return cached(s, periodKey("ai_cost", req), func() (*m.AICostStatsRes, error) { return s.storage.StatsS.AICost(req) })
}

func (s *StatsService) PremiumCost(ctx context.Context, req *m.StatsPeriodReq) (*m.PremiumCostStatsRes, error) {
	if err := NormalizePeriod(req); err != nil {
		return nil, err
	}
	return cached(s, periodKey("premium_cost", req), func() (*m.PremiumCostStatsRes, error) { return s.storage.StatsS.PremiumCost(req) })
}

func (s *StatsService) Engagement(ctx context.Context, req *m.StatsPeriodReq) (*m.EngagementStatsRes, error) {
	if err := NormalizePeriod(req); err != nil {
		return nil, err
	}
	return cached(s, periodKey("engagement", req), func() (*m.EngagementStatsRes, error) { return s.storage.StatsS.Engagement(req) })
}

func (s *StatsService) UserStats(ctx context.Context, userId string) (*m.UserStatsRes, error) {
	return cached(s, "stats:user:"+userId, func() (*m.UserStatsRes, error) { return s.storage.StatsS.UserStats(userId) })
}
