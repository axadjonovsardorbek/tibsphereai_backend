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

// GetById godoc
// @Summary Get admin by ID
// @Description Get a admin by their ID
// @Tags admin
// @Accept  json
// @Produce  json
// @Param id path string false "Admin ID"
// @Success 200 {object} models.AdminRes
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/admin/{id} [get]
func (h *Handler) GetByIdAdmin(c *gin.Context) {
	id := c.Param("id")

	slog.Info("Get admin with id", id)
	res, err := h.Clients.Admin.GetById(context.Background(), &m.ById{Id: id})
	slog.Info("Admin details", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting admin by ID: ", err.Error())
		return
	}

	slog.Info("admin retrieved successfully")
	c.JSON(200, res)
}

// CreateAdmin godoc
// @Summary Create admin/employee
// @Description Yangi admin panel foydalanuvchisi (xodim) qo'shish, rol biriktirish
// @Tags admin
// @Accept json
// @Produce json
// @Param admin body models.AdminCreateReq true "Admin details"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/admin [post]
func (h *Handler) CreateAdmin(c *gin.Context) {
	var req m.AdminCreateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body", err)
		return
	}

	res, err := h.Clients.Admin.Create(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error creating admin: ", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "admin created successfully", "id": res.Id})
}

// GetAllAdmins godoc
// @Summary Get all admins
// @Tags admin
// @Produce json
// @Param login query string false "Login"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.AdminList
// @Security BearerAuth
// @Router /web/admin [get]
func (h *Handler) GetAllAdmins(c *gin.Context) {
	pagination := m.ParsePagination(c.Query("limit"), c.Query("page"))
	req := &m.AdminGetListReq{Login: c.Query("login"), Filter: pagination}

	data, _, err := h.Clients.Admin.GetAll(context.Background(), req)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting admins: ", err.Error())
		return
	}

	c.JSON(http.StatusOK, data)
}

// UpdateAdmin godoc
// @Summary Update admin
// @Description login/rol va (ixtiyoriy) parolni yangilaydi
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "Admin ID"
// @Param admin body models.AdminUpdateReq true "Admin update"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/admin/{id}/update [put]
func (h *Handler) UpdateAdmin(c *gin.Context) {
	var req m.AdminUpdateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.Id = c.Param("id")

	if _, err := h.Clients.Admin.Update(context.Background(), &req); err != nil {
		errs.Bad(c, err)
		slog.Error("Error updating admin: ", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "admin updated"})
}

// DeleteAdmin godoc
// @Summary Delete admin
// @Tags admin
// @Produce json
// @Param id path string true "Admin ID"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/admin/{id}/delete [delete]
func (h *Handler) DeleteAdmin(c *gin.Context) {
	adminId := claimData(c, "user_id")
	if adminId != "" && adminId == c.Param("id") {
		errs.Bad(c, fmt.Errorf("cannot delete your own account while logged in"))
		return
	}

	if _, err := h.Clients.Admin.Delete(context.Background(), &m.ById{Id: c.Param("id")}); err != nil {
		errs.Bad(c, err)
		slog.Error("Error deleting admin: ", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "admin deleted successfully"})
}

// GetProfileAdmin godoc
// @Summary Get my admin profile
// @Description Tokendagi id bo'yicha o'z profilini olish
// @Tags admin
// @Produce json
// @Success 200 {object} models.AdminRes
// @Security BearerAuth
// @Router /web/admin/profile [get]
func (h *Handler) GetProfileAdmin(c *gin.Context) {
	adminId := claimData(c, "user_id")
	if adminId == "" {
		return
	}

	res, err := h.Clients.Admin.GetById(context.Background(), &m.ById{Id: adminId})
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting admin profile: ", err.Error())
		return
	}

	c.JSON(http.StatusOK, res)
}

// Login godoc
// @Summary Login admin
// @Description Authenticate admin with login and password
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body models.LoginReq true "Admin login credentials"
// @Success 200 {object} models.TokenRes "JWT tokens"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/auth/admin/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req m.LoginReq

	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body", err)
		return
	}

	slog.Info("Admin login credentials ", req)
	res, err := h.Clients.Admin.Login(context.Background(), &req)
	slog.Info("Admin login response", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error while login", err)
		return
	}

	c.JSON(http.StatusOK, res)
}
