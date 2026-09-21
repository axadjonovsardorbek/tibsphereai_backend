package handler

import (
	"context"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// Create godoc
// @Summary Create a new appRoute
// @Description Create a new appRoute with the provided details
// @Tags appRoute
// @Accept  json
// @Produce  json
// @Param appRoute body models.AppRouteCreateReq true "AppRoute Details"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/app-route [post]
func (h *Handler) CreateAppRoute(c *gin.Context) {
	req := m.AppRouteCreateReq{}
	err := c.BindJSON(&req)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body: ", err)
		return
	}

	err = validateAppRouteCreate(req)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error from validation")
		return
	}

	slog.Info("Create appRoute", "AppRoute", req)
	res, err := h.Clients.AppRoute.Create(context.Background(), &req)
	slog.Info("Create appRoute response", "AppRoute", res)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error creating appRoute: ", err)
		return
	}

	slog.Info("AppRoute created successfully")
	c.JSON(200, gin.H{
		"message": "app route created successfully",
		"Id":      res.Id,
	})
}

// GetById godoc
// @Summary Get appRoute by ID
// @Description Get a appRoute by their ID
// @Tags appRoute
// @Accept  json
// @Produce  json
// @Param id path string false "AppRoute ID"
// @Success 200 {object} models.AppRouteRes
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/app-route/{id} [get]
// @Router /mobile/app-route/{id} [get]
func (h *Handler) GetByIdAppRoute(c *gin.Context) {
	id := c.Param("id")

	slog.Info("Get appRoute with id", id)
	res, err := h.Clients.AppRoute.GetById(context.Background(), &m.ById{Id: id})
	slog.Info("AppRoute details", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting appRoute by ID: ", err.Error())
		return
	}

	slog.Info("AppRoute retrieved successfully")
	c.JSON(200, res)
}

// GetAllAppRoutes godoc
// @Summary Get all appRoute
// @Description Get all appRoute with optional filtering
// @Tags appRoute
// @Accept  json
// @Produce  json
// @Success 200 {object} models.AppRouteList
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/app-route [get]
// @Router /mobile/app-route [get]
func (h *Handler) GetAllAppRoutes(c *gin.Context) {

	var req m.AppRouteGetListReq

	slog.Info("Get all appRoute with filtes", req)
	data, _, err := h.Clients.AppRoute.GetAll(context.Background(), &req)
	slog.Info("All appRoute", data)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting appRoute: ", err.Error())
		return
	}

	slog.Info("AppRoutes retrieved successfully")
	c.JSON(200, data)
}

// Update godoc
// @Summary Update appRoute
// @Description Update appRoute
// @Tags appRoute
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string false "AppRoute ID"
// @Param credentials body models.AppRouteUpdateBody true "AppRoute update credentials"
// @Success 200 {object} string "AppRoute updated"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/app-route/{id}/update [put]
func (h *Handler) UpdateAppRoute(c *gin.Context) {
	var req m.AppRouteUpdateReq
	var body m.AppRouteUpdateBody

	id := c.Param("id")

	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body", err.Error())
		return
	}

	req.Id = id
	req.CallCenter = body.CallCenter
	req.AppVersion = body.AppVersion
	req.SupportUrl = body.SupportUrl
	req.AppLinks = body.AppLinks
	req.PaymentMinVersion = body.PaymentMinVersion
	req.BuyCourse = body.BuyCourse

	err := validateAppRouteCreate(m.AppRouteCreateReq{
		CallCenter:        req.CallCenter,
		AppVersion:        req.AppVersion,
		SupportUrl:        req.SupportUrl,
		AppLinks:          req.AppLinks,
		PaymentMinVersion: req.PaymentMinVersion,
	})
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error from validation")
		return
	}

	slog.Info("Update AppRoute", req)
	_, err = h.Clients.AppRoute.Update(context.Background(), &req)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error updating AppRoute: ", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "app route updated"})
}

// Delete godoc
// @Summary Delete appRoute
// @Description Delete appRoute
// @Tags appRoute
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Id"
// @Success 200 {object} string "AppRoute deleted"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/app-route/{id}/delete [delete]
func (h *Handler) DeleteAppRoute(c *gin.Context) {
	var req m.ById

	id := c.Param("id")

	req.Id = id

	slog.Info("AppRoute delete with id", req)
	_, err := h.Clients.AppRoute.Delete(context.Background(), &req)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error deleting appRoute", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "app route deleted successfully"})
}
