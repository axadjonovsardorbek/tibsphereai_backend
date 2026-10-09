package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"tibsphereai/internal/helper"
	"tibsphereai/internal/inpay"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
	"time"

	"github.com/go-redis/redis"
)

type OrderService struct {
	storage st.Storage
	rdb     *redis.Client
	push    *PushService

	inpay      *inpay.Client
	baseURL    string
	websiteURL string
}

// WithInpay - inPAY provayderini ulaydi (kalitlar env'da yo'q bo'lsa provider o'chiq turadi).
func (s *OrderService) WithInpay(c *inpay.Client, baseURL, websiteURL string) *OrderService {
	s.inpay, s.baseURL, s.websiteURL = c, baseURL, websiteURL
	return s
}

func NewOrderService(storage *st.Storage, rdb *redis.Client, push *PushService) *OrderService {
	return &OrderService{storage: *storage, rdb: rdb, push: push}
}

// Create - TZ 2.3-bo'lim: to'liq tangaga yopilsa (CoinsUsed yetarli) storage darhol
// PAID qilib qaytaradi (JustPaid=true) - shu holatda ham push/notification ishga tushadi,
// xuddi Click/Payme muvaffaqiyatli to'lovi kabi.
func (s *OrderService) Create(ctx context.Context, req *m.OrderCreateReq) (*m.OrderCreateRes, error) {
	res, err := s.storage.OrderS.Create(req)
	if err != nil || res == nil {
		return res, err
	}
	if res.JustPaid {
		s.onJustPaid(req.UserId, res.Id)
		return res, nil
	}
	if req.Provider == "inpay" {
		url, err := s.createInpayPayment(ctx, res.Id)
		if err != nil {
			return nil, err
		}
		res.Url = url
	}
	return res, nil
}

// createInpayPayment - inPAY'da to'lov yaratadi va uning order_id'sini bizning order bilan bog'laydi.
func (s *OrderService) createInpayPayment(ctx context.Context, orderId string) (string, error) {
	if !s.inpay.Enabled() {
		return "", errors.New("inpay provider is not configured")
	}
	o, err := s.storage.OrderS.GetById(&m.ById{Id: orderId})
	if err != nil {
		return "", err
	}
	if o == nil {
		return "", errors.New("order not found")
	}
	pay, err := s.inpay.Create(ctx, inpay.CreateReq{
		Amount:      o.Amount,
		Description: "TibSphere order " + orderId,
		CallbackURL: strings.TrimRight(s.baseURL, "/") + "/inpay/callback",
		ReturnURL:   s.websiteURL,
	})
	if err != nil {
		return "", err
	}
	if err := s.storage.OrderS.InpayAttach(orderId, pay.OrderID); err != nil {
		return "", err
	}
	return pay.PayURL, nil
}

// InpayCallback - inPAY webhook'i imzosiz, shuning uchun body'ga ishonmaymiz: holat va summani
// inPAY'dan (GET /transactions/) server-to-server qayta olib, shunga qarab to'lovni yakunlaymiz.
func (s *OrderService) InpayCallback(ctx context.Context, inpayOrderId string, transactionId int64) (*m.CompleteResult, error) {
	if !s.inpay.Enabled() {
		return nil, errors.New("inpay provider is not configured")
	}
	tx, err := s.inpay.GetTransaction(ctx, inpayOrderId)
	if err != nil {
		return nil, err
	}
	switch tx.Status {
	case inpay.StatusSuccess:
	case inpay.StatusFailed, inpay.StatusCancelled:
	default: // pending - hali yakunlanmagan
		return &m.CompleteResult{Error: 0, ErrorNote: "pending"}, nil
	}

	res, err := s.storage.OrderS.InpayComplete(inpayOrderId, tx.Status == inpay.StatusSuccess, transactionId, tx.Amount)
	if err != nil || res == nil {
		return res, err
	}
	if res.JustPaid {
		s.onJustPaid(res.PaidUserId, res.PaidOrderId)
	}
	return res, nil
}

func (s *OrderService) GetById(ctx context.Context, req *m.ById) (*m.OrderStatisticRes, error) {
	return s.storage.OrderS.GetById(req)
}

func (s *OrderService) GetAll(ctx context.Context, req *m.OrderGetListReq) (*m.PaginationResponse, *m.OrderList, error) {
	res := &m.PaginationResponse{}

	key := fmt.Sprintf("user_id:%s_tariff_id:%s_status:%s_payment_type:%s_type:%s_day:%s_from:%s_to:%s_page:%d_limit:%d",
		req.UserId, req.TariffId, req.Status, req.PaymentType, req.Type, req.Day, req.From, req.To, req.Filter.Page, req.Filter.Limit)

	if s.rdb != nil {
		if b, err := s.rdb.Get(key).Bytes(); err == nil && len(b) > 0 {
			_ = json.Unmarshal(b, &res)
		}
	}

	if res.Total == 0 {
		res, _, err := s.storage.OrderS.GetAll(req)
		if err != nil {
			return nil, nil, err
		}

		if s.rdb != nil {
			if b, err := json.Marshal(res); err == nil {
				_ = s.rdb.Set(key, b, 60*time.Second).Err()
			}
		}

		return res, nil, nil
	}

	return res, nil, nil
}

