package service

import (
	"context"
	"errors"
	"testing"

	m "tibsphereai/pkg/models"
	"tibsphereai/storage/mocks"
	st "tibsphereai/storage/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newPromoCodeTestService(pm *mocks.PromoCodeMock, um *mocks.UserMock) *PromoCodeService {
	pgStorage := &st.Storage{PromoCodeS: pm, UserS: um}
	userS := NewUserService(pgStorage, nil, nil, nil)
	return NewPromoCodeService(pgStorage, nil, userS)
}

func TestPromoCodeService_Create(t *testing.T) {
	pm := mocks.NewPromoCodeMock()
	svc := newPromoCodeTestService(pm, mocks.NewUserMock())

	req := &m.PromoCodeCreateReq{}
	want := &m.ById{Id: "promo-1"}
	pm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestPromoCodeService_GetById(t *testing.T) {
	pm := mocks.NewPromoCodeMock()
	svc := newPromoCodeTestService(pm, mocks.NewUserMock())

	req := &m.ById{Id: "promo-1"}
	want := &m.PromoCodeRes{}
	pm.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestPromoCodeService_GetAll(t *testing.T) {
	pm := mocks.NewPromoCodeMock()
	svc := newPromoCodeTestService(pm, mocks.NewUserMock())

	req := &m.PromoCodeGetListReq{}
	wantPag := &m.PaginationResponse{Total: 2}
	wantList := &m.PromoCodeList{}
	pm.On("GetAll", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetAll(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Equal(t, wantList, list)
}

func TestPromoCodeService_GetMy(t *testing.T) {
	pm := mocks.NewPromoCodeMock()
	svc := newPromoCodeTestService(pm, mocks.NewUserMock())

	req := &m.PromoCodeGetListReq{}
	wantPag := &m.PaginationResponse{Total: 1}
	wantList := &m.PromoCodeList{}
	pm.On("GetMy", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetMy(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Equal(t, wantList, list)
}

// Redeem - kod "used" qilinadi va qaytgan tanga miqdori UserS.AddPoints orqali
// balansga qo'shiladi (BACKEND_STRUCTURE.md 5-bo'lim: tanga hisoblash markazlashgan).
func TestPromoCodeService_Redeem_Success(t *testing.T) {
	pm := mocks.NewPromoCodeMock()
	um := mocks.NewUserMock()
	svc := newPromoCodeTestService(pm, um)

	req := &m.PromoCodeRedeemReq{Code: "WELCOME10", UserId: "user-1"}
	pm.On("Redeem", req).Return(10, nil)
	um.On("AddPoints", &m.AddPointsReq{UserId: "user-1", CoinsDelta: 10, Reason: "promo_redeem"}).
		Return(&m.Void{}, nil)

	got, err := svc.Redeem(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, &m.PromoCodeRedeemRes{CoinsAdded: 10}, got)
	pm.AssertExpectations(t)
	um.AssertExpectations(t)
}

// Redeem - kod yaroqsiz/ishlatilgan bo'lsa, AddPoints umuman chaqirilmasligi kerak.
func TestPromoCodeService_Redeem_StorageError(t *testing.T) {
	pm := mocks.NewPromoCodeMock()
	um := mocks.NewUserMock()
	svc := newPromoCodeTestService(pm, um)

	req := &m.PromoCodeRedeemReq{Code: "USED", UserId: "user-1"}
	wantErr := errors.New("promo code already used")
	pm.On("Redeem", req).Return(0, wantErr)

	got, err := svc.Redeem(context.Background(), req)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
	um.AssertNotCalled(t, "AddPoints", mock.Anything)
}

// Redeem - kod muvaffaqiyatli ishlatilgan, lekin balansga qo'shishda xato bo'lsa,
// xato foydalanuvchiga qaytarilishi kerak (tanga yo'qolib qolmasligi uchun log/monitoring
// kerak bo'lsa-da, hozircha service shunchaki xatoni qaytaradi).
func TestPromoCodeService_Redeem_AddPointsError(t *testing.T) {
	pm := mocks.NewPromoCodeMock()
	um := mocks.NewUserMock()
	svc := newPromoCodeTestService(pm, um)

	req := &m.PromoCodeRedeemReq{Code: "WELCOME10", UserId: "user-1"}
	wantErr := errors.New("user not found")
	pm.On("Redeem", req).Return(10, nil)
	um.On("AddPoints", mock.Anything).Return(nil, wantErr)

	got, err := svc.Redeem(context.Background(), req)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}

func TestPromoCodeService_Delete(t *testing.T) {
	pm := mocks.NewPromoCodeMock()
	svc := newPromoCodeTestService(pm, mocks.NewUserMock())

	req := &m.ById{Id: "promo-1"}
	pm.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}
