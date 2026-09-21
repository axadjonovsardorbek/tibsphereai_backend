package handler

import (
	"context"
	"fmt"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// CreateCustomPermission godoc
// @Summary Add root or child module
// @Description parent_id bo'sh bo'lsa root modul, berilsa shu modulning bo'lim-moduli (U-Code custom-permission patterni)
// @Tags custom-permission
// @Accept json
// @Produce json
// @Param module body models.CustomPermissionCreateReq true "Module details"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/custom-permission [post]
func (h *Handler) CreateCustomPermission(c *gin.Context) {
	var req m.CustomPermissionCreateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}

	res, err := h.Clients.CustomPermission.Create(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error creating custom permission: ", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "module created successfully", "id": res.Id})
}

// UpdateCustomPermission godoc
// @Summary Update module
// @Tags custom-permission
// @Accept json
// @Produce json
// @Param id path string true "Module ID"
// @Param credentials body models.CustomPermissionUpdateReq true "Module update"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/custom-permission/{id}/update [put]
func (h *Handler) UpdateCustomPermission(c *gin.Context) {
	var req m.CustomPermissionUpdateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.Id = c.Param("id")

	if _, err := h.Clients.CustomPermission.Update(context.Background(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "module updated"})
}

// DeleteCustomPermission godoc
// @Summary Delete module
// @Tags custom-permission
// @Produce json
// @Param id path string true "Module ID"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/custom-permission/{id}/delete [delete]
func (h *Handler) DeleteCustomPermission(c *gin.Context) {
	if _, err := h.Clients.CustomPermission.Delete(context.Background(), &m.ById{Id: c.Param("id")}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "module deleted successfully"})
}

// GetRootCustomPermissions godoc
// @Summary Get root modules
// @Tags custom-permission
// @Produce json
// @Success 200 {object} models.CustomPermissionList
// @Security BearerAuth
// @Router /web/custom-permission [get]
func (h *Handler) GetRootCustomPermissions(c *gin.Context) {
	res, err := h.Clients.CustomPermission.GetRoots(context.Background())
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetChildCustomPermissions godoc
// @Summary Get child modules of a parent module
// @Tags custom-permission
// @Produce json
// @Param id path string true "Parent module ID"
// @Success 200 {object} models.CustomPermissionList
// @Security BearerAuth
// @Router /web/custom-permission/{id}/children [get]
func (h *Handler) GetChildCustomPermissions(c *gin.Context) {
	res, err := h.Clients.CustomPermission.GetChildren(context.Background(), c.Param("id"))
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetAccesses godoc
// @Summary Get modules with a role's current permissions
// @Description role_id majburiy; parent_id berilmasa root modullar, berilsa shu modulning bo'lim-modullari qaytadi
// @Tags custom-permission
// @Produce json
// @Param role_id query string true "Role ID"
// @Param parent_id query string false "Parent module ID"
// @Success 200 {object} models.RoleAccessList
// @Security BearerAuth
// @Router /web/custom-permission/accesses [get]
func (h *Handler) GetAccesses(c *gin.Context) {
	roleId := c.Query("role_id")
	if roleId == "" {
		errs.Bad(c, fmt.Errorf("role_id is required"))
		return
	}

	req := &m.RoleAccessGetReq{RoleId: roleId, ParentId: c.Query("parent_id")}
	res, err := h.Clients.CustomPermission.GetAccesses(context.Background(), req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// UpdateAccesses godoc
// @Summary Grant/revoke a role's permissions for a module
// @Tags custom-permission
// @Accept json
// @Produce json
// @Param body body models.RoleAccessUpsertReq true "Access details"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/custom-permission/accesses [put]
func (h *Handler) UpdateAccesses(c *gin.Context) {
	var req m.RoleAccessUpsertReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}

	if _, err := h.Clients.CustomPermission.UpsertAccess(context.Background(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "accesses updated"})
}
