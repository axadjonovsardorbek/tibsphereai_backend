package handler

import (
	"context"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// CreateCase godoc
// @Summary Create clinical case
// @Tags case
// @Accept json
// @Produce json
// @Param case body models.CaseCreateReq true "Case details"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/case [post]
func (h *Handler) CreateCase(c *gin.Context) {
	var req m.CaseCreateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}

	res, err := h.Clients.Case.Create(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "case created successfully", "id": res.Id})
}

// GenerateCaseWithAI godoc
// @Summary AI yordamida keys yaratish (BACKEND_STRUCTURE.md 7-bo'lim)
// @Tags case
// @Accept json
// @Produce json
// @Param body body object{topic=string,difficulty=string} true "Topic and difficulty"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/case/ai-generate [post]
func (h *Handler) GenerateCaseWithAI(c *gin.Context) {
	var body struct {
		Topic      string `json:"topic"`
		Difficulty string `json:"difficulty"`
	}
	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		return
	}

	res, err := h.Clients.Case.GenerateWithAI(context.Background(), body.Topic, body.Difficulty)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "case draft generated, review and publish", "id": res.Id})
}

// PublishCase godoc
// @Summary Publish case (admin ko'rib chiqib tasdiqlaydi)
// @Tags case
// @Produce json
// @Param id path string true "Case ID"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/case/{id}/publish [put]
func (h *Handler) PublishCase(c *gin.Context) {
	if _, err := h.Clients.Case.Publish(context.Background(), &m.ById{Id: c.Param("id")}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "case published"})
}

// GetByIdCase godoc
// @Summary Get case by ID (admin)
// @Tags case
// @Produce json
// @Param id path string true "Case ID"
// @Success 200 {object} models.CaseRes
// @Security BearerAuth
// @Router /web/case/{id} [get]
func (h *Handler) GetByIdCase(c *gin.Context) {
	res, err := h.Clients.Case.GetById(context.Background(), &m.ById{Id: c.Param("id")})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetAllCases godoc
// @Summary Get all cases (admin)
// @Tags case
// @Produce json
// @Param topic_id query string false "Topic ID"
// @Param category_id query string false "Category ID"
// @Param difficulty query string false "easy|medium|hard"
// @Param status query string false "draft|published"
// @Param patient_gender query string false "male|female"
// @Param is_ai_generated query string false "true|false"
// @Param search query string false "Search (title/subtitle/chief_complaint)"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.CaseList
// @Security BearerAuth
// @Router /web/case [get]
func (h *Handler) GetAllCases(c *gin.Context) {
	pagination := m.ParsePagination(c.Query("limit"), c.Query("page"))
	req := &m.CaseGetListReq{
		TopicId:       c.Query("topic_id"),
		CategoryId:    c.Query("category_id"),
		Difficulty:    c.Query("difficulty"),
		Status:        c.Query("status"),
		PatientGender: c.Query("patient_gender"),
		IsAIGenerated: c.Query("is_ai_generated"),
		Search:        c.Query("search"),
		Filter:        pagination,
	}

	data, _, err := h.Clients.Case.GetAll(context.Background(), req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// UpdateCase godoc
// @Summary Update case
// @Tags case
// @Accept json
// @Produce json
// @Param id path string true "Case ID"
// @Param case body models.CaseUpdateReq true "Case update"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/case/{id}/update [put]
func (h *Handler) UpdateCase(c *gin.Context) {
	var req m.CaseUpdateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.Id = c.Param("id")

	if _, err := h.Clients.Case.Update(context.Background(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "case updated"})
}

// DeleteCase godoc
// @Summary Delete case
// @Tags case
// @Produce json
// @Param id path string true "Id"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/case/{id}/delete [delete]
func (h *Handler) DeleteCase(c *gin.Context) {
	if _, err := h.Clients.Case.Delete(context.Background(), &m.ById{Id: c.Param("id")}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "case deleted successfully"})
}

// GetByIdMobileCase godoc
// @Summary Get case detail (mobile)
// @Tags case
// @Produce json
// @Param id path string true "Case ID"
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.CaseMobileRes
// @Security BearerAuth
// @Router /mobile/case/{id} [get]
func (h *Handler) GetByIdMobileCase(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	res, err := h.Clients.Case.GetByIdMobile(context.Background(), &m.ByIdMobile{Id: c.Param("id"), UserId: userId, Language: getLanguage(c)})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetAllMobileCases godoc
// @Summary Get cases catalog (mobile)
// @Tags case
// @Produce json
// @Param topic_id query string false "Topic ID"
// @Param category_id query string false "Category ID"
// @Param difficulty query string false "easy|medium|hard"
// @Param status query string false "new|active|completed|failed|abandoned (foydalanuvchi holati)"
// @Param search query string false "Search"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.CaseMobileList
// @Security BearerAuth
// @Router /mobile/case [get]
func (h *Handler) GetAllMobileCases(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	pagination := m.ParsePagination(c.Query("limit"), c.Query("page"))
	req := &m.CaseGetListMobileReq{
		UserId:     userId,
		Language:   getLanguage(c),
		TopicId:    c.Query("topic_id"),
		CategoryId: c.Query("category_id"),
		Difficulty: c.Query("difficulty"),
		Status:     c.Query("status"),
		Search:     c.Query("search"),
		Filter:     pagination,
	}

	data, _, err := h.Clients.Case.GetAllMobile(context.Background(), req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// GetRandomCase godoc
// @Summary "Try your luck" - tasodifiy keys (BACKEND_STRUCTURE.md 3-bo'lim)
// @Tags case
// @Produce json
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.CaseMobileRes
// @Security BearerAuth
// @Router /mobile/case/random [get]
func (h *Handler) GetRandomCase(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	res, err := h.Clients.Case.GetRandom(context.Background(), &m.CaseRandomReq{
		UserId:     userId,
		Language:   getLanguage(c),
		CategoryId: c.Query("category_id"),
		Difficulty: c.Query("difficulty"),
	})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
