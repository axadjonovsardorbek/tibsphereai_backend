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

func newCategoryTestService(cm *mocks.CategoryMock) *CategoryService {
	return NewCategoryService(&st.Storage{CategoryS: cm})
}

func TestCategoryService_Create(t *testing.T) {
	cm := mocks.NewCategoryMock()
	svc := newCategoryTestService(cm)

	req := &m.CategoryCreateReq{}
	want := &m.ById{Id: "cat-1"}
	cm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCategoryService_Create_Error(t *testing.T) {
	cm := mocks.NewCategoryMock()
	svc := newCategoryTestService(cm)

	req := &m.CategoryCreateReq{}
	wantErr := errors.New("db error")
	cm.On("Create", req).Return(nil, wantErr)

	_, err := svc.Create(context.Background(), req)
	assert.ErrorIs(t, err, wantErr)
}

func TestCategoryService_GetById(t *testing.T) {
	cm := mocks.NewCategoryMock()
	svc := newCategoryTestService(cm)

	req := &m.ById{Id: "cat-1"}
	want := &m.CategoryRes{}
	cm.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCategoryService_GetAll(t *testing.T) {
	cm := mocks.NewCategoryMock()
	svc := newCategoryTestService(cm)

	req := &m.CategoryGetListReq{}
	wantPag := &m.PaginationResponse{Total: 2}
	wantList := &m.CategoryList{}
	cm.On("GetAll", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetAll(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Equal(t, wantList, list)
}

func TestCategoryService_Update(t *testing.T) {
	cm := mocks.NewCategoryMock()
	svc := newCategoryTestService(cm)

	req := &m.CategoryUpdateReq{}
	cm.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
}

func TestCategoryService_Delete(t *testing.T) {
	cm := mocks.NewCategoryMock()
	svc := newCategoryTestService(cm)

	req := &m.ById{Id: "cat-1"}
	cm.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}

func TestCategoryService_GetByIdMobile(t *testing.T) {
	cm := mocks.NewCategoryMock()
	svc := newCategoryTestService(cm)

	req := &m.ByIdMobile{Id: "cat-1"}
	want := &m.CategoryMobileRes{}
	cm.On("GetByIdMobile", req).Return(want, nil)

	got, err := svc.GetByIdMobile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestCategoryService_GetAllMobile(t *testing.T) {
	cm := mocks.NewCategoryMock()
	svc := newCategoryTestService(cm)

	req := &m.CategoryGetListMobileReq{}
	wantPag := &m.PaginationResponse{Total: 1}
	wantList := &m.CategoryMobileList{}
	cm.On("GetAllMobile", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetAllMobile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Equal(t, wantList, list)
}
