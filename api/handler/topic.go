package handler

import (
	"context"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// CreateTopic godoc
// @Summary Bo'lim ichidagi mavzu yaratish (TZ 6-bo'lim: Bo'lim -> Mavzu -> Keys)
// @Tags topic
// @Accept json
// @Produce json
// @Param topic body models.TopicCreateReq true "Topic details"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/topic [post]
func (h *Handler) CreateTopic(c *gin.Context) {
	var req m.TopicCreateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	res, err := h.Clients.Topic.Create(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "topic created successfully", "id": res.Id})
}

// GetByIdTopic godoc
// @Tags topic
// @Produce json
// @Param id path string true "Topic ID"
// @Success 200 {object} models.TopicRes
// @Security BearerAuth
// @Router /web/topic/{id} [get]
func (h *Handler) GetByIdTopic(c *gin.Context) {
	res, err := h.Clients.Topic.GetById(context.Background(), &m.ById{Id: c.Param("id")})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetAllTopics godoc
// @Tags topic
// @Produce json
// @Param category_id query string false "Category ID"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.TopicList
// @Security BearerAuth
// @Router /web/topic [get]
func (h *Handler) GetAllTopics(c *gin.Context) {
	pagination := m.ParsePagination(c.Query("limit"), c.Query("page"))
	data, _, err := h.Clients.Topic.GetAll(context.Background(), &m.TopicGetListReq{CategoryId: c.Query("category_id"), Filter: pagination})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// UpdateTopic godoc
// @Tags topic
// @Accept json
// @Produce json
// @Param id path string true "Topic ID"
// @Param topic body models.TopicUpdateReq true "Topic update"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/topic/{id}/update [put]
func (h *Handler) UpdateTopic(c *gin.Context) {
	var req m.TopicUpdateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.Id = c.Param("id")
	if _, err := h.Clients.Topic.Update(context.Background(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "topic updated"})
}

// DeleteTopic godoc
// @Tags topic
// @Produce json
// @Param id path string true "Id"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/topic/{id}/delete [delete]
func (h *Handler) DeleteTopic(c *gin.Context) {
	if _, err := h.Clients.Topic.Delete(context.Background(), &m.ById{Id: c.Param("id")}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "topic deleted successfully"})
}

// GetAllMobileTopics godoc
// @Summary Mavzular ro'yxati (mobile)
// @Tags topic
// @Produce json
// @Param category_id query string false "Category ID"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.TopicMobileList
// @Security BearerAuth
// @Router /mobile/topic [get]
func (h *Handler) GetAllMobileTopics(c *gin.Context) {
	pagination := m.ParsePagination(c.Query("limit"), c.Query("page"))
	req := &m.TopicGetListMobileReq{CategoryId: c.Query("category_id"), Language: getLanguage(c), Filter: pagination}
	data, _, err := h.Clients.Topic.GetAllMobile(context.Background(), req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}
