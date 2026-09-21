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

func newPartnerTestService(pm *mocks.PartnerMock) *PartnerService {
	return NewPartnerService(&st.Storage{PartnerS: pm})
}

func TestPartnerService_Create(t *testing.T) {
	pm := mocks.NewPartnerMock()
	svc := newPartnerTestService(pm)

	req := &m.PartnerCreateReq{}
	want := &m.ById{Id: "partner-1"}
	pm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestPartnerService_Create_Error(t *testing.T) {
	pm := mocks.NewPartnerMock()
	svc := newPartnerTestService(pm)

	req := &m.PartnerCreateReq{}
	wantErr := errors.New("db error")
	pm.On("Create", req).Return(nil, wantErr)

	_, err := svc.Create(context.Background(), req)
	assert.ErrorIs(t, err, wantErr)
}

func TestPartnerService_GetById(t *testing.T) {
	pm := mocks.NewPartnerMock()
	svc := newPartnerTestService(pm)

	req := &m.ById{Id: "partner-1"}
	want := &m.PartnerRes{}
	pm.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestPartnerService_GetAll(t *testing.T) {
	pm := mocks.NewPartnerMock()
	svc := newPartnerTestService(pm)

	req := &m.PartnerGetListReq{}
	wantPag := &m.PaginationResponse{Total: 2}
	wantList := &m.PartnerList{}
	pm.On("GetAll", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetAll(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Equal(t, wantList, list)
}

func TestPartnerService_Update(t *testing.T) {
	pm := mocks.NewPartnerMock()
	svc := newPartnerTestService(pm)

	req := &m.PartnerUpdateReq{}
	pm.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
}

func TestPartnerService_Delete(t *testing.T) {
	pm := mocks.NewPartnerMock()
	svc := newPartnerTestService(pm)

	req := &m.ById{Id: "partner-1"}
	pm.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}

func TestPartnerService_GetAllMobile(t *testing.T) {
	pm := mocks.NewPartnerMock()
	svc := newPartnerTestService(pm)

	want := &m.PartnerMobileList{}
	pm.On("GetAllMobile").Return(want, nil)

	got, err := svc.GetAllMobile(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestPartnerService_GetAllMobile_Error(t *testing.T) {
	pm := mocks.NewPartnerMock()
	svc := newPartnerTestService(pm)

	wantErr := errors.New("db down")
	pm.On("GetAllMobile").Return(nil, wantErr)

	got, err := svc.GetAllMobile(context.Background())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}
