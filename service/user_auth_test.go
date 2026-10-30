package service

import (
	"context"
	"strings"
	"testing"

	"tibsphereai/internal/config"
	"tibsphereai/internal/tglink"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type authEnv struct {
	*practiceEnv
	svc   *UserService
	rdb   *redis.Client
	sent  []string // yuborilgan OTP'lar (email|otp)
	phone string
}

func newAuthEnv(t *testing.T) *authEnv {
	e := newPracticeEnv(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	e.store.UserS = st.NewUserRepo(e.pool, rdb)
	cfg := &config.Config{TG_BOT_USERNAME: "tib_test_bot", WEBSITE_URL: "https://x"}
	a := &authEnv{practiceEnv: e, rdb: rdb, phone: "+99890" + strings.ReplaceAll(uuid.NewString()[:8], "-", "1")}
	a.svc = NewUserService(e.store, rdb, cfg, nil)
	a.svc.emailSender = func(email, otp string) error { a.sent = append(a.sent, email+"|"+otp); return nil }
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = e.pool.Exec(bg, `DELETE FROM users WHERE phone_number = $1 OR email LIKE 'ptest-%@example.uz'`, a.phone)
	})
	return a
}

func TestAuth_TelegramLogin_Deeplink(t *testing.T) {
	a := newAuthEnv(t)
	ctx := context.Background()

	start, err := a.svc.TelegramLoginStart(ctx, false)
	require.NoError(t, err)
	assert.Equal(t, 300, start.ExpiresIn)
	assert.Equal(t, "https://t.me/tib_test_bot?start=login_"+start.Token, start.BotUrl)
	purpose, tok, ok := tglink.ParseStartPayload("login_" + start.Token)
	require.True(t, ok)
	assert.Equal(t, tglink.PurposeLogin, purpose)
	assert.Equal(t, start.Token, tok)

	// bot hali tasdiqlamagan
	res, err := a.svc.TelegramLoginCheck(ctx, start.Token, "")
	require.NoError(t, err)
	assert.Equal(t, "pending", res.Status)
	assert.Nil(t, res.Auth)

	// noma'lum token
	_, err = a.svc.TelegramLoginCheck(ctx, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", "")
	assert.ErrorIs(t, err, errs.ErrOTPExpired)

	// bot tasdiqlaydi -> yangi foydalanuvchi yaratiladi va tokenlar qaytadi
	require.NoError(t, tglink.Confirm(a.rdb, tglink.PurposeLogin, start.Token, tglink.Info{Phone: a.phone, Name: "Telegram Foydalanuvchi", TgID: 5}))
	res, err = a.svc.TelegramLoginCheck(ctx, start.Token, "")
	require.NoError(t, err)
	assert.Equal(t, "confirmed", res.Status)
	require.NotNil(t, res.Auth)
	assert.NotEmpty(t, res.Auth.AccessToken)
	userId := res.Auth.Id

	// token bir martalik
	_, err = a.svc.TelegramLoginCheck(ctx, start.Token, "")
	assert.ErrorIs(t, err, errs.ErrOTPExpired)

	// ikkinchi kirish xuddi shu foydalanuvchi
	s2, _ := a.svc.TelegramLoginStart(ctx, false)
	require.NoError(t, tglink.Confirm(a.rdb, tglink.PurposeLogin, s2.Token, tglink.Info{Phone: a.phone, Name: "Boshqa nom"}))
	res2, err := a.svc.TelegramLoginCheck(ctx, s2.Token, "")
	require.NoError(t, err)
	assert.Equal(t, userId, res2.Auth.Id)
}

func TestAuth_PhoneLink_Flow(t *testing.T) {
	a := newAuthEnv(t)
	ctx := context.Background()

	start, err := a.svc.PhoneLinkStart(ctx, a.userId, false)
	require.NoError(t, err)
	assert.Contains(t, start.BotUrl, "start=phone_"+start.Token)

	res, err := a.svc.PhoneLinkConfirm(ctx, a.userId, start.Token)
	require.NoError(t, err)
	assert.Equal(t, "pending", res.Status)

	// boshqa foydalanuvchi bu tokenni ishlata olmaydi
	_, err = a.svc.PhoneLinkConfirm(ctx, uuid.NewString(), start.Token)
	assert.ErrorIs(t, err, errs.ErrForbidden)

	require.NoError(t, tglink.Confirm(a.rdb, tglink.PurposePhone, start.Token, tglink.Info{Phone: a.phone}))
	res, err = a.svc.PhoneLinkConfirm(ctx, a.userId, start.Token)
	require.NoError(t, err)
	assert.Equal(t, "confirmed", res.Status)
	var got string
	_ = a.pool.QueryRow(ctx, `SELECT COALESCE(phone_number,'') FROM users WHERE id = $1`, a.userId).Scan(&got)
	assert.Equal(t, a.phone, got)
	// token bir martalik
	_, err = a.svc.PhoneLinkConfirm(ctx, a.userId, start.Token)
	assert.ErrorIs(t, err, errs.ErrOTPExpired)

	// raqam boshqa hisobda band
	other := uuid.NewString()
	_, err = a.pool.Exec(ctx, `INSERT INTO users (id, name, email) VALUES ($1, 'other', $2)`, other, "ptest-"+other[:8]+"@example.uz")
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = a.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, other) })
	s2, _ := a.svc.PhoneLinkStart(ctx, other, false)
	require.NoError(t, tglink.Confirm(a.rdb, tglink.PurposePhone, s2.Token, tglink.Info{Phone: a.phone}))
	_, err = a.svc.PhoneLinkConfirm(ctx, other, s2.Token)
	assert.ErrorIs(t, err, errs.ErrIdentifierTaken)

	// uzish: boshqa kirish usuli yo'q (faqat telefon) -> rad etiladi; email ulangach ruxsat
	assert.ErrorIs(t, a.svc.UnlinkPhone(ctx, a.userId), errs.ErrAuthMethodRequired)
	email := "ptest-" + a.userId[:8] + "@example.uz"
	_, err = a.svc.EmailLinkSend(ctx, a.userId, email)
	require.NoError(t, err)
	_, otp, _ := strings.Cut(a.sent[len(a.sent)-1], "|")
	_, err = a.svc.EmailLinkConfirm(ctx, a.userId, email, otp)
	require.NoError(t, err)
	require.NoError(t, a.svc.UnlinkPhone(ctx, a.userId))
	_ = a.pool.QueryRow(ctx, `SELECT COALESCE(phone_number,'-') FROM users WHERE id = $1`, a.userId).Scan(&got)
	assert.Equal(t, "-", got)
	// endi email'ni uzib bo'lmaydi (oxirgi usul)
	assert.ErrorIs(t, a.svc.UnlinkEmail(ctx, a.userId), errs.ErrAuthMethodRequired)
}

