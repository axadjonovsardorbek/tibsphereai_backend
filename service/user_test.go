package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"tibsphereai/internal/config"
	"tibsphereai/internal/push"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"
	"tibsphereai/storage/mocks"
	st "tibsphereai/storage/postgres"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testRedis - haqiqiy go-redis protokoli bilan gaplashadigan xotiradagi server
// (miniredis), UserService'ning OTP oqimi to'g'ridan-to'g'ri *redis.Client bilan
// ishlaydi (nil-check yo'q), shuning uchun real klient shart.
func testRedis(t *testing.T) *goredis.Client {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
}

func newUserTestService(um *mocks.UserMock, dm *mocks.UserDeviceMock, rdb *goredis.Client, cfg *config.Config, pushClient *push.Client) *UserService {
	return NewUserService(&st.Storage{UserS: um, UserDeviceS: dm}, rdb, cfg, pushClient)
}

// --- SendConfirmationCode ---

func TestUserService_SendConfirmationCode_Telegram(t *testing.T) {
	rdb := testRedis(t)
	svc := newUserTestService(mocks.NewUserMock(), mocks.NewUserDeviceMock(), rdb, &config.Config{}, nil)

	_, err := svc.SendConfirmationCode(context.Background(), &m.SendConfirmationCodeReq{Identifier: "998901234567", Type: "telegram"})
	require.NoError(t, err)

	val, err := rdb.Get("998901234567").Result()
	require.NoError(t, err)
	assert.Len(t, val, 6) // GenerateOtp 6 xonali kod qaytaradi
}

func TestUserService_SendConfirmationCode_Email_EmptyIdentifier(t *testing.T) {
	rdb := testRedis(t)
	svc := newUserTestService(mocks.NewUserMock(), mocks.NewUserDeviceMock(), rdb, &config.Config{}, nil)

	_, err := svc.SendConfirmationCode(context.Background(), &m.SendConfirmationCodeReq{Identifier: "", Type: "email"})
	assert.Error(t, err)
}

func TestUserService_SendConfirmationCode_UnknownType(t *testing.T) {
	rdb := testRedis(t)
	svc := newUserTestService(mocks.NewUserMock(), mocks.NewUserDeviceMock(), rdb, &config.Config{}, nil)

	_, err := svc.SendConfirmationCode(context.Background(), &m.SendConfirmationCodeReq{Identifier: "x", Type: "bogus"})
	assert.Error(t, err)
}

// --- Confirmation ---

func TestUserService_Confirmation_Telegram_Success(t *testing.T) {
	rdb := testRedis(t)
	um := mocks.NewUserMock()
	svc := newUserTestService(um, mocks.NewUserDeviceMock(), rdb, &config.Config{}, nil)

	rdb.Set("998901234567", "111111", 5*time.Minute)
	rdb.Set("tg:998901234567", "Alice", 5*time.Minute)

	want := &m.TokenRes{AccessToken: "at"}
	um.On("TelegramLogin", &m.TelegramLoginReq{Phone: "998901234567", Name: "Alice"}).Return(want, nil)

	got, err := svc.Confirmation(context.Background(), &m.ConfirmationReq{Identifier: "998901234567", ConfirmationCode: "111111", Type: "telegram"})
	require.NoError(t, err)
	assert.Equal(t, want, got)

	// muvaffaqiyatli tasdiqdan keyin OTP Redis'dan o'chirilishi kerak.
	_, err = rdb.Get("998901234567").Result()
	assert.ErrorIs(t, err, goredis.Nil)
}

func TestUserService_Confirmation_Email_Success(t *testing.T) {
	rdb := testRedis(t)
	um := mocks.NewUserMock()
	svc := newUserTestService(um, mocks.NewUserDeviceMock(), rdb, &config.Config{}, nil)

	rdb.Set("user@mail.com", "222222", 5*time.Minute)

	want := &m.TokenRes{AccessToken: "at2"}
	um.On("EmailLogin", &m.ConfirmationReq{Identifier: "user@mail.com"}).Return(want, nil)

	got, err := svc.Confirmation(context.Background(), &m.ConfirmationReq{Identifier: "user@mail.com", ConfirmationCode: "222222", Type: "email"})
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestUserService_Confirmation_Expired(t *testing.T) {
	rdb := testRedis(t)
	svc := newUserTestService(mocks.NewUserMock(), mocks.NewUserDeviceMock(), rdb, &config.Config{}, nil)

	_, err := svc.Confirmation(context.Background(), &m.ConfirmationReq{Identifier: "no-such-key", ConfirmationCode: "111111", Type: "telegram"})
	assert.ErrorIs(t, err, errs.ErrOTPExpired)
}

func TestUserService_Confirmation_InvalidCode(t *testing.T) {
	rdb := testRedis(t)
	svc := newUserTestService(mocks.NewUserMock(), mocks.NewUserDeviceMock(), rdb, &config.Config{}, nil)

	rdb.Set("998901234567", "111111", 5*time.Minute)

	_, err := svc.Confirmation(context.Background(), &m.ConfirmationReq{Identifier: "998901234567", ConfirmationCode: "999999", Type: "telegram"})
	assert.ErrorIs(t, err, errs.ErrInvalidOTP)
}

func TestUserService_Confirmation_EmptyIdentifier(t *testing.T) {
	rdb := testRedis(t)
	svc := newUserTestService(mocks.NewUserMock(), mocks.NewUserDeviceMock(), rdb, &config.Config{}, nil)

	_, err := svc.Confirmation(context.Background(), &m.ConfirmationReq{Identifier: "", ConfirmationCode: "111111", Type: "telegram"})
	assert.Error(t, err)
}

// --- GoogleLogin (tarmoqqa chiqmasdan tekshiriladigan xato yo'llari) ---

func TestUserService_GoogleLogin_EmptyToken(t *testing.T) {
	svc := newUserTestService(mocks.NewUserMock(), mocks.NewUserDeviceMock(), nil, &config.Config{GOOGLE_CLIENT_ID: "client-id"}, nil)

	_, err := svc.GoogleLogin(context.Background(), &m.GoogleLoginReq{IdToken: ""})
	assert.Error(t, err)
}

func TestUserService_GoogleLogin_NotConfigured(t *testing.T) {
	svc := newUserTestService(mocks.NewUserMock(), mocks.NewUserDeviceMock(), nil, &config.Config{GOOGLE_CLIENT_ID: ""}, nil)

	_, err := svc.GoogleLogin(context.Background(), &m.GoogleLoginReq{IdToken: "x"})
	var appErr *errs.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, errs.ErrInternal.Slug, appErr.Slug)
}

