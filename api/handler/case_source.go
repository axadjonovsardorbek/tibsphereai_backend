package handler

import (
	"errors"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// CreateCaseSource godoc
// @Summary AI case generatsiyasi uchun ishonchli manba qo'shish (havola yoki matn)
// @Tags case-source
// @Accept json
// @Produce json
// @Param body body models.CaseSourceCreateReq true "Source"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/case-source [post]
func (h *Handler) CreateCaseSource(c *gin.Context) {
	var req m.CaseSourceCreateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	if req.Name == "" || req.Content == "" {
		errs.Bad(c, errors.New("name and content are required"))
		return
	}
	res, err := h.Clients.CaseSource.Create(c.Request.Context(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "case source created successfully", "id": res.Id})
}

// GetByIdCaseSource godoc
// @Tags case-source
// @Produce json
// @Param id path string true "Source ID"
// @Success 200 {object} models.CaseSourceRes
// @Security BearerAuth
// @Router /web/case-source/{id} [get]
func (h *Handler) GetByIdCaseSource(c *gin.Context) {
	res, err := h.Clients.CaseSource.GetById(c.Request.Context(), &m.ById{Id: c.Param("id")})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetAllCaseSources godoc
// @Tags case-source
// @Produce json
// @Param name query string false "Name"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.CaseSourceList
// @Security BearerAuth
// @Router /web/case-source [get]
func (h *Handler) GetAllCaseSources(c *gin.Context) {
	pagination := m.ParsePagination(c.Query("limit"), c.Query("page"))
	data, _, err := h.Clients.CaseSource.GetAll(c.Request.Context(), &m.CaseSourceGetListReq{Name: c.Query("name"), Filter: pagination})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// UpdateCaseSource godoc
// @Tags case-source
// @Accept json
// @Produce json
// @Param id path string true "Source ID"
// @Param body body models.CaseSourceUpdateReq true "Source update"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/case-source/{id}/update [put]
func (h *Handler) UpdateCaseSource(c *gin.Context) {
	var req m.CaseSourceUpdateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.Id = c.Param("id")
	if _, err := h.Clients.CaseSource.Update(c.Request.Context(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "case source updated"})
}

// DeleteCaseSource godoc
// @Tags case-source
// @Produce json
// @Param id path string true "Id"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/case-source/{id}/delete [delete]
func (h *Handler) DeleteCaseSource(c *gin.Context) {
	if _, err := h.Clients.CaseSource.Delete(c.Request.Context(), &m.ById{Id: c.Param("id")}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "case source deleted successfully"})
}