func TestAuth_EmailLink_Validation(t *testing.T) {
	a := newAuthEnv(t)
	ctx := context.Background()
	_, err := a.svc.EmailLinkSend(ctx, a.userId, "not-an-email")
	assert.Error(t, err)
	email := "ptest-" + a.userId[:8] + "@example.uz"
	_, err = a.svc.EmailLinkSend(ctx, a.userId, email)
	require.NoError(t, err)
	_, otp, _ := strings.Cut(a.sent[len(a.sent)-1], "|")

	// noto'g'ri kod / boshqa email
	_, err = a.svc.EmailLinkConfirm(ctx, a.userId, email, "000000x")
	assert.ErrorIs(t, err, errs.ErrInvalidOTP)
	_, err = a.svc.EmailLinkConfirm(ctx, a.userId, "other@example.uz", otp)
	assert.ErrorIs(t, err, errs.ErrInvalidOTP)
	_, err = a.svc.EmailLinkConfirm(ctx, a.userId, email, otp)
	require.NoError(t, err)
	// kod bir martalik
	_, err = a.svc.EmailLinkConfirm(ctx, a.userId, email, otp)
	assert.ErrorIs(t, err, errs.ErrOTPExpired)

	// email boshqa hisobda band
	other := uuid.NewString()
	_, err = a.pool.Exec(ctx, `INSERT INTO users (id, name) VALUES ($1, 'other')`, other)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = a.pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, other) })
	_, err = a.svc.EmailLinkSend(ctx, other, email)
	assert.ErrorIs(t, err, errs.ErrIdentifierTaken)
}

func TestAuth_OTP_BruteForce_And_SendThrottle(t *testing.T) {
	a := newAuthEnv(t)
	ctx := context.Background()
	id := a.phone

	// yuborish: 5 tadan keyin 429
	for i := 0; i < otpSendLimit; i++ {
		_, err := a.svc.SendConfirmationCode(ctx, &m.SendConfirmationCodeReq{Type: "telegram", Identifier: id})
		require.NoError(t, err)
	}
	_, err := a.svc.SendConfirmationCode(ctx, &m.SendConfirmationCodeReq{Type: "telegram", Identifier: id})
	assert.ErrorIs(t, err, errs.ErrTooManyRequests)

	// kod: 5 ta noto'g'ri urinishdan keyin 6-chisida qulflanadi va kod o'chadi
	for i := 0; i < otpMaxAttempts; i++ {
		_, err = a.svc.Confirmation(ctx, &m.ConfirmationReq{Type: "telegram", Identifier: id, ConfirmationCode: "x"})
		assert.ErrorIs(t, err, errs.ErrInvalidOTP)
	}
	_, err = a.svc.Confirmation(ctx, &m.ConfirmationReq{Type: "telegram", Identifier: id, ConfirmationCode: "x"})
	assert.ErrorIs(t, err, errs.ErrOTPLocked)
	// qulflangach to'g'ri kod ham ishlamaydi (kod o'chirilgan)
	_, err = a.svc.Confirmation(ctx, &m.ConfirmationReq{Type: "telegram", Identifier: id, ConfirmationCode: "123456"})
	assert.ErrorIs(t, err, errs.ErrOTPExpired)
}

func TestAuth_OTP_CorrectCodeResetsAttempts(t *testing.T) {
	a := newAuthEnv(t)
	ctx := context.Background()
	id := a.phone
	require.NoError(t, a.rdb.Set(id, "654321", 0).Err())
	_, err := a.svc.Confirmation(ctx, &m.ConfirmationReq{Type: "telegram", Identifier: id, ConfirmationCode: "bad"})
	assert.ErrorIs(t, err, errs.ErrInvalidOTP)
	res, err := a.svc.Confirmation(ctx, &m.ConfirmationReq{Type: "telegram", Identifier: id, ConfirmationCode: "654321"})
	require.NoError(t, err)
	assert.NotEmpty(t, res.AccessToken)
	assert.Equal(t, int64(0), a.rdb.Exists("otp_try:"+id).Val(), "urinishlar hisobi tozalanadi")
}
