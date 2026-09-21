package service

import (
	"context"
	"fmt"
	"tibsphereai/internal/push"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

type PushService struct {
	storage st.Storage
	push    *push.Client
}

func NewPushService(storage *st.Storage, push *push.Client) *PushService {
	return &PushService{storage: *storage, push: push}
}

// SendPush - TargetAllUsers bo'lsa uz/ru/en FCM topic'larga (all_uz/all_ru/all_en) alohida
// tarjima bilan, TargetUser bo'lsa o'sha foydalanuvchining tiliga mos matn bilan uning
// tokenlariga to'g'ridan-to'g'ri (topicsiz) yuboriladi.
func (s *PushService) SendPush(ctx context.Context, target m.PushTarget, event m.PushEvent) error {
	if s.push == nil {
		return nil
	}

	switch target.Type {
	case m.TargetAllUsers:
		var firstErr error
		for _, lang := range []string{"uz", "ru", "en"} {
			msg := m.BuildLocalizedPushMessage(event, lang)
			if err := s.push.SendToTopic(ctx, m.BroadcastTopic(lang), msg.Title, msg.Body, msg.Data); err != nil {
				fmt.Println("failed to send push to topic", m.BroadcastTopic(lang), ":", err)
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		return firstErr

	case m.TargetUser:
		if target.UserID == "" {
			return fmt.Errorf("user_id required")
		}
		tokens, err := s.storage.UserDeviceS.GetTokensByUser(ctx, target.UserID)
		if err != nil {
			return err
		}
		if len(tokens) == 0 {
			return nil
		}

		lang := "uz"
		if u, uerr := s.storage.UserS.GetById(&m.ById{Id: target.UserID}); uerr == nil && u != nil && u.Language != nil && *u.Language != "" {
			lang = *u.Language
		}
		msg := m.BuildLocalizedPushMessage(event, lang)

		_, err = s.push.SendToManyTokens(ctx, tokens, msg.Title, msg.Body, msg.Data)
		return err

	default:
		return fmt.Errorf("unknown target")
	}
}
