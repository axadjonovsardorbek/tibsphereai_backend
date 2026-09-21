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

	req := &m.TariffCreateReq{}
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

	req := &m.TariffUpdateReq{}
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
