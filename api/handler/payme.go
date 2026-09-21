package handler

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// PaymeRPC - Payme checkout.paycom.uz webhook (JSON-RPC 2.0, bitta endpoint: POST /payme).
// Referens: furasentr-payment/internal/api/handlers/payme.go, MedicAI'ga moslashtirilgan
// (marketplace order emas, faqat tarif-obuna - shuning uchun ancha sodda: account.order_id
// bizning orders.id, fulfillment = subscriptions yaratish, xuddi Click/ApplyComplete kabi).
func (h *Handler) PaymeRPC(c *gin.Context) {
	var req m.PaymeRequest
	if err := c.BindJSON(&req); err != nil {
		h.paymeSendError(c, nil, m.PaymeErrorInternal, "Invalid JSON-RPC request", "")
		return
	}

	if !h.paymeAuthenticate(c) {
		h.paymeSendError(c, req.ID, m.PaymeErrorAuth, "Authentication failed", "")
		return
	}

	switch req.Method {
	case "CheckPerformTransaction":
		h.paymeCheckPerformTransaction(c, req)
	case "CreateTransaction":
		h.paymeCreateTransaction(c, req)
	case "PerformTransaction":
		h.paymePerformTransaction(c, req)
	case "CancelTransaction":
		h.paymeCancelTransaction(c, req)
	case "CheckTransaction":
		h.paymeCheckTransaction(c, req)
	case "GetStatement":
		h.paymeGetStatement(c, req)
	default:
		h.paymeSendError(c, req.ID, m.PaymeErrorMethodNotFound, "Method not found", "")
	}
}

// paymeAuthenticate - Basic auth: "Paycom:<PAYME_SECRET_KEY>" base64. Faqat parol (secret key)
// solishtiriladi - Payme ba'zan merchant_id'ni username sifatida yuborishi mumkin.
func (h *Handler) paymeAuthenticate(c *gin.Context) bool {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Basic ") {
		return false
	}
	payload, err := base64.StdEncoding.DecodeString(authHeader[len("Basic "):])
	if err != nil {
		return false
	}
	pair := strings.SplitN(string(payload), ":", 2)
	if len(pair) != 2 {
		return false
	}
	return pair[1] != "" && pair[1] == h.Cfg.PAYME_SECRET_KEY
}

func (h *Handler) paymeCheckPerformTransaction(c *gin.Context, req m.PaymeRequest) {
	account := req.GetAccount()
	orderId, _ := account["order_id"].(string)
	amount := req.GetParamInt64("amount")

	if orderId == "" {
		h.paymeSendError(c, req.ID, m.PaymeErrorAccountNotFound, "order_id missing", "account")
		return
	}

	if err := h.Clients.Order.PaymeCheckPerform(context.Background(), orderId, amount); err != nil {
		h.paymeSendServiceError(c, req.ID, err)
		return
	}

	h.paymeSendSuccess(c, req.ID, map[string]any{"allow": true})
}

func (h *Handler) paymeCreateTransaction(c *gin.Context, req m.PaymeRequest) {
	paymeId := req.GetParamString("id")
	timeMs := req.GetParamInt64("time")
	amount := req.GetParamInt64("amount")
	account := req.GetAccount()
	orderId, _ := account["order_id"].(string)

	if orderId == "" || paymeId == "" {
		h.paymeSendError(c, req.ID, m.PaymeErrorAccountNotFound, "order_id/id missing", "account")
		return
	}

	res, err := h.Clients.Order.PaymeCreateTransaction(context.Background(), orderId, paymeId, amount, timeMs)
	if err != nil {
		h.paymeSendServiceError(c, req.ID, err)
		return
	}

	h.paymeSendSuccess(c, req.ID, map[string]any{
		"create_time":  res.CreateTime,
		"perform_time": res.PerformTime,
		"cancel_time":  res.CancelTime,
		"transaction":  res.OrderId,
		"state":        res.State,
		"reason":       res.Reason,
	})
}

