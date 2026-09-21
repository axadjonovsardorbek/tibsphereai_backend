package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"tibsphereai/internal/config"
	"tibsphereai/internal/helper"
	"tibsphereai/internal/push"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
	"time"

	"github.com/go-redis/redis"
	"google.golang.org/api/idtoken"
)

type UserService struct {
	storage st.Storage
	rdb     *redis.Client
	cfg     *config.Config
	push    *push.Client
}

func NewUserService(storage *st.Storage, rdb *redis.Client, cfg *config.Config, pushClient *push.Client) *UserService {
	return &UserService{storage: *storage, rdb: rdb, cfg: cfg, push: pushClient}
}

func (s *UserService) SendConfirmationCode(ctx context.Context, req *m.SendConfirmationCodeReq) (*m.Void, error) {
	otp := helper.GenerateOtp()

	if req.Type == "telegram" {
		s.rdb.Set(req.Identifier, otp, 5*time.Minute)
		fmt.Printf("Generated OTP for Telegram user %s: %s\n", req.Identifier, otp)
		return &m.Void{}, nil
	} else if req.Type == "email" {
		if req.Identifier == "" || req.Identifier == "string" {
			return nil, errors.New("email is required")
		}
		s.rdb.Set(req.Identifier, otp, 5*time.Minute)
		fmt.Printf("Generated OTP for email %s: %s\n", req.Identifier, otp)

		from := s.cfg.EMAIL_NAME
		logoURL := s.cfg.EMAIL_LOGO

		htmlBody := fmt.Sprintf(`<!doctype html>
	<html lang="uz">
	<head>
	  <meta charset="utf-8" />
	  <meta name="viewport" content="width=device-width, initial-scale=1" />
	</head>
	<body style="margin:0;padding:0;background:#f6f7fb;font-family:Arial,Helvetica,sans-serif;">
	  <div style="max-width:560px;margin:0 auto;padding:24px;">
		<div style="background:#ffffff;border-radius:14px;overflow:hidden;border:1px solid #e9ecf3;">
		  <div style="height:6px;background:#e11d48;"></div>
		  <div style="padding:22px 24px 24px 24px;">
			<div style="display:flex;align-items:center;gap:12px;">
			  <img src="%s" alt="MedicAI" height="38" style="display:block;border:0;outline:none;text-decoration:none;" />
			  <div>
				<div style="font-size:18px;font-weight:800;color:#111827;line-height:1.2;">MedicAI</div>
				<div style="font-size:13px;color:#6b7280;margin-top:2px;">Kirish uchun tasdiqlash kodi</div>
			  </div>
			</div>
			<div style="margin-top:18px;font-size:14px;line-height:1.6;color:#111827;">
			  Assalomu alaykum!<br><br>
			  Siz <b>MedicAI</b> platformasiga kirish uchun so‘rov yubordingiz.
			</div>
			<div style="margin:20px 0;padding:16px;border-radius:12px;background:#fff1f2;border:1px solid #fecdd3;text-align:center;">
			  <div style="font-size:12px;color:#9f1239;margin-bottom:8px;font-weight:700;">Tasdiqlash kodingiz</div>
			  <div style="font-size:30px;font-weight:900;letter-spacing:2px;color:#111827;
						  font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,'Liberation Mono','Courier New',monospace;">
				%s
			  </div>
			</div>
			<div style="font-size:13px;line-height:1.6;color:#b91c1c;font-weight:700;margin-bottom:8px;">
			  ⚠️ Hech qachon ushbu kodni boshqalar bilan ulashmang.
			</div>
			<div style="font-size:13px;line-height:1.6;color:#374151;">
			  ⏳ Kod <b>5 daqiqa</b> davomida amal qiladi.<br>
			  Agar bu so‘rovni siz yubormagan bo‘lsangiz, ushbu xabarni e’tiborsiz qoldiring.
			</div>
			<hr style="border:none;border-top:1px solid #eef0f6;margin:18px 0;">
			<div style="font-size:12px;color:#6b7280;">Hurmat bilan,<br>MedicAI</div>
		  </div>
		</div>
		<div style="text-align:center;color:#9ca3af;font-size:12px;margin-top:12px;">© MedicAI</div>
	  </div>
	</body>
	</html>`, logoURL, otp)

		subject := "MedicAI — Kirish uchun tasdiqlash kodi"
		message := fmt.Sprintf("Tasdiqlash kodingiz: %s\nKod 5 daqiqa davomida amal qiladi.", otp)

		err := helper.SendVerificationCode(m.Params{
			From:     from,
			Subject:  subject,
			To:       req.Identifier,
			Message:  message,
			HTMLBody: htmlBody,
		}, *s.cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to send verification code: %v", err)
		}

		return &m.Void{}, nil
	}

	return nil, errors.New("invalid type")
}

