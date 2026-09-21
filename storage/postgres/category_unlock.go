package postgres

import (
	"context"
	"fmt"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryUnlockRepo struct {
	db *pgxpool.Pool
}

func NewCategoryUnlockRepo(db *pgxpool.Pool) *CategoryUnlockRepo {
	return &CategoryUnlockRepo{db: db}
}

// Purchase - bitta tranzaksiya ichida: balansni FOR UPDATE bilan qulflab o'qiydi,
// yetarli bo'lmasa ErrInsufficientCoins, bo'lsa ayiradi + point_transactions'ga
// yozadi + category_unlocks.cases_remaining'ni upsert qiladi.
func (r *CategoryUnlockRepo) Purchase(req *m.CategoryUnlockPurchaseReq, coinCost int) (*m.CategoryUnlockRes, error) {
	ctx := context.Background()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var balance int
	err = tx.QueryRow(ctx, `SELECT coins FROM users WHERE id = $1 AND deleted_at = 0 FOR UPDATE`, req.UserId).Scan(&balance)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found: %s", req.UserId)
	}
	if err != nil {
		return nil, fmt.Errorf("error while reading user balance: %w", err)
	}
	if balance < coinCost {
		return nil, errs.ErrInsufficientCoins
	}

	if _, err := tx.Exec(ctx, `UPDATE users SET coins = coins - $1, updated_at = now() WHERE id = $2`, coinCost, req.UserId); err != nil {
		return nil, fmt.Errorf("error while deducting coins: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO point_transactions (id, user_id, kind, delta, reason, reference_id)
		VALUES ($1, $2, 'coin', $3, 'category_unlock', $4)
	`, uuid.NewString(), req.UserId, -coinCost, req.CategoryId); err != nil {
		return nil, fmt.Errorf("error while logging point transaction: %w", err)
	}

	var remaining int
	err = tx.QueryRow(ctx, `
		INSERT INTO category_unlocks (id, user_id, category_id, cases_remaining)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, category_id)
		DO UPDATE SET cases_remaining = category_unlocks.cases_remaining + $4, updated_at = now()
		RETURNING cases_remaining
	`, uuid.NewString(), req.UserId, req.CategoryId, req.CaseCount).Scan(&remaining)
	if err != nil {
		return nil, fmt.Errorf("error while upserting category unlock: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("error while committing category unlock purchase: %w", err)
	}

	return &m.CategoryUnlockRes{CategoryId: req.CategoryId, CasesRemaining: remaining, CoinsSpent: coinCost}, nil
}

func (r *CategoryUnlockRepo) GetRemaining(userId, categoryId string) (int, error) {
	var remaining int
	err := r.db.QueryRow(context.Background(), `
		SELECT cases_remaining FROM category_unlocks WHERE user_id = $1 AND category_id = $2
	`, userId, categoryId).Scan(&remaining)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("error while getting category unlock: %w", err)
	}
	return remaining, nil
}

func (r *CategoryUnlockRepo) ConsumeOne(userId, categoryId string) (bool, error) {
	var remaining int
	err := r.db.QueryRow(context.Background(), `
		UPDATE category_unlocks SET cases_remaining = cases_remaining - 1, updated_at = now()
		WHERE user_id = $1 AND category_id = $2 AND cases_remaining > 0
		RETURNING cases_remaining
	`, userId, categoryId).Scan(&remaining)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error while consuming category unlock: %w", err)
	}
	return true, nil
}
