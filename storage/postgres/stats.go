package postgres

import (
	"context"
	"fmt"
	m "tibsphereai/pkg/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StatsRepo - admin statistikasi (TZ_STATISTICS). Har so'rov $1/$2 = from/to (YYYY-MM-DD,
// Toshkent kuni, ikkalasi ham kiradi) oladi; vaqt chegaralari statsBounds orqali hisoblanadi.
type StatsRepo struct {
	db *pgxpool.Pool
}

func NewStatsRepo(db *pgxpool.Pool) *StatsRepo {
	return &StatsRepo{db: db}
}

const (
	// statsFrom/statsTo - $1/$2 sanalarning Toshkent yarim tunidagi vaqti ([from, to+1) oraliq).
	statsFrom = `(($1::date)::timestamp AT TIME ZONE 'Asia/Tashkent')`
	statsTo   = `((($2::date) + 1)::timestamp AT TIME ZONE 'Asia/Tashkent')`
	tkDay     = `AT TIME ZONE 'Asia/Tashkent')::date`

	// userActivitySQL - "faol" = sessiya boshlash, practice javobi yoki fleshkarta takrorlash (login emas).
	userActivitySQL = `
		SELECT user_id, started_at AS ts FROM practice_sessions
		UNION ALL SELECT user_id, started_at FROM simulation_sessions
		UNION ALL SELECT ps.user_id, pi.answered_at FROM practice_items pi
			JOIN practice_sessions ps ON ps.id = pi.session_id WHERE pi.answered_at IS NOT NULL
		UNION ALL SELECT user_id, reviewed_at FROM flashcard_reviews`

	// paidOrdersSQL - haqiqiy pul tushgan buyurtmalar (to'liq tangaga yopilganlari tushum emas).
	paidOrdersSQL = `SELECT * FROM orders WHERE status = 'PAID' AND amount > 0`

	activeSubSQL = `EXISTS (SELECT 1 FROM subscriptions s WHERE s.user_id = %s AND s.status = 'active' AND s.ended_at > now())`
)

func (r *StatsRepo) usdRate(ctx context.Context) float64 {
	var rate float64
	_ = r.db.QueryRow(ctx, `SELECT COALESCE(NULLIF(value, '')::numeric, 0) FROM app_settings WHERE key = 'usd_to_uzs_rate'`).Scan(&rate)
	if rate <= 0 {
		rate = 12800
	}
	return rate
}

func div(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func (r *StatsRepo) Finance(req *m.StatsPeriodReq) (*m.FinanceStatsRes, error) {
	ctx := context.Background()
	res := &m.FinanceStatsRes{Period: m.StatsPeriod{From: req.From, To: req.To}, UsdToUzsRate: r.usdRate(ctx)}

	err := r.db.QueryRow(ctx, `
		WITH o AS (`+paidOrdersSQL+` AND paid_at >= `+statsFrom+` AND paid_at < `+statsTo+`)
		SELECT
			(SELECT COALESCE(SUM(amount), 0)::float8 FROM o),
			(SELECT COUNT(*) FROM o),
			(SELECT COUNT(DISTINCT user_id) FROM o),
			(SELECT COALESCE(SUM(cost_usd), 0)::float8 FROM ai_usage_logs WHERE created_at >= `+statsFrom+` AND created_at < `+statsTo+`),
			(SELECT COUNT(DISTINCT user_id) FROM (`+userActivitySQL+`) a WHERE a.ts >= `+statsFrom+` AND a.ts < `+statsTo+`)
	`, req.From, req.To).Scan(&res.RevenueUZS, &res.PaidOrders, &res.PayingUsers, &res.AICostUSD, &res.ActiveUsers)
	if err != nil {
		return nil, fmt.Errorf("error while getting finance totals: %w", err)
	}
	res.AICostUZS = res.AICostUSD * res.UsdToUzsRate
	res.ProfitUZS = res.RevenueUZS - res.AICostUZS
	res.ARPU = div(res.RevenueUZS, float64(res.ActiveUsers))
	res.ARPPU = div(res.RevenueUZS, float64(res.PayingUsers))

	rows, err := r.db.Query(ctx, `
		SELECT d::date::text,
			COALESCE((SELECT SUM(amount) FROM orders o WHERE o.status = 'PAID' AND o.amount > 0 AND (o.paid_at `+tkDay+` = d::date), 0)::float8,
			COALESCE((SELECT SUM(cost_usd) FROM ai_usage_logs a WHERE (a.created_at `+tkDay+` = d::date), 0)::float8
		FROM generate_series($1::date, $2::date, interval '1 day') d
		ORDER BY d
	`, req.From, req.To)
	if err != nil {
		return nil, fmt.Errorf("error while getting finance daily: %w", err)
	}
	defer rows.Close()
	res.Daily = []m.FinanceDay{}
	for rows.Next() {
		var d m.FinanceDay
		if err := rows.Scan(&d.Date, &d.RevenueUZS, &d.AICostUSD); err != nil {
			return nil, err
		}
		d.AICostUZS = d.AICostUSD * res.UsdToUzsRate
		res.Daily = append(res.Daily, d)
	}
	rows.Close()

	rows, err = r.db.Query(ctx, `
		SELECT t.id::text, t.name, t.kind, COUNT(o.id), COALESCE(SUM(o.amount), 0)::float8
		FROM (`+paidOrdersSQL+` AND paid_at >= `+statsFrom+` AND paid_at < `+statsTo+`) o
		JOIN tariffs t ON t.id = o.tariff_id
		GROUP BY t.id, t.name, t.kind
		ORDER BY 5 DESC
	`, req.From, req.To)
	if err != nil {
		return nil, fmt.Errorf("error while getting finance by tariff: %w", err)
	}
	defer rows.Close()
	res.ByTariff = []m.FinanceTariff{}
	for rows.Next() {
		var t m.FinanceTariff
		if err := rows.Scan(&t.TariffId, &t.Name, &t.Kind, &t.Sold, &t.RevenueUZS); err != nil {
			return nil, err
		}
		res.ByTariff = append(res.ByTariff, t)
	}
	return res, rows.Err()
}

func (r *StatsRepo) AICost(req *m.StatsPeriodReq) (*m.AICostStatsRes, error) {
	ctx := context.Background()
	res := &m.AICostStatsRes{Period: m.StatsPeriod{From: req.From, To: req.To}}
	inRange := ` created_at >= ` + statsFrom + ` AND created_at < ` + statsTo

	var freeCoinCost float64
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(cost_usd), 0)::float8, COALESCE(SUM(tokens_used), 0)::bigint, COUNT(*),
			COALESCE(SUM(cost_usd) FILTER (WHERE user_tier IN ('free', 'coin')), 0)::float8,
			(SELECT COALESCE(SUM(-delta), 0) FROM point_transactions WHERE kind = 'coin' AND delta < 0 AND `+inRange+`)
		FROM ai_usage_logs WHERE`+inRange, req.From, req.To).
		Scan(&res.TotalCostUSD, &res.TotalTokens, &res.Calls, &freeCoinCost, &res.CoinsSpent)
	if err != nil {
		return nil, fmt.Errorf("error while getting ai cost totals: %w", err)
	}
	res.CostPerCoinUSD = div(freeCoinCost, float64(res.CoinsSpent))

	rows, err := r.db.Query(ctx, `
		SELECT feature, COUNT(*), COALESCE(SUM(tokens_used), 0)::bigint, COALESCE(SUM(cost_usd), 0)::float8
		FROM ai_usage_logs WHERE`+inRange+` GROUP BY feature ORDER BY 4 DESC`, req.From, req.To)
	if err != nil {
		return nil, fmt.Errorf("error while getting ai cost by feature: %w", err)
	}
	defer rows.Close()
	res.ByFeature = []m.AICostFeature{}
	for rows.Next() {
		var f m.AICostFeature
		if err := rows.Scan(&f.Feature, &f.Calls, &f.Tokens, &f.CostUSD); err != nil {
			return nil, err
		}
		res.ByFeature = append(res.ByFeature, f)
	}
	rows.Close()

	// Bitta amal: sessiyalar - sessiya ichidagi barcha AI chaqiruvlari (session_id bo'yicha) va
	// coins_charged; fleshkarta to'plami/yo'l xaritasi - bitta log = bitta amal, tanga Charge yozuvidan.
	rows, err = r.db.Query(ctx, `
		WITH ps AS (
			SELECT p.mode, p.coins_charged,
				(SELECT COALESCE(SUM(cost_usd), 0) FROM ai_usage_logs a WHERE a.session_id = p.id) AS cost
			FROM practice_sessions p WHERE p.started_at >= `+statsFrom+` AND p.started_at < `+statsTo+`
		), ss AS (
			SELECT s.coins_charged,
				(SELECT COALESCE(SUM(cost_usd), 0) FROM ai_usage_logs a WHERE a.session_id = s.id) AS cost
			FROM simulation_sessions s WHERE s.started_at >= `+statsFrom+` AND s.started_at < `+statsTo+`
		), single AS (
			SELECT feature, COUNT(*) AS n, COALESCE(SUM(cost_usd), 0) AS cost FROM ai_usage_logs
			WHERE feature IN ('flashcard_deck', 'roadmap') AND`+inRange+` GROUP BY feature
		), single_coins AS (
			SELECT reason, COALESCE(SUM(-delta), 0) AS coins FROM point_transactions
			WHERE kind = 'coin' AND delta < 0 AND reason IN ('flashcard_deck', 'roadmap') AND`+inRange+` GROUP BY reason
		)
		SELECT 'case_session', COUNT(*), COALESCE(AVG(cost), 0)::float8, COALESCE(AVG(coins_charged), 0)::float8 FROM ps WHERE mode = 'case'
		UNION ALL SELECT 'test_session', COUNT(*), COALESCE(AVG(cost), 0)::float8, COALESCE(AVG(coins_charged), 0)::float8 FROM ps WHERE mode = 'test'
		UNION ALL SELECT 'simulation', COUNT(*), COALESCE(AVG(cost), 0)::float8, COALESCE(AVG(coins_charged), 0)::float8 FROM ss
		UNION ALL SELECT f.name, COALESCE(s.n, 0), COALESCE(s.cost / NULLIF(s.n, 0), 0)::float8, COALESCE(c.coins::float8 / NULLIF(s.n, 0), 0)::float8
			FROM (VALUES ('flashcard_deck'), ('roadmap')) f(name)
			LEFT JOIN single s ON s.feature = f.name LEFT JOIN single_coins c ON c.reason = f.name
	`, req.From, req.To)
	if err != nil {
		return nil, fmt.Errorf("error while getting ai cost per action: %w", err)
	}
	defer rows.Close()
	res.PerAction = []m.AICostAction{}
	for rows.Next() {
		var a m.AICostAction
		if err := rows.Scan(&a.Action, &a.Count, &a.AvgCostUSD, &a.AvgCoins); err != nil {
			return nil, err
		}
		res.PerAction = append(res.PerAction, a)
	}
	rows.Close()

	rows, err = r.db.Query(ctx, `
		SELECT CASE WHEN user_id IS NULL THEN 'system' WHEN user_tier = '' THEN 'unknown' ELSE user_tier END AS tier,
			COUNT(*), COUNT(DISTINCT user_id), COALESCE(SUM(cost_usd), 0)::float8
		FROM ai_usage_logs WHERE`+inRange+` GROUP BY 1 ORDER BY 4 DESC`, req.From, req.To)
	if err != nil {
		return nil, fmt.Errorf("error while getting ai cost by tier: %w", err)
	}
	defer rows.Close()
	res.ByTier = []m.AICostTier{}
	for rows.Next() {
		var t m.AICostTier
		if err := rows.Scan(&t.Tier, &t.Calls, &t.Users, &t.CostUSD); err != nil {
			return nil, err
		}
		res.ByTier = append(res.ByTier, t)
	}
	return res, rows.Err()
}

func (r *StatsRepo) PremiumCost(req *m.StatsPeriodReq) (*m.PremiumCostStatsRes, error) {
	ctx := context.Background()
	rate := r.usdRate(ctx)
	res := &m.PremiumCostStatsRes{Period: m.StatsPeriod{From: req.From, To: req.To}}

	err := r.db.QueryRow(ctx, `
		WITH prem AS (
			SELECT DISTINCT user_id FROM subscriptions
			WHERE status IN ('active', 'expired') AND started_at < `+statsTo+` AND ended_at > `+statsFrom+`
		), cost AS (
			SELECT p.user_id,
				(SELECT COALESCE(SUM(cost_usd), 0) FROM ai_usage_logs a
					WHERE a.user_id = p.user_id AND a.created_at >= `+statsFrom+` AND a.created_at < `+statsTo+`) AS period_cost,
				(SELECT COALESCE(SUM(cost_usd), 0) FROM ai_usage_logs a WHERE a.user_id = p.user_id) AS life_cost,
				(SELECT COALESCE(SUM(amount), 0) FROM orders o WHERE o.user_id = p.user_id AND o.status = 'PAID' AND o.amount > 0) AS life_paid
			FROM prem p
		)
		SELECT COUNT(*), COALESCE(AVG(period_cost), 0)::float8, COUNT(*) FILTER (WHERE life_cost * $3 > life_paid)
		FROM cost
	`, req.From, req.To, rate).Scan(&res.PremiumUsers, &res.AvgAICostUSD, &res.LossMakingUsers)
	if err != nil {
		return nil, fmt.Errorf("error while getting premium cost: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		WITH top AS (
			SELECT user_id, SUM(cost_usd) AS cost, SUM(tokens_used) AS tokens
			FROM ai_usage_logs
			WHERE user_id IS NOT NULL AND created_at >= `+statsFrom+` AND created_at < `+statsTo+`
			GROUP BY user_id ORDER BY cost DESC LIMIT 20
		)
		SELECT t.user_id::text, COALESCE(u.name, ''),
			CASE WHEN `+fmt.Sprintf(activeSubSQL, "t.user_id")+` THEN 'premium'
				WHEN EXISTS (SELECT 1 FROM orders o WHERE o.user_id = t.user_id AND o.status = 'PAID' AND o.amount > 0) THEN 'paying'
				ELSE 'free' END,
			t.cost::float8, t.tokens::bigint,
			(SELECT COUNT(*) FROM practice_sessions p WHERE p.user_id = t.user_id AND p.started_at >= `+statsFrom+` AND p.started_at < `+statsTo+`)
			+ (SELECT COUNT(*) FROM simulation_sessions s WHERE s.user_id = t.user_id AND s.started_at >= `+statsFrom+` AND s.started_at < `+statsTo+`),
			(SELECT COALESCE(SUM(amount), 0) FROM orders o WHERE o.user_id = t.user_id AND o.status = 'PAID' AND o.amount > 0)::float8
		FROM top t JOIN users u ON u.id = t.user_id
		ORDER BY t.cost DESC
	`, req.From, req.To)
	if err != nil {
		return nil, fmt.Errorf("error while getting top spenders: %w", err)
	}
	defer rows.Close()
	res.TopSpenders = []m.TopSpender{}
	for rows.Next() {
		var s m.TopSpender
		if err := rows.Scan(&s.UserId, &s.Name, &s.Type, &s.CostUSD, &s.Tokens, &s.Sessions, &s.PaidUZS); err != nil {
			return nil, err
		}
		res.TopSpenders = append(res.TopSpenders, s)
	}
	return res, rows.Err()
}

