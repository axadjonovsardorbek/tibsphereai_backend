package postgres

import (
	"context"
	"fmt"
	m "tibsphereai/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FavoriteRepo struct {
	db *pgxpool.Pool
}

func NewFavoriteRepo(db *pgxpool.Pool) *FavoriteRepo {
	return &FavoriteRepo{db: db}
}

func (r *FavoriteRepo) Toggle(req *m.FavoriteToggleReq) (*m.FavoriteToggleRes, error) {
	ctx := context.Background()

	res, err := r.db.Exec(ctx, `DELETE FROM favorites WHERE user_id = $1 AND case_id = $2`, req.UserId, req.CaseId)
	if err != nil {
		return nil, fmt.Errorf("error while toggling favorite: %w", err)
	}
	if res.RowsAffected() > 0 {
		return &m.FavoriteToggleRes{IsFavorite: false}, nil
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO favorites (id, user_id, case_id) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, case_id) DO NOTHING
	`, uuid.NewString(), req.UserId, req.CaseId)
	if err != nil {
		return nil, fmt.Errorf("error while adding favorite: %w", err)
	}

	return &m.FavoriteToggleRes{IsFavorite: true}, nil
}

func (r *FavoriteRepo) GetAll(req *m.FavoriteGetListReq) (*m.PaginationResponse, *m.CaseMobileList, error) {
	query := `
	SELECT
		COUNT(c.id) OVER() AS total_count,
		c.id, c.topic_id, c.title ->> $1, c.cover_image_url, c.difficulty,
		c.chief_complaint ->> $1, c.patient_age, c.patient_gender, c.expected_duration_minutes, c.max_attempts,
		TRUE AS is_favorite
	FROM favorites f
	JOIN cases c ON c.id = f.case_id AND c.deleted_at = 0
	WHERE f.user_id = $2
	ORDER BY f.created_at DESC
	`

	var args []interface{}
	args = append(args, req.Language, req.UserId)
	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting favorites: %w", err)
	}
	defer rows.Close()

	list := m.CaseMobileList{}
	for rows.Next() {
		c := &m.CaseMobileRes{}
		var total int
		if err := rows.Scan(
			&total, &c.Id, &c.TopicId, &c.Title, &c.CoverImageUrl, &c.Difficulty,
			&c.ChiefComplaint, &c.PatientAge, &c.PatientGender, &c.ExpectedDurationMinutes, &c.MaxAttempts,
			&c.IsFavorite,
		); err != nil {
			return nil, nil, fmt.Errorf("error while scanning favorite case: %w", err)
		}
		list.Cases = append(list.Cases, c)
		list.Count = total
	}

	response := m.NewPaginationResponse(req.Filter, list.Count, list.Cases)
	return &response, &list, nil
}
