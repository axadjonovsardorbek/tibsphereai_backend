package service

import (
	"context"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"

	"github.com/go-redis/redis"
)

type PromoCodeService struct {
	storage st.Storage
	rdb     *redis.Client
	userS   *UserService
}

func NewPromoCodeService(storage *st.Storage, rdb *redis.Client, userS *UserService) *PromoCodeService {
	return &PromoCodeService{storage: *storage, rdb: rdb, userS: userS}
}

// Create - Hamkor/Super Admin OLDINDAN tanga qiymati bilan promokod yaratadi (TZ 5-6-bo'lim).
func (s *PromoCodeService) Create(ctx context.Context, req *m.PromoCodeCreateReq) (*m.ById, error) {
	return s.storage.PromoCodeS.Create(req)
}

func (s *PromoCodeService) GetById(ctx context.Context, req *m.ById) (*m.PromoCodeRes, error) {
	return s.storage.PromoCodeS.GetById(req)
}

func (s *PromoCodeService) GetAll(ctx context.Context, req *m.PromoCodeGetListReq) (*m.PaginationResponse, *m.PromoCodeList, error) {
	return s.storage.PromoCodeS.GetAll(req)
}

// GetMy - "Mening Promokodlarim" (BACKEND_STRUCTURE.md 5-bo'lim).
func (s *PromoCodeService) GetMy(ctx context.Context, req *m.PromoCodeGetListReq) (*m.PaginationResponse, *m.PromoCodeList, error) {
	return s.storage.PromoCodeS.GetMy(req)
}

// Redeem - foydalanuvchi kodni kiritganda (TZ 2.3-bo'lim): kodni "used" qiladi va
// tegishli tanga miqdorini UserS.AddPoints orqali balansga qo'shadi.
func (s *PromoCodeService) Redeem(ctx context.Context, req *m.PromoCodeRedeemReq) (*m.PromoCodeRedeemRes, error) {
	value, err := s.storage.PromoCodeS.Redeem(req)
	if err != nil {
		return nil, err
	}

	if _, err := s.userS.AddPoints(ctx, &m.AddPointsReq{
		UserId:     req.UserId,
		CoinsDelta: value,
		Reason:     "promo_redeem",
	}); err != nil {
		return nil, err
	}

	return &m.PromoCodeRedeemRes{CoinsAdded: value}, nil
}

func (s *PromoCodeService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.PromoCodeS.Delete(req)
}
