package handler

import (
	"context"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// CreateLevel godoc
// @Summary Level qo'shish (statik jadval)
// @Tags level
// @Accept json
// @Produce json
// @Param level body models.LevelCreateReq true "Level"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/level [post]
func (h *Handler) CreateLevel(c *gin.Context) {
	var req m.LevelCreateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	res, err := h.Clients.Level.Create(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "level created successfully", "id": res.Id})
}

// GetAllLevels godoc
// @Tags level
// @Produce json
// @Success 200 {array} models.LevelRes
// @Security BearerAuth
// @Router /web/level [get]
func (h *Handler) GetAllLevels(c *gin.Context) {
	data, err := h.Clients.Level.GetAll(context.Background())
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// UpdateLevel godoc
// @Tags level
// @Accept json
// @Produce json
// @Param id path string true "Level ID"
// @Param level body models.LevelUpdateReq true "Level update"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/level/{id}/update [put]
func (h *Handler) UpdateLevel(c *gin.Context) {
	var req m.LevelUpdateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.Id = c.Param("id")
	if _, err := h.Clients.Level.Update(context.Background(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "level updated"})
}

// DeleteLevel godoc
// @Tags level
// @Produce json
// @Param id path string true "Id"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/level/{id}/delete [delete]
func (h *Handler) DeleteLevel(c *gin.Context) {
	if _, err := h.Clients.Level.Delete(context.Background(), &m.ById{Id: c.Param("id")}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "level deleted successfully"})
}
