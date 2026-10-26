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

// CoinsPer1kTokens - admin belgilagan tarif: 1 000 AI token necha tanga. Sozlanmagan yoki
// manfiy bo'lsa 0 (tanga yechilmaydi).
func (s *AppSettingService) CoinsPer1kTokens(ctx context.Context) float64 {
	if v := s.getFloatSetting(m.AppSettingCoinsPer1kTokens, 0); v > 0 {
		return v
	}
	return 0
}

// SetCoinsPer1kTokens - tarifni saqlaydi; keyingi barcha hisob-kitoblar yangi qiymatni ishlatadi.
func (s *AppSettingService) SetCoinsPer1kTokens(ctx context.Context, v float64) error {
	_, err := s.storage.AppSettingS.Set(&m.AppSettingSetReq{Key: m.AppSettingCoinsPer1kTokens, Value: strconv.FormatFloat(v, 'f', -1, 64)})
	return err
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

// VoiceEnabled - platforma ovozli funksiyasi (case shikoyati va chat javoblarini
// ovozda o'qib berish) global yoqilganmi (admin killswitch, BACKEND_STRUCTURE.md
// 7-bo'lim). Audio backendda generatsiya qilinmaydi - frontend qurilma TTS'idan
// (iOS/Android/brauzer) foydalanadi, backend faqat shu funksiyani ko'rsatish/yashirish
// uchun bitta bool beradi. Sozlanmagan bo'lsa default yoqilgan (true).
func (s *AppSettingService) VoiceEnabled(ctx context.Context) bool {
	setting, err := s.storage.AppSettingS.Get(m.AppSettingVoiceEnabled)
	if err != nil || setting == nil {
		return true
	}
	v, err := strconv.ParseBool(setting.Value)
	if err != nil {
		return true
	}
	return v
}

// Case qiyinlik darajasi koeffitsientlari - app_settings'da sozlanmagan bo'lsa
// ishlatiladigan default qiymatlar ("easy" har doim 1x, bu yerda yo'q).
const (
	defaultMediumMultiplier = 1.5
	defaultHardMultiplier   = 2.0
)

func (s *AppSettingService) getFloatSetting(key string, def float64) float64 {
	setting, err := s.storage.AppSettingS.Get(key)
	if err != nil || setting == nil {
		return def
	}
	if v, err := strconv.ParseFloat(setting.Value, 64); err == nil {
		return v
	}
	return def
}

// DifficultyMultiplier - case.difficulty (easy|medium|hard) bo'yicha koeffitsient
// (admin "Mukofot sozlamalari" orqali sozlaydi, masalan medium=1.5x, hard=2x).
// Bu koeffitsient foydalanuvchining o'z darajasidagi bazaviy XP/Coin'ga
// (levels.xp_reward/coin_reward) qo'llanadi - ko'ring [[SimulationService.Finish]].
// "easy" yoki noma'lum/bo'sh difficulty - 1x (ko'paytirilmagan).
func (s *AppSettingService) DifficultyMultiplier(ctx context.Context, difficulty string) float64 {
	switch difficulty {
	case "medium":
		return s.getFloatSetting(m.AppSettingMediumRewardMultiplier, defaultMediumMultiplier)
	case "hard":
		return s.getFloatSetting(m.AppSettingHardRewardMultiplier, defaultHardMultiplier)
	default:
		return 1.0
	}
}

// GetDifficultyRewards - admin panelning "Mukofot sozlamalari" formasi uchun,
// joriy koeffitsientlar bir so'rovda.
func (s *AppSettingService) GetDifficultyRewards(ctx context.Context) *m.DifficultyRewardRes {
	return &m.DifficultyRewardRes{
		MediumMultiplier: s.getFloatSetting(m.AppSettingMediumRewardMultiplier, defaultMediumMultiplier),
		HardMultiplier:   s.getFloatSetting(m.AppSettingHardRewardMultiplier, defaultHardMultiplier),
	}
}

// SetDifficultyRewards - admin formadagi 2 ta koeffitsientni app_settings'ga yozadi.
func (s *AppSettingService) SetDifficultyRewards(ctx context.Context, req *m.DifficultyRewardRes) error {
	if _, err := s.storage.AppSettingS.Set(&m.AppSettingSetReq{Key: m.AppSettingMediumRewardMultiplier, Value: strconv.FormatFloat(req.MediumMultiplier, 'f', -1, 64)}); err != nil {
		return err
	}
	if _, err := s.storage.AppSettingS.Set(&m.AppSettingSetReq{Key: m.AppSettingHardRewardMultiplier, Value: strconv.FormatFloat(req.HardMultiplier, 'f', -1, 64)}); err != nil {
		return err
	}
	return nil
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
	return &m.UserLimitStatusRes{Used: used, Total: total, Remaining: remaining, CoinsPer1kTokens: s.CoinsPer1kTokens(ctx)}, nil
}

// DailyFreeCoins - har 24 soatda har foydalanuvchiga beriladigan bepul tangalar (default 10; 0 - yo'q).
func (s *AppSettingService) DailyFreeCoins(ctx context.Context) int {
	v := s.getFloatSetting(m.AppSettingDailyFreeCoins, 10)
	if v < 0 {
		return 10
	}
	return int(v)
}

// PracticeImagesEnabled - case'larga AI tibbiy tasvir (EKG/rentgen...) generatsiyasi yoqilganmi (default o'chiq:
// tasvirlar illyustrativ va qo'shimcha xarajat keltiradi, admin ongli ravishda yoqadi).
func (s *AppSettingService) PracticeImagesEnabled(ctx context.Context) bool {
	setting, err := s.storage.AppSettingS.Get(m.AppSettingPracticeImages)
	if err != nil || setting == nil {
		return false
	}
	v, err := strconv.ParseBool(setting.Value)
	return err == nil && v
}
