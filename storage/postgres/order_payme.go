package postgres

import (
	"context"
	"fmt"
	"math"
	"time"

	m "tibsphereai/pkg/models"

	"github.com/jackc/pgx/v5"
)

// Payme JSON-RPC oqimi (docs: furasentr-payment/internal/api/handlers/payme.go, moslashtirilgan:
// bizda marketplace order emas, faqat tarif-obuna, shuning uchun ancha soddaroq).
// account.order_id = bizning orders.id (UUID), amount - tiyin (so'm*100).
//
// Xato turlari: protokol darajasidagi rad javoblari (*m.PaymeOpError) - handler ularni
// aynan shu Code/Message bilan JSON-RPC error qiladi; qolgan (DB) xatolar generic
// PaymeErrorInternal bo'lib ketadi.

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func amountToTiyin(amount float64) int64 {
	return int64(math.Round(amount * 100))
}

// PaymeCheckPerform - CheckPerformTransaction: order mavjudligi, holati va summasini tekshiradi.
func (r *OrderRepo) PaymeCheckPerform(orderId string, amountTiyin int64) error {
	var amount float64
	var status string

	err := r.db.QueryRow(context.Background(), `
		SELECT amount, status FROM orders WHERE id = $1::uuid
	`, orderId).Scan(&amount, &status)
	if err != nil {
		return &m.PaymeOpError{Code: m.PaymeErrorAccountNotFound, Message: "Order not found", Data: "account"}
	}

	switch status {
	case "PAID":
		return &m.PaymeOpError{Code: m.PaymeErrorAlreadyPaid, Message: "Order already paid", Data: "account"}
	case "CANCELLED", "EXPIRED":
		return &m.PaymeOpError{Code: m.PaymeErrorAccountNotFound, Message: "Order cancelled or expired", Data: "account"}
	}

	if amountToTiyin(amount) != amountTiyin {
		return &m.PaymeOpError{Code: m.PaymeErrorInvalidAmount, Message: "Invalid amount", Data: "amount"}
	}
	return nil
}

