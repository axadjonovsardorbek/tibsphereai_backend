package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"sync"
	models "tibsphereai/pkg/models"
	"time"

	"tibsphereai/internal/helper"
	"tibsphereai/internal/ratelimit"
	"tibsphereai/internal/tglink"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/jackc/pgx/v5/pgconn"
)

// --- Telegram bot orqali OTP'siz kirish va hisobga telefon/email ulash (TZ_MVP: Auth) ---

type TelegramLinkStartRes struct {
	Token     string `json:"token"`
	BotUrl    string `json:"bot_url"`
	ExpiresIn int    `json:"expires_in"` // soniya
}

type TelegramLinkCheckRes struct {
	Status string           `json:"status"` // pending | confirmed
	Auth   *models.TokenRes `json:"auth,omitempty"`
}

var (
	botNameMu   sync.Mutex
	botNameVal  string
	botNameTime time.Time
)

// botUsername - login botining username'i: TG_BOT_USERNAME env, bo'lmasa Telegram getMe (1 soat keshlanadi).
func (s *UserService) botUsername(ctx context.Context) (string, error) {
	if s.cfg.TG_BOT_USERNAME != "" {
		return s.cfg.TG_BOT_USERNAME, nil
	}
	botNameMu.Lock()
	defer botNameMu.Unlock()
	if botNameVal != "" && time.Since(botNameTime) < time.Hour {
		return botNameVal, nil
	}
	if s.cfg.NEW_BOT_TOKEN == "" {
		return "", errors.New("telegram bot is not configured (NEW_BOT_TOKEN / TG_BOT_USERNAME)")
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.telegram.org/bot"+s.cfg.NEW_BOT_TOKEN+"/getMe", nil)
	resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("telegram getMe failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	var out struct {
		OK     bool `json:"ok"`
		Result struct {
			Username string `json:"username"`
		} `json:"result"`
	}
	if json.Unmarshal(body, &out) != nil || !out.OK || out.Result.Username == "" {
		return "", errors.New("telegram getMe returned no username")
	}
	botNameVal, botNameTime = out.Result.Username, time.Now()
	return botNameVal, nil
}

func (s *UserService) linkStart(ctx context.Context, purpose, owner string, web bool) (*TelegramLinkStartRes, error) {
	bot, err := s.botUsername(ctx)
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	token := tglink.NewToken()
	if err := tglink.Create(s.rdb, purpose, token, owner, web); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	return &TelegramLinkStartRes{
		Token: token, ExpiresIn: int(tglink.TTL.Seconds()),
		BotUrl: fmt.Sprintf("https://t.me/%s?start=%s_%s", bot, purpose, token),
	}, nil
}

// TelegramLoginStart - OTP'siz kirishni boshlaydi: bir martalik token va bot havolasini qaytaradi.
func (s *UserService) TelegramLoginStart(ctx context.Context, web bool) (*TelegramLinkStartRes, error) {
	return s.linkStart(ctx, tglink.PurposeLogin, "", web)
}

// TelegramLoginCheck - ilova poll qiladi: bot tasdiqlagan bo'lsa tokenlar (kirish yoki ro'yxatdan o'tish),
// aks holda pending. Token bir martalik.
func (s *UserService) TelegramLoginCheck(ctx context.Context, token, referral string) (*TelegramLinkCheckRes, error) {
	info, ok, err := tglink.Consume(s.rdb, tglink.PurposeLogin, token)
	if errors.Is(err, tglink.ErrNotFound) {
		return nil, errs.ErrOTPExpired
	}
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if !ok {
		return &TelegramLinkCheckRes{Status: tglink.StatusPending}, nil
	}
	tok, err := s.storage.UserS.TelegramLogin(&m.TelegramLoginReq{Phone: info.Phone, Name: info.Name, ReferralCode: referral})
	if err != nil {
		return nil, err
	}
	return &TelegramLinkCheckRes{Status: tglink.StatusConfirmed, Auth: tok}, nil
}

// PhoneLinkStart - joriy hisobga telefon raqam ulash/almashtirishni boshlaydi (raqam Telegram orqali tasdiqlanadi).
func (s *UserService) PhoneLinkStart(ctx context.Context, userId string, web bool) (*TelegramLinkStartRes, error) {
	return s.linkStart(ctx, tglink.PurposePhone, userId, web)
}

// PhoneLinkConfirm - bot tasdiqlagan raqamni hisobga yozadi. Raqam boshqa hisobda bo'lsa 409.
func (s *UserService) PhoneLinkConfirm(ctx context.Context, userId, token string) (*TelegramLinkCheckRes, error) {
	info, err := tglink.Get(s.rdb, tglink.PurposePhone, token)
	if errors.Is(err, tglink.ErrNotFound) {
		return nil, errs.ErrOTPExpired
	}
	if err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if info.Owner != userId {
		return nil, errs.ErrForbidden
	}
	if info.Status != tglink.StatusConfirmed {
		return &TelegramLinkCheckRes{Status: tglink.StatusPending}, nil
	}
	if err := s.setIdentifier(ctx, userId, "phone_number", info.Phone); err != nil {
		return nil, err
	}
	s.rdb.Del("tgl:" + tglink.PurposePhone + ":" + token)
	return &TelegramLinkCheckRes{Status: tglink.StatusConfirmed}, nil
}

// setIdentifier - phone_number/email ni yangilaydi; boshqa faol hisobda band bo'lsa ErrIdentifierTaken.
func (s *UserService) setIdentifier(ctx context.Context, userId, column, value string) error {
	if column != "phone_number" && column != "email" {
		return errors.New("invalid column")
	}
	res, err := s.storage.Db.Exec(ctx, `
		UPDATE users SET `+column+` = $2, updated_at = now()
		WHERE id = $1 AND deleted_at = 0
		  AND NOT EXISTS (SELECT 1 FROM users o WHERE o.`+column+` = $2 AND o.deleted_at = 0 AND o.id <> $1)
	`, userId, value)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return errs.ErrIdentifierTaken
		}
		return err
	}
	if res.RowsAffected() == 0 {
		return errs.ErrIdentifierTaken
	}
	return nil
}

