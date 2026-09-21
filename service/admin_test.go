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

func newAdminTestService(am *mocks.AdminMock) *AdminService {
	return NewAdminService(&st.Storage{AdminS: am})
}

func TestAdminService_Login(t *testing.T) {
	am := mocks.NewAdminMock()
	svc := newAdminTestService(am)

	req := &m.LoginReq{Login: "root", Password: "secret"}
	want := &m.TokenRes{AccessToken: "at", RefreshToken: "rt"}
	am.On("Login", req).Return(want, nil)

	got, err := svc.Login(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAdminService_Login_Error(t *testing.T) {
	am := mocks.NewAdminMock()
	svc := newAdminTestService(am)

	req := &m.LoginReq{Login: "root", Password: "wrong"}
	wantErr := errors.New("invalid credentials")
	am.On("Login", req).Return(nil, wantErr)

	got, err := svc.Login(context.Background(), req)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}

func TestAdminService_GetById(t *testing.T) {
	am := mocks.NewAdminMock()
	svc := newAdminTestService(am)

	req := &m.ById{Id: "admin-1"}
	want := &m.AdminRes{}
	am.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
