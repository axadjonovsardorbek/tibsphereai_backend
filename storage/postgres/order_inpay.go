package postgres

import (
	"context"
	"fmt"
	"time"

	m "tibsphereai/pkg/models"
)

// InpayAttach - inPAY yaratgan order_id'ni bizning orders qatoriga bog'laydi (payment_type='inpay').
func (r *OrderRepo) InpayAttach(orderId, inpayOrderId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	_, err := r.db.Exec(ctx, `
		UPDATE orders SET inpay_order_id = $2, payment_type = 'inpay', updated_at = now()
		WHERE id = $1::uuid AND status IN ('NEW','EXPIRED')
	`, orderId, inpayOrderId)
	if err != nil {
		return fmt.Errorf("inpay attach: %w", err)
	}
	return nil
}

// InpayComplete - inPAY webhook'i (server-to-server tasdiqlangandan keyin) natijasini qo'llaydi.
// Idempotent: PAID order qayta kelsa hech narsa o'zgarmaydi (JustPaid=false).
// success=false bo'lsa NEW order CANCELLED qilinadi. paidAmount order summasiga mos bo'lishi shart.
func (r *OrderRepo) InpayComplete(inpayOrderId string, success bool, transactionId int64, paidAmount float64) (*m.CompleteResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("inpay complete: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var orderId, status, userId, tariffId string
	var amount float64
	var coinsUsed, duration int
	err = tx.QueryRow(ctx, `
		SELECT id::text, status, amount, coins_used, user_id::text, tariff_id::text, duration
		FROM orders WHERE inpay_order_id = $1 FOR UPDATE
	`, inpayOrderId).Scan(&orderId, &status, &amount, &coinsUsed, &userId, &tariffId, &duration)
	if err != nil {
		return &m.CompleteResult{Error: ErrOrderNotExist, ErrorNote: "Order not found"}, nil
	}

	if status == "PAID" {
		return &m.CompleteResult{Error: ErrOK, ErrorNote: "Already paid"}, nil
	}

	if !success {
		if _, err := tx.Exec(ctx, `UPDATE orders SET status='CANCELLED', updated_at=now() WHERE id=$1::uuid AND status IN ('NEW','EXPIRED')`, orderId); err != nil {
			return nil, fmt.Errorf("inpay complete: cancel: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit: %w", err)
		}
		return &m.CompleteResult{Error: ErrCancelled, ErrorNote: "Transaction cancelled"}, nil
	}

	if amountToTiyin(amount) != amountToTiyin(paidAmount) {
		return &m.CompleteResult{Error: ErrIncorrectAmount, ErrorNote: "Incorrect amount"}, nil
	}

	if _, err := tx.Exec(ctx, `
		UPDATE orders SET status='PAID', paid_at=now(), inpay_transaction_id=$2, updated_at=now()
		WHERE id=$1::uuid
	`, orderId, transactionId); err != nil {
		return nil, fmt.Errorf("inpay complete: mark paid: %w", err)
	}

	if coinsUsed > 0 {
		if _, err := tx.Exec(ctx, `UPDATE users SET coins = coins - $1, updated_at = now() WHERE id = $2::uuid`, coinsUsed, userId); err != nil {
			return nil, fmt.Errorf("inpay complete: deduct coins: %w", err)
		}
	}

	if err := r.activateTariff(ctx, tx, userId, tariffId, orderId, duration); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &m.CompleteResult{
		Error: ErrOK, ErrorNote: "Success",
		JustPaid: true, PaidUserId: userId, PaidOrderId: orderId,
	}, nil
}
