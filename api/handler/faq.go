package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// Create godoc
// @Summary Create a new faq
// @Description Create a new faq with the provided details
// @Tags faq
// @Accept  json
// @Produce  json
// @Param faq body models.FaqCreateBody true "Faq Details"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/faq [post]
func (h *Handler) CreateFaq(c *gin.Context) {
	req := m.FaqCreateReq{}
	body := m.FaqCreateBody{}
	err := c.BindJSON(&body)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body: ", err)
		return
	}

	questionJSON, err := json.Marshal(body.Question)
	if err != nil {
		errs.Conflict(c, fmt.Errorf("failed to serialize question"))
		slog.Error("Error serializing question: ", err)
		return
	}

	answerJSON, err := json.Marshal(body.Answer)
	if err != nil {
		errs.Conflict(c, fmt.Errorf("failed to serialize answer"))
		slog.Error("Error serializing answer: ", err)
		return
	}

	req.Question = string(questionJSON)
	req.Answer = string(answerJSON)
	req.OrderNum = body.OrderNum

	slog.Info("Create faq", "Faq", req)
	res, err := h.Clients.Faq.Create(context.Background(), &req)
	slog.Info("Create faq response", "Faq", res)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error creating faq: ", err)
		return
	}

	slog.Info("Faq created successfully")
	c.JSON(200, gin.H{
		"message": "faq created successfully",
		"Id":      res.Id,
	})
}

// GetById godoc
// @Summary Get faq by ID
// @Description Get a faq by their ID
// @Tags faq
// @Accept  json
// @Produce  json
// @Param id path string false "Faq ID"
// @Success 200 {object} models.FaqRes
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/faq/{id} [get]
func (h *Handler) GetByIdFaq(c *gin.Context) {
	id := c.Param("id")

	slog.Info("Get faq with id", id)
	res, err := h.Clients.Faq.GetById(context.Background(), &m.ById{Id: id})
	slog.Info("Faq details", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting faq by ID: ", err.Error())
		return
	}

	slog.Info("Faq retrieved successfully")
	c.JSON(200, res)
}

// GetAllFaqs godoc
// @Summary Get all faq
// @Description Get all faq with optional filtering
// @Tags faq
// @Accept  json
// @Produce  json
// @Param question query string false "Question"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.FaqList
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/faq [get]
func (h *Handler) GetAllFaqs(c *gin.Context) {
	question := c.Query("question")
	limit := c.Query("limit")
	page := c.Query("page")

	pagination := m.ParsePagination(limit, page)

	req := &m.FaqGetListReq{
		Question: question,
		Filter:   pagination,
	}

	slog.Info("Get all faq with filtes", req)
	data, _, err := h.Clients.Faq.GetAll(context.Background(), req)
	slog.Info("All faq", data)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting faq: ", err.Error())
		return
	}

	slog.Info("Faqs retrieved successfully")
	c.JSON(200, data)
}

// Update godoc
// @Summary Update faq
// @Description Update faq
// @Tags faq
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string false "Faq ID"
// @Param credentials body models.FaqUpdateBody true "Faq update credentials"
// @Success 200 {object} string "Faq updated"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/faq/{id}/update [put]
func (h *Handler) UpdateFaq(c *gin.Context) {
	var req m.FaqUpdateReq
	var body m.FaqUpdateBody

	id := c.Param("id")

	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body", err.Error())
		return
	}

	questionJSON, err := json.Marshal(body.Question)
	if err != nil {
		errs.Conflict(c, fmt.Errorf("failed to serialize question"))
		slog.Error("Error serializing question: ", err)
		return
	}

	answerJSON, err := json.Marshal(body.Answer)
	if err != nil {
		errs.Conflict(c, fmt.Errorf("failed to serialize answer"))
		slog.Error("Error serializing answer: ", err)
		return
	}

	req.Id = id
	req.Question = string(questionJSON)
	req.Answer = string(answerJSON)
	req.OrderNum = body.OrderNum

	slog.Info("Update Faq", req)
	_, err = h.Clients.Faq.Update(context.Background(), &req)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error updating Faq: ", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "faq updated"})
}

// Delete godoc
// @Summary Delete faq
// @Description Delete faq
// @Tags faq
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Id"
// @Success 200 {object} string "Faq deleted"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/faq/{id}/delete [delete]
func (h *Handler) DeleteFaq(c *gin.Context) {
	var req m.ById

	id := c.Param("id")

	req.Id = id

	slog.Info("Faq delete with id", req)
	_, err := h.Clients.Faq.Delete(context.Background(), &req)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error deleting faq", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "faq deleted successfully"})
}

// GetById godoc
// @Summary Get faq by ID
// @Description Get a faq by their ID
// @Tags faq
// @Accept  json
// @Produce  json
// @Param id path string false "Faq ID"
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.FaqMobileRes
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /mobile/faq/{id} [get]
func (h *Handler) GetByIdMobileFaq(c *gin.Context) {
	id := c.Param("id")
	lang := getLanguage(c)

	slog.Info("Get faq with id", id)
	res, err := h.Clients.Faq.GetByIdMobile(context.Background(), &m.ByIdMobile{Id: id, Language: lang})
	slog.Info("Faq details", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting faq by ID: ", err.Error())
		return
	}

	slog.Info("Faq retrieved successfully")
	c.JSON(200, res)
}

// GetAllFaqs godoc
// @Summary Get all faq
// @Description Get all faq with optional filtering
// @Tags faq
// @Accept  json
// @Produce  json
// @Param question query string false "Question"
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.FaqMobileList
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /mobile/faq [get]
func (h *Handler) GetAllMobileFaqs(c *gin.Context) {
	question := c.Query("question")

	lang := getLanguage(c)

	req := &m.FaqGetListMobileReq{
		Question: question,
		Language: lang,
	}

	slog.Info("Get all faq with filtes", req)
	data, _, err := h.Clients.Faq.GetAllMobile(context.Background(), req)
	slog.Info("All faq", data)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting faq: ", err.Error())
		return
	}

	slog.Info("Faqs retrieved successfully")
	c.JSON(200, data)
}
