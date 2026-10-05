package service

import (
	"context"
	"tibsphereai/internal/push"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// UserDeviceService - ko'p qurilma push token (users.fcm_token o'rnini bosadi). Topic'larga
// obuna/unobuna backendda qilinmaydi - frontning o'zi FCM SDK orqali boshqaradi.
type UserDeviceService struct {
	storage st.Storage
	push    *push.Client
}

func NewUserDeviceService(storage *st.Storage, pushClient *push.Client) *UserDeviceService {
	return &UserDeviceService{storage: *storage, push: pushClient}
}

func (s *UserDeviceService) Register(ctx context.Context, req *m.UserDeviceRegisterReq) (*m.Void, error) {
	return s.storage.UserDeviceS.Register(req)
}

func (s *UserDeviceService) Unregister(ctx context.Context, req *m.UserDeviceUnregisterReq) (*m.Void, error) {
	if _, err := s.storage.UserDeviceS.Unregister(req); err != nil {
		return nil, err
	}
	return &m.Void{}, nil
}
