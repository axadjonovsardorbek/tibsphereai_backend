package postgres

import (
	"context"
	"fmt"
	m "tibsphereai/pkg/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardRepo struct {
	db *pgxpool.Pool
}

func NewDashboardRepo(db *pgxpool.Pool) *DashboardRepo {
	return &DashboardRepo{
		db: db,
	}
}

func (r *DashboardRepo) GetDashboard(req *m.GetDashboardReq) (*m.DashboardRes, error) {
	query := `
	WITH bounds AS (
	  SELECT
		CASE
		  WHEN COALESCE($1,'') = '' THEN NULL
		  WHEN $1 = 'week'  THEN date_trunc('week',  now())
		  WHEN $1 = 'month' THEN date_trunc('month', now())
		  WHEN $1 = 'year'  THEN date_trunc('year',  now())
		  WHEN $1 = 'day' THEN date_trunc('day', COALESCE(NULLIF($2,'')::timestamptz, now()))
		  WHEN $1 = 'range' THEN COALESCE(NULLIF($3,'')::timestamptz, NULL)
		  ELSE NULL
		END AS from_ts,
		CASE
		  WHEN COALESCE($1,'') = '' THEN NULL
		  WHEN $1 IN ('week','month','year') THEN now()
		  WHEN $1 = 'day' THEN date_trunc('day', COALESCE(NULLIF($2,'')::timestamptz, now())) + interval '1 day'
		  WHEN $1 = 'range' THEN (COALESCE(NULLIF($4,'')::timestamptz, NULL) + interval '1 day')
		  ELSE NULL
		END AS to_ts
	)
	SELECT
	  (SELECT COUNT(*) FROM categories c, bounds b
		WHERE c.deleted_at=0
		  AND (b.from_ts IS NULL OR c.created_at >= b.from_ts)
		  AND (b.to_ts   IS NULL OR c.created_at <  b.to_ts)
	  ) AS categories,

	  (SELECT COUNT(*) FROM cases c, bounds b
		WHERE c.deleted_at=0
		  AND (b.from_ts IS NULL OR c.created_at >= b.from_ts)
		  AND (b.to_ts   IS NULL OR c.created_at <  b.to_ts)
	  ) AS cases,

	  (SELECT COUNT(*) FROM users u, bounds b
		WHERE u.deleted_at=0
		  AND (b.from_ts IS NULL OR u.created_at >= b.from_ts)
		  AND (b.to_ts   IS NULL OR u.created_at <  b.to_ts)
	  ) AS users,

	  (SELECT COUNT(DISTINCT ua.user_id)
		FROM user_activities ua
		JOIN users u ON u.id = ua.user_id
		CROSS JOIN bounds b
		WHERE u.deleted_at = 0
		  AND ua.activity <> 0
		  AND (b.from_ts IS NULL OR ua.day >= b.from_ts::date)
		  AND (b.to_ts   IS NULL OR ua.day <  b.to_ts::date)
	  ) AS active_users,

	  (SELECT COUNT(*) FROM subscriptions s
		WHERE s.status = 'active' AND s.ended_at > now()
	  ) AS active_subscriptions,

	  (SELECT COUNT(*) FROM simulation_sessions s, bounds b
		WHERE s.status = 'completed'
		  AND (b.from_ts IS NULL OR s.ended_at >= b.from_ts)
		  AND (b.to_ts   IS NULL OR s.ended_at <  b.to_ts)
	  ) AS completed_sessions,

	  (SELECT COALESCE(SUM(a.cost_usd), 0) FROM ai_usage_logs a, bounds b
		WHERE (b.from_ts IS NULL OR a.created_at >= b.from_ts)
		  AND (b.to_ts   IS NULL OR a.created_at <  b.to_ts)
	  ) AS ai_total_cost_usd
	`

	row := r.db.QueryRow(context.Background(), query, req.Type, req.Day, req.From, req.To)

	var dashboard m.DashboardRes
	err := row.Scan(
		&dashboard.Categories,
		&dashboard.Cases,
		&dashboard.Users,
		&dashboard.ActiveUsers,
		&dashboard.ActiveSubscriptions,
		&dashboard.CompletedSessions,
		&dashboard.AITotalCostUSD,
	)
	if err != nil {
		return nil, fmt.Errorf("error while getting dashboard: %w", err)
	}

	return &dashboard, nil
}
