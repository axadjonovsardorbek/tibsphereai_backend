package handler

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	m "tibsphereai/pkg/models"

	"github.com/gin-gonic/gin"
)

// Click error codes (Supplier side)
const (
	ErrOK              = 0
	ErrSignFailed      = -1
	ErrIncorrectAmount = -2
	ErrActionNotFound  = -3
	ErrAlreadyPaid     = -4
	ErrOrderNotExist   = -5
	ErrTransNotExist   = -6
	ErrFailedUpdate    = -7
	ErrBadRequest      = -8
	ErrCancelled       = -9
)

type clickReq struct {
	ClickTransID      int64
	ServiceID         int
	ClickPaydocID     int64
	MerchantTransID   string
	AmountStr         string  // ✅ SIGN uchun aynan shu ishlatiladi
	Amount            float64 // optional: ichki logika uchun (sign uchun emas)
	Action            int
	Error             int
	ErrorNote         string
	SignTime          string
	SignString        string
	MerchantPrepareID int64
}

type clickResp struct {
	Error           int    `json:"error"`
	ErrorNote       string `json:"error_note"`
	ClickTransID    int64  `json:"click_trans_id,omitempty"`
	MerchantTransID string `json:"merchant_trans_id,omitempty"`
	MerchantPrepID  int64  `json:"merchant_prepare_id,omitempty"`
	MerchantConfID  int64  `json:"merchant_confirm_id,omitempty"`
}

// isUUID minimal check (senda order_id UUID bo‘lsa)
func isUUID(s string) bool {
	return len(s) >= 32 && strings.Count(s, "-") >= 4
}

func parseClickForm(c *gin.Context) (clickReq, map[string]string, error) {
	// form (x-www-form-urlencoded) yoki query bo‘lsa ham ishlaydi
	if err := c.Request.ParseForm(); err != nil {
		return clickReq{}, nil, err
	}
	f := func(k string) string { return strings.TrimSpace(c.Request.FormValue(k)) }

	// required
	clickTransID, err := strconv.ParseInt(f("click_trans_id"), 10, 64)
	if err != nil {
		return clickReq{}, nil, fmt.Errorf("click_trans_id: %w", err)
	}
	serviceID, err := strconv.Atoi(f("service_id"))
	if err != nil {
		return clickReq{}, nil, fmt.Errorf("service_id: %w", err)
	}
	clickPaydocID, err := strconv.ParseInt(f("click_paydoc_id"), 10, 64)
	if err != nil {
		return clickReq{}, nil, fmt.Errorf("click_paydoc_id: %w", err)
	}
	merchantTransID := f("merchant_trans_id")
	if merchantTransID == "" {
		return clickReq{}, nil, fmt.Errorf("merchant_trans_id missing")
	}

	amountStr := f("amount")
	if amountStr == "" {
		return clickReq{}, nil, fmt.Errorf("amount missing")
	}
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return clickReq{}, nil, fmt.Errorf("amount: %w", err)
	}

	action, err := strconv.Atoi(f("action"))
	if err != nil {
		return clickReq{}, nil, fmt.Errorf("action: %w", err)
	}

	// optional
	errCode := 0
	if v := f("error"); v != "" {
		if x, e := strconv.Atoi(v); e == nil {
			errCode = x
		}
	}
	errNote := f("error_note")
	signTime := f("sign_time")
	signString := f("sign_string")

	var mpid int64
	if v := f("merchant_prepare_id"); v != "" {
		mpid, _ = strconv.ParseInt(v, 10, 64)
	}

	raw := map[string]string{
		"click_trans_id":      f("click_trans_id"),
		"service_id":          f("service_id"),
		"click_paydoc_id":     f("click_paydoc_id"),
		"merchant_trans_id":   f("merchant_trans_id"),
		"amount":              amountStr, // ✅ aynan requestdan kelgan ko‘rinish
		"action":              f("action"),
		"error":               f("error"),
		"error_note":          f("error_note"),
		"sign_time":           f("sign_time"),
		"sign_string":         f("sign_string"),
		"merchant_prepare_id": f("merchant_prepare_id"),
	}

	return clickReq{
		ClickTransID:      clickTransID,
		ServiceID:         serviceID,
		ClickPaydocID:     clickPaydocID,
		MerchantTransID:   merchantTransID,
		AmountStr:         amountStr,
		Amount:            amount,
		Action:            action,
		Error:             errCode,
		ErrorNote:         errNote,
		SignTime:          signTime,
		SignString:        signString,
		MerchantPrepareID: mpid,
	}, raw, nil
}

