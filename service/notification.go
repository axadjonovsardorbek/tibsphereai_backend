package service

import (
	"context"
	"encoding/json"
	"fmt"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
	"time"

	"github.com/go-redis/redis"
)

type NotificationService struct {
	storage st.Storage
	rdb     *redis.Client
	push    *PushService
}

func NewNotificationService(storage *st.Storage, rdb *redis.Client, push *PushService) *NotificationService {
	return &NotificationService{storage: *storage, rdb: rdb, push: push}
}

func (s *NotificationService) Create(ctx context.Context, req *m.NotificationCreateReq) (*m.ById, error) {
	res, err := s.storage.NotificationS.Create(req)
	if err != nil {
		return nil, err
	}

	pushTarget := m.PushTarget{Type: m.TargetAllUsers}
	if req.Type == "selected" && req.UserId != "" {
		pushTarget = m.PushTarget{Type: m.TargetUser, UserID: req.UserId}
	}

	// req.Title/req.Message - {"uz":"...","ru":"...","en":"..."} JSON string. Push til bo'yicha
	// (TargetAllUsers - har topic o'z tilida, TargetUser - foydalanuvchi tilida) shu yerdan tanlanadi.
	pushEvent := m.PushEvent{
		Type:           m.EventAdminNotification,
		AdminTitleJSON: req.Title,
		AdminBodyJSON:  req.Message,
	}

	if err := s.push.SendPush(ctx, pushTarget, pushEvent); err != nil {
		fmt.Println("Failed to send push:", err)
	}

	return res, nil
}

func (s *NotificationService) GetById(ctx context.Context, req *m.ById) (*m.MasterNotificationRes, error) {
	return s.storage.NotificationS.GetById(req)
}

func (s *NotificationService) GetAll(ctx context.Context, req *m.NotificationGetListReq) (*m.PaginationResponse, *m.MasterNotificationList, error) {
	res := &m.PaginationResponse{}

	key := fmt.Sprintf("title:%s_page:%d_limit:%d", req.Title, req.Filter.Page, req.Filter.Limit)

	if s.rdb != nil {
		if b, err := s.rdb.Get(key).Bytes(); err == nil && len(b) > 0 {
			_ = json.Unmarshal(b, &res)
		}
	}

	if res.Total == 0 {
		res, _, err := s.storage.NotificationS.GetAll(req)
		if err != nil {
			return nil, nil, err
		}

		if s.rdb != nil {
			if b, err := json.Marshal(res); err == nil {
				_ = s.rdb.Set(key, b, 30*time.Second).Err()
			}
		}

		return res, nil, nil
	}

	return res, nil, nil
}

func (s *NotificationService) Update(ctx context.Context, req *m.MasterNotificationUpdateReq) (*m.Void, error) {
	return s.storage.NotificationS.Update(req)
}

func (s *NotificationService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.NotificationS.Delete(req)
}

func (s *NotificationService) UserNotifications(ctx context.Context, req *m.NotificationGetListReq) (*m.PaginationResponse, *m.UserNotificationList, error) {
	res := &m.PaginationResponse{}

	key := fmt.Sprintf("lang:%s_user_id:%s_is_read:%s", req.Language, req.UserId, req.IsRead)

	if s.rdb != nil {
		if b, err := s.rdb.Get(key).Bytes(); err == nil && len(b) > 0 {
			_ = json.Unmarshal(b, &res)
		}
	}

	if res.Total == 0 {
		res, _, err := s.storage.NotificationS.UserNotifications(req)
		if err != nil {
			return nil, nil, err
		}

		if s.rdb != nil {
			if b, err := json.Marshal(res); err == nil {
				_ = s.rdb.Set(key, b, 30*time.Second).Err()
			}
		}

		return res, nil, nil
	}

	return res, nil, nil
}

func (s *NotificationService) MarkAsRead(ctx context.Context, req *m.NotificationMarkReadReq) (*m.Void, error) {
	return s.storage.NotificationS.MarkAsRead(req)
}
