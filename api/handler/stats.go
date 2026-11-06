package handler

import (
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"
	"tibsphereai/service"

	"github.com/gin-gonic/gin"
)

// statsPeriod - ?from&to (YYYY-MM-DD, Toshkent kuni); berilmasa oxirgi 30 kun. Xato bo'lsa 400 yuboradi.
func statsPeriod(c *gin.Context) (*m.StatsPeriodReq, bool) {
	req := &m.StatsPeriodReq{From: c.Query("from"), To: c.Query("to")}
	if err := service.NormalizePeriod(req); err != nil {
		errs.Bad(c, err)
		return nil, false
	}
	return req, true
}

func statsReply[T any](c *gin.Context, res *T, err error) {
	if err != nil {
		errs.Internal(c, err)
		return
	}
	c.JSON(200, res)
}

// GetFinanceStats godoc
// @Summary Moliya: tushum, AI xarajati, foyda, ARPU/ARPPU, kunlik va tarif bo'yicha
// @Tags stats
// @Produce json
// @Param from query string false "YYYY-MM-DD (default: to - 29 kun)"
// @Param to query string false "YYYY-MM-DD (default: bugun, Toshkent)"
// @Success 200 {object} models.FinanceStatsRes
// @Failure 400 {object} string
// @Security BearerAuth
// @Router /web/stats/finance [get]
func (h *Handler) GetFinanceStats(c *gin.Context) {
	if req, ok := statsPeriod(c); ok {
		res, err := h.Clients.Stats.Finance(c.Request.Context(), req)
		statsReply(c, res, err)
	}
}

// GetAICostStats godoc
// @Summary AI xarajati: funksiya, amal va foydalanuvchi turi bo'yicha; 1 tanga tannarxi
// @Tags stats
// @Produce json
// @Param from query string false "YYYY-MM-DD"
// @Param to query string false "YYYY-MM-DD"
// @Success 200 {object} models.AICostStatsRes
// @Failure 400 {object} string
// @Security BearerAuth
// @Router /web/stats/ai-cost [get]
func (h *Handler) GetAICostStats(c *gin.Context) {
	if req, ok := statsPeriod(c); ok {
		res, err := h.Clients.Stats.AICost(c.Request.Context(), req)
		statsReply(c, res, err)
	}
}

// GetPremiumCostStats godoc
// @Summary Premium foydalanuvchilar AI xarajati va eng ko'p sarflagan 20 foydalanuvchi
// @Tags stats
// @Produce json
// @Param from query string false "YYYY-MM-DD"
// @Param to query string false "YYYY-MM-DD"
// @Success 200 {object} models.PremiumCostStatsRes
// @Failure 400 {object} string
// @Security BearerAuth
// @Router /web/stats/premium-cost [get]
func (h *Handler) GetPremiumCostStats(c *gin.Context) {
	if req, ok := statsPeriod(c); ok {
		res, err := h.Clients.Stats.PremiumCost(c.Request.Context(), req)
		statsReply(c, res, err)
	}
}

// GetEngagementStats godoc
// @Summary Faollik: DAU/WAU/MAU, kunlik ko'rsatkichlar, voronka
// @Tags stats
// @Produce json
// @Param from query string false "YYYY-MM-DD"
// @Param to query string false "YYYY-MM-DD"
// @Success 200 {object} models.EngagementStatsRes
// @Failure 400 {object} string
// @Security BearerAuth
// @Router /web/stats/engagement [get]
func (h *Handler) GetEngagementStats(c *gin.Context) {
	if req, ok := statsPeriod(c); ok {
		res, err := h.Clients.Stats.Engagement(c.Request.Context(), req)
		statsReply(c, res, err)
	}
}

// GetUserStats godoc
// @Summary Bitta foydalanuvchi: oxirgi 30 kun grafigi, eng ko'p bo'limlar, oxirgi tranzaksiya/buyurtmalar
// @Tags stats
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} models.UserStatsRes
// @Security BearerAuth
// @Router /web/user/{id}/stats [get]
func (h *Handler) GetUserStats(c *gin.Context) {
	res, err := h.Clients.Stats.UserStats(c.Request.Context(), c.Param("id"))
	statsReply(c, res, err)
}
