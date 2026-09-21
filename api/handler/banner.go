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
// @Summary Create a new banner
// @Description Create a new banner with the provided details
// @Tags banner
// @Accept  json
// @Produce  json
// @Param banner body models.BannerCreateBody true "Banner Details"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/banner [post]
func (h *Handler) CreateBanner(c *gin.Context) {
	req := m.BannerCreateReq{}
	body := m.BannerCreateBody{}
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
	imageurlJson, err := json.Marshal(body.ImageUrl)
	if err != nil {
		errs.Conflict(c, fmt.Errorf("failed to serialize image"))
		slog.Error("Error serializing image: ", err)
		return
	}

	req.Title = string(titleJson)
	req.Description = string(descriptionJson)
	req.ImageUrl = string(imageurlJson)
	req.LinkUrl = body.LinkUrl
	req.OrderNum = body.OrderNum

	slog.Info("Create banner", "Banner", req)
	res, err := h.Clients.Banner.Create(context.Background(), &req)
	slog.Info("Create banner response", "Banner", res)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error creating banner: ", err)
		return
	}

	slog.Info("Banner created successfully")
	c.JSON(200, gin.H{
		"message": "banner created successfully",
		"Id":      res.Id,
	})
}

// GetById godoc
// @Summary Get banner by ID
// @Description Get a banner by their ID
// @Tags banner
// @Accept  json
// @Produce  json
// @Param id path string false "Banner ID"
// @Success 200 {object} models.BannerRes
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/banner/{id} [get]
func (h *Handler) GetByIdBanner(c *gin.Context) {
	id := c.Param("id")

	slog.Info("Get banner with id", id)
	res, err := h.Clients.Banner.GetById(context.Background(), &m.ById{Id: id})
	slog.Info("Banner details", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting banner by ID: ", err.Error())
		return
	}

	slog.Info("Banner retrieved successfully")
	c.JSON(200, res)
}

// GetAllBanners godoc
// @Summary Get all banner
// @Description Get all banner with optional filtering
// @Tags banner
// @Accept  json
// @Produce  json
// @Param title query string false "Title"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.BannerList
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/banner [get]
func (h *Handler) GetAllBanners(c *gin.Context) {
	title := c.Query("title")
	limit := c.Query("limit")
	page := c.Query("page")

	pagination := m.ParsePagination(limit, page)

	req := &m.BannerGetListReq{
		Title:  title,
		Filter: pagination,
	}

	slog.Info("Get all banner with filtes", req)
	data, _, err := h.Clients.Banner.GetAll(context.Background(), req)
	slog.Info("All banner", data)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting banner: ", err.Error())
		return
	}

	slog.Info("Banners retrieved successfully")
	c.JSON(200, data)
}

// Update godoc
// @Summary Update banner
// @Description Update banner
// @Tags banner
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string false "Banner ID"
// @Param credentials body models.BannerUpdateBody true "Banner update credentials"
// @Success 200 {object} string "Banner updated"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/banner/{id}/update [put]
func (h *Handler) UpdateBanner(c *gin.Context) {
	var req m.BannerUpdateReq
	var body m.BannerUpdateBody

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
	imageurlJson, err := json.Marshal(body.ImageUrl)
	if err != nil {
		errs.Conflict(c, fmt.Errorf("failed to serialize image"))
		slog.Error("Error serializing image: ", err)
		return
	}

	req.Id = id
	req.Title = string(titleJson)
	req.Description = string(descriptionJson)
	req.ImageUrl = string(imageurlJson)
	req.LinkUrl = body.LinkUrl
	req.OrderNum = body.OrderNum

	slog.Info("Update Banner", req)
	_, err = h.Clients.Banner.Update(context.Background(), &req)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error updating Banner: ", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "banner updated"})
}

// Delete godoc
// @Summary Delete banner
// @Description Delete banner
// @Tags banner
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Id"
// @Success 200 {object} string "Banner deleted"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/banner/{id}/delete [delete]
func (h *Handler) DeleteBanner(c *gin.Context) {
	var req m.ById

	id := c.Param("id")

	req.Id = id

	slog.Info("Banner delete with id", req)
	_, err := h.Clients.Banner.Delete(context.Background(), &req)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error deleting banner", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "banner deleted successfully"})
}

// GetById godoc
// @Summary Get banner by ID
// @Description Get a banner by their ID
// @Tags banner
// @Accept  json
// @Produce  json
// @Param id path string false "Banner ID"
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.BannerMobileRes
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /mobile/banner/{id} [get]
func (h *Handler) GetByIdMobileBanner(c *gin.Context) {
	id := c.Param("id")

	lang := getLanguage(c)

	slog.Info("Get banner with id", id)
	res, err := h.Clients.Banner.GetByIdMobile(context.Background(), &m.ByIdMobile{Id: id, Language: lang})
	slog.Info("Banner details", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting banner by ID: ", err.Error())
		return
	}

	slog.Info("Banner retrieved successfully")
	c.JSON(200, res)
}

// GetAllBanners godoc
// @Summary Get all banner
// @Description Get all banner with optional filtering
// @Tags banner
// @Accept  json
// @Produce  json
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.BannerMobileList
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /mobile/banner [get]
func (h *Handler) GetAllMobileBanners(c *gin.Context) {
	lang := getLanguage(c)

	req := &m.BannerGetListMobileReq{
		Language: lang,
	}

	slog.Info("Get all banner with filtes", req)
	data, _, err := h.Clients.Banner.GetAllMobile(context.Background(), req)
	slog.Info("All banner", data)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting banner: ", err.Error())
		return
	}

	slog.Info("Banners retrieved successfully")
	c.JSON(200, data)
}
