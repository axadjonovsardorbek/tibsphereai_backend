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

// ChangePassword godoc
// @Summary Change password
// @Description Change password
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param credentials body models.ChangePasswordBody true "Change password credentials"
// @Success 200 {object} string "Password changed successfully"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/auth/password/change [put]
func (h *Handler) ChangePasswordWeb(c *gin.Context) {
	var req m.ChangePasswordReq
	var body m.ChangePasswordBody

	user_id := claimData(c, "user_id")
	fmt.Println("userId", user_id)
	if user_id == "" {
		return
	}
	role := claimData(c, "role")
	fmt.Println("role", role)
	if role == "" {
		return
	}

	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body", err)
		return
	}

	if body.OldPassword == body.NewPassword || body.OldPassword == body.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "the old password must not be the same as the new password.",
		})
		slog.Error("Error changing password")
		return
	}
	if body.NewPassword != body.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "the new password must not be the different as the confirm password.",
		})
		slog.Error("Error changing password")
		return
	}

	req.NewPassword = body.NewPassword
	req.OldPassword = body.OldPassword
	req.Id = user_id
	req.Role = role

	slog.Info("Change password credentials ", req)
	res, err := h.Clients.User.ChangePassword(context.Background(), &req)
	slog.Info("Change password response", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error while check", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed successfully"})
}