func TestUserService_GoogleLogin_MalformedToken(t *testing.T) {
	svc := newUserTestService(mocks.NewUserMock(), mocks.NewUserDeviceMock(), nil, &config.Config{GOOGLE_CLIENT_ID: "client-id"}, nil)

	_, err := svc.GoogleLogin(context.Background(), &m.GoogleLoginReq{IdToken: "not-a-jwt"})
	var appErr *errs.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, errs.ErrGoogleTokenInvalid.Slug, appErr.Slug)
}

// --- Oddiy pass-through metodlar ---

func TestUserService_Check(t *testing.T) {
	um := mocks.NewUserMock()
	svc := newUserTestService(um, mocks.NewUserDeviceMock(), nil, &config.Config{}, nil)

	req := &m.UserCheckReq{Identifier: "998901234567"}
	want := &m.UserCheckRes{HasAccount: true}
	um.On("Check", req).Return(want, nil)

	got, err := svc.Check(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestUserService_GetById(t *testing.T) {
	um := mocks.NewUserMock()
	svc := newUserTestService(um, mocks.NewUserDeviceMock(), nil, &config.Config{}, nil)

	req := &m.ById{Id: "user-1"}
	want := &m.UserRes{}
	um.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestUserService_Delete(t *testing.T) {
	um := mocks.NewUserMock()
	svc := newUserTestService(um, mocks.NewUserDeviceMock(), nil, &config.Config{}, nil)

	req := &m.ById{Id: "user-1"}
	um.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}

func TestUserService_AddPoints(t *testing.T) {
	um := mocks.NewUserMock()
	svc := newUserTestService(um, mocks.NewUserDeviceMock(), nil, &config.Config{}, nil)

	req := &m.AddPointsReq{UserId: "user-1", XPDelta: 10, CoinsDelta: 5}
	um.On("AddPoints", req).Return(&m.Void{}, nil)

	_, err := svc.AddPoints(context.Background(), req)
	require.NoError(t, err)
}

// --- Update ---

func TestUserService_Update_Error(t *testing.T) {
	um := mocks.NewUserMock()
	svc := newUserTestService(um, mocks.NewUserDeviceMock(), nil, &config.Config{}, nil)

	req := &m.UserUpdateReq{Id: "user-1", Language: "en"}
	wantErr := errors.New("db error")
	um.On("Update", req).Return(nil, wantErr)

	_, err := svc.Update(context.Background(), req)
	assert.ErrorIs(t, err, wantErr)
}

// --- GetRating ---

func TestUserService_GetRating_ClampsLimit(t *testing.T) {
	um := mocks.NewUserMock()
	svc := newUserTestService(um, mocks.NewUserDeviceMock(), nil, &config.Config{}, nil)

	um.On("GetTop", mock.MatchedBy(func(r *m.RatingRequest) bool { return r.Limit == 10 })).
		Return([]m.RatingUserItem{}, nil)

	got, err := svc.GetRating(context.Background(), &m.RatingRequest{Limit: -5})
	require.NoError(t, err)
	assert.Equal(t, 10, got.Limit)
}

func TestUserService_GetRating_ClampsLimit_Max(t *testing.T) {
	um := mocks.NewUserMock()
	svc := newUserTestService(um, mocks.NewUserDeviceMock(), nil, &config.Config{}, nil)

	um.On("GetTop", mock.MatchedBy(func(r *m.RatingRequest) bool { return r.Limit == 100 })).
		Return([]m.RatingUserItem{}, nil)

	got, err := svc.GetRating(context.Background(), &m.RatingRequest{Limit: 1000})
	require.NoError(t, err)
	assert.Equal(t, 100, got.Limit)
}

func TestUserService_GetRating_WithUserId_IncludesMe(t *testing.T) {
	um := mocks.NewUserMock()
	svc := newUserTestService(um, mocks.NewUserDeviceMock(), nil, &config.Config{}, nil)

	req := &m.RatingRequest{Limit: 10, UserId: "user-1"}
	items := []m.RatingUserItem{{UserID: "user-2", Rank: 1}}
	me := &m.RatingUserItem{UserID: "user-1", Rank: 5, IsMe: true}
	um.On("GetTop", req).Return(items, nil)
	um.On("GetMe", req).Return(me, nil)

	got, err := svc.GetRating(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, items, got.Items)
	assert.Equal(t, me, got.Me)
}

func TestUserService_GetRating_GetTopError(t *testing.T) {
	um := mocks.NewUserMock()
	svc := newUserTestService(um, mocks.NewUserDeviceMock(), nil, &config.Config{}, nil)

	wantErr := errors.New("db down")
	um.On("GetTop", mock.Anything).Return(nil, wantErr)

	got, err := svc.GetRating(context.Background(), &m.RatingRequest{Limit: 10})
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}
