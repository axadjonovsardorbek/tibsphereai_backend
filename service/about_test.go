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

func newAboutTestService(am *mocks.AboutMock) *AboutService {
	return NewAboutService(&st.Storage{AboutS: am}, nil)
}

func TestAboutService_Create(t *testing.T) {
	am := mocks.NewAboutMock()
	svc := newAboutTestService(am)

	req := &m.AboutCreateReq{}
	want := &m.ById{Id: "about-1"}
	am.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	am.AssertExpectations(t)
}

func TestAboutService_Create_Error(t *testing.T) {
	am := mocks.NewAboutMock()
	svc := newAboutTestService(am)

	req := &m.AboutCreateReq{}
	wantErr := errors.New("db error")
	am.On("Create", req).Return(nil, wantErr)

	got, err := svc.Create(context.Background(), req)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}

func TestAboutService_GetById(t *testing.T) {
	am := mocks.NewAboutMock()
	svc := newAboutTestService(am)

	req := &m.ById{Id: "about-1"}
	want := &m.AboutRes{Id: "about-1"}
	am.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAboutService_Update(t *testing.T) {
	am := mocks.NewAboutMock()
	svc := newAboutTestService(am)

	req := &m.AboutUpdateReq{}
	am.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
}

func TestAboutService_Delete(t *testing.T) {
	am := mocks.NewAboutMock()
	svc := newAboutTestService(am)

	req := &m.ById{Id: "about-1"}
	am.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}

func TestAboutService_GetByIdMobile(t *testing.T) {
	am := mocks.NewAboutMock()
	svc := newAboutTestService(am)

	req := &m.ByIdMobile{Id: "about-1"}
	want := &m.AboutMobileRes{Id: "about-1"}
	am.On("GetByIdMobile", req).Return(want, nil)

	got, err := svc.GetByIdMobile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// GetAll/GetAllMobile: rdb=nil bo'lsa keshlash butunlay chetlab o'tiladi,
// har doim storage'ga boradi (res.Total==0 shart bajariladi). Diqqat: service
// qatlami storage'dan qaytgan *AboutList qiymatini ataylab e'tiborsiz qoldiradi
// (handler ham uni ishlatmaydi, PaginationResponse.Data'ni ishlatadi) - shuning
// uchun list qiymati doim nil qaytadi, bu real xatti-harakat.
func TestAboutService_GetAll(t *testing.T) {
	am := mocks.NewAboutMock()
	svc := newAboutTestService(am)

	req := &m.AboutGetListReq{Title: "x"}
	wantPag := &m.PaginationResponse{Total: 3}
	am.On("GetAll", req).Return(wantPag, &m.AboutList{}, nil)

	pag, list, err := svc.GetAll(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Nil(t, list)
}

func TestAboutService_GetAll_Error(t *testing.T) {
	am := mocks.NewAboutMock()
	svc := newAboutTestService(am)

	req := &m.AboutGetListReq{}
	wantErr := errors.New("db down")
	am.On("GetAll", req).Return(nil, nil, wantErr)

	pag, list, err := svc.GetAll(context.Background(), req)
	assert.Nil(t, pag)
	assert.Nil(t, list)
	assert.ErrorIs(t, err, wantErr)
}

func TestAboutService_GetAllMobile(t *testing.T) {
	am := mocks.NewAboutMock()
	svc := newAboutTestService(am)

	req := &m.AboutGetListMobileReq{Language: "uz"}
	wantPag := &m.PaginationResponse{Total: 1}
	am.On("GetAllMobile", req).Return(wantPag, &m.AboutMobileList{}, nil)

	pag, list, err := svc.GetAllMobile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Nil(t, list)
}
