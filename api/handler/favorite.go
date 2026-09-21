package handler

import (
	"context"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// ToggleFavorite godoc
// @Summary Sevimlilarga qo'shish/olib tashlash (BACKEND_STRUCTURE.md 3-bo'lim)
// @Tags favorite
// @Accept json
// @Produce json
// @Param body body object{case_id=string} true "Case ID"
// @Success 200 {object} models.FavoriteToggleRes
// @Security BearerAuth
// @Router /mobile/favorite [post]
func (h *Handler) ToggleFavorite(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}

	var body struct {
		CaseId string `json:"case_id"`
	}
	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		return
	}

	res, err := h.Clients.Favorite.Toggle(context.Background(), &m.FavoriteToggleReq{UserId: userId, CaseId: body.CaseId})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetFavorites godoc
// @Summary Sevimlilar ro'yxati (GET /api/v1/favorites - BACKEND_STRUCTURE.md 3-bo'lim)
// @Tags favorite
// @Produce json
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.CaseMobileList
// @Security BearerAuth
// @Router /mobile/favorite [get]
func (h *Handler) GetFavorites(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	pagination := m.ParsePagination(c.Query("limit"), c.Query("page"))

	data, _, err := h.Clients.Favorite.GetAll(context.Background(), &m.FavoriteGetListReq{
		UserId: userId, Language: getLanguage(c), Filter: pagination,
	})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}