func (r *StatsRepo) Engagement(req *m.StatsPeriodReq) (*m.EngagementStatsRes, error) {
	ctx := context.Background()
	res := &m.EngagementStatsRes{Period: m.StatsPeriod{From: req.From, To: req.To}}

	err := r.db.QueryRow(ctx, `
		WITH a AS (`+userActivitySQL+`), today AS (SELECT (now() `+tkDay+` AS d)
		SELECT
			(SELECT COUNT(DISTINCT user_id) FROM a, today WHERE (a.ts `+tkDay+` = today.d),
			(SELECT COUNT(DISTINCT user_id) FROM a, today WHERE (a.ts `+tkDay+` > today.d - 7),
			(SELECT COUNT(DISTINCT user_id) FROM a, today WHERE (a.ts `+tkDay+` > today.d - 30),
			(SELECT COUNT(*) FROM users WHERE deleted_at = 0 AND created_at >= `+statsFrom+` AND created_at < `+statsTo+`)
	`, req.From, req.To).Scan(&res.DAU, &res.WAU, &res.MAU, &res.NewUsers)
	if err != nil {
		return nil, fmt.Errorf("error while getting engagement totals: %w", err)
	}
	res.DAUMAU = div(float64(res.DAU), float64(res.MAU))

	rows, err := r.db.Query(ctx, `
		WITH a AS (
			SELECT user_id, (ts `+tkDay+` AS d FROM (`+userActivitySQL+`) x
			WHERE ts >= `+statsFrom+` AND ts < `+statsTo+`
		), s AS (
			SELECT (started_at `+tkDay+` AS d FROM practice_sessions WHERE started_at >= `+statsFrom+` AND started_at < `+statsTo+`
			UNION ALL SELECT (started_at `+tkDay+` FROM simulation_sessions WHERE started_at >= `+statsFrom+` AND started_at < `+statsTo+`
		)
		SELECT g::date::text,
			(SELECT COUNT(DISTINCT user_id) FROM a WHERE a.d = g::date),
			(SELECT COUNT(*) FROM users u WHERE u.deleted_at = 0 AND (u.created_at `+tkDay+` = g::date),
			(SELECT COUNT(*) FROM s WHERE s.d = g::date),
			(SELECT COUNT(*) FROM flashcard_reviews f WHERE (f.reviewed_at `+tkDay+` = g::date)
		FROM generate_series($1::date, $2::date, interval '1 day') g
		ORDER BY g
	`, req.From, req.To)
	if err != nil {
		return nil, fmt.Errorf("error while getting engagement daily: %w", err)
	}
	defer rows.Close()
	res.Daily = []m.EngagementDay{}
	for rows.Next() {
		var d m.EngagementDay
		if err := rows.Scan(&d.Date, &d.DAU, &d.NewUsers, &d.Sessions, &d.FlashcardReviews); err != nil {
			return nil, err
		}
		res.Daily = append(res.Daily, d)
	}
	rows.Close()

	err = r.db.QueryRow(ctx, `
		WITH c AS (
			SELECT id, (created_at `+tkDay+` AS d FROM users
			WHERE deleted_at = 0 AND created_at >= `+statsFrom+` AND created_at < `+statsTo+`
		), a AS (`+userActivitySQL+`)
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM practice_sessions p WHERE p.user_id = c.id)
				OR EXISTS (SELECT 1 FROM simulation_sessions s WHERE s.user_id = c.id)),
			COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM practice_sessions p WHERE p.user_id = c.id AND p.status = 'finished')
				OR EXISTS (SELECT 1 FROM simulation_sessions s WHERE s.user_id = c.id AND s.status IN ('completed', 'failed'))),
			COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM a WHERE a.user_id = c.id AND (a.ts `+tkDay+` = c.d + 1)),
			COUNT(*) FILTER (WHERE EXISTS (SELECT 1 FROM orders o WHERE o.user_id = c.id AND o.status = 'PAID' AND o.amount > 0))
		FROM c
	`, req.From, req.To).Scan(&res.Funnel.Registered, &res.Funnel.StartedSession, &res.Funnel.FinishedSession,
		&res.Funnel.ReturnedNextDay, &res.Funnel.Paid)
	if err != nil {
		return nil, fmt.Errorf("error while getting engagement funnel: %w", err)
	}
	return res, nil
}