// ChangePassword godoc
// @Summary Change password
// @Description Change password
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param credentials body models.ChangePasswordBody true "Change password credentials"
// @Success 200 {object} string "Password changed successfully"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// RefreshPassword godoc
// @Summary Refresh password
// @Description Refresh password
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param credentials body models.RefreshPasswordReq true "Refresh password credentials"
// @Success 200 {object} models.RefreshPassword
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/auth/password/refresh [put]
func (h *Handler) RefreshPassword(c *gin.Context) {

	var req m.RefreshPasswordReq

	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body", err)
		return
	}

	slog.Info("Refresh password", req.UserId)
	res, err := h.Clients.User.RefreshPassword(context.Background(), &req)
	slog.Info("Refreshed password", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error refreshing password: ", err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// Check godoc
// @Summary Check user
// @Description Check user
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param credentials body models.UserCheckReq true "User check credentials"
// @Success 200 {object} models.UserCheckRes
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /mobile/auth/user/check [post]
func (h *Handler) CheckUser(c *gin.Context) {
	var req m.UserCheckReq

	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body", err)
		return
	}

	check := isValidIdentifier(req.Identifier)
	if !check {
		errs.Bad(c, fmt.Errorf("identifier: email yoki telefon raqam formati noto'g'ri (\"%s\") — masalan: user@example.com yoki +998901234567", req.Identifier))
		slog.Error("Error binding request body: Invalid identifier")
		return
	}

	slog.Info("User check credentials ", req)
	res, err := h.Clients.User.Check(context.Background(), &req)
	slog.Info("User check response", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error while check", err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetReferralInfo godoc
// @Summary Referral ma'lumotlari (do'stlarni taklif qilish)
// @Description O'z referral_code'i, taklif qilganlar soni va ular evaziga olingan tangalar
// @Tags user
// @Produce json
// @Success 200 {object} models.ReferralInfoRes
// @Security BearerAuth
// @Router /mobile/referral [get]
func (h *Handler) GetReferralInfo(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		errs.Bad(c, fmt.Errorf("unauthorized"))
		return
	}

	res, err := h.Clients.User.GetReferralInfo(context.Background(), userId)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error while getting referral info", err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// GoogleLogin godoc
// @Summary Google orqali kirish (1 bosqichli)
// @Description BACKEND_STRUCTURE.md 1-bo'lim: Google ID token orqali bir bosqichli kirish
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body models.GoogleLoginReq true "Google ID token"
// @Success 200 {object} models.TokenRes
// @Failure 400 {object} string
// @Router /mobile/auth/google [post]
func (h *Handler) GoogleLogin(c *gin.Context) {
	var req m.GoogleLoginReq

	if err := c.BindJSON(&req); err != nil {
		errs.Bad(c, err)
		return
	}

	res, err := h.Clients.User.GoogleLogin(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error while google login", err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetById godoc
// @Summary Get user by ID
// @Description Get an user by their ID
// @Tags user
// @Accept  json
// @Produce  json
// @Param id path string false "User ID"
// @Success 200 {object} models.UserRes
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/user/{id} [get]
func (h *Handler) GetByIdUser(c *gin.Context) {
	id := c.Param("id")

	slog.Info("Get user with id", id)
	res, err := h.Clients.User.GetById(context.Background(), &m.ById{Id: id})
	slog.Info("User details", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting user by ID: ", err.Error())
		return
	}

	slog.Info("User retrieved successfully")
	c.JSON(200, res)
}

// GetAllUsers godoc
// @Summary Get all user
// @Description Get all user with optional filtering
// @Tags user
// @Accept  json
// @Produce  json
// @Param name query string false "Name"
// @Param phone_number query string false "PhoneNumber"
// @Param limit query int false "Limit"
// @Param page query int false "Page"
// @Success 200 {object} models.UserList
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /web/user [get]
func (h *Handler) GetAllUsers(c *gin.Context) {
	name := c.Query("name")
	phone_number := c.Query("phone_number")
	limit := c.Query("limit")
	page := c.Query("page")

	pagination := m.ParsePagination(limit, page)

	req := &m.UserGetListReq{
		Name:        name,
		PhoneNumber: phone_number,
		Filter:      pagination,
	}

	slog.Info("Get all user with filtes", req)
	data, _, err := h.Clients.User.GetAll(context.Background(), req)
	slog.Info("All user", data)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting user: ", err.Error())
		return
	}

	slog.Info("Users retrieved successfully")
	c.JSON(200, data)
}

// Delete godoc
// @Summary Delete user
// @Description Delete user
// @Tags user
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Id"
// @Success 200 {object} string "User deleted"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /web/user/{id}/delete [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	var req m.ById

	id := c.Param("id")

	req.Id = id

	slog.Info("User delete with id", req)
	_, err := h.Clients.User.Delete(context.Background(), &req)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error deleting user", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

// GetProfile godoc
// @Summary Get user profile
// @Description Get user profile
// @Tags profile
// @Accept  json
// @Produce  json
// @Success 200 {object} models.UserRes
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Security BearerAuth
// @Router /mobile/user/get/profile [get]
func (h *Handler) GetProfileUser(c *gin.Context) {

	user_id := claimData(c, "user_id")
	if user_id == "" {
		return
	}

	slog.Info("Get user with id", user_id)
	res, err := h.Clients.User.GetById(context.Background(), &m.ById{Id: user_id})
	slog.Info("User details", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting user by ID: ", err.Error())
		return
	}

	slog.Info("user retrieved successfully")
	c.JSON(200, res)
}

// GetUserLimit godoc
// @Summary Kunlik bepul urinish limiti holati (home ekranidagi "35/50 Limit" belgisi)
// @Tags profile
// @Produce json
// @Success 200 {object} models.UserLimitStatusRes
// @Security BearerAuth
// @Router /mobile/user/limit [get]
func (h *Handler) GetUserLimit(c *gin.Context) {
	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}
	res, err := h.Clients.AppSetting.GetLimitStatus(context.Background(), userId)
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// UpdateProfile godoc
// @Summary Update a user profile
// @Description Updates user profile fields via multipart/form-data. Image is optional.
// @Tags profile
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param name formData string false "User name"
// @Param phone_number formData string false "Phone number"
// @Param email formData string false "Email"
// @Param language formData string false "Language (e.g. uz, ru, en)"
// @Param image formData file false "Profile image (jpg/png/webp)"
// @Success 200 {object} string "Profile updated"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /mobile/user/update/profile [put]
func (h *Handler) UpdateProfileUser(c *gin.Context) {

	req := m.UserUpdateReq{
		Name:           formStr(c, "name"),
		PhoneNumber:    formStr(c, "phone_number"),
		Language:       formStr(c, "language"),
		Email:          formStr(c, "email"),
		Specialization: formStr(c, "specialization"),
	}

	user_id := claimData(c, "user_id")
	if user_id == "" {
		return
	}

	req.Id = user_id

	if req.PhoneNumber != "" {
		check := isValidPhone(req.PhoneNumber)
		if !check {
			errs.Bad(c, fmt.Errorf("phone_number: telefon raqam noto'g'ri formatda (\"%s\") — to'g'ri format: +998901234567 yoki 901234567", req.PhoneNumber))
			slog.Error("Error binding request body: Phone number is not valid")
			return
		}
	}

	fileName, tempFilePath, err := formFile(c, "image")
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error processing image: ", err)
		return
	}

	if tempFilePath != "" && fileName != "" {
		minioURL, err := h.MinIO.Upload(fileName, tempFilePath)
		if err != nil {
			errs.Bad(c, fmt.Errorf("failed to upload to MinIO"))
			return
		}
		req.ImageUrl = minioURL
	}

	slog.Info("Update User Profile", req)
	_, err = h.Clients.User.Update(context.Background(), &req)
	slog.Info("User profile updated")

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error updating user profile: ", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}

// Delete godoc
// @Summary Delete a profile
// @Description Delete a profile
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} string "Profile deleted"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /mobile/user/delete/profile [delete]
func (h *Handler) DeleteProfileUser(c *gin.Context) {
	var req m.ById

	user_id := claimData(c, "user_id")
	if user_id == "" {
		return
	}

	req.Id = user_id

	slog.Info("User delete profile with id", req)
	_, err := h.Clients.User.Delete(context.Background(), &req)
	slog.Info("User profile deleted")

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error deleting user profile", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile deleted successfully"})
}

