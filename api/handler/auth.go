package handler

import (
	"context"
	"fmt"
	"net/http"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// SendConfirmation godoc
// @Summary Send confirmation code to email
// @Description Send a confirmation code to the provided email address
// @Tags auth
// @Accept  json
// @Produce  json
// @Param banner body models.SendConfirmationCodeReq true "Code Details"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /mobile/auth/user/otp/send [post]
func (h *Handler) SendConfirmation(c *gin.Context) {
	var req m.SendConfirmationCodeReq

	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body", err)
		return
	}

	if req.Type != "email" && req.Type != "telegram" {
		errs.Bad(c, fmt.Errorf("invalid type, must be 'email' or 'telegram'"))
		slog.Error("Invalid type, must be 'email' or 'telegram'")
		return
	}

	check := isValidIdentifier(req.Identifier)
	if !check {
		errs.Bad(c, fmt.Errorf("invalid identifier format"))
		slog.Error("Invalid identifier format")
		return
	}

	slog.Info("Identifier:", req)
	res, err := h.Clients.User.SendConfirmationCode(context.Background(), &req)
	slog.Info("Send Confirmation res:", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error while sending otp", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "confirmation code sent successfully"})
}

// VerifyConfirmation godoc
// @Summary Verify confirmation code
// @Description Verify the provided confirmation code and return a token if valid
// @Tags auth
// @Accept  json
// @Produce  json
// @Param banner body models.ConfirmationReq true "Code Details"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /mobile/auth/user/otp/confirm [post]
func (h *Handler) VerifyConfirmation(c *gin.Context) {
	var req m.ConfirmationReq

	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body", err)
		return
	}

	if req.Identifier != "" && req.Identifier != "string" {

		check := isValidIdentifier(req.Identifier)

		if !check {
			errs.Bad(c, fmt.Errorf("invalid identifier format"))
			slog.Error("Invalid identifier format")
			return
		}
	}

	if req.ConfirmationCode == "" || req.ConfirmationCode == "string" {
		errs.Bad(c, fmt.Errorf("confirmation code is required"))
		slog.Error("Confirmation code is required")
		return
	}

	if req.Type != "email" && req.Type != "telegram" {
		errs.Bad(c, fmt.Errorf("invalid type, must be 'email' or 'telegram'"))
		slog.Error("Invalid type, must be 'email' or 'telegram'")
		return
	}

	slog.Info("Verify Confirmation Code:", req)
	res, err := h.Clients.User.Confirmation(context.Background(), &req)
	slog.Info("Verify Confirmation res:", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error while verifying confirmation code", err)
		return
	}

	c.JSON(http.StatusOK, res)
}
