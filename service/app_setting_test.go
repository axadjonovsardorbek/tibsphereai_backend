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

func newAppSettingTestService(am *mocks.AppSettingMock, defaultLim int) *AppSettingService {
	return NewAppSettingService(&st.Storage{AppSettingS: am}, defaultLim)
}

func TestAppSettingService_DailyFreeLimit_UsesSettingValue(t *testing.T) {
	am := mocks.NewAppSettingMock()
	svc := newAppSettingTestService(am, 3)

	am.On("Get", m.AppSettingDailyFreeLimit).Return(&m.AppSettingRes{Value: "7"}, nil)

	got := svc.DailyFreeLimit(context.Background())
	assert.Equal(t, 7, got)
}

// Sozlama topilmasa yoki xato bo'lsa - config'dagi default qiymatga qaytadi.
func TestAppSettingService_DailyFreeLimit_FallsBackToDefault_OnError(t *testing.T) {
	am := mocks.NewAppSettingMock()
	svc := newAppSettingTestService(am, 3)

	am.On("Get", m.AppSettingDailyFreeLimit).Return(nil, errors.New("not found"))

	got := svc.DailyFreeLimit(context.Background())
	assert.Equal(t, 3, got)
}

// Sozlama qiymati butun songa aylanmasa - ham default qiymatga qaytadi.
func TestAppSettingService_DailyFreeLimit_FallsBackToDefault_OnBadValue(t *testing.T) {
	am := mocks.NewAppSettingMock()
	svc := newAppSettingTestService(am, 3)

	am.On("Get", m.AppSettingDailyFreeLimit).Return(&m.AppSettingRes{Value: "not-a-number"}, nil)

	got := svc.DailyFreeLimit(context.Background())
	assert.Equal(t, 3, got)
}

func TestAppSettingService_HasFreeAttemptLeft_True(t *testing.T) {
	am := mocks.NewAppSettingMock()
	svc := newAppSettingTestService(am, 3)

	am.On("Get", m.AppSettingDailyFreeLimit).Return(&m.AppSettingRes{Value: "3"}, nil)
	am.On("GetTodayFreeAttemptsUsed", "user-1").Return(2, nil)

	ok, err := svc.HasFreeAttemptLeft(context.Background(), "user-1")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestAppSettingService_HasFreeAttemptLeft_False_WhenLimitReached(t *testing.T) {
	am := mocks.NewAppSettingMock()
	svc := newAppSettingTestService(am, 3)

	am.On("Get", m.AppSettingDailyFreeLimit).Return(&m.AppSettingRes{Value: "3"}, nil)
	am.On("GetTodayFreeAttemptsUsed", "user-1").Return(3, nil)

	ok, err := svc.HasFreeAttemptLeft(context.Background(), "user-1")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestAppSettingService_HasFreeAttemptLeft_Error(t *testing.T) {
	am := mocks.NewAppSettingMock()
	svc := newAppSettingTestService(am, 3)

	wantErr := errors.New("db down")
	am.On("GetTodayFreeAttemptsUsed", "user-1").Return(0, wantErr)

	ok, err := svc.HasFreeAttemptLeft(context.Background(), "user-1")
	assert.False(t, ok)
	assert.ErrorIs(t, err, wantErr)
}

func TestAppSettingService_ConsumeFreeAttempt(t *testing.T) {
	am := mocks.NewAppSettingMock()
	svc := newAppSettingTestService(am, 3)

	am.On("IncrTodayFreeAttempts", "user-1").Return(nil)

	err := svc.ConsumeFreeAttempt(context.Background(), "user-1")
	require.NoError(t, err)
}

func TestAppSettingService_GetAll(t *testing.T) {
	am := mocks.NewAppSettingMock()
	svc := newAppSettingTestService(am, 3)

	want := []m.AppSettingRes{{Key: m.AppSettingDailyFreeLimit, Value: "3"}}
	am.On("GetAll").Return(want, nil)

	got, err := svc.GetAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAppSettingService_Set(t *testing.T) {
	am := mocks.NewAppSettingMock()
	svc := newAppSettingTestService(am, 3)

	req := &m.AppSettingSetReq{Key: m.AppSettingDailyFreeLimit, Value: "5"}
	am.On("Set", req).Return(&m.Void{}, nil)

	_, err := svc.Set(context.Background(), req)
	require.NoError(t, err)
}
