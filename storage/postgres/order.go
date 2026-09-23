package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderRepo struct {
	db  *pgxpool.Pool
	cfg orderRepoConfig
}

// orderRepoConfig - Click pay URL yasash uchun kerakli minimal maydonlar
// (config.Config'ni to'liq import qilmaslik uchun).
type orderRepoConfig struct {
	ClickMerchantId int
	ClickServiceId  int
}

func NewOrderRepo(db *pgxpool.Pool, clickMerchantId, clickServiceId int) *OrderRepo {
	return &OrderRepo{db: db, cfg: orderRepoConfig{ClickMerchantId: clickMerchantId, ClickServiceId: clickServiceId}}
}

func (r *OrderRepo) payURL(amount float64, orderId string) string {
	return fmt.Sprintf(
		"https://my.click.uz/services/pay?merchant_id=%d&service_id=%d&amount=%.2f&transaction_param=%s",
		r.cfg.ClickMerchantId, r.cfg.ClickServiceId, amount, orderId,
	)
}

// activateTariff - to'lov PAID bo'lgandan keyin (gateway'siz, Click yoki Payme -
// uchala joyda ham) tariff.kind bo'yicha natijani beradi: "subscription" bo'lsa
// subscriptions qatori, "coin_package" bo'lsa users.coins ga qo'shiladi.
// tx ichida chaqiriladi, alohida commit qilmaydi.
func (r *OrderRepo) activateTariff(ctx context.Context, tx pgx.Tx, userId, tariffId, orderId string, duration int) error {
	var kind string
	var coins int
	if err := tx.QueryRow(ctx, `SELECT kind, coins FROM tariffs WHERE id = $1`, tariffId).Scan(&kind, &coins); err != nil {
		return fmt.Errorf("activateTariff: get tariff: %w", err)
	}

	if kind == "coin_package" {
		if _, err := tx.Exec(ctx, `UPDATE users SET coins = coins + $1, updated_at = now() WHERE id = $2`, coins, userId); err != nil {
			return fmt.Errorf("activateTariff: credit coins: %w", err)
		}
		return nil
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO subscriptions (id, user_id, tariff_id, order_id, started_at, ended_at, status)
		VALUES (gen_random_uuid(), $1, $2, $3, now(), now() + ($4 || ' months')::interval, 'active')
	`, userId, tariffId, orderId, duration); err != nil {
		return fmt.Errorf("activateTariff: create subscription: %w", err)
	}
	return nil
}

// Create - Premium obunaga (tariff) buyurtma yaratadi. TZ 2.3-bo'lim: to'liq pulga
// (Click/Payme), to'liq tangaga, yoki aralash to'lanadi. CoinsUsed so'mga
// app_settings.coin_to_uzs_rate bo'yicha aylantirilib narxdan ayiriladi; qolgan
// summa <= 0 bo'lsa gateway'siz to'g'ridan-to'g'ri PAID qilinadi (JustPaid=true).
func (r *OrderRepo) Create(req *m.OrderCreateReq) (*m.OrderCreateRes, error) {
	ctx := context.Background()

	// 1 kun ichida yaratilgan, hali to'lanmagan (pul talab qiladigan) buyurtma bo'lsa - o'shani qaytaramiz (idempotent)
	var checkId string
	var checkAmount float64
	_ = r.db.QueryRow(ctx, `
		SELECT id, amount FROM orders
		WHERE user_id = $1 AND tariff_id = $2
		  AND status IN ('NEW','EXPIRED')
		  AND created_at >= now() - interval '1 days'
		ORDER BY created_at DESC LIMIT 1
	`, req.UserId, req.TariffId).Scan(&checkId, &checkAmount)

	if checkId != "" && checkAmount > 0 {
		return &m.OrderCreateRes{Id: checkId, Url: r.payURL(checkAmount, checkId)}, nil
	}

	var duration int
	var price float64
	err := r.db.QueryRow(ctx, `
		SELECT duration, price FROM tariffs WHERE id = $1 AND deleted_at = 0
	`, req.TariffId).Scan(&duration, &price)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errs.ErrTariffNotFound
		}
		return nil, err
	}

	coinsUsed := req.CoinsUsed
	if coinsUsed < 0 {
		coinsUsed = 0
	}

	if coinsUsed > 0 {
		var balance int
		if err := r.db.QueryRow(ctx, `SELECT coins FROM users WHERE id = $1 AND deleted_at = 0`, req.UserId).Scan(&balance); err != nil {
			return nil, fmt.Errorf("error while checking coin balance: %w", err)
		}
		if coinsUsed > balance {
			return nil, errs.ErrInsufficientCoins
		}
	}

	rate := 0
	if coinsUsed > 0 {
		var rateStr string
		if err := r.db.QueryRow(ctx, `SELECT value FROM app_settings WHERE key = 'coin_to_uzs_rate'`).Scan(&rateStr); err == nil {
			fmt.Sscanf(rateStr, "%d", &rate)
		}
	}

	amount := price - float64(coinsUsed*rate)
	if amount < 0 {
		amount = 0
	}

	paymentType := "click"
	if coinsUsed > 0 && amount <= 0 {
		paymentType = "coins"
	}

	id := uuid.NewString()
	meta, _ := json.Marshal(map[string]any{"created_from": "api"})

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO orders (id, user_id, tariff_id, duration, amount, coins_used, tariff_amount, payment_type, meta)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)
	`, id, req.UserId, req.TariffId, duration, amount, coinsUsed, price, paymentType, string(meta))
	if err != nil {
		return nil, fmt.Errorf("error while creating order: %w", err)
	}

	if amount > 0 {
		// Pul talab qilinadi (to'liq yoki aralash) - gateway flow, tangalar hozircha
		// deduksiya qilinmaydi (ApplyComplete/PaymePerform muvaffaqiyatli PAID qilganda ayiriladi).
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit: %w", err)
		}
		return &m.OrderCreateRes{Id: id, Url: r.payURL(amount, id)}, nil
	}

	// To'liq tangaga yopilgan - gateway'siz darhol PAID.
	if coinsUsed > 0 {
		if _, err := tx.Exec(ctx, `UPDATE users SET coins = coins - $1, updated_at = now() WHERE id = $2`, coinsUsed, req.UserId); err != nil {
			return nil, fmt.Errorf("error while deducting coins: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE orders SET status = 'PAID', paid_at = now(), updated_at = now() WHERE id = $1
	`, id); err != nil {
		return nil, fmt.Errorf("error while marking order paid: %w", err)
	}
	if err := r.activateTariff(ctx, tx, req.UserId, req.TariffId, id, duration); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &m.OrderCreateRes{Id: id, Url: "", JustPaid: true}, nil
}

func (r *OrderRepo) GetById(req *m.ById) (*m.OrderStatisticRes, error) {
	query := `
	SELECT
		o.id, o.user_id, u.name, u.phone_number,
		o.tariff_id, t.name AS tariff_name, o.duration,
		o.amount, o.coins_used, o.tariff_amount,
		o.payment_type, o.status, o.created_at::text, o.paid_at::text
	FROM orders o
	JOIN users u ON u.id = o.user_id
	JOIN tariffs t ON t.id = o.tariff_id
	WHERE o.id = $1
	`

	row := r.db.QueryRow(context.Background(), query, req.Id)

	order := m.OrderStatisticRes{}
	err := row.Scan(
		&order.Id, &order.UserId, &order.UserName, &order.UserPhone,
		&order.TariffId, &order.TariffName, &order.Duration,
		&order.Amount, &order.CoinsUsed, &order.TariffAmount,
		&order.PaymentType, &order.Status, &order.CreatedAt, &order.PaidAt,
	)

	if err == pgx.ErrNoRows {
		return nil, errs.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting order by id: %w", err)
	}

	return &order, nil
}

func (r *OrderRepo) GetAll(req *m.OrderGetListReq) (*m.PaginationResponse, *m.OrderList, error) {
	query := `
	SELECT
		COUNT(o.id) OVER() AS total_count,
		o.id, o.user_id, u.name, u.phone_number,
		o.tariff_id, t.name AS tariff_name, o.duration,
		o.amount, o.coins_used, o.tariff_amount,
		o.payment_type, o.status, o.created_at::text, o.paid_at::text
	FROM orders o
	JOIN users u ON u.id = o.user_id
	JOIN tariffs t ON t.id = o.tariff_id
	WHERE 1=1
	`

	var args []interface{}

	if req.UserId != "" && req.UserId != "string" {
		args = append(args, req.UserId)
		query += fmt.Sprintf(" AND o.user_id = $%d", len(args))
	}
	if req.TariffId != "" && req.TariffId != "string" {
		args = append(args, req.TariffId)
		query += fmt.Sprintf(" AND o.tariff_id = $%d", len(args))
	}
	if req.Status != "" && req.Status != "string" {
		args = append(args, req.Status)
		query += fmt.Sprintf(" AND o.status = $%d", len(args))
	}
	if req.PaymentType != "" && req.PaymentType != "string" {
		args = append(args, req.PaymentType)
		query += fmt.Sprintf(" AND o.payment_type = $%d", len(args))
	}

	if req.Type != "" && req.Type != "string" {
		loc, err := time.LoadLocation("Asia/Tashkent")
		if err != nil {
			slog.Warn("tzdata missing? fallback to fixed zone", "err", err)
			loc = time.FixedZone("Asia/Tashkent", 5*60*60)
		}
		parseDate := func(s string) (time.Time, error) { return time.ParseInLocation("2006-01-02", s, loc) }
		startOfDay := func(t time.Time) time.Time {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		}
		startOfWeekMonday := func(t time.Time) time.Time {
			t = startOfDay(t)
			wd := int(t.Weekday())
			if wd == 0 {
				wd = 7
			}
			return t.AddDate(0, 0, -(wd - 1))
		}
		now := time.Now().In(loc)

		switch req.Type {
		case "day":
			if req.Day == "" || req.Day == "string" {
				return nil, nil, fmt.Errorf("day is required for type=day")
			}
			d, err := parseDate(req.Day)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid day format, expected YYYY-MM-DD")
			}
			from := startOfDay(d)
			to := from.AddDate(0, 0, 1)
			args = append(args, from)
			query += fmt.Sprintf(" AND o.created_at >= $%d", len(args))
			args = append(args, to)
			query += fmt.Sprintf(" AND o.created_at < $%d", len(args))
		case "range":
			if req.From == "" || req.From == "string" || req.To == "" || req.To == "string" {
				return nil, nil, fmt.Errorf("from and to are required for type=range")
			}
			f, err := parseDate(req.From)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid from format, expected YYYY-MM-DD")
			}
			t, err := parseDate(req.To)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid to format, expected YYYY-MM-DD")
			}
			from := startOfDay(f)
			to := startOfDay(t).AddDate(0, 0, 1)
			args = append(args, from)
			query += fmt.Sprintf(" AND o.created_at >= $%d", len(args))
			args = append(args, to)
			query += fmt.Sprintf(" AND o.created_at < $%d", len(args))
		case "week":
			from := startOfWeekMonday(now)
			to := from.AddDate(0, 0, 7)
			args = append(args, from)
			query += fmt.Sprintf(" AND o.created_at >= $%d", len(args))
			args = append(args, to)
			query += fmt.Sprintf(" AND o.created_at < $%d", len(args))
		case "month":
			from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
			to := from.AddDate(0, 1, 0)
			args = append(args, from)
			query += fmt.Sprintf(" AND o.created_at >= $%d", len(args))
			args = append(args, to)
			query += fmt.Sprintf(" AND o.created_at < $%d", len(args))
		case "year":
			from := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, loc)
			to := from.AddDate(1, 0, 0)
			args = append(args, from)
			query += fmt.Sprintf(" AND o.created_at >= $%d", len(args))
			args = append(args, to)
			query += fmt.Sprintf(" AND o.created_at < $%d", len(args))
		default:
			return nil, nil, fmt.Errorf("invalid type: %s", req.Type)
		}
	}

	query += " ORDER BY o.created_at DESC"
	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting orders: %w", err)
	}
	defer rows.Close()

	orders := m.OrderList{}

	for rows.Next() {
		order := &m.OrderStatisticRes{}
		var totalCount int

		err := rows.Scan(
			&totalCount,
			&order.Id, &order.UserId, &order.UserName, &order.UserPhone,
			&order.TariffId, &order.TariffName, &order.Duration,
			&order.Amount, &order.CoinsUsed, &order.TariffAmount,
			&order.PaymentType, &order.Status, &order.CreatedAt, &order.PaidAt,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("error while scanning order: %w", err)
		}

		orders.Orders = append(orders.Orders, order)
		orders.Count = totalCount
	}

	response := m.NewPaginationResponse(req.Filter, orders.Count, orders.Orders)
	return &response, &orders, nil
}