func md5hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

// action=0
func VerifyPrepare(secret string,
	clickTransID int64,
	serviceID int,
	merchantTransID string,
	amountStr string,
	action int,
	signTime string,
	signString string,
) bool {
	raw := fmt.Sprintf("%d%d%s%s%s%d%s",
		clickTransID,
		serviceID,
		secret,
		merchantTransID,
		strings.TrimSpace(amountStr), // ✅ no rounding
		action,
		signTime,
	)
	expected := md5hex(raw)
	return strings.EqualFold(expected, signString)
}

// action=1
func VerifyComplete(secret string,
	clickTransID int64,
	serviceID int,
	merchantTransID string,
	merchantPrepareID int64,
	amountStr string,
	action int,
	signTime string,
	signString string,
) bool {
	raw := fmt.Sprintf("%d%d%s%s%d%s%d%s",
		clickTransID,
		serviceID,
		secret,
		merchantTransID,
		merchantPrepareID,
		strings.TrimSpace(amountStr), // ✅ no rounding
		action,
		signTime,
	)
	expected := md5hex(raw)
	return strings.EqualFold(expected, signString)
}

// ---- HANDLERS ----

// POST /click/prepare (action=0)
func (h *Handler) ClickPrepare(c *gin.Context) {
	req, raw, err := parseClickForm(c)

	fmt.Println("----------------------------------")
	fmt.Println("CLICK PREPARE REQ:", req)
	fmt.Println("----------------------------------")
	fmt.Println("CLICK PREPARE RAW:", raw)
	fmt.Println("----------------------------------")
	fmt.Println("CLICK PREPARE ERR:", err)
	fmt.Println("----------------------------------")

	if err != nil {
		fmt.Println("Check api 1")
		slog.Error("click prepare parse error", "err", err)
		c.JSON(200, clickResp{Error: ErrBadRequest, ErrorNote: "Error in request from click"})
		return
	}

	if req.Action != 0 {
		fmt.Println("Check api 2")
		c.JSON(200, clickResp{Error: ErrActionNotFound, ErrorNote: "Action not found"})
		return
	}

	// service_id check
	if req.ServiceID != h.Cfg.CLICK_SERVICE_ID {
		fmt.Println("Check api 3")
		c.JSON(200, clickResp{Error: ErrBadRequest, ErrorNote: "service_id mismatch"})
		return
	}

	// merchant_trans_id = order_id (UUID)
	if !isUUID(req.MerchantTransID) {
		fmt.Println("Check api 4")
		c.JSON(200, clickResp{Error: ErrOrderNotExist, ErrorNote: "Order not found"})
		return
	}

	// SIGN CHECK (✅ amountStr ishlatyapmiz)
	if !VerifyPrepare(
		h.Cfg.CLICK_SECRET_KEY,
		req.ClickTransID,
		req.ServiceID,
		req.MerchantTransID,
		req.AmountStr,
		req.Action,
		req.SignTime,
		req.SignString,
	) {
		slog.Warn("click prepare sign failed",
			"click_trans_id", req.ClickTransID,
			"merchant_trans_id", req.MerchantTransID,
			"amount", req.AmountStr,
			"sign_time", req.SignTime,
		)
		fmt.Println("Check api 5")
		c.JSON(200, clickResp{Error: ErrSignFailed, ErrorNote: "SIGN CHECK FAILED!"})
		return
	}

	// unique merchant_prepare_id (senda int64)
	merchantPrepareID := time.Now().UnixNano()

	patch := map[string]any{
		"click_prepare_raw": raw,
		"click_prepare_at":  time.Now().Format(time.RFC3339),
	}
	patchJSON, _ := json.Marshal(patch)

	res, err := h.Clients.Order.ApplyPrepare(c.Request.Context(), &m.ApplyPrepareParams{
		OrderId:           req.MerchantTransID,
		ClickTransId:      req.ClickTransID,
		ClickPaydocId:     req.ClickPaydocID,
		MerchantPrepareId: merchantPrepareID,
		Amount:            req.Amount, // ichki logika uchun float (xohlasang amountStrdan decimal qil)
		MetaPatchJSON:     patchJSON,
	})
	if err != nil {
		fmt.Println("Check api 6")
		slog.Error("click prepare repo error", "err", err)
		c.JSON(200, clickResp{Error: ErrFailedUpdate, ErrorNote: "Failed to update"})
		return
	}

	c.JSON(200, clickResp{
		Error:           res.Error,
		ErrorNote:       res.ErrorNote,
		ClickTransID:    req.ClickTransID,
		MerchantTransID: req.MerchantTransID,
		MerchantPrepID:  res.MerchantPrepareId,
	})
}