// authMethods - hisobda nechta kirish usuli bor (telefon, email, google).
func (s *UserService) authMethods(ctx context.Context, userId string) (phone, email, google bool, err error) {
	var p, e, g *string
	err = s.storage.Db.QueryRow(ctx, `SELECT phone_number, email, google_id FROM users WHERE id = $1 AND deleted_at = 0`, userId).Scan(&p, &e, &g)
	return p != nil && *p != "", e != nil && *e != "", g != nil && *g != "", err
}

// UnlinkPhone - telefon raqamni uzadi (kamida bitta boshqa kirish usuli qolishi shart).
func (s *UserService) UnlinkPhone(ctx context.Context, userId string) error {
	phone, email, google, err := s.authMethods(ctx, userId)
	if err != nil {
		return err
	}
	if !phone {
		return nil
	}
	if !email && !google {
		return errs.ErrAuthMethodRequired
	}
	_, err = s.storage.Db.Exec(ctx, `UPDATE users SET phone_number = NULL, updated_at = now() WHERE id = $1`, userId)
	return err
}

// UnlinkEmail - emailni uzadi (kamida bitta boshqa kirish usuli qolishi shart).
func (s *UserService) UnlinkEmail(ctx context.Context, userId string) error {
	phone, email, google, err := s.authMethods(ctx, userId)
	if err != nil {
		return err
	}
	if !email {
		return nil
	}
	if !phone && !google {
		return errs.ErrAuthMethodRequired
	}
	_, err = s.storage.Db.Exec(ctx, `UPDATE users SET email = NULL, updated_at = now() WHERE id = $1`, userId)
	return err
}

func emailLinkKey(userId string) string { return "emlk:" + userId }

// EmailLinkSend - hisobga email ulash: tasdiqlash kodi emailga yuboriladi (10 daqiqa).
func (s *UserService) EmailLinkSend(ctx context.Context, userId, email string) (*m.Void, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > 150 {
		return nil, errors.New("invalid email")
	}
	if !s.cfg.RATE_LIMIT_DISABLED {
		if ok, _ := ratelimit.Allow(s.rdb, "rl:emlk_send:"+userId, 3, otpSendWindow); !ok {
			return nil, errs.ErrTooManyRequests
		}
	}
	var taken bool
	_ = s.storage.Db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at = 0 AND id <> $2)`, email, userId).Scan(&taken)
	if taken {
		return nil, errs.ErrIdentifierTaken
	}
	otp := helper.GenerateOtp()
	if err := s.rdb.Set(emailLinkKey(userId), otp+"|"+email, 10*time.Minute).Err(); err != nil {
		return nil, errs.ErrInternal.Wrap(err)
	}
	if err := s.emailSender(email, otp); err != nil {
		s.rdb.Del(emailLinkKey(userId))
		return nil, err
	}
	return &m.Void{}, nil
}

// EmailLinkConfirm - kod to'g'ri bo'lsa email hisobga yoziladi.
func (s *UserService) EmailLinkConfirm(ctx context.Context, userId, email, code string) (*m.Void, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	raw, err := s.rdb.Get(emailLinkKey(userId)).Result()
	if err != nil {
		return nil, errs.ErrOTPExpired
	}
	otp, stored, _ := strings.Cut(raw, "|")
	if stored != email || otp != code {
		if !s.cfg.RATE_LIMIT_DISABLED {
			if ok, _ := ratelimit.Allow(s.rdb, "otp_try:"+emailLinkKey(userId), otpMaxAttempts, otpAttemptsTTL); !ok {
				s.rdb.Del(emailLinkKey(userId))
				return nil, errs.ErrOTPLocked
			}
		}
		return nil, errs.ErrInvalidOTP
	}
	if err := s.setIdentifier(ctx, userId, "email", email); err != nil {
		return nil, err
	}
	s.rdb.Del(emailLinkKey(userId), "otp_try:"+emailLinkKey(userId))
	return &m.Void{}, nil
}
