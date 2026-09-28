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

// difficultyRewardDefaults - app_settings'da sozlanmagan bo'lsa ishlatiladigan
// bazaviy qiymatlar (case.difficulty: easy|medium|hard).
var difficultyRewardDefaults = map[string]struct{ XP, Coin int }{
	"easy":   {XP: 50, Coin: 5},
	"medium": {XP: 100, Coin: 10},
	"hard":   {XP: 150, Coin: 15},
}

func (s *AppSettingService) getIntSetting(key string, def int) int {
	setting, err := s.storage.AppSettingS.Get(key)
	if err != nil || setting == nil {
		return def
	}
	if v, err := strconv.Atoi(setting.Value); err == nil {
		return v
	}
	return def
}

// DifficultyReward - case.difficulty (easy|medium|hard) bo'yicha bazaviy XP/Coin
// (admin "Mukofot sozlamalari" orqali sozlaydi). Yakuniy mukofot bu bazaga
// health_percent nisbatan mutanosib beriladi - ko'ring [[SimulationService.Finish]].
// Noma'lum/bo'sh difficulty kelsa - "medium" default'iga tushadi.
func (s *AppSettingService) DifficultyReward(ctx context.Context, difficulty string) (xpBase int, coinBase int) {
	def, ok := difficultyRewardDefaults[difficulty]
	if !ok {
		def = difficultyRewardDefaults["medium"]
	}
	switch difficulty {
	case "easy":
		return s.getIntSetting(m.AppSettingXPRewardEasy, def.XP), s.getIntSetting(m.AppSettingCoinRewardEasy, def.Coin)
	case "hard":
		return s.getIntSetting(m.AppSettingXPRewardHard, def.XP), s.getIntSetting(m.AppSettingCoinRewardHard, def.Coin)
	default:
		return s.getIntSetting(m.AppSettingXPRewardMedium, def.XP), s.getIntSetting(m.AppSettingCoinRewardMedium, def.Coin)
	}
}

// GetDifficultyRewards - admin panelning "Mukofot sozlamalari" formasi uchun,
// uchala darajaning joriy qiymatlari bir so'rovda.
func (s *AppSettingService) GetDifficultyRewards(ctx context.Context) *m.DifficultyRewardRes {
	xpE, coinE := s.DifficultyReward(ctx, "easy")
	xpM, coinM := s.DifficultyReward(ctx, "medium")
	xpH, coinH := s.DifficultyReward(ctx, "hard")
	return &m.DifficultyRewardRes{
		XPEasy: xpE, CoinEasy: coinE,
		XPMedium: xpM, CoinMedium: coinM,
		XPHard: xpH, CoinHard: coinH,
	}
}

// SetDifficultyRewards - admin formadagi 6 ta qiymatni app_settings'ga bittalab yozadi.
func (s *AppSettingService) SetDifficultyRewards(ctx context.Context, req *m.DifficultyRewardRes) error {
	kv := map[string]int{
		m.AppSettingXPRewardEasy:     req.XPEasy,
		m.AppSettingCoinRewardEasy:   req.CoinEasy,
		m.AppSettingXPRewardMedium:   req.XPMedium,
		m.AppSettingCoinRewardMedium: req.CoinMedium,
		m.AppSettingXPRewardHard:     req.XPHard,
		m.AppSettingCoinRewardHard:   req.CoinHard,
	}
	for key, val := range kv {
		if _, err := s.storage.AppSettingS.Set(&m.AppSettingSetReq{Key: key, Value: strconv.Itoa(val)}); err != nil {
			return err
		}
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
	return &m.UserLimitStatusRes{Used: used, Total: total, Remaining: remaining}, nil
}
