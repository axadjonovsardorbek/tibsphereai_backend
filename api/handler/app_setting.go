package handler

import (
	"context"
	"net/http"
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
// @Summary Case qiyinlik darajasi (easy/medium/hard) bo'yicha bazaviy XP/Coin mukofoti
// @Description Yakuniy mukofot shu bazaga health_percent nisbatan mutanosib beriladi
// @Description (SimulationService.Finish). Sozlanmagan bo'lsa default: easy 50/5, medium 100/10, hard 150/15.
// @Tags setting
// @Produce json
// @Success 200 {object} models.DifficultyRewardRes
// @Security BearerAuth
// @Router /web/setting/difficulty-reward [get]
func (h *Handler) GetDifficultyRewards(c *gin.Context) {
	c.JSON(http.StatusOK, h.Clients.AppSetting.GetDifficultyRewards(context.Background()))
}

// SetDifficultyRewards godoc
// @Summary Case qiyinlik darajasi bo'yicha bazaviy XP/Coin mukofotini sozlash
// @Tags setting
// @Accept json
// @Produce json
// @Param body body models.DifficultyRewardRes true "6 ta qiymat (easy/medium/hard uchun xp+coin)"
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
