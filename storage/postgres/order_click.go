package postgres

import (
	"context"
	"fmt"
	m "tibsphereai/pkg/models"
	"time"
)

const (
	ErrOK              = 0
	ErrSignFailed      = -1
	ErrIncorrectAmount = -2
	ErrActionNotFound  = -3
	ErrAlreadyPaid     = -4
	ErrOrderNotExist   = -5
	ErrTransNotExist   = -6
	ErrFailedUpdate    = -7
	ErrBadRequest      = -8
	ErrCancelled       = -9
)

func (r *OrderRepo) ApplyPrepare(p *m.ApplyPrepareParams) (*m.PrepareResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var amount float64
	var status string
	var existingPrepare *int64
	var metaOld []byte

	err = tx.QueryRow(ctx, `
		SELECT amount::float8, status::text, merchant_prepare_id, meta
		FROM orders
		WHERE id=$1::uuid
		FOR UPDATE
	`, p.OrderId).Scan(&amount, &status, &existingPrepare, &metaOld)

	if err != nil {
		return &m.PrepareResult{Error: ErrOrderNotExist, ErrorNote: "Order not found"}, nil
	}
	if amount != p.Amount {
		return &m.PrepareResult{Error: ErrIncorrectAmount, ErrorNote: "Incorrect parameter amount"}, nil
	}
	if status == "PAID" {
		var ep int64
		if existingPrepare != nil {
			ep = *existingPrepare
		}
		return &m.PrepareResult{Error: ErrAlreadyPaid, ErrorNote: "Already paid", MerchantPrepareId: ep}, nil
	}
	if status == "CANCELLED" {
		var ep int64
		if existingPrepare != nil {
			ep = *existingPrepare
		}
		return &m.PrepareResult{Error: ErrCancelled, ErrorNote: "Transaction cancelled", MerchantPrepareId: ep}, nil
	}

	_, _ = tx.Exec(ctx, `
		UPDATE orders
		SET meta = COALESCE(meta,'{}'::jsonb) || COALESCE($2::jsonb,'{}'::jsonb),
		    updated_at=now()
		WHERE id=$1::uuid
	`, p.OrderId, string(p.MetaPatchJSON))

	if existingPrepare != nil {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit: %w", err)
		}
		return &m.PrepareResult{Error: ErrOK, ErrorNote: "Success", MerchantPrepareId: *existingPrepare}, nil
	}

	_, err = tx.Exec(ctx, `
		UPDATE orders
		SET status='RESERVED',
		    click_trans_id=$2,
		    click_paydoc_id=$3,
		    merchant_prepare_id=$4,
		    updated_at=now()
		WHERE id=$1::uuid
	`, p.OrderId, p.ClickTransId, p.ClickPaydocId, p.MerchantPrepareId)
	if err != nil {
		return &m.PrepareResult{Error: ErrFailedUpdate, ErrorNote: "Failed to update"}, nil
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &m.PrepareResult{Error: ErrOK, ErrorNote: "Success", MerchantPrepareId: p.MerchantPrepareId}, nil
}

// ApplyComplete - to'lovni PAID qiladi va faol subscription'ni yaratadi/uzaytiradi
// (imed'dagi "user_courses" o'rniga "subscriptions"). Muvaffaqiyatli PAID bo'lsa
// JustPaid/PaidUserId to'ldiriladi - service qatlami shu orqali promokod chiqaradi
// (BACKEND_STRUCTURE.md 5-bo'lim).
func (r *OrderRepo) ApplyComplete(p *m.ApplyCompleteParams) (*m.CompleteResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var status string
	var prepare *int64
	var paidAt *time.Time
	var coinsUsed int
	var userId, tariffId string
	var duration int

	err = tx.QueryRow(ctx, `
		SELECT status::text, merchant_prepare_id, paid_at, coins_used, user_id, tariff_id, duration
		FROM orders
		WHERE id=$1::uuid
		FOR UPDATE
	`, p.OrderId).Scan(&status, &prepare, &paidAt, &coinsUsed, &userId, &tariffId, &duration)

	if err != nil {
		return &m.CompleteResult{Error: ErrOrderNotExist, ErrorNote: "Order not found"}, nil
	}
	if prepare == nil || *prepare != p.MerchantPrepareId {
		return &m.CompleteResult{Error: ErrTransNotExist, ErrorNote: "Transaction does not exist"}, nil
	}

	_, _ = tx.Exec(ctx, `
		UPDATE orders
		SET meta = COALESCE(meta,'{}'::jsonb) || COALESCE($2::jsonb,'{}'::jsonb),
		    updated_at=now()
		WHERE id=$1::uuid
	`, p.OrderId, string(p.MetaPatchJSON))

	if status == "PAID" {
		conf := int64(0)
		if paidAt != nil {
			conf = paidAt.Unix()
		}
		_ = tx.Commit(ctx)
		return &m.CompleteResult{Error: ErrOK, ErrorNote: "Success", MerchantConfirmId: conf}, nil
	}

	if p.ErrorCode != 0 {
		_, err = tx.Exec(ctx, `
			UPDATE orders SET status='CANCELLED', updated_at=now()
			WHERE id=$1::uuid
		`, p.OrderId)
		if err != nil {
			return &m.CompleteResult{Error: ErrFailedUpdate, ErrorNote: "Failed to update"}, nil
		}
		_ = tx.Commit(ctx)
		return &m.CompleteResult{Error: ErrCancelled, ErrorNote: "Transaction cancelled", MerchantConfirmId: 0}, nil
	}

	_, err = tx.Exec(ctx, `
		UPDATE orders
		SET status='PAID',
		    paid_at=now(),
		    click_trans_id=$2,
		    updated_at=now()
		WHERE id=$1::uuid
	`, p.OrderId, p.ClickTransId)
	if err != nil {
		return &m.CompleteResult{Error: ErrFailedUpdate, ErrorNote: "Failed to update"}, nil
	}

	if coinsUsed > 0 {
		if _, err = tx.Exec(ctx, `UPDATE users SET coins = coins - $1, updated_at = now() WHERE id = $2`, coinsUsed, userId); err != nil {
			return &m.CompleteResult{Error: ErrFailedUpdate, ErrorNote: "Failed to deduct coins"}, nil
		}
	}

	if err := r.activateTariff(ctx, tx, userId, tariffId, p.OrderId, duration); err != nil {
		return &m.CompleteResult{Error: ErrFailedUpdate, ErrorNote: "Failed to activate tariff"}, nil
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &m.CompleteResult{
		Error: ErrOK, ErrorNote: "Success", MerchantConfirmId: time.Now().Unix(),
		JustPaid: userId != "", PaidUserId: userId, PaidOrderId: p.OrderId,
	}, nil
}
