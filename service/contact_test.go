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

func newContactTestService(cm *mocks.ContactMock) *ContactService {
	return NewContactService(&st.Storage{ContactS: cm}, nil)
}

func TestContactService_Create(t *testing.T) {
	cm := mocks.NewContactMock()
	svc := newContactTestService(cm)

	req := &m.ContactCreateReq{}
	want := &m.ById{Id: "contact-1"}
	cm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestContactService_Create_Error(t *testing.T) {
	cm := mocks.NewContactMock()
	svc := newContactTestService(cm)

	req := &m.ContactCreateReq{}
	wantErr := errors.New("db error")
	cm.On("Create", req).Return(nil, wantErr)

	_, err := svc.Create(context.Background(), req)
	assert.ErrorIs(t, err, wantErr)
}

func TestContactService_GetById(t *testing.T) {
	cm := mocks.NewContactMock()
	svc := newContactTestService(cm)

	req := &m.ById{Id: "contact-1"}
	want := &m.ContactRes{}
	cm.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestContactService_Update(t *testing.T) {
	cm := mocks.NewContactMock()
	svc := newContactTestService(cm)

	req := &m.ContactUpdateReq{}
	cm.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
}

func TestContactService_Delete(t *testing.T) {
	cm := mocks.NewContactMock()
	svc := newContactTestService(cm)

	req := &m.ById{Id: "contact-1"}
	cm.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}

func TestContactService_GetAll(t *testing.T) {
	cm := mocks.NewContactMock()
	svc := newContactTestService(cm)

	req := &m.ContactGetListReq{}
	wantPag := &m.PaginationResponse{Total: 2}
	cm.On("GetAll", req).Return(wantPag, &m.ContactList{}, nil)

	pag, list, err := svc.GetAll(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Nil(t, list)
}

func TestContactService_GetAll_Error(t *testing.T) {
	cm := mocks.NewContactMock()
	svc := newContactTestService(cm)

	req := &m.ContactGetListReq{}
	wantErr := errors.New("db down")
	cm.On("GetAll", req).Return(nil, nil, wantErr)

	pag, list, err := svc.GetAll(context.Background(), req)
	assert.Nil(t, pag)
	assert.Nil(t, list)
	assert.ErrorIs(t, err, wantErr)
}
