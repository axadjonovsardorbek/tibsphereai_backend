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

// CreateOrder godoc
// @Summary Premium obunaga buyurtma (BACKEND_STRUCTURE.md 5-bo'lim, Click orqali to'lov)
// @Tags subscription
// @Accept json
// @Produce json
// @Param order body models.OrderCreateBody true "Tariff"
// @Success 200 {object} string
// @Security BearerAuth
// @Router /mobile/subscription [post]
func (h *Handler) CreateOrder(c *gin.Context) {
	var body m.OrderCreateBody
	if err := c.BindJSON(&body); err != nil {
		errs.Bad(c, err)
		return
	}

	userId := claimData(c, "user_id")
	if userId == "" {
		return
	}

	req := m.OrderCreateReq{UserId: userId, TariffId: body.TariffId, CoinsUsed: body.CoinsUsed}

	res, err := h.Clients.Order.Create(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "order created successfully", "order_id": res.Id, "url": res.Url, "just_paid": res.JustPaid,
	})
}

// GetByIdOrder godoc
// @Tags order
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} models.OrderStatisticRes
// @Security BearerAuth
// @Router /web/order/{id} [get]
func (h *Handler) GetByIdOrder(c *gin.Context) {
	res, err := h.Clients.Order.GetById(context.Background(), &m.ById{Id: c.Param("id")})
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"order": res})
}

// GetAllOrders godoc
// @Tags order
// @Produce json
// @Param user_id query string false "User ID"
// @Param tariff_id query string false "Tariff ID"
// @Param status query string false "Order Status" Enums(NEW, PAID, CANCELLED, RESERVED, EXPIRED)
// @Param payment_type query string false "Payment Type" Enums(click, admin)
// @Param type query string false "range, day, week, month, year"
// @Param day query string false "YYYY-MM-DD"
// @Param from query string false "YYYY-MM-DD"
// @Param to query string false "YYYY-MM-DD"
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} models.OrderList
// @Security BearerAuth
// @Router /web/order [get]
func (h *Handler) GetAllOrders(c *gin.Context) {
	status := c.Query("status")
	paymentType := c.Query("payment_type")
	reqType := c.Query("type")
	day := c.Query("day")
	from := c.Query("from")
	to := c.Query("to")
	pagination := m.ParsePagination(c.Query("limit"), c.Query("page"))

	if status != "" && status != "string" {
		switch status {
		case "NEW", "PAID", "CANCELLED", "RESERVED", "EXPIRED":
		default:
			errs.Bad(c, fmt.Errorf("status must be one of NEW, PAID, CANCELLED, RESERVED, EXPIRED"))
			return
		}
	}

	if reqType == "day" && day != "" && !IsYYYYMMDD(day) {
		errs.Bad(c, fmt.Errorf("date must be YYYY-MM-DD"))
		return
	}
	if reqType == "range" && (from == "" || to == "" || !IsYYYYMMDD(from) || !IsYYYYMMDD(to)) {
		errs.Bad(c, fmt.Errorf("from/to are required and must be YYYY-MM-DD"))
		return
	}

	req := m.OrderGetListReq{
		UserId:      c.Query("user_id"),
		TariffId:    c.Query("tariff_id"),
		Status:      status,
		PaymentType: paymentType,
		Type:        reqType,
		Day:         day,
		From:        from,
		To:          to,
		Filter:      pagination,
	}

	res, _, err := h.Clients.Order.GetAll(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		slog.Error("Error getting all orders: ", err)
		return
	}

	c.JSON(http.StatusOK, res)
}
