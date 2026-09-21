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

func newFaqTestService(fm *mocks.FaqMock) *FaqService {
	return NewFaqService(&st.Storage{FaqS: fm}, nil)
}

func TestFaqService_Create(t *testing.T) {
	fm := mocks.NewFaqMock()
	svc := newFaqTestService(fm)

	req := &m.FaqCreateReq{}
	want := &m.ById{Id: "faq-1"}
	fm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestFaqService_Create_Error(t *testing.T) {
	fm := mocks.NewFaqMock()
	svc := newFaqTestService(fm)

	req := &m.FaqCreateReq{}
	wantErr := errors.New("db error")
	fm.On("Create", req).Return(nil, wantErr)

	_, err := svc.Create(context.Background(), req)
	assert.ErrorIs(t, err, wantErr)
}

func TestFaqService_GetById(t *testing.T) {
	fm := mocks.NewFaqMock()
	svc := newFaqTestService(fm)

	req := &m.ById{Id: "faq-1"}
	want := &m.FaqRes{}
	fm.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestFaqService_Update(t *testing.T) {
	fm := mocks.NewFaqMock()
	svc := newFaqTestService(fm)

	req := &m.FaqUpdateReq{}
	fm.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
}

func TestFaqService_Delete(t *testing.T) {
	fm := mocks.NewFaqMock()
	svc := newFaqTestService(fm)

	req := &m.ById{Id: "faq-1"}
	fm.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}

func TestFaqService_GetByIdMobile(t *testing.T) {
	fm := mocks.NewFaqMock()
	svc := newFaqTestService(fm)

	req := &m.ByIdMobile{Id: "faq-1"}
	want := &m.FaqMobileRes{}
	fm.On("GetByIdMobile", req).Return(want, nil)

	got, err := svc.GetByIdMobile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestFaqService_GetAll(t *testing.T) {
	fm := mocks.NewFaqMock()
	svc := newFaqTestService(fm)

	req := &m.FaqGetListReq{}
	wantPag := &m.PaginationResponse{Total: 2}
	fm.On("GetAll", req).Return(wantPag, &m.FaqList{}, nil)

	pag, list, err := svc.GetAll(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Nil(t, list)
}

func TestFaqService_GetAll_Error(t *testing.T) {
	fm := mocks.NewFaqMock()
	svc := newFaqTestService(fm)

	req := &m.FaqGetListReq{}
	wantErr := errors.New("db down")
	fm.On("GetAll", req).Return(nil, nil, wantErr)

	pag, list, err := svc.GetAll(context.Background(), req)
	assert.Nil(t, pag)
	assert.Nil(t, list)
	assert.ErrorIs(t, err, wantErr)
}

func TestFaqService_GetAllMobile(t *testing.T) {
	fm := mocks.NewFaqMock()
	svc := newFaqTestService(fm)

	req := &m.FaqGetListMobileReq{}
	wantPag := &m.PaginationResponse{Total: 1}
	fm.On("GetAllMobile", req).Return(wantPag, &m.FaqMobileList{}, nil)

	pag, list, err := svc.GetAllMobile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Nil(t, list)
}