// PaymeCreateTransaction - CreateTransaction: idempotent (Payme id qayta kelsa - eski holatni
// qaytaradi), boshqa faol tranzaksiya bilan band bo'lsa xato, aks holda payme_id'ni bog'lab
// state=1 (created) qiladi.
func (r *OrderRepo) PaymeCreateTransaction(orderId, paymeId string, amountTiyin, timeMs int64) (*m.PaymeCreateResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("payme create: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var amount float64
	var status string
	var existingPaymeId *string
	var existingState *int
	var createTime, performTime, cancelTime *int64
	var reason *int

	err = tx.QueryRow(ctx, `
		SELECT amount, status, payme_id, payme_state, payme_create_time, payme_perform_time, payme_cancel_time, payme_reason
		FROM orders WHERE id = $1::uuid
		FOR UPDATE
	`, orderId).Scan(&amount, &status, &existingPaymeId, &existingState, &createTime, &performTime, &cancelTime, &reason)
	if err != nil {
		return nil, &m.PaymeOpError{Code: m.PaymeErrorAccountNotFound, Message: "Order not found", Data: "account"}
	}

	// 1. Idempotency - shu Payme id bilan allaqachon yaratilgan
	if existingPaymeId != nil && *existingPaymeId == paymeId {
		return &m.PaymeCreateResult{
			OrderId: orderId, CreateTime: derefInt64(createTime), PerformTime: derefInt64(performTime),
			CancelTime: derefInt64(cancelTime), State: derefIntOr(existingState, 1), Reason: reason,
		}, nil
	}

	// 2. Account band - boshqa Payme id bilan faol (state=1) tranzaksiya mavjud
	if existingPaymeId != nil && existingState != nil && *existingState == m.PaymeStateCreated {
		return nil, &m.PaymeOpError{Code: m.PaymeErrorOtherTransactionPending, Message: "Account is busy with another transaction", Data: "account"}
	}

	if status == "PAID" {
		return nil, &m.PaymeOpError{Code: m.PaymeErrorAlreadyPaid, Message: "Order already paid", Data: "account"}
	}
	if status == "CANCELLED" || status == "EXPIRED" {
		return nil, &m.PaymeOpError{Code: m.PaymeErrorAccountNotFound, Message: "Order cancelled or expired", Data: "account"}
	}
	if amountToTiyin(amount) != amountTiyin {
		return nil, &m.PaymeOpError{Code: m.PaymeErrorInvalidAmount, Message: "Invalid amount", Data: "amount"}
	}

	_, err = tx.Exec(ctx, `
		UPDATE orders
		SET payme_id = $2, payme_state = $3, payme_create_time = $4,
		    payment_type = 'payme', status = 'RESERVED', updated_at = now()
		WHERE id = $1::uuid
	`, orderId, paymeId, m.PaymeStateCreated, timeMs)
	if err != nil {
		return nil, fmt.Errorf("payme create: update order: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("payme create: commit: %w", err)
	}

	return &m.PaymeCreateResult{OrderId: orderId, CreateTime: timeMs, State: m.PaymeStateCreated}, nil
}

// PaymePerformTransaction - PerformTransaction: state 1->2, order PAID + subscription yaratiladi
// (Click'dagi ApplyComplete bilan bir xil natija). JustPaid=true bo'lsa service qatlami
// promokod/push logikasini ishga tushiradi.
func (r *OrderRepo) PaymePerformTransaction(paymeId string) (*m.PaymeTransactionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("payme perform: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var orderId, userId, tariffId string
	var duration int
	var state int
	var createTime, performTime, cancelTime *int64
	var reason *int

	var coinsUsed int
	err = tx.QueryRow(ctx, `
		SELECT id, user_id, tariff_id, duration, payme_state,
		       payme_create_time, payme_perform_time, payme_cancel_time, payme_reason, coins_used
		FROM orders WHERE payme_id = $1
		FOR UPDATE
	`, paymeId).Scan(&orderId, &userId, &tariffId, &duration, &state, &createTime, &performTime, &cancelTime, &reason, &coinsUsed)
	if err != nil {
		return nil, &m.PaymeOpError{Code: m.PaymeErrorTransactionNotFound, Message: "Transaction not found"}
	}

	if state == m.PaymeStatePerformed {
		// Idempotent - allaqachon perform qilingan
		return &m.PaymeTransactionResult{
			OrderId: orderId, CreateTime: derefInt64(createTime), PerformTime: derefInt64(performTime),
			State: state, Reason: reason,
		}, nil
	}
	if state != m.PaymeStateCreated {
		return nil, &m.PaymeOpError{Code: m.PaymeErrorCantCancel, Message: "Transaction in invalid state"}
	}

	performTimeMs := time.Now().UnixMilli()

	_, err = tx.Exec(ctx, `
		UPDATE orders
		SET payme_state = $2, payme_perform_time = $3, status = 'PAID', paid_at = now(), updated_at = now()
		WHERE id = $1::uuid
	`, orderId, m.PaymeStatePerformed, performTimeMs)
	if err != nil {
		return nil, fmt.Errorf("payme perform: update order: %w", err)
	}

	if coinsUsed > 0 {
		if _, err = tx.Exec(ctx, `UPDATE users SET coins = coins - $1, updated_at = now() WHERE id = $2`, coinsUsed, userId); err != nil {
			return nil, fmt.Errorf("payme perform: deduct coins: %w", err)
		}
	}

	if err := r.activateTariff(ctx, tx, userId, tariffId, orderId, duration); err != nil {
		return nil, fmt.Errorf("payme perform: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("payme perform: commit: %w", err)
	}

	return &m.PaymeTransactionResult{
		OrderId: orderId, CreateTime: derefInt64(createTime), PerformTime: performTimeMs,
		State: m.PaymeStatePerformed, JustPaid: true,
	}, nil
}

// PaymeCancelTransaction - CancelTransaction: state 1->-1 (oddiy bekor) yoki 2->-2 (refund).
// Refund holatida shu order_id bilan bog'langan faol subscription ham "refunded" qilinadi
// (ended_at=now(), foydalanuvchi Premium huquqini yo'qotadi).
func (r *OrderRepo) PaymeCancelTransaction(paymeId string, reason int) (*m.PaymeTransactionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("payme cancel: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var orderId string
	var state int
	var createTime, performTime, cancelTime *int64
	var existingReason *int

	err = tx.QueryRow(ctx, `
		SELECT id, payme_state, payme_create_time, payme_perform_time, payme_cancel_time, payme_reason
		FROM orders WHERE payme_id = $1
		FOR UPDATE
	`, paymeId).Scan(&orderId, &state, &createTime, &performTime, &cancelTime, &existingReason)
	if err != nil {
		return nil, &m.PaymeOpError{Code: m.PaymeErrorTransactionNotFound, Message: "Transaction not found"}
	}

	if state == m.PaymeStateCancelled || state == m.PaymeStateRefunded {
		// Idempotent - allaqachon bekor qilingan
		return &m.PaymeTransactionResult{
			OrderId: orderId, CreateTime: derefInt64(createTime), PerformTime: derefInt64(performTime),
			CancelTime: derefInt64(cancelTime), State: state, Reason: existingReason,
		}, nil
	}

	newState := m.PaymeStateCancelled
	if state == m.PaymeStatePerformed {
		newState = m.PaymeStateRefunded
	}
	cancelTimeMs := time.Now().UnixMilli()

	_, err = tx.Exec(ctx, `
		UPDATE orders
		SET payme_state = $2, payme_cancel_time = $3, payme_reason = $4, status = 'CANCELLED', updated_at = now()
		WHERE id = $1::uuid
	`, orderId, newState, cancelTimeMs, reason)
	if err != nil {
		return nil, fmt.Errorf("payme cancel: update order: %w", err)
	}

	if newState == m.PaymeStateRefunded {
		_, err = tx.Exec(ctx, `
			UPDATE subscriptions
			SET status = 'refunded', ended_at = now(), updated_at = now()
			WHERE order_id = $1::uuid AND status = 'active'
		`, orderId)
		if err != nil {
			return nil, fmt.Errorf("payme cancel: expire subscription: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("payme cancel: commit: %w", err)
	}

	return &m.PaymeTransactionResult{
		OrderId: orderId, CreateTime: derefInt64(createTime), PerformTime: derefInt64(performTime),
		CancelTime: cancelTimeMs, State: newState, Reason: &reason,
	}, nil
}

// PaymeCheckTransaction - CheckTransaction: holatni faqat o'qiydi.
func (r *OrderRepo) PaymeCheckTransaction(paymeId string) (*m.PaymeTransactionResult, error) {
	var orderId string
	var state int
	var createTime, performTime, cancelTime *int64
	var reason *int

	err := r.db.QueryRow(context.Background(), `
		SELECT id, payme_state, payme_create_time, payme_perform_time, payme_cancel_time, payme_reason
		FROM orders WHERE payme_id = $1
	`, paymeId).Scan(&orderId, &state, &createTime, &performTime, &cancelTime, &reason)
	if err != nil {
		return nil, &m.PaymeOpError{Code: m.PaymeErrorTransactionNotFound, Message: "Transaction not found"}
	}

	return &m.PaymeTransactionResult{
		OrderId: orderId, CreateTime: derefInt64(createTime), PerformTime: derefInt64(performTime),
		CancelTime: derefInt64(cancelTime), State: state, Reason: reason,
	}, nil
}

// PaymeGetStatement - GetStatement: berilgan vaqt oralig'idagi Payme tranzaksiyalari ro'yxati.
func (r *OrderRepo) PaymeGetStatement(fromMs, toMs int64) ([]m.PaymeStatementItem, error) {
	rows, err := r.db.Query(context.Background(), `
		SELECT id, payme_id, amount, payme_create_time, payme_perform_time, payme_cancel_time, payme_state, payme_reason
		FROM orders
		WHERE payme_id IS NOT NULL AND payme_create_time >= $1 AND payme_create_time <= $2
		ORDER BY payme_create_time ASC
	`, fromMs, toMs)
	if err != nil {
		if err == pgx.ErrNoRows {
			return []m.PaymeStatementItem{}, nil
		}
		return nil, fmt.Errorf("payme get statement: %w", err)
	}
	defer rows.Close()

	var list []m.PaymeStatementItem
	for rows.Next() {
		var item m.PaymeStatementItem
		var paymeId string
		var createTime, performTime, cancelTime *int64
		var reason *int
		if err := rows.Scan(&item.OrderId, &paymeId, &item.Amount, &createTime, &performTime, &cancelTime, &item.State, &reason); err != nil {
			return nil, fmt.Errorf("payme get statement: scan: %w", err)
		}
		item.PaymeId = paymeId
		item.CreateTime = derefInt64(createTime)
		item.PerformTime = derefInt64(performTime)
		item.CancelTime = derefInt64(cancelTime)
		item.Reason = reason
		list = append(list, item)
	}
	return list, nil
}

func derefIntOr(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}
