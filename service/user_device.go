package service

import (
	"context"
	"fmt"
	"tibsphereai/internal/push"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// UserDeviceService - ko'p qurilma push token (users.fcm_token o'rnini bosadi). Register/Unregister
// paytida foydalanuvchi tiliga mos FCM topicga (all_uz/all_ru/all_en) HAMDA shaxsiy
// UserTopic("user_"+userId)ga obuna/unobuna qilinadi - TargetUser push (PushService)
// shu shaxsiy topic orqali yuboriladi, tokenni DB'dan alohida o'qish shart emas.
type UserDeviceService struct {
	storage st.Storage
	push    *push.Client
}

func NewUserDeviceService(storage *st.Storage, pushClient *push.Client) *UserDeviceService {
	return &UserDeviceService{storage: *storage, push: pushClient}
}

func (s *UserDeviceService) Register(ctx context.Context, req *m.UserDeviceRegisterReq) (*m.Void, error) {
	res, err := s.storage.UserDeviceS.Register(req)
	if err != nil {
		return nil, err
	}

	lang := "uz"
	if u, uerr := s.storage.UserS.GetById(&m.ById{Id: req.UserId}); uerr == nil && u != nil && u.Language != nil && *u.Language != "" {
		lang = *u.Language
	}
	if s.push != nil && req.FCMToken != "" {
		if err := s.push.SubscribeToTopic(ctx, []string{req.FCMToken}, m.BroadcastTopic(lang)); err != nil {
			fmt.Println("failed to subscribe device to topic:", err)
		}
		if err := s.push.SubscribeToTopic(ctx, []string{req.FCMToken}, m.UserTopic(req.UserId)); err != nil {
			fmt.Println("failed to subscribe device to user topic:", err)
		}
	}

	return res, nil
}

func (s *UserDeviceService) Unregister(ctx context.Context, req *m.UserDeviceUnregisterReq) (*m.Void, error) {
	userId, err := s.storage.UserDeviceS.Unregister(req)
	if err != nil {
		return nil, err
	}

	if s.push != nil && userId != "" && req.FCMToken != "" {
		lang := "uz"
		if u, uerr := s.storage.UserS.GetById(&m.ById{Id: userId}); uerr == nil && u != nil && u.Language != nil && *u.Language != "" {
			lang = *u.Language
		}
		if err := s.push.UnsubscribeFromTopic(ctx, []string{req.FCMToken}, m.BroadcastTopic(lang)); err != nil {
			fmt.Println("failed to unsubscribe device from topic:", err)
		}
		if err := s.push.UnsubscribeFromTopic(ctx, []string{req.FCMToken}, m.UserTopic(userId)); err != nil {
			fmt.Println("failed to unsubscribe device from user topic:", err)
		}
	}

	return &m.Void{}, nil
}
