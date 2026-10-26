package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// GetAllSettings godoc
// @Summary Tizim sozlamalari (BACKEND_STRUCTURE.md 7-bo'lim: masalan kunlik bepul limit)
// @Tags setting
// @Produce json
// @Success 200 {array} models.AppSettingRes
// @Security BearerAuth
// @Router /web/setting [get]
func (h *Handler) GetAllSettings(c *gin.Context) {
	data, err := h.Clients.AppSetting.GetAll(context.Background())
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// SetSetting godoc
// @Tags setting
// @Accept json
// @Produce json
// @Param setting body models.AppSettingSetReq true "key/value"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/setting [put]
func (h *Handler) SetSetting(c *gin.Context) {
	var req m.AppSettingSetReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	if _, err := h.Clients.AppSetting.Set(context.Background(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "setting saved"})
}

// GetDifficultyRewards godoc
// @Summary Case qiyinlik darajasi (medium/hard) mukofot koeffitsienti
// @Description Bazaviy XP/Coin foydalanuvchining darajasidan (levels.xp_reward/coin_reward)
// @Description olinadi, bu yerdagi koeffitsient ("easy"=1x, bu yerda yo'q) shunga qo'llanadi,
// @Description keyin health_percent'ga mutanosib beriladi (SimulationService.Finish).
// @Description Sozlanmagan bo'lsa default: medium=1.5, hard=2.
// @Tags setting
// @Produce json
// @Success 200 {object} models.DifficultyRewardRes
// @Security BearerAuth
// @Router /web/setting/difficulty-reward [get]
func (h *Handler) GetDifficultyRewards(c *gin.Context) {
	c.JSON(http.StatusOK, h.Clients.AppSetting.GetDifficultyRewards(context.Background()))
}

// SetDifficultyRewards godoc
// @Summary Case qiyinlik darajasi (medium/hard) mukofot koeffitsientini sozlash
// @Tags setting
// @Accept json
// @Produce json
// @Param body body models.DifficultyRewardRes true "medium_multiplier, hard_multiplier"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/setting/difficulty-reward [put]
func (h *Handler) SetDifficultyRewards(c *gin.Context) {
	var req m.DifficultyRewardRes
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	if err := h.Clients.AppSetting.SetDifficultyRewards(context.Background(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "setting saved"})
}

// GetVoiceSetting godoc
// @Summary Ovozli funksiya (shikoyat/chat matnini o'qib berish) global yoqilganmi
// @Description Audio backendda generatsiya qilinmaydi - frontend shu bayroqqa qarab
// @Description qurilma TTS'idan (device text-to-speech) foydalanadi yoki yashiradi.
// @Tags setting
// @Produce json
// @Success 200 {object} models.VoiceSettingRes
// @Security BearerAuth
// @Router /mobile/setting/voice [get]
func (h *Handler) GetVoiceSetting(c *gin.Context) {
	c.JSON(http.StatusOK, m.VoiceSettingRes{Enabled: h.Clients.AppSetting.VoiceEnabled(context.Background())})
}

// GetCoinRate godoc
// @Summary Token narxi: 1 000 token = X tanga
// @Tags setting
// @Produce json
// @Success 200 {object} models.CoinRateRes
// @Security BearerAuth
// @Router /web/setting/coin-rate [get]
func (h *Handler) GetCoinRate(c *gin.Context) {
	c.JSON(http.StatusOK, m.CoinRateRes{CoinsPer1kTokens: h.Clients.AppSetting.CoinsPer1kTokens(context.Background())})
}

// SetCoinRate godoc
// @Summary Token narxini o'zgartirish (1 000 token = X tanga)
// @Description AI provayder tannarxi + platforma xarajatlari + marja yig'indisi koeffitsient sifatida
// @Description kiritiladi. Keyingi barcha so'rovlar yangi tarif bo'yicha hisoblanadi.
// @Tags setting
// @Accept json
// @Produce json
// @Param body body models.CoinRateRes true "coins_per_1k_tokens"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/setting/coin-rate [put]
func (h *Handler) SetCoinRate(c *gin.Context) {
	var req m.CoinRateRes
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	if req.CoinsPer1kTokens < 0 {
		errs.Bad(c, fmt.Errorf("coins_per_1k_tokens must be >= 0"))
		return
	}
	if err := h.Clients.AppSetting.SetCoinsPer1kTokens(context.Background(), req.CoinsPer1kTokens); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "setting saved"})
}

// GetDailyFreeCoins godoc
// @Summary Kunlik bepul tangalar (har foydalanuvchiga har 24 soatda)
// @Tags setting
// @Produce json
// @Success 200 {object} models.DailyFreeCoinsRes
// @Security BearerAuth
// @Router /web/setting/daily-free-coins [get]
func (h *Handler) GetDailyFreeCoins(c *gin.Context) {
	c.JSON(http.StatusOK, m.DailyFreeCoinsRes{DailyFreeCoins: h.Clients.AppSetting.DailyFreeCoins(context.Background())})
}

// SetDailyFreeCoins godoc
// @Summary Kunlik bepul tangalarni o'zgartirish (0 - bepul tanga yo'q)
// @Description Bugun allaqachon berilgan tangalarga ta'sir qilmaydi, ertadan kuchga kiradi.
// @Tags setting
// @Accept json
// @Produce json
// @Param body body models.DailyFreeCoinsRes true "daily_free_coins"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/setting/daily-free-coins [put]
func (h *Handler) SetDailyFreeCoins(c *gin.Context) {
	var req m.DailyFreeCoinsRes
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	if req.DailyFreeCoins < 0 || req.DailyFreeCoins > 10000 {
		errs.Bad(c, fmt.Errorf("daily_free_coins must be between 0 and 10000"))
		return
	}
	if _, err := h.Clients.AppSetting.Set(context.Background(), &m.AppSettingSetReq{Key: m.AppSettingDailyFreeCoins, Value: strconv.Itoa(req.DailyFreeCoins)}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "setting saved"})
}

// GetPracticeImages godoc
// @Summary AI practice case'lariga tibbiy tasvir (EKG, rentgen...) generatsiyasi yoqilganmi
// @Tags setting
// @Produce json
// @Success 200 {object} models.PracticeImagesRes
// @Security BearerAuth
// @Router /web/setting/practice-images [get]
func (h *Handler) GetPracticeImages(c *gin.Context) {
	c.JSON(http.StatusOK, m.PracticeImagesRes{Enabled: h.Clients.AppSetting.PracticeImagesEnabled(context.Background())})
}

// SetPracticeImages godoc
// @Summary AI tasvir generatsiyasini yoqish/o'chirish
// @Description Yoqilsa case'larning bir qismiga AI tasvir (illyustrativ) qo'shiladi; qo'shimcha token sarfi bor.
// @Tags setting
// @Accept json
// @Produce json
// @Param body body models.PracticeImagesRes true "enabled"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/setting/practice-images [put]
func (h *Handler) SetPracticeImages(c *gin.Context) {
	var req m.PracticeImagesRes
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	if _, err := h.Clients.AppSetting.Set(context.Background(), &m.AppSettingSetReq{Key: m.AppSettingPracticeImages, Value: strconv.FormatBool(req.Enabled)}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "setting saved"})
}
