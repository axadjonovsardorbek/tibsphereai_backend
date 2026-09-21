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

// push=nil bo'lsa PushService.SendPush darhol nil qaytaradi (real FCM ulanishi
// kerak emas) - NotificationService orkestratsiyasini sinash uchun yetarli.
func newNotificationTestService(nm *mocks.NotificationMock) *NotificationService {
	pgStorage := &st.Storage{NotificationS: nm}
	pushSvc := NewPushService(pgStorage, nil)
	return NewNotificationService(pgStorage, nil, pushSvc)
}

func TestNotificationService_Create_Broadcast(t *testing.T) {
	nm := mocks.NewNotificationMock()
	svc := newNotificationTestService(nm)

	req := &m.NotificationCreateReq{Type: "all", Title: `{"uz":"Salom"}`, Message: `{"uz":"Xabar"}`}
	want := &m.ById{Id: "notif-1"}
	nm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestNotificationService_Create_Selected(t *testing.T) {
	nm := mocks.NewNotificationMock()
	svc := newNotificationTestService(nm)

	req := &m.NotificationCreateReq{Type: "selected", UserId: "user-1", Title: `{"uz":"Salom"}`, Message: `{"uz":"Xabar"}`}
	want := &m.ById{Id: "notif-2"}
	nm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

// storage.Create xato bersa, push umuman urinilmasligi kerak (xato darhol qaytadi).
func TestNotificationService_Create_StorageError(t *testing.T) {
	nm := mocks.NewNotificationMock()
	svc := newNotificationTestService(nm)

	req := &m.NotificationCreateReq{Title: `{"uz":"x"}`, Message: `{"uz":"y"}`}
	wantErr := errors.New("db error")
	nm.On("Create", req).Return(nil, wantErr)

	got, err := svc.Create(context.Background(), req)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}

func TestNotificationService_GetById(t *testing.T) {
	nm := mocks.NewNotificationMock()
	svc := newNotificationTestService(nm)

	req := &m.ById{Id: "notif-1"}
	want := &m.MasterNotificationRes{}
	nm.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestNotificationService_GetAll(t *testing.T) {
	nm := mocks.NewNotificationMock()
	svc := newNotificationTestService(nm)

	req := &m.NotificationGetListReq{}
	wantPag := &m.PaginationResponse{Total: 2}
	wantList := &m.MasterNotificationList{}
	nm.On("GetAll", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetAll(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Nil(t, list) // service kesh bo'lmasa ham storage list qiymatini uzatmaydi
}

func TestNotificationService_Update(t *testing.T) {
	nm := mocks.NewNotificationMock()
	svc := newNotificationTestService(nm)

	req := &m.MasterNotificationUpdateReq{}
	nm.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
}

func TestNotificationService_Delete(t *testing.T) {
	nm := mocks.NewNotificationMock()
	svc := newNotificationTestService(nm)

	req := &m.ById{Id: "notif-1"}
	nm.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}

func TestNotificationService_UserNotifications(t *testing.T) {
	nm := mocks.NewNotificationMock()
	svc := newNotificationTestService(nm)

	req := &m.NotificationGetListReq{UserId: "user-1"}
	wantPag := &m.PaginationResponse{Total: 1}
	wantList := &m.UserNotificationList{}
	nm.On("UserNotifications", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.UserNotifications(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Nil(t, list)
}

func TestNotificationService_MarkAsRead(t *testing.T) {
	nm := mocks.NewNotificationMock()
	svc := newNotificationTestService(nm)

	req := &m.NotificationMarkReadReq{}
	nm.On("MarkAsRead", req).Return(&m.Void{}, nil)

	_, err := svc.MarkAsRead(context.Background(), req)
	require.NoError(t, err)
}
