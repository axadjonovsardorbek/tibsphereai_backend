package handler

import (
	"context"
	"net/http"
	"tibsphereai/pkg/errs"

	"github.com/gin-gonic/gin"
)

// GetDebrief godoc
// @Summary AI orqali Debriefing hisoboti (BACKEND_STRUCTURE.md 5-bo'lim)
// @Tags debrief
// @Produce json
// @Param session_id path string true "Session ID"
// @Success 200 {object} models.DebriefReportRes
// @Security BearerAuth
// @Router /mobile/debrief/{session_id} [get]
func (h *Handler) GetDebrief(c *gin.Context) {
	res, err := h.Clients.Debrief.GetBySessionId(context.Background(), c.Param("session_id"))
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
