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

func newAppRouteTestService(rm *mocks.AppRouteMock) *AppRouteService {
	return NewAppRouteService(&st.Storage{AppRouteS: rm}, nil)
}

func TestAppRouteService_Create(t *testing.T) {
	rm := mocks.NewAppRouteMock()
	svc := newAppRouteTestService(rm)

	req := &m.AppRouteCreateReq{}
	want := &m.ById{Id: "route-1"}
	rm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAppRouteService_GetById(t *testing.T) {
	rm := mocks.NewAppRouteMock()
	svc := newAppRouteTestService(rm)

	req := &m.ById{Id: "route-1"}
	want := &m.AppRouteRes{}
	rm.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// AppRouteService.GetAll - hech qanday kesh qatlami yo'q, to'g'ridan-to'g'ri
// storage natijasini (list ham) qaytaradi - bu yerda ataylab qoldirish yo'q.
func TestAppRouteService_GetAll(t *testing.T) {
	rm := mocks.NewAppRouteMock()
	svc := newAppRouteTestService(rm)

	req := &m.AppRouteGetListReq{}
	wantPag := &m.PaginationResponse{Total: 2}
	wantList := &m.AppRouteList{}
	rm.On("GetAll", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetAll(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Equal(t, wantList, list)
}

func TestAppRouteService_GetAll_Error(t *testing.T) {
	rm := mocks.NewAppRouteMock()
	svc := newAppRouteTestService(rm)

	req := &m.AppRouteGetListReq{}
	wantErr := errors.New("db down")
	rm.On("GetAll", req).Return(nil, nil, wantErr)

	pag, list, err := svc.GetAll(context.Background(), req)
	assert.Nil(t, pag)
	assert.Nil(t, list)
	assert.ErrorIs(t, err, wantErr)
}

func TestAppRouteService_Update(t *testing.T) {
	rm := mocks.NewAppRouteMock()
	svc := newAppRouteTestService(rm)

	req := &m.AppRouteUpdateReq{}
	rm.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
}

func TestAppRouteService_Delete(t *testing.T) {
	rm := mocks.NewAppRouteMock()
	svc := newAppRouteTestService(rm)

	req := &m.ById{Id: "route-1"}
	rm.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}
