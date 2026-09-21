package handler

import (
	"context"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// CreatePartner godoc
// @Summary Hamkor o'quv markazi qo'shish (BACKEND_STRUCTURE.md 7-bo'lim)
// @Tags partner
// @Accept json
// @Produce json
// @Param partner body models.PartnerCreateReq true "Partner"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/partner [post]
func (h *Handler) CreatePartner(c *gin.Context) {
	var req m.PartnerCreateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	res, err := h.Clients.Partner.Create(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "partner created successfully", "id": res.Id})
}

// GetByIdPartner godoc
// @Tags partner
// @Produce json
// @Param id path string true "Partner ID"
// @Success 200 {object} models.PartnerRes
// @Security BearerAuth
// @Router /web/partner/{id} [get]
func (h *Handler) GetByIdPartner(c *gin.Context) {
	res, err := h.Clients.Partner.GetById(context.Background(), &m.ById{Id: c.Param("id")})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetAllPartners godoc
// @Tags partner
// @Produce json
// @Param name query string false "Name"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.PartnerList
// @Security BearerAuth
// @Router /web/partner [get]
func (h *Handler) GetAllPartners(c *gin.Context) {
	pagination := m.ParsePagination(c.Query("limit"), c.Query("page"))
	data, _, err := h.Clients.Partner.GetAll(context.Background(), &m.PartnerGetListReq{Name: c.Query("name"), Filter: pagination})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// UpdatePartner godoc
// @Tags partner
// @Accept json
// @Produce json
// @Param id path string true "Partner ID"
// @Param partner body models.PartnerUpdateReq true "Partner update"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/partner/{id}/update [put]
func (h *Handler) UpdatePartner(c *gin.Context) {
	var req m.PartnerUpdateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.Id = c.Param("id")
	if _, err := h.Clients.Partner.Update(context.Background(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "partner updated"})
}

// DeletePartner godoc
// @Tags partner
// @Produce json
// @Param id path string true "Id"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/partner/{id}/delete [delete]
func (h *Handler) DeletePartner(c *gin.Context) {
	if _, err := h.Clients.Partner.Delete(context.Background(), &m.ById{Id: c.Param("id")}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "partner deleted successfully"})
}

// GetAllMobilePartners godoc
// @Summary Home ekranidagi "Hamkor kurslari" banneri (TZ 2.1-bo'lim)
// @Tags partner
// @Produce json
// @Success 200 {object} models.PartnerMobileList
// @Security BearerAuth
// @Router /mobile/partner [get]
func (h *Handler) GetAllMobilePartners(c *gin.Context) {
	data, err := h.Clients.Partner.GetAllMobile(context.Background())
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}
