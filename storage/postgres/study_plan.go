package postgres

import (
	"context"
	"fmt"
	m "tibsphereai/pkg/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StudyPlanRepo - O'quv rejasi va push bildirishnoma vaqti (BACKEND_STRUCTURE.md 2-bo'lim).
type StudyPlanRepo struct {
	db *pgxpool.Pool
}

func NewStudyPlanRepo(db *pgxpool.Pool) *StudyPlanRepo {
	return &StudyPlanRepo{db: db}
}

func (r *StudyPlanRepo) Upsert(req *m.StudyPlanUpsertReq) (*m.Void, error) {
	_, err := r.db.Exec(context.Background(), `
		INSERT INTO study_plans (user_id, remind_time, remind_minutes_before, is_enabled)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			remind_time = EXCLUDED.remind_time,
			remind_minutes_before = EXCLUDED.remind_minutes_before,
			is_enabled = EXCLUDED.is_enabled
	`, req.UserId, req.RemindTime, req.RemindMinutesBefore, req.IsEnabled)
	if err != nil {
		return nil, fmt.Errorf("error while upserting study plan: %w", err)
	}
	return &m.Void{}, nil
}

func (r *StudyPlanRepo) GetByUserId(userId string) (*m.StudyPlanRes, error) {
	row := r.db.QueryRow(context.Background(), `
		SELECT user_id, remind_time::text, remind_minutes_before, is_enabled
		FROM study_plans WHERE user_id = $1
	`, userId)

	s := m.StudyPlanRes{}
	err := row.Scan(&s.UserId, &s.RemindTime, &s.RemindMinutesBefore, &s.IsEnabled)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting study plan: %w", err)
	}
	return &s, nil
}

// GetAllEnabled - push scheduler (TODO: cron) uchun barcha faol reja egalari.
func (r *StudyPlanRepo) GetAllEnabled() ([]m.StudyPlanRes, error) {
	rows, err := r.db.Query(context.Background(), `
		SELECT user_id, remind_time::text, remind_minutes_before, is_enabled
		FROM study_plans WHERE is_enabled = TRUE
	`)
	if err != nil {
		return nil, fmt.Errorf("error while getting enabled study plans: %w", err)
	}
	defer rows.Close()

	var list []m.StudyPlanRes
	for rows.Next() {
		s := m.StudyPlanRes{}
		if err := rows.Scan(&s.UserId, &s.RemindTime, &s.RemindMinutesBefore, &s.IsEnabled); err != nil {
			return nil, fmt.Errorf("error while scanning study plan: %w", err)
		}
		list = append(list, s)
	}
	return list, nil
}
