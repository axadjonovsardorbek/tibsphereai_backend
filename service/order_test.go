package service

import (
	"context"
	"errors"
	"testing"
	"time"

	m "tibsphereai/pkg/models"
	"tibsphereai/storage/mocks"
	st "tibsphereai/storage/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newOrderTestService(om *mocks.OrderMock, nm *mocks.NotificationMock) *OrderService {
	pgStorage := &st.Storage{OrderS: om, NotificationS: nm}
	// push=nil ichki push.Client - PushService.SendPush darhol nil qaytaradi (real FCM shart emas).
	pushSvc := NewPushService(pgStorage, nil)
	return NewOrderService(pgStorage, nil, pushSvc)
}

// waitForNotification - onJustPaid fon rejimida (goroutine) ishga tushadi, shuning
// uchun uning yakunlanishini kutish uchun kanal orqali sinxronlashtiramiz (uxlash o'rniga).
func waitForNotification(t *testing.T, done chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for onJustPaid background notification")
	}
}

// Create - to'liq tangaga yopilib (JustPaid=true) darhol PAID bo'lganda, fon
// rejimida push + in-app bildirishnoma ishga tushishi kerak (TZ 2.3-bo'lim).
func TestOrderService_Create_JustPaid_TriggersNotification(t *testing.T) {
	om := mocks.NewOrderMock()
	nm := mocks.NewNotificationMock()
	svc := newOrderTestService(om, nm)

	req := &m.OrderCreateReq{UserId: "user-1", TariffId: "tariff-1", CoinsUsed: 1000}
	want := &m.OrderCreateRes{Id: "order-1", JustPaid: true}
	om.On("Create", req).Return(want, nil)

	done := make(chan struct{})
	nm.On("Create", mock.Anything).Return(&m.ById{}, nil).Run(func(args mock.Arguments) {
		close(done)
	})

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	waitForNotification(t, done)
	nm.AssertExpectations(t)
}

// Create - to'liq tangaga yopilmagan (JustPaid=false, gateway'ga yo'naltiriladi)
// holatda hech qanday bildirishnoma yaratilmasligi kerak.
func TestOrderService_Create_NotJustPaid_NoNotification(t *testing.T) {
	om := mocks.NewOrderMock()
	nm := mocks.NewNotificationMock()
	svc := newOrderTestService(om, nm)

	req := &m.OrderCreateReq{UserId: "user-1", TariffId: "tariff-1"}
	want := &m.OrderCreateRes{Id: "order-2", Url: "https://checkout.paycom.uz/xxx", JustPaid: false}
	om.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	time.Sleep(50 * time.Millisecond)
	nm.AssertNotCalled(t, "Create", mock.Anything)
}

func TestOrderService_Create_Error(t *testing.T) {
	om := mocks.NewOrderMock()
	nm := mocks.NewNotificationMock()
	svc := newOrderTestService(om, nm)

	req := &m.OrderCreateReq{UserId: "user-1", TariffId: "tariff-1"}
	wantErr := errors.New("tariff not found")
	om.On("Create", req).Return(nil, wantErr)

	got, err := svc.Create(context.Background(), req)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}

