package postgres

import (
	"context"
	"fmt"

	m "tibsphereai/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserDeviceRepo - ko'p qurilma push token (users.fcm_token o'rnini bosadi).
type UserDeviceRepo struct {
	db *pgxpool.Pool
}

func NewUserDeviceRepo(db *pgxpool.Pool) *UserDeviceRepo {
	return &UserDeviceRepo{db: db}
}

// Register - upsert: fcm_token UNIQUE, shu token boshqa userga tegishli bo'lsa qayta bog'lanadi
// (masalan qurilma boshqa hisobga login qilinganda).
func (r *UserDeviceRepo) Register(req *m.UserDeviceRegisterReq) (*m.Void, error) {
	_, err := r.db.Exec(context.Background(), `
		INSERT INTO user_devices (id, user_id, fcm_token, platform)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (fcm_token) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			platform = EXCLUDED.platform,
			updated_at = now()
	`, uuid.NewString(), req.UserId, req.FCMToken, req.Platform)
	if err != nil {
		return nil, fmt.Errorf("error while registering device: %w", err)
	}
	return &m.Void{}, nil
}

// Unregister - o'chirilgan qurilmaning user_id'sini qaytaradi (topilmasa bo'sh string) - service
// qatlami shu userning tilini topib, FCM topicdan unsubscribe qiladi.
func (r *UserDeviceRepo) Unregister(req *m.UserDeviceUnregisterReq) (string, error) {
	var userId string
	err := r.db.QueryRow(context.Background(), `
		DELETE FROM user_devices WHERE fcm_token = $1 RETURNING user_id
	`, req.FCMToken).Scan(&userId)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("error while unregistering device: %w", err)
	}
	return userId, nil
}

func (r *UserDeviceRepo) GetTokensAll(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT DISTINCT fcm_token FROM user_devices`)
	if err != nil {
		return nil, fmt.Errorf("get all device tokens: %w", err)
	}
	defer rows.Close()

	tokens := make([]string, 0, 256)
	for rows.Next() {
		var tok string
		if err := rows.Scan(&tok); err != nil {
			return nil, fmt.Errorf("scan device token: %w", err)
		}
		tokens = append(tokens, tok)
	}
	return tokens, rows.Err()
}

func (r *UserDeviceRepo) GetTokensByUser(ctx context.Context, userId string) ([]string, error) {
	rows, err := r.db.Query(ctx, `SELECT fcm_token FROM user_devices WHERE user_id = $1`, userId)
	if err != nil {
		return nil, fmt.Errorf("get user device tokens: %w", err)
	}
	defer rows.Close()

	tokens := make([]string, 0, 4)
	for rows.Next() {
		var tok string
		if err := rows.Scan(&tok); err != nil {
			return nil, fmt.Errorf("scan device token: %w", err)
		}
		tokens = append(tokens, tok)
	}
	return tokens, rows.Err()
}
