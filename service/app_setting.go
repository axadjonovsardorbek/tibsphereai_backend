package service

import (
	"context"
	"strconv"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

type AppSettingService struct {
	storage    st.Storage
	defaultLim int
}

func NewAppSettingService(storage *st.Storage, defaultDailyFreeLimit int) *AppSettingService {
	return &AppSettingService{storage: *storage, defaultLim: defaultDailyFreeLimit}
}

func (s *AppSettingService) GetAll(ctx context.Context) ([]m.AppSettingRes, error) {
	return s.storage.AppSettingS.GetAll()
}

func (s *AppSettingService) Set(ctx context.Context, req *m.AppSettingSetReq) (*m.Void, error) {
	return s.storage.AppSettingS.Set(req)
}

// DailyFreeLimit - admin panel orqali o'zgartirilishi mumkin bo'lgan kunlik
// bepul urinishlar soni (BACKEND_STRUCTURE.md 6,7-bo'lim). Sozlanmagan bo'lsa
// config'dagi DAILY_FREE_LIMIT default qiymati ishlatiladi.
func (s *AppSettingService) DailyFreeLimit(ctx context.Context) int {
	setting, err := s.storage.AppSettingS.Get(m.AppSettingDailyFreeLimit)
	if err != nil || setting == nil {
		return s.defaultLim
	}
	if v, err := strconv.Atoi(setting.Value); err == nil {
		return v
	}
	return s.defaultLim
}

// HasFreeAttemptLeft - kunlik limit tekshiruvi (Premium bo'lmagan foydalanuvchilar uchun).
func (s *AppSettingService) HasFreeAttemptLeft(ctx context.Context, userId string) (bool, error) {
	used, err := s.storage.AppSettingS.GetTodayFreeAttemptsUsed(userId)
	if err != nil {
		return false, err
	}
	return used < s.DailyFreeLimit(ctx), nil
}

func (s *AppSettingService) ConsumeFreeAttempt(ctx context.Context, userId string) error {
	return s.storage.AppSettingS.IncrTodayFreeAttempts(userId)
}

// GetLimitStatus - home ekranidagi "35/50 Limit" belgisi uchun. Premium
// obunachi bo'lsa cheksiz (has_subscription=true, total/used=0).
func (s *AppSettingService) GetLimitStatus(ctx context.Context, userId string) (*m.UserLimitStatusRes, error) {
	var hasSub bool
	_ = s.storage.Db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM subscriptions WHERE user_id = $1 AND status = 'active' AND ended_at > now())
	`, userId).Scan(&hasSub)

	if hasSub {
		return &m.UserLimitStatusRes{HasSubscription: true}, nil
	}

	used, err := s.storage.AppSettingS.GetTodayFreeAttemptsUsed(userId)
	if err != nil {
		return nil, err
	}
	total := s.DailyFreeLimit(ctx)
	remaining := total - used
	if remaining < 0 {
		remaining = 0
	}
	return &m.UserLimitStatusRes{Used: used, Total: total, Remaining: remaining}, nil
}
