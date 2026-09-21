package service

import (
	"context"
	"errors"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// CategoryUnlockService - Premium obunasiz, tanga evaziga bitta bo'limdan (kardiologiya,
// reanimatsiya, tez yordam va h.k.) N ta case ishlash huquqini sotib olish.
// Narx category.coins_per_case (admin sozlaydi, real raxod asosida) * case_count.
type CategoryUnlockService struct {
	storage st.Storage
}

func NewCategoryUnlockService(storage *st.Storage) *CategoryUnlockService {
	return &CategoryUnlockService{storage: *storage}
}

func (s *CategoryUnlockService) Purchase(ctx context.Context, req *m.CategoryUnlockPurchaseReq) (*m.CategoryUnlockRes, error) {
	if req.CaseCount <= 0 {
		return nil, errors.New("case_count must be positive")
	}

	category, err := s.storage.CategoryS.GetById(&m.ById{Id: req.CategoryId})
	if err != nil {
		return nil, err
	}
	coinsPerCase := 0
	if category.CoinsPerCase != nil {
		coinsPerCase = *category.CoinsPerCase
	}

	return s.storage.CategoryUnlockS.Purchase(req, coinsPerCase*req.CaseCount)
}

func (s *CategoryUnlockService) GetStatus(ctx context.Context, userId, categoryId string) (*m.CategoryUnlockStatusRes, error) {
	category, err := s.storage.CategoryS.GetById(&m.ById{Id: categoryId})
	if err != nil {
		return nil, err
	}
	coinsPerCase := 0
	if category.CoinsPerCase != nil {
		coinsPerCase = *category.CoinsPerCase
	}

	remaining, err := s.storage.CategoryUnlockS.GetRemaining(userId, categoryId)
	if err != nil {
		return nil, err
	}

	return &m.CategoryUnlockStatusRes{CategoryId: categoryId, CoinsPerCase: coinsPerCase, CasesRemaining: remaining}, nil
}
