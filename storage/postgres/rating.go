package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	m "tibsphereai/pkg/models"
)

// ratingScoresCTE - reyting uchun "scores" CTE: har foydalanuvchining davr ichidagi case soni (activity)
// va yulduzlari (stars). Davr "total" bo'lsa yulduz users.stars'dan (promokod/level mukofotlari bilan),
// aks holda point_transactions (kind='stars') yozuvlari yig'indisidan olinadi (Toshkent kuni bo'yicha).
// $1, $2 - davr boshi/oxiri (sana).
func ratingScoresCTE(rt m.RatingType) string {
	starsExpr := `COALESCE((
			SELECT SUM(pt.delta) FROM point_transactions pt
			WHERE pt.user_id = u.id AND pt.kind = 'stars'
			  AND (pt.created_at AT TIME ZONE 'Asia/Tashkent')::date BETWEEN $1::date AND $2::date
		), 0)`
	if rt == m.RatingTotal || rt == "" {
		starsExpr = `COALESCE(u.stars, 0)`
	}
	return `
	WITH scores AS (
		SELECT
		  u.id AS user_id,
		  u.name AS name,
		  u.image_url AS avatar,
		  COALESCE((
			SELECT SUM(ua.activity) FROM user_activities ua
			WHERE ua.user_id = u.id AND ua.day >= $1::date AND ua.day <= $2::date
		  ), 0)::bigint AS activity,
		  ` + starsExpr + `::bigint AS stars
		FROM users u
	)`
}

// ratingOrder - tartiblash (whitelist, SQL'ga faqat shu qiymatlar tushadi). Default - case soni.
func ratingOrder(sort string) string {
	if sort == m.RatingSortStars {
		return "stars DESC, activity DESC, name ASC, user_id ASC"
	}
	return "activity DESC, stars DESC, name ASC, user_id ASC"
}

func ratingRange(req *m.RatingRequest) (m.RatingType, time.Time, time.Time) {
	loc := time.FixedZone("UTC+5", 5*3600)
	rt := m.RatingType(req.Type)
	start, end := calcRange(rt, loc)
	return rt, start, end
}

func (r *UserRepo) GetRating(req *m.RatingRequest) (*m.RatingResponse, error) {
	ctx := context.Background()

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 10
	}
	if req.Type == "" {
		req.Type = "total"
	}
	rt, start, end := ratingRange(req)

	res := &m.RatingResponse{
		Type:  rt,
		Sort:  m.NormalizeRatingSort(req.Sort),
		Limit: req.Limit,
		Items: make([]m.RatingUserItem, 0, req.Limit),
	}

	query := ratingScoresCTE(rt) + `,
	ranked AS (
		SELECT user_id, name, avatar, activity, stars,
		  ROW_NUMBER() OVER (ORDER BY ` + ratingOrder(res.Sort) + `) AS rank
		FROM scores
	),
	topn AS (
		SELECT user_id, name, avatar, activity, stars, rank FROM ranked ORDER BY rank LIMIT $3
	),
	me AS (
		SELECT user_id, name, avatar, activity, stars, rank FROM ranked WHERE user_id = $4::uuid
	)
	SELECT 'top' AS kind, user_id, name, avatar, activity, stars, rank FROM topn
	UNION ALL
	SELECT 'me'  AS kind, user_id, name, avatar, activity, stars, rank FROM me;
	`

	rows, err := r.db.Query(ctx, query, start, end, req.Limit, req.UserId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var kind string
		var it m.RatingUserItem
		if err := rows.Scan(&kind, &it.UserID, &it.Name, &it.ImageUrl, &it.Activity, &it.Stars, &it.Rank); err != nil {
			return nil, err
		}
		if kind == "me" {
			it.IsMe = true
			res.Me = &it
		} else {
			res.Items = append(res.Items, it)
		}
	}
	return res, rows.Err()
}

func (r *UserRepo) GetTop(req *m.RatingRequest) ([]m.RatingUserItem, error) {
	ctx := context.Background()
	if req.Type == "" {
		req.Type = "total"
	}
	rt, start, end := ratingRange(req)

	query := ratingScoresCTE(rt) + `,
	ranked AS (
		SELECT user_id, name, avatar, activity, stars,
		  ROW_NUMBER() OVER (ORDER BY ` + ratingOrder(m.NormalizeRatingSort(req.Sort)) + `) AS rank
		FROM scores
	)
	SELECT user_id, name, avatar, activity, stars, rank, (user_id = $3) AS is_me
	FROM ranked ORDER BY rank LIMIT $4
	`

	rows, err := r.db.Query(ctx, query, start, end, req.UserId, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("top rating query: %w", err)
	}
	defer rows.Close()

	items := make([]m.RatingUserItem, 0, req.Limit)
	for rows.Next() {
		var it m.RatingUserItem
		if err := rows.Scan(&it.UserID, &it.Name, &it.ImageUrl, &it.Activity, &it.Stars, &it.Rank, &it.IsMe); err != nil {
			return nil, fmt.Errorf("top rating scan: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("top rating rows: %w", err)
	}
	slog.Debug("GetTop rating", "type", req.Type, "sort", req.Sort, "limit", req.Limit, "items_count", len(items))
	return items, nil
}

func (r *UserRepo) GetMe(req *m.RatingRequest) (*m.RatingUserItem, error) {
	ctx := context.Background()
	if req.Type == "" {
		req.Type = "total"
	}
	rt, start, end := ratingRange(req)

	query := ratingScoresCTE(rt) + `,
	ranked AS (
		SELECT user_id, name, avatar, activity, stars,
		  ROW_NUMBER() OVER (ORDER BY ` + ratingOrder(m.NormalizeRatingSort(req.Sort)) + `) AS rank
		FROM scores
	)
	SELECT user_id, name, avatar, activity, stars, rank FROM ranked WHERE user_id = $3::uuid
	`

	var it m.RatingUserItem
	err := r.db.QueryRow(ctx, query, start, end, req.UserId).
		Scan(&it.UserID, &it.Name, &it.ImageUrl, &it.Activity, &it.Stars, &it.Rank)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("me rating query: %w", err)
	}
	it.IsMe = true
	return &it, nil
}
