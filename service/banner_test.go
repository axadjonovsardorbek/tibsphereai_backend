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

func newBannerTestService(bm *mocks.BannerMock) *BannerService {
	return NewBannerService(&st.Storage{BannerS: bm}, nil)
}

func TestBannerService_Create(t *testing.T) {
	bm := mocks.NewBannerMock()
	svc := newBannerTestService(bm)

	req := &m.BannerCreateReq{}
	want := &m.ById{Id: "banner-1"}
	bm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestBannerService_Create_Error(t *testing.T) {
	bm := mocks.NewBannerMock()
	svc := newBannerTestService(bm)

	req := &m.BannerCreateReq{}
	wantErr := errors.New("db error")
	bm.On("Create", req).Return(nil, wantErr)

	_, err := svc.Create(context.Background(), req)
	assert.ErrorIs(t, err, wantErr)
}

func TestBannerService_GetById(t *testing.T) {
	bm := mocks.NewBannerMock()
	svc := newBannerTestService(bm)

	req := &m.ById{Id: "banner-1"}
	want := &m.BannerRes{}
	bm.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestBannerService_Update(t *testing.T) {
	bm := mocks.NewBannerMock()
	svc := newBannerTestService(bm)

	req := &m.BannerUpdateReq{}
	bm.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
}

func TestBannerService_Delete(t *testing.T) {
	bm := mocks.NewBannerMock()
	svc := newBannerTestService(bm)

	req := &m.ById{Id: "banner-1"}
	bm.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}

func TestBannerService_GetByIdMobile(t *testing.T) {
	bm := mocks.NewBannerMock()
	svc := newBannerTestService(bm)

	req := &m.ByIdMobile{Id: "banner-1"}
	want := &m.BannerMobileRes{}
	bm.On("GetByIdMobile", req).Return(want, nil)

	got, err := svc.GetByIdMobile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestBannerService_GetAll(t *testing.T) {
	bm := mocks.NewBannerMock()
	svc := newBannerTestService(bm)

	req := &m.BannerGetListReq{Title: "x"}
	wantPag := &m.PaginationResponse{Total: 2}
	bm.On("GetAll", req).Return(wantPag, &m.BannerList{}, nil)

	pag, list, err := svc.GetAll(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Nil(t, list) // service qatlami ataylab list qiymatini e'tiborsiz qoldiradi
}

func TestBannerService_GetAll_Error(t *testing.T) {
	bm := mocks.NewBannerMock()
	svc := newBannerTestService(bm)

	req := &m.BannerGetListReq{}
	wantErr := errors.New("db down")
	bm.On("GetAll", req).Return(nil, nil, wantErr)

	pag, list, err := svc.GetAll(context.Background(), req)
	assert.Nil(t, pag)
	assert.Nil(t, list)
	assert.ErrorIs(t, err, wantErr)
}

func TestBannerService_GetAllMobile(t *testing.T) {
	bm := mocks.NewBannerMock()
	svc := newBannerTestService(bm)

	req := &m.BannerGetListMobileReq{Language: "uz"}
	wantPag := &m.PaginationResponse{Total: 1}
	bm.On("GetAllMobile", req).Return(wantPag, &m.BannerMobileList{}, nil)

	pag, list, err := svc.GetAllMobile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Nil(t, list)
}
