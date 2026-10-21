package handler

import (
	"context"
	"net/http"
	"strconv"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// Create godoc
// @Summary Create a new tariff
// @Description Create a new tariff with the provided details
// @Tags tariff
// @Accept  json
// @Produce  json
// @Param tariff body models.TariffCreateReq true "Tariff Details"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/tariff [post]
func (h *Handler) CreateTariff(c *gin.Context) {
	req := m.TariffCreateReq{}
	err := c.BindJSON(&req)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body: ", err)
		return
	}

	slog.Info("Create tariff", "Tariff", req)
	res, err := h.Clients.Tariff.Create(context.Background(), &req)
	slog.Info("Create tariff response", "Tariff", res)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error creating tariff: ", err)
		return
	}

	slog.Info("Tariff created successfully")
	c.JSON(200, gin.H{
		"message": "tariff created successfully",
		"Id":      res.Id,
	})
}

// GetById godoc
// @Summary Get tariff by ID
// @Description Get a tariff by their ID
// @Tags tariff
// @Accept  json
// @Produce  json
// @Param id path string false "Tariff ID"
// @Success 200 {object} models.TariffRes
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/tariff/{id} [get]
// @Router /mobile/tariff/{id} [get]
func (h *Handler) GetByIdTariff(c *gin.Context) {
	id := c.Param("id")

	slog.Info("Get tariff with id", id)
	res, err := h.Clients.Tariff.GetById(context.Background(), &m.ById{Id: id})
	slog.Info("Tariff details", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting tariff by ID: ", err.Error())
		return
	}

	slog.Info("Tariff retrieved successfully")
	c.JSON(200, res)
}

// GetAllTariffs godoc
// @Summary Get all tariff
// @Description Get all tariff with optional filtering
// @Tags tariff
// @Accept  json
// @Produce  json
// @Param duration query int false "Duration"
// @Success 200 {object} models.TariffList
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/tariff [get]
// @Router /mobile/tariff [get]
func (h *Handler) GetAllTariffs(c *gin.Context) {
	duration := c.Query("duration")

	var durationValue int

	if duration != "" {
		parsedDuration, err := strconv.Atoi(duration)
		if err != nil {
			parsedDuration = 0
		}
		durationValue = parsedDuration
	}

	req := &m.TariffGetListReq{
		Duration: durationValue,
	}

	slog.Info("Get all tariff with filtes", req)
	data, _, err := h.Clients.Tariff.GetAll(context.Background(), req)
	slog.Info("All tariff", data)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting tariff: ", err.Error())
		return
	}

	slog.Info("Tariffs retrieved successfully")
	c.JSON(200, data)
}

// Update godoc
// @Summary Update tariff
// @Description Update tariff
// @Tags tariff
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string false "Tariff ID"
// @Param credentials body models.TariffUpdateBody true "Tariff update credentials"
// @Success 200 {object} string "Tariff updated"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/tariff/{id}/update [put]
func (h *Handler) UpdateTariff(c *gin.Context) {
	var req m.TariffUpdateReq
	var body m.TariffUpdateBody

	id := c.Param("id")

	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body", err.Error())
		return
	}

	req.Id = id
	req.Kind = body.Kind
	req.Duration = body.Duration
	req.Coins = body.Coins
	req.Price = body.Price
	req.Name = body.Name
	req.Description = body.Description

	slog.Info("Update Tariff", req)
	_, err := h.Clients.Tariff.Update(context.Background(), &req)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error updating Tariff: ", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tariff updated"})
}

// Delete godoc
// @Summary Delete tariff
// @Description Delete tariff
// @Tags tariff
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Id"
// @Success 200 {object} string "Tariff deleted"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/tariff/{id}/delete [delete]
func (h *Handler) DeleteTariff(c *gin.Context) {
	var req m.ById

	id := c.Param("id")

	req.Id = id

	slog.Info("Tariff delete with id", req)
	_, err := h.Clients.Tariff.Delete(context.Background(), &req)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error deleting tariff", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tariff deleted successfully"})
}
