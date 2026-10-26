package handler

import (
	"net/http"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// GetRoadmap godoc
// @Summary AI yo'l xaritasi: bilim xaritasi (kuchli/zaif mavzular) va shaxsiy o'qish yo'li
// @Description Xarita va qadamlar bepul, har so'rovda hisoblanadi. AI izohlari (is_ai=true) POST /mobile/roadmap/refresh
// @Description orqali yaratiladi va 24 soat keshlanadi. Har qadamdan to'g'ridan-to'g'ri mashq boshlash mumkin (action/topic_id/difficulty/count).
// @Tags roadmap
// @Produce json
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.RoadmapRes
// @Security BearerAuth
// @Router /mobile/roadmap [get]
func (h *Handler) GetRoadmap(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	res, err := h.Clients.Roadmap.Get(c.Request.Context(), userId, getLanguage(c))
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// RefreshRoadmap godoc
// @Summary AI yo'l xaritasi izohlarini yangilash (AI token sarfiga tanga yechiladi)
// @Tags roadmap
// @Produce json
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.RoadmapRes
// @Security BearerAuth
// @Router /mobile/roadmap/refresh [post]
func (h *Handler) RefreshRoadmap(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	res, err := h.Clients.Roadmap.Refresh(c.Request.Context(), userId, getLanguage(c))
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// CreateFeedback godoc
// @Summary Ilova haqida fikr/baho yuborish (rating 1-5 yoki xabar)
// @Tags feedback
// @Accept json
// @Produce json
// @Param body body models.FeedbackCreateReq true "feedback"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /mobile/feedback [post]
func (h *Handler) CreateFeedback(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	var req m.FeedbackCreateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.UserId = userId
	res, err := h.Clients.Feedback.Create(c.Request.Context(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "thank you", "id": res.Id})
}

// GetAllFeedback godoc
// @Summary Foydalanuvchilar fikrlari (admin)
// @Tags feedback
// @Produce json
// @Param kind query string false "general|bug|idea|content|nps"
// @Param status query string false "new|seen|done"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.PaginationResponse
// @Security BearerAuth
// @Router /web/feedback [get]
func (h *Handler) GetAllFeedback(c *gin.Context) {
	data, sum, err := h.Clients.Feedback.GetAll(c.Request.Context(), &m.FeedbackListReq{
		Kind: c.Query("kind"), Status: c.Query("status"), Filter: m.ParsePagination(c.Query("limit"), c.Query("page")),
	})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": sum, "list": data})
}

// SetFeedbackStatus godoc
// @Summary Fikr holatini belgilash (new|seen|done)
// @Tags feedback
// @Accept json
// @Produce json
// @Param id path string true "Feedback ID"
// @Param body body object true "{status}"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/feedback/{id}/status [put]
func (h *Handler) SetFeedbackStatus(c *gin.Context) {
	var body struct {
		Status string `json:"status"`
	}
	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		return
	}
	if err := h.Clients.Feedback.SetStatus(c.Request.Context(), c.Param("id"), body.Status); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "status updated"})
}
