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
// @Summary Create a new about
// @Description Create a new about with the provided details
// @Tags about
// @Accept  json
// @Produce  json
// @Param about body models.AboutCreateBody true "About Details"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/about [post]
func (h *Handler) CreateAbout(c *gin.Context) {
	req := m.AboutCreateReq{}
	body := m.AboutCreateBody{}
	err := c.BindJSON(&body)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body: ", err)
		return
	}

	titleJson, err := json.Marshal(body.Title)
	if err != nil {
		errs.Conflict(c, fmt.Errorf("failed to serialize title"))
		slog.Error("Error serializing title: ", err)
		return
	}

	descriptionJson, err := json.Marshal(body.Description)
	if err != nil {
		errs.Conflict(c, fmt.Errorf("failed to serialize description"))
		slog.Error("Error serializing description: ", err)
		return
	}

	req.Title = string(titleJson)
	req.Description = string(descriptionJson)
	req.LinkUrl = body.LinkUrl
	req.OrderNum = body.OrderNum

	slog.Info("Create about", "About", req)
	res, err := h.Clients.About.Create(context.Background(), &req)
	slog.Info("Create about response", "About", res)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error creating about: ", err)
		return
	}

	slog.Info("About created successfully")
	c.JSON(200, gin.H{
		"message": "about created successfully",
		"Id":      res.Id,
	})
}

// GetById godoc
// @Summary Get about by ID
// @Description Get a about by their ID
// @Tags about
// @Accept  json
// @Produce  json
// @Param id path string false "About ID"
// @Success 200 {object} models.AboutRes
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/about/{id} [get]
func (h *Handler) GetByIdAbout(c *gin.Context) {
	id := c.Param("id")

	slog.Info("Get about with id", id)
	res, err := h.Clients.About.GetById(context.Background(), &m.ById{Id: id})
	slog.Info("About details", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting about by ID: ", err.Error())
		return
	}

	slog.Info("About retrieved successfully")
	c.JSON(200, res)
}

// GetAllAbouts godoc
// @Summary Get all about
// @Description Get all about with optional filtering
// @Tags about
// @Accept  json
// @Produce  json
// @Param title query string false "Title"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.AboutList
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/about [get]
func (h *Handler) GetAllAbouts(c *gin.Context) {
	title := c.Query("title")
	limit := c.Query("limit")
	page := c.Query("page")

	pagination := m.ParsePagination(limit, page)

	req := &m.AboutGetListReq{
		Title:  title,
		Filter: pagination,
	}

	slog.Info("Get all about with filtes", req)
	data, _, err := h.Clients.About.GetAll(context.Background(), req)
	slog.Info("All about", data)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting about: ", err.Error())
		return
	}

	slog.Info("Abouts retrieved successfully")
	c.JSON(200, data)
}

// Update godoc
// @Summary Update about
// @Description Update about
// @Tags about
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string false "About ID"
// @Param credentials body models.AboutUpdateBody true "About update credentials"
// @Success 200 {object} string "About updated"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/about/{id}/update [put]
func (h *Handler) UpdateAbout(c *gin.Context) {
	var req m.AboutUpdateReq
	var body m.AboutUpdateBody

	id := c.Param("id")

	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body", err.Error())
		return
	}

	titleJson, err := json.Marshal(body.Title)
	if err != nil {
		errs.Conflict(c, fmt.Errorf("failed to serialize title"))
		slog.Error("Error serializing title: ", err)
		return
	}

	descriptionJson, err := json.Marshal(body.Description)
	if err != nil {
		errs.Conflict(c, fmt.Errorf("failed to serialize description"))
		slog.Error("Error serializing description: ", err)
		return
	}

	req.Id = id
	req.Title = string(titleJson)
	req.Description = string(descriptionJson)
	req.LinkUrl = body.LinkUrl
	req.OrderNum = body.OrderNum

	slog.Info("Update About", req)
	_, err = h.Clients.About.Update(context.Background(), &req)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error updating About: ", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "About updated"})
}

// Delete godoc
// @Summary Delete about
// @Description Delete about
// @Tags about
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Id"
// @Success 200 {object} string "About deleted"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/about/{id}/delete [delete]
func (h *Handler) DeleteAbout(c *gin.Context) {
	var req m.ById

	id := c.Param("id")

	req.Id = id

	slog.Info("About delete with id", req)
	_, err := h.Clients.About.Delete(context.Background(), &req)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error deleting about", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "about deleted successfully"})
}

// GetById godoc
// @Summary Get about by ID
// @Description Get a about by their ID
// @Tags about
// @Accept  json
// @Produce  json
// @Param id path string false "About ID"
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.AboutMobileRes
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /mobile/about/{id} [get]
func (h *Handler) GetByIdMobileAbout(c *gin.Context) {
	id := c.Param("id")

	lang := getLanguage(c)

	slog.Info("Get about with id", id)
	res, err := h.Clients.About.GetByIdMobile(context.Background(), &m.ByIdMobile{Id: id, Language: lang})
	slog.Info("About details", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting about by ID: ", err.Error())
		return
	}

	slog.Info("About retrieved successfully")
	c.JSON(200, res)
}

// GetAllAbouts godoc
// @Summary Get all about
// @Description Get all about with optional filtering
// @Tags about
// @Accept  json
// @Produce  json
// @Param title query string false "Title"
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.AboutMobileList
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /mobile/about [get]
func (h *Handler) GetAllMobileAbouts(c *gin.Context) {
	title := c.Query("title")

	lang := getLanguage(c)

	req := &m.AboutGetListMobileReq{
		Title:    title,
		Language: lang,
	}

	slog.Info("Get all about with filtes", req)
	data, _, err := h.Clients.About.GetAllMobile(context.Background(), req)
	slog.Info("All about", data)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting about: ", err.Error())
		return
	}

	slog.Info("Abouts retrieved successfully")
	c.JSON(200, data)
}