func (r *StatsRepo) UserStats(userId string) (*m.UserStatsRes, error) {
	ctx := context.Background()
	res := &m.UserStatsRes{UserId: userId}

	rows, err := r.db.Query(ctx, `
		WITH today AS (SELECT (now() `+tkDay+` AS d)
		SELECT g::date::text,
			(SELECT COUNT(*) FROM practice_sessions p WHERE p.user_id = $1 AND (p.started_at `+tkDay+` = g::date)
			+ (SELECT COUNT(*) FROM simulation_sessions s WHERE s.user_id = $1 AND (s.started_at `+tkDay+` = g::date),
			(SELECT COALESCE(SUM(tokens_used), 0) FROM ai_usage_logs a WHERE a.user_id = $1 AND (a.created_at `+tkDay+` = g::date)::bigint,
			(SELECT COALESCE(SUM(-delta), 0) FROM point_transactions t
				WHERE t.user_id = $1 AND t.kind = 'coin' AND t.delta < 0 AND (t.created_at `+tkDay+` = g::date)::int
		FROM today, generate_series(today.d - 29, today.d, interval '1 day') g
		ORDER BY g
	`, userId)
	if err != nil {
		return nil, fmt.Errorf("error while getting user daily stats: %w", err)
	}
	defer rows.Close()
	res.Daily = []m.UserStatsDay{}
	for rows.Next() {
		var d m.UserStatsDay
		if err := rows.Scan(&d.Date, &d.Sessions, &d.Tokens, &d.CoinsSpent); err != nil {
			return nil, err
		}
		res.Daily = append(res.Daily, d)
	}
	rows.Close()

	rows, err = r.db.Query(ctx, `
		SELECT c.id::text, COALESCE(c.name->>'uz', ''), COUNT(*)
		FROM (
			SELECT category_id FROM practice_sessions WHERE user_id = $1 AND category_id IS NOT NULL
			UNION ALL SELECT t.category_id FROM simulation_sessions s
				JOIN cases cs ON cs.id = s.case_id JOIN topics t ON t.id = cs.topic_id
				WHERE s.user_id = $1 AND t.category_id IS NOT NULL
		) x JOIN categories c ON c.id = x.category_id
		GROUP BY c.id, c.name ORDER BY 3 DESC LIMIT 5
	`, userId)
	if err != nil {
		return nil, fmt.Errorf("error while getting user top categories: %w", err)
	}
	defer rows.Close()
	res.TopCategories = []m.UserStatsCategory{}
	for rows.Next() {
		var c m.UserStatsCategory
		if err := rows.Scan(&c.CategoryId, &c.Name, &c.Sessions); err != nil {
			return nil, err
		}
		res.TopCategories = append(res.TopCategories, c)
	}
	rows.Close()

	rows, err = r.db.Query(ctx, `
		SELECT delta, COALESCE(reason, ''), source, created_at::text FROM point_transactions
		WHERE user_id = $1 AND kind = 'coin' AND delta <> 0
		ORDER BY created_at DESC LIMIT 10
	`, userId)
	if err != nil {
		return nil, fmt.Errorf("error while getting user transactions: %w", err)
	}
	defer rows.Close()
	res.Transactions = []m.UserStatsCoinTx{}
	for rows.Next() {
		var t m.UserStatsCoinTx
		if err := rows.Scan(&t.Delta, &t.Reason, &t.Source, &t.CreatedAt); err != nil {
			return nil, err
		}
		res.Transactions = append(res.Transactions, t)
	}
	rows.Close()

	rows, err = r.db.Query(ctx, `
		SELECT o.id::text, COALESCE(t.name, ''), o.amount::float8, o.status, o.payment_type, o.created_at::text, o.paid_at::text
		FROM orders o LEFT JOIN tariffs t ON t.id = o.tariff_id
		WHERE o.user_id = $1 ORDER BY o.created_at DESC LIMIT 10
	`, userId)
	if err != nil {
		return nil, fmt.Errorf("error while getting user orders: %w", err)
	}
	defer rows.Close()
	res.Orders = []m.UserStatsOrder{}
	for rows.Next() {
		var o m.UserStatsOrder
		if err := rows.Scan(&o.Id, &o.TariffName, &o.Amount, &o.Status, &o.PaymentType, &o.CreatedAt, &o.PaidAt); err != nil {
			return nil, err
		}
		res.Orders = append(res.Orders, o)
	}
	return res, rows.Err()
}
