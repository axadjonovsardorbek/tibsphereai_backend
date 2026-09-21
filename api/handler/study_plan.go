package handler

import (
	"context"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// UpsertStudyPlan godoc
// @Summary O'quv rejasi va eslatma vaqtini belgilash (BACKEND_STRUCTURE.md 2-bo'lim)
// @Tags study-plan
// @Accept json
// @Produce json
// @Param plan body models.StudyPlanUpsertReq true "Study plan"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /mobile/study-plan [put]
func (h *Handler) UpsertStudyPlan(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}

	var req m.StudyPlanUpsertReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.UserId = userId

	if _, err := h.Clients.StudyPlan.Upsert(context.Background(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "study plan saved"})
}

// GetStudyPlan godoc
// @Tags study-plan
// @Produce json
// @Success 200 {object} models.StudyPlanRes
// @Security BearerAuth
// @Router /mobile/study-plan [get]
func (h *Handler) GetStudyPlan(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	res, err := h.Clients.StudyPlan.GetByUserId(context.Background(), userId)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
