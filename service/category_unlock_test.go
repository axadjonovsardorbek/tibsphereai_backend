package service

import (
	"context"
	"errors"
	"testing"

	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"
	"tibsphereai/storage/mocks"
	st "tibsphereai/storage/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCategoryUnlockTestService(cm *mocks.CategoryMock, cum *mocks.CategoryUnlockMock) *CategoryUnlockService {
	return NewCategoryUnlockService(&st.Storage{CategoryS: cm, CategoryUnlockS: cum})
}

// Purchase - narx category.coins_per_case * case_count formulasi bilan hisoblanadi
// va storage.CategoryUnlockS.Purchase'ga aynan shu tanga miqdori bilan uzatiladi.
func TestCategoryUnlockService_Purchase_ComputesCost(t *testing.T) {
	cm := mocks.NewCategoryMock()
	cum := mocks.NewCategoryUnlockMock()
	svc := newCategoryUnlockTestService(cm, cum)

	coinsPerCase := 5
	cm.On("GetById", &m.ById{Id: "cat-1"}).Return(&m.CategoryRes{CoinsPerCase: &coinsPerCase}, nil)

	req := &m.CategoryUnlockPurchaseReq{UserId: "user-1", CategoryId: "cat-1", CaseCount: 10}
	want := &m.CategoryUnlockRes{CategoryId: "cat-1", CasesRemaining: 10, CoinsSpent: 50}
	cum.On("Purchase", req, 50).Return(want, nil)

	got, err := svc.Purchase(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCategoryUnlockService_Purchase_InvalidCaseCount(t *testing.T) {
	svc := newCategoryUnlockTestService(mocks.NewCategoryMock(), mocks.NewCategoryUnlockMock())

	req := &m.CategoryUnlockPurchaseReq{UserId: "user-1", CategoryId: "cat-1", CaseCount: 0}
	got, err := svc.Purchase(context.Background(), req)
	assert.Nil(t, got)
	assert.Error(t, err)
}

func TestCategoryUnlockService_Purchase_CategoryNotFound(t *testing.T) {
	cm := mocks.NewCategoryMock()
	svc := newCategoryUnlockTestService(cm, mocks.NewCategoryUnlockMock())

	wantErr := errors.New("category not found")
	cm.On("GetById", &m.ById{Id: "cat-1"}).Return(nil, wantErr)

	req := &m.CategoryUnlockPurchaseReq{UserId: "user-1", CategoryId: "cat-1", CaseCount: 5}
	got, err := svc.Purchase(context.Background(), req)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}

// Purchase - balans yetarli bo'lmasa storage ErrInsufficientCoins qaytaradi, service
// uni o'zgarishsiz tarqatadi.
func TestCategoryUnlockService_Purchase_InsufficientCoins(t *testing.T) {
	cm := mocks.NewCategoryMock()
	cum := mocks.NewCategoryUnlockMock()
	svc := newCategoryUnlockTestService(cm, cum)

	coinsPerCase := 100
	cm.On("GetById", &m.ById{Id: "cat-1"}).Return(&m.CategoryRes{CoinsPerCase: &coinsPerCase}, nil)

	req := &m.CategoryUnlockPurchaseReq{UserId: "user-1", CategoryId: "cat-1", CaseCount: 5}
	cum.On("Purchase", req, 500).Return(nil, errs.ErrInsufficientCoins)

	got, err := svc.Purchase(context.Background(), req)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, errs.ErrInsufficientCoins)
}

func TestCategoryUnlockService_GetStatus(t *testing.T) {
	cm := mocks.NewCategoryMock()
	cum := mocks.NewCategoryUnlockMock()
	svc := newCategoryUnlockTestService(cm, cum)

	coinsPerCase := 5
	cm.On("GetById", &m.ById{Id: "cat-1"}).Return(&m.CategoryRes{CoinsPerCase: &coinsPerCase}, nil)
	cum.On("GetRemaining", "user-1", "cat-1").Return(7, nil)

	got, err := svc.GetStatus(context.Background(), "user-1", "cat-1")
	require.NoError(t, err)
	assert.Equal(t, &m.CategoryUnlockStatusRes{CategoryId: "cat-1", CoinsPerCase: 5, CasesRemaining: 7}, got)
}

func TestCategoryUnlockService_GetStatus_CategoryNotFound(t *testing.T) {
	cm := mocks.NewCategoryMock()
	svc := newCategoryUnlockTestService(cm, mocks.NewCategoryUnlockMock())

	wantErr := errors.New("not found")
	cm.On("GetById", &m.ById{Id: "cat-1"}).Return(nil, wantErr)

	got, err := svc.GetStatus(context.Background(), "user-1", "cat-1")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}