func (h *Handler) paymePerformTransaction(c *gin.Context, req m.PaymeRequest) {
	paymeId := req.GetParamString("id")
	if paymeId == "" {
		h.paymeSendError(c, req.ID, m.PaymeErrorTransactionNotFound, "id missing", "")
		return
	}

	res, err := h.Clients.Order.PaymePerformTransaction(context.Background(), paymeId)
	if err != nil {
		h.paymeSendServiceError(c, req.ID, err)
		return
	}

	h.paymeSendSuccess(c, req.ID, map[string]any{
		"transaction":  res.OrderId,
		"perform_time": res.PerformTime,
		"state":        res.State,
		"reason":       res.Reason,
	})
}

func (h *Handler) paymeCancelTransaction(c *gin.Context, req m.PaymeRequest) {
	paymeId := req.GetParamString("id")
	reason := int(req.GetParamInt64("reason"))
	if paymeId == "" {
		h.paymeSendError(c, req.ID, m.PaymeErrorTransactionNotFound, "id missing", "")
		return
	}

	res, err := h.Clients.Order.PaymeCancelTransaction(context.Background(), paymeId, reason)
	if err != nil {
		h.paymeSendServiceError(c, req.ID, err)
		return
	}

	h.paymeSendSuccess(c, req.ID, map[string]any{
		"transaction": res.OrderId,
		"cancel_time": res.CancelTime,
		"state":       res.State,
	})
}

func (h *Handler) paymeCheckTransaction(c *gin.Context, req m.PaymeRequest) {
	paymeId := req.GetParamString("id")
	if paymeId == "" {
		h.paymeSendError(c, req.ID, m.PaymeErrorTransactionNotFound, "id missing", "")
		return
	}

	res, err := h.Clients.Order.PaymeCheckTransaction(context.Background(), paymeId)
	if err != nil {
		h.paymeSendServiceError(c, req.ID, err)
		return
	}

	h.paymeSendSuccess(c, req.ID, map[string]any{
		"create_time":  res.CreateTime,
		"perform_time": res.PerformTime,
		"cancel_time":  res.CancelTime,
		"transaction":  res.OrderId,
		"state":        res.State,
		"reason":       res.Reason,
	})
}

func (h *Handler) paymeGetStatement(c *gin.Context, req m.PaymeRequest) {
	from := req.GetParamInt64("from")
	to := req.GetParamInt64("to")

	items, err := h.Clients.Order.PaymeGetStatement(context.Background(), from, to)
	if err != nil {
		h.paymeSendServiceError(c, req.ID, err)
		return
	}

	transactions := make([]map[string]any, 0, len(items))
	for _, it := range items {
		transactions = append(transactions, map[string]any{
			"id":           it.PaymeId,
			"time":         it.CreateTime,
			"amount":       int64(it.Amount * 100),
			"account":      map[string]any{"order_id": it.OrderId},
			"create_time":  it.CreateTime,
			"perform_time": it.PerformTime,
			"cancel_time":  it.CancelTime,
			"transaction":  it.OrderId,
			"state":        it.State,
			"reason":       it.Reason,
		})
	}

	h.paymeSendSuccess(c, req.ID, map[string]any{"transactions": transactions})
}

// paymeSendServiceError - service/storage qatlamidan kelgan xatoni JSON-RPC javobiga aylantiradi:
// *models.PaymeOpError bo'lsa aynan shu Code/Message/Data bilan, aks holda generic Internal.
func (h *Handler) paymeSendServiceError(c *gin.Context, id any, err error) {
	var opErr *m.PaymeOpError
	if errors.As(err, &opErr) {
		h.paymeSendError(c, id, opErr.Code, opErr.Message, opErr.Data)
		return
	}
	h.paymeSendError(c, id, m.PaymeErrorInternal, "Internal error", "")
}

func (h *Handler) paymeSendSuccess(c *gin.Context, id any, result map[string]any) {
	c.JSON(200, m.PaymeResponse{JsonRPC: "2.0", ID: id, Result: result})
}

func (h *Handler) paymeSendError(c *gin.Context, id any, code int, message string, data string) {
	c.JSON(200, m.PaymeResponse{
		JsonRPC: "2.0",
		ID:      id,
		Error: &m.PaymeError{
			Code:    code,
			Message: map[string]any{"uz": message, "ru": message, "en": message},
			Data:    data,
		},
	})
}
