package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type inpayCallbackBody struct {
	Amount        any    `json:"amount"`
	Status        string `json:"status"`
	OrderID       string `json:"order_id"`
	TransactionID any    `json:"transaction_id"`
	CreatedAt     string `json:"created_at"`
}

// InpayCallback godoc
// @Summary inPAY webhook (imzosiz; holat inPAY API orqali qayta tekshiriladi)
// @Tags payment
// @Accept json
// @Produce json
// @Success 200 {object} map[string]any
// @Router /inpay/callback [post]
func (h *Handler) InpayCallback(c *gin.Context) {
	var body inpayCallbackBody
	if err := c.ShouldBindJSON(&body); err != nil || body.OrderID == "" {
		// inPAY 200 bo'lmasa qayta yuboradi - yaroqsiz so'rovni retry qildirmaymiz.
		slog.Warn("inpay callback bad body", "err", err)
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "bad request"})
		return
	}

	var trID int64
	switch v := body.TransactionID.(type) {
	case float64:
		trID = int64(v)
	case string:
		trID, _ = strconv.ParseInt(v, 10, 64)
	}

	res, err := h.Clients.Order.InpayCallback(c.Request.Context(), body.OrderID, trID)
	if err != nil {
		// Vaqtinchalik xato (tarmoq/DB): 500 qaytaramiz - inPAY qayta urinadi.
		slog.Error("inpay callback failed", "order_id", body.OrderID, "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"success": false})
		return
	}

	slog.Info("inpay callback", "order_id", body.OrderID, "status", body.Status, "result", res.ErrorNote)
	c.JSON(http.StatusOK, gin.H{"success": res.Error == 0, "message": res.ErrorNote})
}
