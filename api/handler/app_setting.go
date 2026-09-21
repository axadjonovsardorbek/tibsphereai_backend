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
