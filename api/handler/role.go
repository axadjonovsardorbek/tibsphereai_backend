package handler

import (
	"context"
	"net/http"
	"tibsphereai/pkg/errs"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// CreateRole godoc
// @Summary Create a new role
// @Description Admin panel roli (superadmin tizim roli, o'zgartirilmaydi/o'chirilmaydi)
// @Tags role
// @Accept json
// @Produce json
// @Param role body models.RoleCreateReq true "Role details"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/role [post]
func (h *Handler) CreateRole(c *gin.Context) {
	var req m.RoleCreateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}

	res, err := h.Clients.Role.Create(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error creating role: ", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role created successfully", "id": res.Id})
}

// GetByIdRole godoc
// @Summary Get role by ID
// @Tags role
// @Produce json
// @Param id path string true "Role ID"
// @Success 200 {object} models.RoleRes
// @Security BearerAuth
// @Router /web/role/{id} [get]
func (h *Handler) GetByIdRole(c *gin.Context) {
	res, err := h.Clients.Role.GetById(context.Background(), &m.ById{Id: c.Param("id")})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetAllRoles godoc
// @Summary Get all roles
// @Tags role
// @Produce json
// @Param name query string false "Name"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.RoleList
// @Security BearerAuth
// @Router /web/role [get]
func (h *Handler) GetAllRoles(c *gin.Context) {
	pagination := m.ParsePagination(c.Query("limit"), c.Query("page"))
	req := &m.RoleGetListReq{Name: c.Query("name"), Filter: pagination}

	data, _, err := h.Clients.Role.GetAll(context.Background(), req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// UpdateRole godoc
// @Summary Update role
// @Tags role
// @Accept json
// @Produce json
// @Param id path string true "Role ID"
// @Param credentials body models.RoleUpdateReq true "Role update"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/role/{id}/update [put]
func (h *Handler) UpdateRole(c *gin.Context) {
	var req m.RoleUpdateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.Id = c.Param("id")

	if _, err := h.Clients.Role.Update(context.Background(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

// DeleteRole godoc
// @Summary Delete role
// @Tags role
// @Produce json
// @Param id path string true "Id"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/role/{id}/delete [delete]
func (h *Handler) DeleteRole(c *gin.Context) {
	if _, err := h.Clients.Role.Delete(context.Background(), &m.ById{Id: c.Param("id")}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "role deleted successfully"})
}
