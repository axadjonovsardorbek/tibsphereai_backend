package handler

import (
	"context"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// GetAllAIPrompts godoc
// @Summary AI promptlarini ko'rish (BACKEND_STRUCTURE.md 7-bo'lim)
// @Tags ai-prompt
// @Produce json
// @Success 200 {array} models.AIPromptRes
// @Security BearerAuth
// @Router /web/ai-prompt [get]
func (h *Handler) GetAllAIPrompts(c *gin.Context) {
	data, err := h.Clients.AIPrompt.GetAll(context.Background())
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// UpsertAIPrompt godoc
// @Summary AI promptni kodni o'zgartirmasdan sozlash
// @Tags ai-prompt
// @Accept json
// @Produce json
// @Param prompt body models.AIPromptUpsertReq true "Prompt"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/ai-prompt [put]
func (h *Handler) UpsertAIPrompt(c *gin.Context) {
	var req m.AIPromptUpsertReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	if _, err := h.Clients.AIPrompt.Upsert(context.Background(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "prompt saved"})
}

// GetAIUsageSummary godoc
// @Summary Gemini AI xarajatlari va sarflangan tokenlar (BACKEND_STRUCTURE.md 7-bo'lim)
// @Tags ai-prompt
// @Produce json
// @Success 200 {object} models.AIUsageSummaryRes
// @Security BearerAuth
// @Router /web/ai-prompt/usage [get]
func (h *Handler) GetAIUsageSummary(c *gin.Context) {
	res, err := h.Clients.AIPrompt.GetUsageSummary(context.Background())
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// TestAIPatient godoc
// @Summary Bemor AI'sini bir martalik so'rov bilan sinash ("case yechish"ni tekshirish)
// @Tags ai-prompt
// @Accept json
// @Produce json
// @Param body body models.AITestPatientReq true "case_id + question"
// @Success 200 {object} models.AITestPatientRes
// @Security BearerAuth
// @Router /web/ai-prompt/test-patient [post]
func (h *Handler) TestAIPatient(c *gin.Context) {
	var req m.AITestPatientReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	res, err := h.Clients.AIPrompt.TestPatient(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// TestAIDebrief godoc
// @Summary Debrief AI'sini bir martalik so'rov bilan sinash (qadamlar logi -> hisobot)
// @Tags ai-prompt
// @Accept json
// @Produce json
// @Param body body models.AITestDebriefReq true "case_id + events"
// @Success 200 {object} models.AITestDebriefRes
// @Security BearerAuth
// @Router /web/ai-prompt/test-debrief [post]
func (h *Handler) TestAIDebrief(c *gin.Context) {
	var req m.AITestDebriefReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	res, err := h.Clients.AIPrompt.TestDebrief(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
