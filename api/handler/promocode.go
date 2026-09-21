package handler

import (
	"context"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// CreatePromoCode godoc
// @Summary Hamkor/admin tanga qiymatli promokod yaratadi (TZ 5-6-bo'lim)
// @Tags promocode
// @Accept json
// @Produce json
// @Param promocode body models.PromoCodeCreateReq true "PromoCode details"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/promocode [post]
func (h *Handler) CreatePromoCode(c *gin.Context) {
	var req m.PromoCodeCreateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	adminId := claimData(c, "user_id")
	if adminId != "" {
		req.CreatedBy = adminId
	}
	res, err := h.Clients.PromoCode.Create(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "promo code created successfully", "id": res.Id})
}

// GetByIdPromoCode godoc
// @Tags promocode
// @Produce json
// @Param id path string true "PromoCode ID"
// @Success 200 {object} models.PromoCodeRes
// @Security BearerAuth
// @Router /web/promocode/{id} [get]
func (h *Handler) GetByIdPromoCode(c *gin.Context) {
	res, err := h.Clients.PromoCode.GetById(context.Background(), &m.ById{Id: c.Param("id")})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetAllPromoCodes godoc
// @Summary Generatsiya qilingan promokodlar ro'yxati (BACKEND_STRUCTURE.md 7-bo'lim)
// @Tags promocode
// @Produce json
// @Param partner_id query string false "Partner ID"
// @Param status query string false "active|used|expired"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.PromoCodeList
// @Security BearerAuth
// @Router /web/promocode [get]
func (h *Handler) GetAllPromoCodes(c *gin.Context) {
	pagination := m.ParsePagination(c.Query("limit"), c.Query("page"))
	data, _, err := h.Clients.PromoCode.GetAll(context.Background(), &m.PromoCodeGetListReq{
		PartnerId: c.Query("partner_id"), Status: c.Query("status"), Filter: pagination,
	})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// DeletePromoCode godoc
// @Tags promocode
// @Produce json
// @Param id path string true "Id"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/promocode/{id}/delete [delete]
func (h *Handler) DeletePromoCode(c *gin.Context) {
	if _, err := h.Clients.PromoCode.Delete(context.Background(), &m.ById{Id: c.Param("id")}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "promo code deleted successfully"})
}

// GetMyPromoCodes godoc
// @Summary Mening Promokodlarim (ishlatganlarim, BACKEND_STRUCTURE.md 5-bo'lim)
// @Tags promocode
// @Produce json
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.PromoCodeList
// @Security BearerAuth
// @Router /mobile/promocode [get]
func (h *Handler) GetMyPromoCodes(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	pagination := m.ParsePagination(c.Query("limit"), c.Query("page"))
	data, _, err := h.Clients.PromoCode.GetMy(context.Background(), &m.PromoCodeGetListReq{UserId: userId, Filter: pagination})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// RedeemPromoCode godoc
// @Summary Promokodni kiritib tanga olish (TZ 2.3-bo'lim)
// @Tags promocode
// @Accept json
// @Produce json
// @Param body body object{code=string} true "Promo code"
// @Success 200 {object} models.PromoCodeRedeemRes
// @Security BearerAuth
// @Router /mobile/promocode/redeem [post]
func (h *Handler) RedeemPromoCode(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		return
	}
	res, err := h.Clients.PromoCode.Redeem(context.Background(), &m.PromoCodeRedeemReq{Code: body.Code, UserId: userId})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
