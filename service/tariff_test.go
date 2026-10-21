package service

import (
	"context"
	"errors"
	"testing"

	m "tibsphereai/pkg/models"
	"tibsphereai/storage/mocks"
	st "tibsphereai/storage/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTariffTestService(tm *mocks.TariffMock) *TariffService {
	return NewTariffService(&st.Storage{TariffS: tm}, nil)
}

func TestTariffService_Create(t *testing.T) {
	tm := mocks.NewTariffMock()
	svc := newTariffTestService(tm)

	req := &m.TariffCreateReq{Kind: "subscription", Duration: 1}
	want := &m.ById{Id: "tariff-1"}
	tm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestTariffService_GetById(t *testing.T) {
	tm := mocks.NewTariffMock()
	svc := newTariffTestService(tm)

	req := &m.ById{Id: "tariff-1"}
	want := &m.TariffRes{}
	tm.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestTariffService_GetAll(t *testing.T) {
	tm := mocks.NewTariffMock()
	svc := newTariffTestService(tm)

	req := &m.TariffGetListReq{}
	wantPag := &m.PaginationResponse{Total: 2}
	tm.On("GetAll", req).Return(wantPag, &m.TariffList{}, nil)

	pag, list, err := svc.GetAll(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Nil(t, list) // service qatlami ataylab list qiymatini e'tiborsiz qoldiradi
}

func TestTariffService_GetAll_Error(t *testing.T) {
	tm := mocks.NewTariffMock()
	svc := newTariffTestService(tm)

	req := &m.TariffGetListReq{}
	wantErr := errors.New("db down")
	tm.On("GetAll", req).Return(nil, nil, wantErr)

	pag, list, err := svc.GetAll(context.Background(), req)
	assert.Nil(t, pag)
	assert.Nil(t, list)
	assert.ErrorIs(t, err, wantErr)
}

func TestTariffService_Update(t *testing.T) {
	tm := mocks.NewTariffMock()
	svc := newTariffTestService(tm)

	req := &m.TariffUpdateReq{Kind: "subscription", Duration: 1}
	tm.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
}

func TestTariffService_Delete(t *testing.T) {
	tm := mocks.NewTariffMock()
	svc := newTariffTestService(tm)

	req := &m.ById{Id: "tariff-1"}
	tm.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}

// kind bo'yicha tekshiruv: coin_package'da duration, subscription'da coins e'tiborga olinmaydi.
func TestTariffService_Create_KindValidation(t *testing.T) {
	tm := mocks.NewTariffMock()
	svc := newTariffTestService(tm)

	coin := &m.TariffCreateReq{Kind: "coin_package", Coins: 500, Price: 10} // duration yo'q - xato emas
	tm.On("Create", coin).Return(&m.ById{Id: "t1"}, nil)
	_, err := svc.Create(context.Background(), coin)
	require.NoError(t, err)

	sub := &m.TariffCreateReq{Kind: "subscription", Duration: 3, Price: 10} // coins yo'q - xato emas
	tm.On("Create", sub).Return(&m.ById{Id: "t2"}, nil)
	_, err = svc.Create(context.Background(), sub)
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), &m.TariffCreateReq{Kind: "coin_package", Duration: 3})
	assert.Error(t, err) // coins majburiy
	_, err = svc.Create(context.Background(), &m.TariffCreateReq{Kind: "subscription", Coins: 100})
	assert.Error(t, err) // duration majburiy
	_, err = svc.Create(context.Background(), &m.TariffCreateReq{Kind: "x", Duration: 1})
	assert.Error(t, err)

	empty := &m.TariffCreateReq{Duration: 1} // kind bo'sh = subscription
	tm.On("Create", empty).Return(&m.ById{Id: "t3"}, nil)
	_, err = svc.Create(context.Background(), empty)
	require.NoError(t, err)
	assert.Equal(t, "subscription", empty.Kind)
}

// Update'da kind yuborilmasa, mavjud tarifdan olinadi.
func TestTariffService_Update_KindFromExisting(t *testing.T) {
	tm := mocks.NewTariffMock()
	svc := newTariffTestService(tm)

	kind := "coin_package"
	tm.On("GetById", &m.ById{Id: "t1"}).Return(&m.TariffRes{Kind: &kind}, nil)
	req := &m.TariffUpdateReq{Id: "t1", Coins: 200}
	tm.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "coin_package", req.Kind)
}