func (s *UserService) Confirmation(ctx context.Context, req *m.ConfirmationReq) (*m.TokenRes, error) {
	if req.Identifier == "" || req.Identifier == "string" {
		return nil, errors.New("identifier is required")
	}

	storedOtp, err := s.rdb.Get(req.Identifier).Result()
	if err == redis.Nil {
		return nil, errs.ErrOTPExpired
	} else if err != nil {
		return nil, errs.ErrInternal.Wrap(fmt.Errorf("failed to retrieve verification code from redis: %w", err))
	}

	if storedOtp != req.ConfirmationCode {
		return nil, errs.ErrInvalidOTP
	}

	defer func() {
		if err := s.rdb.Del(req.Identifier).Err(); err != nil {
			fmt.Printf("Failed to delete OTP from Redis for identifier %s: %v\n", req.Identifier, err)
		}
	}()

	switch req.Type {
	case "email":
		return s.storage.UserS.EmailLogin(&m.ConfirmationReq{Identifier: req.Identifier})
	case "telegram":
		name, _ := s.rdb.Get(fmt.Sprintf("tg:%s", req.Identifier)).Result()
		return s.storage.UserS.TelegramLogin(&m.TelegramLoginReq{Phone: req.Identifier, Name: name})
	}

	return nil, errors.New("invalid type")
}

// GoogleLogin - "1 bosqichli kirish" (BACKEND_STRUCTURE.md 1-bo'lim). Google
// id_token'ni google.golang.org/api/idtoken orqali tekshiradi, keyin storage
// qatlamiga faqat tasdiqlangan claim'larni uzatadi.
func (s *UserService) GoogleLogin(ctx context.Context, req *m.GoogleLoginReq) (*m.TokenRes, error) {
	if req.IdToken == "" {
		return nil, errors.New("id_token is required")
	}
	if s.cfg.GOOGLE_CLIENT_ID == "" {
		return nil, errs.ErrInternal.Wrap(errors.New("google login is not configured (GOOGLE_CLIENT_ID)"))
	}

	payload, err := idtoken.Validate(ctx, req.IdToken, s.cfg.GOOGLE_CLIENT_ID)
	if err != nil {
		return nil, errs.ErrGoogleTokenInvalid.Wrap(err)
	}

	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)

	return s.storage.UserS.GoogleLogin(&m.GoogleAuthReq{
		GoogleId: payload.Subject,
		Email:    email,
		Name:     name,
	})
}

func (s *UserService) Check(ctx context.Context, req *m.UserCheckReq) (*m.UserCheckRes, error) {
	return s.storage.UserS.Check(req)
}
func (s *UserService) GetById(ctx context.Context, req *m.ById) (*m.UserRes, error) {
	return s.storage.UserS.GetById(req)
}
func (s *UserService) GetAll(ctx context.Context, req *m.UserGetListReq) (*m.PaginationResponse, *m.UserList, error) {
	res := &m.PaginationResponse{}

	key := fmt.Sprintf("user_name:%s_user_phone:%s_page:%d_limit:%d",
		req.Name, req.PhoneNumber, req.Filter.Page, req.Filter.Limit)

	if s.rdb != nil {
		if b, err := s.rdb.Get(key).Bytes(); err == nil && len(b) > 0 {
			_ = json.Unmarshal(b, &res)
		}
	}

	if res.Total == 0 {
		res, _, err := s.storage.UserS.GetAll(req)
		if err != nil {
			return nil, nil, err
		}

		if s.rdb != nil {
			if b, err := json.Marshal(res); err == nil {
				_ = s.rdb.Set(key, b, 60*time.Second).Err()
			}
		}

		return res, nil, nil
	}

	return res, nil, nil
}

