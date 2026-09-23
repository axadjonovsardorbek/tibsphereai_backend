package postgres

import (
	"context"
	"fmt"
	m "tibsphereai/pkg/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

// CoinTransactionRepo - admin "Tangalar savdosi monitoringi" (point_transactions,
// kind='coin' bo'yicha). Yozuvlar AddPoints/category_unlock/activateTariff/applyReferral
// orqali qo'shiladi, bu yerda faqat o'qiladi.
type CoinTransactionRepo struct {
	db *pgxpool.Pool
}

func NewCoinTransactionRepo(db *pgxpool.Pool) *CoinTransactionRepo {
	return &CoinTransactionRepo{db: db}
}

func (r *CoinTransactionRepo) GetAll(req *m.CoinTransactionGetListReq) (*m.PaginationResponse, *m.CoinTransactionList, error) {
	query := `
	SELECT
		COUNT(pt.id) OVER() AS total_count,
		pt.id, pt.user_id, u.name, pt.delta, pt.reason, pt.reference_type, pt.reference_id, pt.created_at::text
	FROM point_transactions pt
	JOIN users u ON u.id = pt.user_id
	WHERE pt.kind = 'coin'
	`

	var args []interface{}
	if req.UserId != "" && req.UserId != "string" {
		args = append(args, req.UserId)
		query += fmt.Sprintf(" AND pt.user_id = $%d", len(args))
	}
	if req.Reason != "" && req.Reason != "string" {
		args = append(args, req.Reason)
		query += fmt.Sprintf(" AND pt.reason = $%d", len(args))
	}
	if req.From != "" && req.From != "string" {
		args = append(args, req.From)
		query += fmt.Sprintf(" AND pt.created_at >= $%d::date", len(args))
	}
	if req.To != "" && req.To != "string" {
		args = append(args, req.To)
		query += fmt.Sprintf(" AND pt.created_at < ($%d::date + interval '1 day')", len(args))
	}

	query += " ORDER BY pt.created_at DESC"

	limit := req.Filter.GetLimit()
	offset := req.Filter.GetOffset()
	args = append(args, limit)
	query += fmt.Sprintf(" LIMIT $%d", len(args))
	args = append(args, offset)
	query += fmt.Sprintf(" OFFSET $%d", len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting coin transactions: %w", err)
	}
	defer rows.Close()

	list := m.CoinTransactionList{}
	for rows.Next() {
		item := m.CoinTransactionRes{}
		var totalCount int
		if err := rows.Scan(&totalCount, &item.Id, &item.UserId, &item.UserName, &item.Delta, &item.Reason, &item.ReferenceType, &item.ReferenceId, &item.CreatedAt); err != nil {
			return nil, nil, fmt.Errorf("error while scanning coin transaction row: %w", err)
		}
		list.Transactions = append(list.Transactions, &item)
		list.Count = totalCount
	}

	response := m.NewPaginationResponse(req.Filter, list.Count, list.Transactions)
	return &response, &list, nil
}

func (r *CoinTransactionRepo) GetSummary() (*m.CoinMonitoringSummaryRes, error) {
	ctx := context.Background()
	res := &m.CoinMonitoringSummaryRes{}

	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(delta), 0) FROM point_transactions WHERE kind = 'coin' AND reason = 'coin_purchase'
	`).Scan(&res.TotalPurchased); err != nil {
		return nil, fmt.Errorf("error while summing coin_purchase: %w", err)
	}

	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(delta), 0) FROM point_transactions
		WHERE kind = 'coin' AND delta > 0 AND reason != 'coin_purchase'
	`).Scan(&res.TotalEarned); err != nil {
		return nil, fmt.Errorf("error while summing earned: %w", err)
	}

	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(-delta), 0) FROM point_transactions WHERE kind = 'coin' AND delta < 0
	`).Scan(&res.TotalSpent); err != nil {
		return nil, fmt.Errorf("error while summing spent: %w", err)
	}

	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(coins), 0) FROM users WHERE deleted_at = 0
	`).Scan(&res.CoinsInCirculation); err != nil {
		return nil, fmt.Errorf("error while summing coins in circulation: %w", err)
	}

	return res, nil
}
