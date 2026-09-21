package service

import (
	"context"
	"testing"

	"tibsphereai/internal/push"
	m "tibsphereai/pkg/models"
	"tibsphereai/storage/mocks"
	st "tibsphereai/storage/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// &push.Client{} - "msg" maydoni nil (init qilinmagan), lekin har bir metod shu
// holatni tekshiradi va "push client not initialized" xatosini qaytaradi -
// tarmoqqa chiqmasdan PushService orkestratsiya logikasini sinash uchun yetarli.
func uninitializedPushClient() *push.Client {
	return &push.Client{}
}

func newPushTestService(um *mocks.UserMock, dm *mocks.UserDeviceMock, pc *push.Client) *PushService {
	return NewPushService(&st.Storage{UserS: um, UserDeviceS: dm}, pc)
}

func TestPushService_SendPush_NilClient_NoOp(t *testing.T) {
	svc := newPushTestService(mocks.NewUserMock(), mocks.NewUserDeviceMock(), nil)

	err := svc.SendPush(context.Background(), m.PushTarget{Type: m.TargetAllUsers}, m.PushEvent{Type: m.EventNewCase})
	require.NoError(t, err)
}

// TargetAllUsers - uz/ru/en uchtala topicga ham urinib ko'rilishi kerak (barchasi
// xato bersa ham), va birinchi uchragan xato qaytarilishi kerak.
func TestPushService_SendPush_TargetAllUsers_PropagatesFirstError(t *testing.T) {
	svc := newPushTestService(mocks.NewUserMock(), mocks.NewUserDeviceMock(), uninitializedPushClient())

	err := svc.SendPush(context.Background(), m.PushTarget{Type: m.TargetAllUsers}, m.PushEvent{Type: m.EventNewCase})
	assert.ErrorContains(t, err, "push client not initialized")
}

func TestPushService_SendPush_TargetUser_EmptyUserID(t *testing.T) {
	svc := newPushTestService(mocks.NewUserMock(), mocks.NewUserDeviceMock(), uninitializedPushClient())

	err := svc.SendPush(context.Background(), m.PushTarget{Type: m.TargetUser}, m.PushEvent{Type: m.EventPaymentSuccess})
	assert.ErrorContains(t, err, "user_id required")
}

// TargetUser - foydalanuvchining hech qanday qurilmasi ro'yxatdan o'tmagan bo'lsa,
// push jo'natilmaydi va xato ham bo'lmaydi.
func TestPushService_SendPush_TargetUser_NoTokens_NoOp(t *testing.T) {
	dm := mocks.NewUserDeviceMock()
	svc := newPushTestService(mocks.NewUserMock(), dm, uninitializedPushClient())

	dm.On("GetTokensByUser", mock.Anything, "user-1").Return([]string{}, nil)

	err := svc.SendPush(context.Background(), m.PushTarget{Type: m.TargetUser, UserID: "user-1"}, m.PushEvent{Type: m.EventPaymentSuccess})
	require.NoError(t, err)
}

// TargetUser - tokenlar bor bo'lsa, foydalanuvchi tilini storage'dan olib,
// SendToManyTokens'ga urinib ko'radi (uninitialized client -> xato qaytadi).
func TestPushService_SendPush_TargetUser_WithTokens_PropagatesSendError(t *testing.T) {
	dm := mocks.NewUserDeviceMock()
	um := mocks.NewUserMock()
	svc := newPushTestService(um, dm, uninitializedPushClient())

	lang := "ru"
	dm.On("GetTokensByUser", mock.Anything, "user-1").Return([]string{"tok1", "tok2"}, nil)
	um.On("GetById", &m.ById{Id: "user-1"}).Return(&m.UserRes{Language: &lang}, nil)

	err := svc.SendPush(context.Background(), m.PushTarget{Type: m.TargetUser, UserID: "user-1"}, m.PushEvent{Type: m.EventPaymentSuccess})
	assert.ErrorContains(t, err, "push client not initialized")
}

func TestPushService_SendPush_UnknownTarget(t *testing.T) {
	svc := newPushTestService(mocks.NewUserMock(), mocks.NewUserDeviceMock(), uninitializedPushClient())

	err := svc.SendPush(context.Background(), m.PushTarget{Type: "bogus"}, m.PushEvent{Type: m.EventPaymentSuccess})
	assert.ErrorContains(t, err, "unknown target")
}
