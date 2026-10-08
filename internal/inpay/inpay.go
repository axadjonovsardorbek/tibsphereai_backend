// Package inpay - inPAY (https://inpay.uz/api/) to'lov provayderi klienti.
// Oqim: Authorization (Bearer, 24 soat, keshlanadi) -> POST /create/ (pay_url) ->
// inPAY callback_url'ga webhook yuboradi (imzosiz!) -> biz GET /transactions/ bilan tasdiqlaymiz.
package inpay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	StatusPending   = "pending"
	StatusSuccess   = "success"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"

	tokenTTL = 23 * time.Hour // inPAY tokeni 24 soat amal qiladi; zaxira bilan yangilaymiz
)

type Client struct {
	baseURL       string
	merchantID    string
	merchantToken string
	http          *http.Client

	mu       sync.Mutex
	bearer   string
	bearerAt time.Time
}

func NewClient(baseURL, merchantID, merchantToken string) *Client {
	if baseURL == "" {
		baseURL = "https://inpay.uz/api/v1"
	}
	return &Client{
		baseURL:       strings.TrimRight(baseURL, "/"),
		merchantID:    merchantID,
		merchantToken: merchantToken,
		http:          &http.Client{Timeout: 15 * time.Second},
	}
}

// Enabled - kalitlar env'da berilganmi.
func (c *Client) Enabled() bool {
	return c != nil && c.merchantID != "" && c.merchantToken != ""
}

type CreateReq struct {
	Amount      float64
	Description string
	CallbackURL string
	ReturnURL   string
	Phone       string
	ClientIP    string
}

type CreateRes struct {
	OrderID string `json:"order_id"`
	PayURL  string `json:"pay_url"`
}

type Transaction struct {
	OrderID string  `json:"order_id"`
	Status  string  `json:"status"`
	Amount  float64 `json:"amount"`
}

type apiError struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ErrorCode string `json:"error_code"`
}

func (c *Client) authorize(ctx context.Context, force bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !force && c.bearer != "" && time.Since(c.bearerAt) < tokenTTL {
		return c.bearer, nil
	}

	q := url.Values{}
	q.Set("merchant_id", c.merchantID)
	q.Set("merchant_token", c.merchantToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/authorization/?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	var out struct {
		apiError
		BearerToken string `json:"bearer_token"`
	}
	if _, err := c.do(req, &out); err != nil {
		return "", fmt.Errorf("inpay authorization: %w", err)
	}
	if out.BearerToken == "" {
		return "", fmt.Errorf("inpay authorization: empty bearer_token (%s)", out.Message)
	}
	c.bearer, c.bearerAt = out.BearerToken, time.Now()
	return c.bearer, nil
}

func (c *Client) do(req *http.Request, out any) (int, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode >= 400 {
		var e apiError
		_ = json.Unmarshal(body, &e)
		if e.Message == "" {
			e.Message = strings.TrimSpace(string(body))
		}
		return resp.StatusCode, fmt.Errorf("http %d %s: %s", resp.StatusCode, e.ErrorCode, e.Message)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return resp.StatusCode, fmt.Errorf("decode response: %w", err)
	}
	return resp.StatusCode, nil
}

// authed - Bearer bilan so'rov; 401 bo'lsa token yangilanib bir marta qayta uriniladi.
func (c *Client) authed(ctx context.Context, build func() (*http.Request, error), out any) error {
	for attempt := 0; attempt < 2; attempt++ {
		tok, err := c.authorize(ctx, attempt > 0)
		if err != nil {
			return err
		}
		req, err := build()
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		code, err := c.do(req, out)
		if err == nil {
			return nil
		}
		if code != http.StatusUnauthorized || attempt == 1 {
			return err
		}
	}
	return errors.New("inpay: unreachable")
}

// Create - to'lov yaratadi; qaytgan OrderID bizning orders qatoriga bog'lanadi.
func (c *Client) Create(ctx context.Context, r CreateReq) (*CreateRes, error) {
	payload := map[string]any{
		"merchant_id": c.merchantID,
		"token":       c.merchantToken,
		"amount":      r.Amount,
	}
	if r.Description != "" {
		payload["description"] = r.Description
	}
	if r.CallbackURL != "" {
		payload["callback_url"] = r.CallbackURL
	}
	if r.ReturnURL != "" {
		payload["return_url"] = r.ReturnURL
	}
	if r.Phone != "" {
		payload["phone"] = r.Phone
	}
	if r.ClientIP != "" {
		payload["client_ip"] = r.ClientIP
	}
	b, _ := json.Marshal(payload)

	var out struct {
		apiError
		CreateRes
	}
	err := c.authed(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/create/", bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	}, &out)
	if err != nil {
		return nil, fmt.Errorf("inpay create: %w", err)
	}
	if out.OrderID == "" || out.PayURL == "" {
		return nil, fmt.Errorf("inpay create: bad response: %s %s", out.ErrorCode, out.Message)
	}
	return &out.CreateRes, nil
}

// GetTransaction - webhook imzosiz bo'lgani uchun holatni shu orqali server-to-server tasdiqlaymiz.
func (c *Client) GetTransaction(ctx context.Context, orderID string) (*Transaction, error) {
	// Javob ba'zan tekis, ba'zan "data" ichida keladi - ikkalasini ham qabul qilamiz.
	var out struct {
		apiError
		Transaction
		Data *Transaction `json:"data"`
	}
	err := c.authed(ctx, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet,
			c.baseURL+"/transactions/?order_id="+url.QueryEscape(orderID), nil)
	}, &out)
	if err != nil {
		return nil, fmt.Errorf("inpay transactions: %w", err)
	}
	if out.Data != nil {
		return out.Data, nil
	}
	return &out.Transaction, nil
}
