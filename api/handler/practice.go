package handler

import (
	"net/http"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// StartPractice godoc
// @Summary AI practice sessiyasini boshlash (bo'lim, mavzu, qiyinlik, rejim)
// @Tags practice
// @Accept json
// @Produce json
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Param body body models.PracticeStartReq true "mode, category_id, topic_ids, difficulty, count"
// @Success 200 {object} models.PracticeSessionRes
// @Security BearerAuth
// @Router /mobile/practice/sessions [post]
func (h *Handler) StartPractice(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	var req m.PracticeStartReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.UserId, req.Language = userId, getLanguage(c)
	res, err := h.Clients.Practice.Start(c.Request.Context(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// NextPractice godoc
// @Summary Keyingi case/savolni AI generatsiya qiladi (javob berilmagan element bo'lsa o'shani qaytaradi)
// @Tags practice
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} models.PracticeItemRes
// @Security BearerAuth
// @Router /mobile/practice/sessions/{id}/next [post]
func (h *Handler) NextPractice(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	res, err := h.Clients.Practice.Next(c.Request.Context(), userId, c.Param("id"))
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// AnswerPractice godoc
// @Summary Javob yuborish: case - erkin matn (AI baholaydi), test - variant kaliti (A-D)
// @Tags practice
// @Accept json
// @Produce json
// @Param id path string true "Item ID"
// @Param body body models.PracticeAnswerReq true "answer"
// @Success 200 {object} service.PracticeAnswerRes
// @Security BearerAuth
// @Router /mobile/practice/items/{id}/answer [post]
func (h *Handler) AnswerPractice(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	var req m.PracticeAnswerReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.UserId, req.ItemId = userId, c.Param("id")
	res, err := h.Clients.Practice.Answer(c.Request.Context(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// FinishPractice godoc
// @Summary Sessiyani yakunlash (natija, XP)
// @Tags practice
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} models.PracticeSessionRes
// @Security BearerAuth
// @Router /mobile/practice/sessions/{id}/finish [post]
func (h *Handler) FinishPractice(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	res, err := h.Clients.Practice.Finish(c.Request.Context(), userId, c.Param("id"))
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetPracticeSession godoc
// @Summary Sessiya tafsiloti (elementlar bilan)
// @Tags practice
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {object} models.PracticeSessionRes
// @Security BearerAuth
// @Router /mobile/practice/sessions/{id} [get]
func (h *Handler) GetPracticeSession(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	res, err := h.Clients.Practice.GetSession(c.Request.Context(), userId, c.Param("id"))
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// ListPracticeSessions godoc
// @Summary Practice tarixi
// @Tags practice
// @Produce json
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.PaginationResponse
// @Security BearerAuth
// @Router /mobile/practice/sessions [get]
func (h *Handler) ListPracticeSessions(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	data, _, err := h.Clients.Practice.ListSessions(c.Request.Context(), &m.PracticeListReq{
		UserId: userId, Filter: m.ParsePagination(c.Query("limit"), c.Query("page")),
	})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// ReportPracticeItem godoc
// @Summary Noto'g'ri case/savol haqida xabar berish
// @Tags practice
// @Accept json
// @Produce json
// @Param id path string true "Item ID"
// @Param body body models.PracticeReportReq true "reason"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /mobile/practice/items/{id}/report [post]
func (h *Handler) ReportPracticeItem(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	var req m.PracticeReportReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.UserId, req.ItemId = userId, c.Param("id")
	if _, err := h.Clients.Practice.Report(c.Request.Context(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "reported"})
}