func TestOrderService_GetById(t *testing.T) {
	om := mocks.NewOrderMock()
	svc := newOrderTestService(om, mocks.NewNotificationMock())

	req := &m.ById{Id: "order-1"}
	want := &m.OrderStatisticRes{}
	om.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestOrderService_ApplyPrepare(t *testing.T) {
	om := mocks.NewOrderMock()
	svc := newOrderTestService(om, mocks.NewNotificationMock())

	req := &m.ApplyPrepareParams{OrderId: "order-1"}
	want := &m.PrepareResult{MerchantPrepareId: 5}
	om.On("ApplyPrepare", req).Return(want, nil)

	got, err := svc.ApplyPrepare(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// ApplyComplete (Click) - JustPaid bo'lsa onJustPaid ishga tushishi kerak.
func TestOrderService_ApplyComplete_JustPaid(t *testing.T) {
	om := mocks.NewOrderMock()
	nm := mocks.NewNotificationMock()
	svc := newOrderTestService(om, nm)

	req := &m.ApplyCompleteParams{OrderId: "order-1", ClickTransId: 123}
	want := &m.CompleteResult{JustPaid: true, PaidUserId: "user-1", PaidOrderId: "order-1"}
	om.On("ApplyComplete", req).Return(want, nil)

	done := make(chan struct{})
	nm.On("Create", mock.Anything).Return(&m.ById{}, nil).Run(func(args mock.Arguments) {
		close(done)
	})

	got, err := svc.ApplyComplete(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	waitForNotification(t, done)
}

func TestOrderService_ApplyComplete_NotJustPaid(t *testing.T) {
	om := mocks.NewOrderMock()
	nm := mocks.NewNotificationMock()
	svc := newOrderTestService(om, nm)

	req := &m.ApplyCompleteParams{OrderId: "order-1"}
	want := &m.CompleteResult{JustPaid: false, Error: -9, ErrorNote: "already paid"}
	om.On("ApplyComplete", req).Return(want, nil)

	got, err := svc.ApplyComplete(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)

	time.Sleep(50 * time.Millisecond)
	nm.AssertNotCalled(t, "Create", mock.Anything)
}

func TestOrderService_PaymeCheckPerform(t *testing.T) {
	om := mocks.NewOrderMock()
	svc := newOrderTestService(om, mocks.NewNotificationMock())

	om.On("PaymeCheckPerform", "order-1", int64(50000)).Return(nil)

	err := svc.PaymeCheckPerform(context.Background(), "order-1", 50000)
	require.NoError(t, err)
}

func TestOrderService_PaymeCreateTransaction(t *testing.T) {
	om := mocks.NewOrderMock()
	svc := newOrderTestService(om, mocks.NewNotificationMock())

	want := &m.PaymeCreateResult{OrderId: "order-1", State: 1}
	om.On("PaymeCreateTransaction", "order-1", "payme-1", int64(50000), int64(1000)).Return(want, nil)

	got, err := svc.PaymeCreateTransaction(context.Background(), "order-1", "payme-1", 50000, 1000)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// PaymePerformTransaction - JustPaid bo'lsa, order egasini OrderS.GetById orqali
// topib onJustPaid ishga tushirishi kerak.
func TestOrderService_PaymePerformTransaction_JustPaid(t *testing.T) {
	om := mocks.NewOrderMock()
	nm := mocks.NewNotificationMock()
	svc := newOrderTestService(om, nm)

	want := &m.PaymeTransactionResult{OrderId: "order-1", State: 2, JustPaid: true}
	om.On("PaymePerformTransaction", "payme-1").Return(want, nil)
	om.On("GetById", &m.ById{Id: "order-1"}).Return(&m.OrderStatisticRes{UserId: "user-1"}, nil)

	done := make(chan struct{})
	nm.On("Create", mock.Anything).Return(&m.ById{}, nil).Run(func(args mock.Arguments) {
		close(done)
	})

	got, err := svc.PaymePerformTransaction(context.Background(), "payme-1")
	require.NoError(t, err)
	assert.Equal(t, want, got)
	waitForNotification(t, done)
}

func TestOrderService_PaymePerformTransaction_NotJustPaid(t *testing.T) {
	om := mocks.NewOrderMock()
	svc := newOrderTestService(om, mocks.NewNotificationMock())

	want := &m.PaymeTransactionResult{OrderId: "order-1", State: 2, JustPaid: false}
	om.On("PaymePerformTransaction", "payme-1").Return(want, nil)

	got, err := svc.PaymePerformTransaction(context.Background(), "payme-1")
	require.NoError(t, err)
	assert.Equal(t, want, got)
	om.AssertNotCalled(t, "GetById", mock.Anything)
}

func TestOrderService_PaymeCancelTransaction(t *testing.T) {
	om := mocks.NewOrderMock()
	svc := newOrderTestService(om, mocks.NewNotificationMock())

	want := &m.PaymeTransactionResult{OrderId: "order-1", State: -1}
	om.On("PaymeCancelTransaction", "payme-1", 1).Return(want, nil)

	got, err := svc.PaymeCancelTransaction(context.Background(), "payme-1", 1)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestOrderService_PaymeCheckTransaction(t *testing.T) {
	om := mocks.NewOrderMock()
	svc := newOrderTestService(om, mocks.NewNotificationMock())

	want := &m.PaymeTransactionResult{OrderId: "order-1", State: 2}
	om.On("PaymeCheckTransaction", "payme-1").Return(want, nil)

	got, err := svc.PaymeCheckTransaction(context.Background(), "payme-1")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestOrderService_PaymeGetStatement(t *testing.T) {
	om := mocks.NewOrderMock()
	svc := newOrderTestService(om, mocks.NewNotificationMock())

	want := []m.PaymeStatementItem{{OrderId: "order-1", Amount: 50000}}
	om.On("PaymeGetStatement", int64(1000), int64(2000)).Return(want, nil)

	got, err := svc.PaymeGetStatement(context.Background(), 1000, 2000)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
