package service

import (
	"context"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// CoinTransactionService - admin "Tangalar savdosi monitoringi" (BACKEND_STRUCTURE.md
// gamifikatsiya bo'limiga qo'shimcha). point_transactions (kind='coin') ni o'qiydi.
type CoinTransactionService struct {
	storage st.Storage
}

func NewCoinTransactionService(storage *st.Storage) *CoinTransactionService {
	return &CoinTransactionService{storage: *storage}
}

func (s *CoinTransactionService) GetAll(ctx context.Context, req *m.CoinTransactionGetListReq) (*m.PaginationResponse, *m.CoinTransactionList, error) {
	return s.storage.CoinTransactionS.GetAll(req)
}

func (s *CoinTransactionService) GetSummary(ctx context.Context) (*m.CoinMonitoringSummaryRes, error) {
	return s.storage.CoinTransactionS.GetSummary()
}