// POST /click/complete (action=1)
func (h *Handler) ClickComplete(c *gin.Context) {
	req, raw, err := parseClickForm(c)
	fmt.Println("----------------------------------")
	fmt.Println("CLICK COMPLETE REQ:", req)
	fmt.Println("----------------------------------")
	fmt.Println("CLICK COMPLETE RAW:", raw)
	fmt.Println("----------------------------------")
	fmt.Println("CLICK COMPLETE ERR:", err)
	fmt.Println("----------------------------------")
	if err != nil {
		fmt.Println("Check api 7")
		slog.Error("click complete parse error", "err", err)
		c.JSON(200, clickResp{Error: ErrBadRequest, ErrorNote: "Error in request from click"})
		return
	}

	if req.Action != 1 {
		fmt.Println("Check api 8")
		c.JSON(200, clickResp{Error: ErrActionNotFound, ErrorNote: "Action not found"})
		return
	}

	if req.ServiceID != h.Cfg.CLICK_SERVICE_ID {
		fmt.Println("Check api 9")
		c.JSON(200, clickResp{Error: ErrBadRequest, ErrorNote: "service_id mismatch"})
		return
	}

	if !isUUID(req.MerchantTransID) {
		fmt.Println("Check api 10")
		c.JSON(200, clickResp{Error: ErrOrderNotExist, ErrorNote: "Order not found"})
		return
	}

	// SIGN CHECK (✅ amountStr ishlatyapmiz)
	if !VerifyComplete(
		h.Cfg.CLICK_SECRET_KEY,
		req.ClickTransID,
		req.ServiceID,
		req.MerchantTransID,
		req.MerchantPrepareID,
		req.AmountStr,
		req.Action,
		req.SignTime,
		req.SignString,
	) {
		slog.Warn("click complete sign failed",
			"click_trans_id", req.ClickTransID,
			"merchant_trans_id", req.MerchantTransID,
			"merchant_prepare_id", req.MerchantPrepareID,
			"amount", req.AmountStr,
			"sign_time", req.SignTime,
		)
		fmt.Println("Check api 11")
		c.JSON(200, clickResp{Error: ErrSignFailed, ErrorNote: "SIGN CHECK FAILED!"})
		return
	}

	patch := map[string]any{
		"click_complete_raw": raw,
		"click_complete_at":  time.Now().Format(time.RFC3339),
	}
	patchJSON, _ := json.Marshal(patch)

	res, err := h.Clients.Order.ApplyComplete(c.Request.Context(), &m.ApplyCompleteParams{
		OrderId:           req.MerchantTransID,
		ClickTransId:      req.ClickTransID,
		MerchantPrepareId: req.MerchantPrepareID,
		ErrorCode:         req.Error,
		MetaPatchJSON:     patchJSON,
	})
	if err != nil {
		fmt.Println("Check api 12")
		slog.Error("click complete repo error", "err", err)
		c.JSON(200, clickResp{Error: ErrFailedUpdate, ErrorNote: "Failed to update"})
		return
	}

	c.JSON(200, clickResp{
		Error:           res.Error,
		ErrorNote:       res.ErrorNote,
		ClickTransID:    req.ClickTransID,
		MerchantTransID: req.MerchantTransID,
		MerchantConfID:  res.MerchantConfirmId,
	})
}
