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
// @Summary Create a new category
// @Description BACKEND_STRUCTURE.md 2-bo'lim: tibbiy yo'nalishlar (Kardiologiya, Terapiya va h.k.)
// @Tags category
// @Accept json
// @Produce json
// @Param category body models.CategoryCreateReq true "Category Details"
// @Success 200 {object} string
// @Failure 400 {object} string
// @Security BearerAuth
// @Router /web/category [post]
func (h *Handler) CreateCategory(c *gin.Context) {
	var req m.CategoryCreateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}

	res, err := h.Clients.Category.Create(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error creating category: ", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "category created successfully", "id": res.Id})
}

// GetById godoc
// @Summary Get category by ID
// @Tags category
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} models.CategoryRes
// @Security BearerAuth
// @Router /web/category/{id} [get]
func (h *Handler) GetByIdCategory(c *gin.Context) {
	res, err := h.Clients.Category.GetById(context.Background(), &m.ById{Id: c.Param("id")})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetAllCategories godoc
// @Summary Get all categories
// @Tags category
// @Produce json
// @Param name query string false "Name"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.CategoryList
// @Security BearerAuth
// @Router /web/category [get]
func (h *Handler) GetAllCategories(c *gin.Context) {
	pagination := m.ParsePagination(c.Query("limit"), c.Query("page"))
	req := &m.CategoryGetListReq{Name: c.Query("name"), Audience: c.Query("audience"), Filter: pagination}

	data, _, err := h.Clients.Category.GetAll(context.Background(), req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// UpdateCategory godoc
// @Summary Update category
// @Tags category
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param credentials body models.CategoryUpdateReq true "Category update"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/category/{id}/update [put]
func (h *Handler) UpdateCategory(c *gin.Context) {
	var req m.CategoryUpdateReq
	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}
	req.Id = c.Param("id")

	if _, err := h.Clients.Category.Update(context.Background(), &req); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "category updated"})
}

// DeleteCategory godoc
// @Summary Delete category
// @Tags category
// @Produce json
// @Param id path string true "Id"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /web/category/{id}/delete [delete]
func (h *Handler) DeleteCategory(c *gin.Context) {
	if _, err := h.Clients.Category.Delete(context.Background(), &m.ById{Id: c.Param("id")}); err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "category deleted successfully"})
}

// GetByIdMobileCategory godoc
// @Summary Get category by ID (mobile)
// @Tags category
// @Produce json
// @Param id path string true "Category ID"
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.CategoryMobileRes
// @Security BearerAuth
// @Router /mobile/category/{id} [get]
func (h *Handler) GetByIdMobileCategory(c *gin.Context) {
	res, err := h.Clients.Category.GetByIdMobile(context.Background(), &m.ByIdMobile{Id: c.Param("id"), Language: getLanguage(c)})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetAllMobileCategories godoc
// @Summary Get all categories (mobile)
// @Tags category
// @Produce json
// @Param Accept-Language header string false "Language (uz|ru|en)" default(uz)
// @Success 200 {object} models.CategoryMobileList
// @Security BearerAuth
// @Router /mobile/category [get]
func (h *Handler) GetAllMobileCategories(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}

	req := &m.CategoryGetListMobileReq{Language: getLanguage(c), UserId: userId, Audience: c.Query("audience")}
	data, _, err := h.Clients.Category.GetAllMobile(context.Background(), req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// PurchaseCategoryUnlock godoc
// @Summary Bo'limni tanga evaziga ochish (Premium obunasiz)
// @Description Bo'limning bitta case narxi (coins_per_case) * case_count miqdorida
// @Description tanga yechiladi va shuncha case ishlash huquqi beriladi.
// @Tags category
// @Accept json
// @Produce json
// @Param id path string true "Category ID"
// @Param body body object{case_count=int} true "Nechta case sotib olinayapti"
// @Success 200 {object} models.CategoryUnlockRes
// @Security BearerAuth
// @Router /mobile/category/{id}/unlock [post]
func (h *Handler) PurchaseCategoryUnlock(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}

	var body struct {
		CaseCount int `json:"case_count"`
	}
	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		return
	}

	req := &m.CategoryUnlockPurchaseReq{UserId: userId, CategoryId: c.Param("id"), CaseCount: body.CaseCount}
	res, err := h.Clients.CategoryUnlock.Purchase(context.Background(), req)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetCategoryUnlockStatus godoc
// @Summary Bo'lim uchun tanga narxi va foydalanuvchining qolgan case-huquqi
// @Tags category
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} models.CategoryUnlockStatusRes
// @Security BearerAuth
// @Router /mobile/category/{id}/unlock [get]
func (h *Handler) GetCategoryUnlockStatus(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}

	res, err := h.Clients.CategoryUnlock.GetStatus(context.Background(), userId, c.Param("id"))
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
