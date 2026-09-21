package handler

import (
	"context"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// RegisterDevice godoc
// @Summary Push token ro'yxatdan o'tkazish (ko'p qurilma support)
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object{fcm_token=string,platform=string} true "Device"
// @Success 200 {object} string
// @Router /mobile/user/device [post]
func (h *Handler) RegisterDevice(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	var body struct {
		FCMToken string `json:"fcm_token"`
		Platform string `json:"platform"`
	}
	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		return
	}
	if _, err := h.Clients.UserDevice.Register(context.Background(), &m.UserDeviceRegisterReq{
		UserId: userId, FCMToken: body.FCMToken, Platform: body.Platform,
	}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "device registered"})
}

// UnregisterDevice godoc
// @Summary Push token'ni o'chirish (logout)
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body object{fcm_token=string} true "Device"
// @Success 200 {object} string
// @Router /mobile/user/device [delete]
func (h *Handler) UnregisterDevice(c *gin.Context) {
	var body struct {
		FCMToken string `json:"fcm_token"`
	}
	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		return
	}
	if _, err := h.Clients.UserDevice.Unregister(context.Background(), &m.UserDeviceUnregisterReq{FCMToken: body.FCMToken}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "device unregistered"})
}
