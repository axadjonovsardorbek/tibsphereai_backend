package handler

import (
	"context"
	"net/http"
	"strconv"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// GetAllCoinTransactions godoc
// @Summary Tangalar savdosi monitoringi - tranzaksiyalar ro'yxati
// @Description point_transactions (kind=coin): coin_purchase, category_unlock, referral_reward, promo_redeem, simulation_finish va h.k.
// @Tags coin-transaction
// @Produce json
// @Param user_id query string false "User ID bo'yicha filter"
// @Param reason query string false "Sabab bo'yicha filter (coin_purchase, referral_reward, ...)"
// @Param from query string false "2024-06-01"
// @Param to query string false "2024-06-30"
// @Param page query int false "Page"
// @Param limit query int false "Limit"
// @Success 200 {object} models.CoinTransactionList
// @Security BearerAuth
// @Router /web/coin-transaction [get]
func (h *Handler) GetAllCoinTransactions(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	req := m.CoinTransactionGetListReq{
		UserId: c.Query("user_id"),
		Reason: c.Query("reason"),
		From:   c.Query("from"),
		To:     c.Query("to"),
		Filter: m.Pagination{Page: page, Limit: limit},
	}

	pagination, res, err := h.Clients.CoinTransaction.GetAll(context.Background(), &req)
	if err != nil {
		errs.Bad(c, err)
		return
	}

	pagination.Data = res
	c.JSON(http.StatusOK, pagination)
}

// GetCoinMonitoringSummary godoc
// @Summary Tangalar savdosi monitoringi - umumiy ko'rinish
// @Description Sotib olingan, ishlab topilgan, sarflangan va aylanmadagi jami tanga
// @Tags coin-transaction
// @Produce json
// @Success 200 {object} models.CoinMonitoringSummaryRes
// @Security BearerAuth
// @Router /web/coin-transaction/summary [get]
func (h *Handler) GetCoinMonitoringSummary(c *gin.Context) {
	res, err := h.Clients.CoinTransaction.GetSummary(context.Background())
	if err != nil {
		errs.Bad(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