// Update - profil yangilanadi. Til o'zgargan bo'lsa, userning barcha registratsiya qilingan
// qurilmalari eski til topic'laridan chiqarilib, yangi til topic'iga (all_uz/all_ru/all_en)
// qayta obuna qilinadi (fon rejimida, javobni kutdirmaydi).
func (s *UserService) Update(ctx context.Context, req *m.UserUpdateReq) (*m.Void, error) {
	res, err := s.storage.UserS.Update(req)
	if err != nil {
		return res, err
	}

	if req.Language == "uz" || req.Language == "ru" || req.Language == "en" {
		go s.resyncLanguageTopic(req.Id, req.Language)
	}

	return res, nil
}

func (s *UserService) resyncLanguageTopic(userId, newLang string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("resync language topic panic:", r)
		}
	}()
	if s.push == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tokens, err := s.storage.UserDeviceS.GetTokensByUser(ctx, userId)
	if err != nil || len(tokens) == 0 {
		return
	}

	newTopic := m.BroadcastTopic(newLang)
	for _, lang := range []string{"uz", "ru", "en"} {
		topic := m.BroadcastTopic(lang)
		if topic == newTopic {
			continue
		}
		_ = s.push.UnsubscribeFromTopic(ctx, tokens, topic)
	}
	if err := s.push.SubscribeToTopic(ctx, tokens, newTopic); err != nil {
		fmt.Println("failed to subscribe to new language topic:", err)
	}
}
func (s *UserService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.UserS.Delete(req)
}
func (s *UserService) ChangePassword(ctx context.Context, req *m.ChangePasswordReq) (*m.Void, error) {
	return s.storage.UserS.ChangePassword(req)
}
func (s *UserService) RefreshPassword(ctx context.Context, req *m.RefreshPasswordReq) (*m.RefreshPassword, error) {
	return s.storage.UserS.RefreshPassword(req)
}
func (s *UserService) CreateActivity(ctx context.Context, req *m.UserActivityCreateReq) (*m.Void, error) {
	return s.storage.UserS.CreateActivity(req)
}
func (s *UserService) GetActivities(ctx context.Context, req *m.UserActivityGetListReq) (*m.ActivityStatsResponse, error) {
	return s.storage.UserS.GetActivities(req)
}
func (s *UserService) GetRating(ctx context.Context, req *m.RatingRequest) (*m.RatingResponse, error) {
	req.Limit = clampLimit(req.Limit)

	res := &m.RatingResponse{
		Type:  req.Type,
		Limit: req.Limit,
		Items: make([]m.RatingUserItem, 0, req.Limit),
	}

	key := topCacheKey(string(req.Type), req.Limit)

	if s.rdb != nil {
		if b, err := s.rdb.Get(key).Bytes(); err == nil && len(b) > 0 {
			_ = json.Unmarshal(b, &res.Items)
		}
	}

	if len(res.Items) == 0 {
		items, err := s.storage.UserS.GetTop(req)
		if err != nil {
			return nil, err
		}
		res.Items = items

		if s.rdb != nil {
			if b, err := json.Marshal(items); err == nil {
				_ = s.rdb.Set(key, b, 30*time.Second).Err()
			}
		}
	}

	if req.UserId != "" {
		me, err := s.storage.UserS.GetMe(req)
		if err != nil {
			return nil, err
		}
		res.Me = me
	}

	return res, nil
}

// AddPoints - keys yakunlanganda XP/Coin qo'shish (simulation service shu orqali chaqiradi).
func (s *UserService) AddPoints(ctx context.Context, req *m.AddPointsReq) (*m.Void, error) {
	return s.storage.UserS.AddPoints(req)
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func topCacheKey(typ string, limit int) string {
	return fmt.Sprintf("rating:top:%s:%d", typ, limit)
}
