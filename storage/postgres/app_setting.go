package postgres

import (
	"context"
	"fmt"
	m "tibsphereai/pkg/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AppSettingRepo struct {
	db *pgxpool.Pool
}

func NewAppSettingRepo(db *pgxpool.Pool) *AppSettingRepo {
	return &AppSettingRepo{db: db}
}

func (r *AppSettingRepo) Get(key string) (*m.AppSettingRes, error) {
	row := r.db.QueryRow(context.Background(), `
		SELECT key, value, updated_at::text FROM app_settings WHERE key = $1
	`, key)

	s := m.AppSettingRes{}
	err := row.Scan(&s.Key, &s.Value, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting app setting: %w", err)
	}
	return &s, nil
}

func (r *AppSettingRepo) Set(req *m.AppSettingSetReq) (*m.Void, error) {
	_, err := r.db.Exec(context.Background(), `
		INSERT INTO app_settings (key, value, updated_at) VALUES ($1, $2, now())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`, req.Key, req.Value)
	if err != nil {
		return nil, fmt.Errorf("error while setting app setting: %w", err)
	}
	return &m.Void{}, nil
}

func (r *AppSettingRepo) GetAll() ([]m.AppSettingRes, error) {
	rows, err := r.db.Query(context.Background(), `SELECT key, value, updated_at::text FROM app_settings ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("error while getting app settings: %w", err)
	}
	defer rows.Close()

	var list []m.AppSettingRes
	for rows.Next() {
		s := m.AppSettingRes{}
		if err := rows.Scan(&s.Key, &s.Value, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error while scanning app setting: %w", err)
		}
		list = append(list, s)
	}
	return list, nil
}

// GetTodayFreeAttemptsUsed / IncrTodayFreeAttempts - kunlik bepul limit
// (BACKEND_STRUCTURE.md 6-bo'lim): har kecha soat 00:00 da (Asia/Tashkent) avtomatik yangilanadi,
// chunki hisob "day" ustuniga bog'langan.
func (r *AppSettingRepo) GetTodayFreeAttemptsUsed(userId string) (int, error) {
	var used int
	err := r.db.QueryRow(context.Background(), `
		SELECT COALESCE(free_attempts_used, 0) FROM user_daily_limits
		WHERE user_id = $1 AND day = (now() AT TIME ZONE 'Asia/Tashkent')::date
	`, userId).Scan(&used)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("error while getting today free attempts: %w", err)
	}
	return used, nil
}

func (r *AppSettingRepo) IncrTodayFreeAttempts(userId string) error {
	_, err := r.db.Exec(context.Background(), `
		INSERT INTO user_daily_limits (user_id, day, free_attempts_used)
		VALUES ($1, (now() AT TIME ZONE 'Asia/Tashkent')::date, 1)
		ON CONFLICT (user_id, day) DO UPDATE SET free_attempts_used = user_daily_limits.free_attempts_used + 1
	`, userId)
	if err != nil {
		return fmt.Errorf("error while incrementing today free attempts: %w", err)
	}
	return nil
}