// Activity godoc
// @Summary Activity user
// @Description Activity user
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param credentials body models.UserActivityCreateBody true "User activity credentials"
// @Success 200 {object} string "Activity created"
// @Failure 400 {object} string "Invalid request payload"
// @Failure 500 {object} string "Server error"
// @Router /mobile/user/activity [post]
func (h *Handler) CreateActivity(c *gin.Context) {
	var req m.UserActivityCreateReq
	var body m.UserActivityCreateBody

	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		slog.Error("Error binding request body", err)
		return
	}

	user_id := claimData(c, "user_id")
	if user_id == "" {
		return
	}

	req.UserId = user_id
	req.Activity = body.Activity

	slog.Info("Create activity ", req)
	res, err := h.Clients.User.CreateActivity(context.Background(), &req)
	slog.Info("Create activity response", res)

	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error while create activity", err)
		return
	}

	c.JSON(http.StatusOK, res)
}

// GetActivityStats godoc
// @Summary Get user activity stats
// @Description Returns user activity stats by type: day|week|month|year|range. For day: optional ?date=YYYY-MM-DD. For range: required ?from=YYYY-MM-DD&to=YYYY-MM-DD
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param type query string true "Stats type: day|week|month|year|range"
// @Param date query string false "For day only: YYYY-MM-DD (default: today)"
// @Param from query string false "For range only: YYYY-MM-DD"
// @Param to query string false "For range only: YYYY-MM-DD"
// @Success 200 {object} models.ActivityStatsResponse "Activity stats"
// @Failure 400 {object} string "Invalid request params"
// @Failure 500 {object} string "Server error"
// @Router /mobile/user/activity [get]
func (h *Handler) GetActivityStats(c *gin.Context) {
	userID := claimData(c, "user_id")
	if userID == "" {
		return
	}

	typ := c.Query("type")
	date := c.Query("date")
	from := c.Query("from")
	to := c.Query("to")

	if typ == "" {
		errs.Bad(c, fmt.Errorf("type is required"))
		return
	}

	req := &m.UserActivityGetListReq{
		UserId: userID,
		Type:   typ,
	}

	if typ == "day" {
		if date != "" && !IsYYYYMMDD(date) {
			slog.Error("Invalid date format for day type")
			errs.Bad(c, fmt.Errorf("date must be YYYY-MM-DD"))
			return
		}
		req.Date = date
	}

	if typ == "range" {
		if from == "" || to == "" {
			slog.Error("from and to are required for range type")
			errs.Bad(c, fmt.Errorf("from and to are required for range"))
			return
		}
		if !IsYYYYMMDD(from) || !IsYYYYMMDD(to) {
			slog.Error("Invalid from/to format for range type")
			errs.Bad(c, fmt.Errorf("from/to must be YYYY-MM-DD"))
			return
		}

		req.FromDate = from
		req.ToDate = to
	}

	slog.Info("Get activity stats request", "req", req)
	statsRes, err := h.Clients.User.GetActivities(context.Background(), req)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error while get activity stats", err)
		return
	}

	c.JSON(http.StatusOK, statsRes)
}

// GetRating godoc
// @Summary Reyting (leaderboard) olish
// @Description Reytingni qaytaradi: day|week|month|year|total. Default: total. 0 activity bo'lgan userlar avtomatik oxirida turadi. Response ichida TOP list va "Me" (mening o'rnim) qaytadi.
// @Tags profile
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param type query string false "Filter type: day|week|month|year|total (default: total)" Enums(day,week,month,year,total)
// @Param limit query int false "Top list limit (default: 10, max: 100)"
// @Success 200 {object} models.RatingResponse "Rating response"
// @Failure 400 {object} string "Invalid request params"
// @Failure 500 {object} string "Server error"
// @Router /mobile/user/rating [get]
func (h *Handler) GetRating(c *gin.Context) {
	limit := c.Query("limit")
	typ := c.Query("type")

	userID := claimData(c, "user_id")
	if userID == "" {
		return
	}

	if typ == "" {
		typ = "total"
	}

	switch typ {
	case "day", "week", "month", "year", "total":
	default:
		slog.Error("Invalid type parameter for rating")
		errs.Bad(c, fmt.Errorf("type must be one of: day|week|month|year|total"))
		return
	}

	pagination := m.ParsePagination(limit, "0")

	req := m.RatingRequest{
		UserId: userID,
		Type:   m.RatingType(typ),
		Limit:  int(pagination.Limit),
	}

	res, err := h.Clients.User.GetRating(context.Background(), &req)
	if err != nil {
		slog.Error("Error while get rating", err)
		errs.Bad(c, err)
		return
	}

	c.JSON(http.StatusOK, res)
}