func (s *OrderService) ApplyPrepare(ctx context.Context, req *m.ApplyPrepareParams) (*m.PrepareResult, error) {
	return s.storage.OrderS.ApplyPrepare(req)
}

// ApplyComplete - to'lov muvaffaqiyatli tasdiqlangach (BACKEND_STRUCTURE.md 5-bo'lim):
//  1. Premium obuna faollashtiriladi (storage qatlamida - subscriptions),
//  2. Tizim avtomatik 1 martalik promokod yaratadi (aktiv hamkorga bog'langan holda),
//  3. Push + in-app bildirishnoma yuboriladi.
func (s *OrderService) ApplyComplete(ctx context.Context, req *m.ApplyCompleteParams) (*m.CompleteResult, error) {
	res, err := s.storage.OrderS.ApplyComplete(req)
	if err != nil || res == nil {
		return res, err
	}

	if res.JustPaid {
		s.onJustPaid(res.PaidUserId, res.PaidOrderId)
	}

	return res, nil
}

// onJustPaid - to'lov birinchi marta PAID bo'lganda ishga tushadigan umumiy oqim
// (push + in-app bildirishnoma). Create (to'liq tanga), Click (ApplyComplete) va
// Payme (PerformTransaction) - uchalasi ham shu orqali ishlaydi. Promokod endi
// avtomatik chiqarilmaydi (TZ 5-6-bo'lim: hamkor o'zi oldindan yaratadi).
func (s *OrderService) onJustPaid(userId, orderId string) {
	go func(userId, orderId string) {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("order complete goroutine panic:", r)
			}
		}()

		pushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		pushEvent := m.PushEvent{Type: m.EventPaymentSuccess, OrderID: orderId}
		pushTarget := m.PushTarget{Type: m.TargetUser, UserID: userId}
		_ = s.push.SendPush(pushCtx, pushTarget, pushEvent)

		msg := m.BuildLocalizedPushMessage(pushEvent, "uz")
		title, body := helper.StringToJSON(msg.Title, msg.Body)
		if _, err := s.storage.NotificationS.Create(&m.NotificationCreateReq{
			Title: title, Message: body, Type: "selected", UserId: userId,
		}); err != nil {
			fmt.Println("failed to create notification:", err)
		}
	}(userId, orderId)
}

// ============ Payme JSON-RPC (docs: furasentr-payment) ============

func (s *OrderService) PaymeCheckPerform(ctx context.Context, orderId string, amountTiyin int64) error {
	return s.storage.OrderS.PaymeCheckPerform(orderId, amountTiyin)
}

func (s *OrderService) PaymeCreateTransaction(ctx context.Context, orderId, paymeId string, amountTiyin, timeMs int64) (*m.PaymeCreateResult, error) {
	return s.storage.OrderS.PaymeCreateTransaction(orderId, paymeId, amountTiyin, timeMs)
}

func (s *OrderService) PaymePerformTransaction(ctx context.Context, paymeId string) (*m.PaymeTransactionResult, error) {
	res, err := s.storage.OrderS.PaymePerformTransaction(paymeId)
	if err != nil || res == nil {
		return res, err
	}

	if res.JustPaid {
		var userId string
		if o, oerr := s.storage.OrderS.GetById(&m.ById{Id: res.OrderId}); oerr == nil && o != nil {
			userId = o.UserId
		}
		if userId != "" {
			s.onJustPaid(userId, res.OrderId)
		}
	}

	return res, nil
}

func (s *OrderService) PaymeCancelTransaction(ctx context.Context, paymeId string, reason int) (*m.PaymeTransactionResult, error) {
	return s.storage.OrderS.PaymeCancelTransaction(paymeId, reason)
}

func (s *OrderService) PaymeCheckTransaction(ctx context.Context, paymeId string) (*m.PaymeTransactionResult, error) {
	return s.storage.OrderS.PaymeCheckTransaction(paymeId)
}

func (s *OrderService) PaymeGetStatement(ctx context.Context, fromMs, toMs int64) ([]m.PaymeStatementItem, error) {
	return s.storage.OrderS.PaymeGetStatement(fromMs, toMs)
}

// RunInpayReconciler - webhook yetib kelmay qolsa ham to'lov qotib qolmasligi uchun
// kutilayotgan inPAY buyurtmalarini har 10 soniyada inPAY'dan tekshirib, yakunlaydi.
func (s *OrderService) RunInpayReconciler(ctx context.Context) {
	if !s.inpay.Enabled() {
		return
	}
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reconcileInpay(ctx)
		}
	}
}

func (s *OrderService) reconcileInpay(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("inpay reconcile panic:", r)
		}
	}()
	ids, err := s.storage.OrderS.InpayListPending()
	if err != nil {
		fmt.Println("inpay reconcile list:", err)
		return
	}
	for _, id := range ids {
		cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		res, err := s.InpayCallback(cctx, id, 0)
		cancel()
		if err != nil {
			fmt.Println("inpay reconcile", id, err)
			continue
		}
		if res != nil && res.JustPaid {
			fmt.Println("inpay reconcile: paid", id)
		}
	}
}
