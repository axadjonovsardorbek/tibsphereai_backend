package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"tibsphereai/pkg/errs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Biller - Pay-as-you-go hisob-kitobi (TZ_MVP): kunlik bepul tangalar (har kuni yangidan,
// to'planmaydi) + sotib olingan balans (users.coins). Premium obunachilar uchun tanga yechilmaydi.
// Practice, fleshkarta va yo'l xaritasi modullari umumiy foydalanadi.
type Biller struct {
	db       *pgxpool.Pool
	settings *AppSettingService
}

func NewBiller(db *pgxpool.Pool, settings *AppSettingService) *Biller {
	return &Biller{db: db, settings: settings}
}

// Wallet - foydalanuvchining joriy hamyoni.
type Wallet struct {
	HasSubscription  bool    `json:"has_subscription"`
	DailyFreeCoins   int     `json:"daily_free_coins"` // bugun berilgan bepul tangalar
	FreeCoinsLeft    int     `json:"free_coins_left"`  // bugungi qolgani
	Coins            int     `json:"coins"`            // sotib olingan/yig'ilgan balans
	CoinsPer1kTokens float64 `json:"coins_per_1k_tokens"`
}

// Spendable - sarflash mumkin bo'lgan jami tanga.
func (w Wallet) Spendable() int { return w.FreeCoinsLeft + w.Coins }

const tashkentToday = `(now() AT TIME ZONE 'Asia/Tashkent')::date`

func (b *Biller) HasSubscription(ctx context.Context, userId string) bool {
	var active bool
	_ = b.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM subscriptions WHERE user_id = $1 AND status = 'active' AND ended_at > now())
	`, userId).Scan(&active)
	return active
}

func (b *Biller) Wallet(ctx context.Context, userId string) (Wallet, error) {
	w := Wallet{HasSubscription: b.HasSubscription(ctx, userId), CoinsPer1kTokens: b.settings.CoinsPer1kTokens(ctx)}
	grant := b.settings.DailyFreeCoins(ctx)
	if _, err := b.db.Exec(ctx, `
		INSERT INTO user_daily_coins (user_id, day, granted, used) VALUES ($1, `+tashkentToday+`, $2, 0)
		ON CONFLICT (user_id, day) DO NOTHING
	`, userId, grant); err != nil {
		return w, fmt.Errorf("error while granting daily coins: %w", err)
	}
	if err := b.db.QueryRow(ctx, `
		SELECT granted, GREATEST(granted - used, 0) FROM user_daily_coins WHERE user_id = $1 AND day = `+tashkentToday,
		userId).Scan(&w.DailyFreeCoins, &w.FreeCoinsLeft); err != nil {
		return w, err
	}
	_ = b.db.QueryRow(ctx, `SELECT coins FROM users WHERE id = $1 AND deleted_at = 0`, userId).Scan(&w.Coins)
	return w, nil
}

// CanSpend - yangi AI chaqiruvini boshlash mumkinmi (obunachi yoki hamyonda tanga bor).
func (b *Biller) CanSpend(ctx context.Context, userId string) (bool, error) {
	w, err := b.Wallet(ctx, userId)
	if err != nil {
		return false, err
	}
	if w.HasSubscription || w.CoinsPer1kTokens <= 0 {
		return true, nil
	}
	return w.Spendable() > 0, nil
}

// RequireFunds - hamyon bo'sh bo'lsa ErrInsufficientCoins.
func (b *Biller) RequireFunds(ctx context.Context, userId string) error {
	ok, err := b.CanSpend(ctx, userId)
	if err != nil {
		return err
	}
	if !ok {
		return errs.ErrInsufficientCoins
	}
	return nil
}

// CoinsForTokens - tokenlarning tangadagi qiymati: ceil(tokens/1000 * tarif).
func CoinsForTokens(tokens int, rate float64) int {
	if tokens <= 0 || rate <= 0 {
		return 0
	}
	return int(math.Ceil(float64(tokens) / 1000 * rate))
}

// Charge - tangani yechadi: avval bugungi bepul tangalardan, qolgani balansdan (balansdan
// ortiq yechilmaydi). Qaytaradi: haqiqatda yechilgan tanga. Obunachilardan yechilmaydi.
func (b *Biller) Charge(ctx context.Context, userId string, coins int, reason, refType, refId string) (int, error) {
	if coins <= 0 || b.HasSubscription(ctx, userId) {
		return 0, nil
	}
	if _, err := b.Wallet(ctx, userId); err != nil { // bugungi qator mavjudligini ta'minlaydi
		return 0, err
	}
	tx, err := b.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var freeLeft int
	if err := tx.QueryRow(ctx, `
		SELECT GREATEST(granted - used, 0) FROM user_daily_coins WHERE user_id = $1 AND day = `+tashkentToday+` FOR UPDATE
	`, userId).Scan(&freeLeft); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	fromFree := min(freeLeft, coins)
	if fromFree > 0 {
		if _, err := tx.Exec(ctx, `UPDATE user_daily_coins SET used = used + $2 WHERE user_id = $1 AND day = `+tashkentToday, userId, fromFree); err != nil {
			return 0, err
		}
	}
	paid := 0
	if rest := coins - fromFree; rest > 0 {
		var balance int
		if err := tx.QueryRow(ctx, `SELECT coins FROM users WHERE id = $1 AND deleted_at = 0 FOR UPDATE`, userId).Scan(&balance); err != nil {
			return 0, err
		}
		paid = min(balance, rest)
		if paid > 0 {
			if _, err := tx.Exec(ctx, `UPDATE users SET coins = coins - $2, updated_at = now() WHERE id = $1`, userId, paid); err != nil {
				return 0, err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO point_transactions (id, user_id, kind, delta, reason, reference_type, reference_id)
				VALUES (gen_random_uuid(), $1, 'coin', $2, $3, $4, $5)
			`, userId, -paid, reason, refType, refId); err != nil {
				return 0, err
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return fromFree + paid, nil
}
