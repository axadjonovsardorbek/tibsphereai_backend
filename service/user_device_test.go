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

func newUserDeviceTestService(dm *mocks.UserDeviceMock, um *mocks.UserMock) *UserDeviceService {
	// push=nil - real FCM ulanishi kerak emas, s.push!=nil tekshiruvi false bo'lib
	// obuna chaqiruvlari chetlab o'tiladi (asosiy Register/Unregister oqimini sinaydi).
	return NewUserDeviceService(&st.Storage{UserDeviceS: dm, UserS: um}, nil)
}

func TestUserDeviceService_Register(t *testing.T) {
	dm := mocks.NewUserDeviceMock()
	um := mocks.NewUserMock()
	svc := newUserDeviceTestService(dm, um)

	req := &m.UserDeviceRegisterReq{UserId: "user-1", FCMToken: "tok1", Platform: "android"}
	want := &m.Void{}
	dm.On("Register", req).Return(want, nil)
	lang := "ru"
	um.On("GetById", &m.ById{Id: "user-1"}).Return(&m.UserRes{Language: &lang}, nil)

	got, err := svc.Register(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestUserDeviceService_Register_StorageError(t *testing.T) {
	dm := mocks.NewUserDeviceMock()
	um := mocks.NewUserMock()
	svc := newUserDeviceTestService(dm, um)

	req := &m.UserDeviceRegisterReq{UserId: "user-1", FCMToken: "tok1"}
	wantErr := errors.New("db error")
	dm.On("Register", req).Return(nil, wantErr)

	got, err := svc.Register(context.Background(), req)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
	um.AssertNotCalled(t, "GetById")
}

func TestUserDeviceService_Unregister(t *testing.T) {
	dm := mocks.NewUserDeviceMock()
	um := mocks.NewUserMock()
	svc := newUserDeviceTestService(dm, um)

	req := &m.UserDeviceUnregisterReq{FCMToken: "tok1"}
	dm.On("Unregister", req).Return("user-1", nil)

	got, err := svc.Unregister(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, &m.Void{}, got)
}

func TestUserDeviceService_Unregister_NotFound(t *testing.T) {
	dm := mocks.NewUserDeviceMock()
	um := mocks.NewUserMock()
	svc := newUserDeviceTestService(dm, um)

	req := &m.UserDeviceUnregisterReq{FCMToken: "unknown"}
	dm.On("Unregister", req).Return("", nil)

	got, err := svc.Unregister(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, &m.Void{}, got)
}

func TestUserDeviceService_Unregister_StorageError(t *testing.T) {
	dm := mocks.NewUserDeviceMock()
	um := mocks.NewUserMock()
	svc := newUserDeviceTestService(dm, um)

	req := &m.UserDeviceUnregisterReq{FCMToken: "tok1"}
	wantErr := errors.New("db error")
	dm.On("Unregister", req).Return("", wantErr)

	got, err := svc.Unregister(context.Background(), req)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}
