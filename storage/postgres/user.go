package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"tibsphereai/api/token"
	"tibsphereai/internal/helper"
	"tibsphereai/pkg/errs"
	"tibsphereai/pkg/models"
	m "tibsphereai/pkg/models"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type UserRepo struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

func NewUserRepo(db *pgxpool.Pool, rdb *redis.Client) *UserRepo {
	return &UserRepo{
		db:  db,
		rdb: rdb,
	}
}

func (r *UserRepo) Check(req *m.UserCheckReq) (*m.UserCheckRes, error) {
	var res m.UserCheckRes

	query := `
	SELECT EXISTS (
		SELECT 1
		FROM users
		WHERE deleted_at = 0
		AND (phone_number = $1 OR email = $1)
	)
	`
	row := r.db.QueryRow(context.Background(), query, req.Identifier)
	err := row.Scan(&res.HasAccount)
	if err != nil {
		return &res, fmt.Errorf("error while checking user existence: %w", err)
	}

	return &res, nil
}

func (r *UserRepo) EmailLogin(req *m.ConfirmationReq) (*m.TokenRes, error) {
	ctx := context.Background()

	var count int
	var user_id string
	var tkn *token.Tokens

	query := `
	SELECT
		COUNT(id) OVER() as count,
		id
	FROM
		users
	WHERE
		email = $1
	AND
		deleted_at = 0
	GROUP BY id
	`

	row := r.db.QueryRow(ctx, query, req.Identifier)

	err := row.Scan(
		&count,
		&user_id,
	)

	if err != nil {
		fmt.Println("error while checking user existence:", err)
	}

	if count == 1 && user_id != "string" {
		tkn = token.GenerateJWTToken(user_id, "user", r.rdb)
		return &m.TokenRes{
			AccessToken:  tkn.AccessToken,
			RefreshToken: tkn.RefreshToken,
			Role:         "user",
			Id:           user_id,
		}, nil
	}

	id := uuid.New().String()
	refCode := r.newReferralCode(ctx)

	insertQuery := `
		INSERT INTO users (
			id,
			email,
			name,
			referral_code
		) VALUES($1, $2, $3, $4)
		`

	_, err = r.db.Exec(ctx, insertQuery, id, req.Identifier, "Foydalanuvchi", refCode)

	if err != nil {
		return nil, err
	}
	r.applyReferral(ctx, id, req.ReferralCode)
	tkn = token.GenerateJWTToken(id, "user", r.rdb)
	return &m.TokenRes{
		AccessToken:  tkn.AccessToken,
		RefreshToken: tkn.RefreshToken,
		Role:         "user",
		Id:           id,
	}, nil
}

func (r *UserRepo) TelegramLogin(req *m.TelegramLoginReq) (*m.TokenRes, error) {
	ctx := context.Background()

	var count int
	var user_id string
	var tkn *token.Tokens

	query := `
	SELECT
		COUNT(id) OVER() as count,
		id
	FROM
		users
	WHERE
		phone_number = $1
	AND
		deleted_at = 0
	GROUP BY id
	`

	row := r.db.QueryRow(ctx, query, req.Phone)

	err := row.Scan(
		&count,
		&user_id,
	)

	if err != nil {
		fmt.Println("error while checking user existence:", err)
	}

	if count == 1 && user_id != "string" {
		tkn = token.GenerateJWTToken(user_id, "user", r.rdb)
		return &m.TokenRes{
			AccessToken:  tkn.AccessToken,
			RefreshToken: tkn.RefreshToken,
			Role:         "user",
			Id:           user_id,
		}, nil
	}

	id := uuid.New().String()
	refCode := r.newReferralCode(ctx)

	insertQuery := `
		INSERT INTO users (
			id,
			phone_number,
			name,
			referral_code
		) VALUES($1, $2, $3, $4)
		`

	_, err = r.db.Exec(ctx, insertQuery, id, req.Phone, req.Name, refCode)

	if err != nil {
		return nil, err
	}
	r.applyReferral(ctx, id, req.ReferralCode)
	tkn = token.GenerateJWTToken(id, "user", r.rdb)
	return &m.TokenRes{
		AccessToken:  tkn.AccessToken,
		RefreshToken: tkn.RefreshToken,
		Role:         "user",
		Id:           id,
	}, nil
}

// GoogleLogin - "1 bosqichli kirish" (BACKEND_STRUCTURE.md 1-bo'lim). Google
// id_token'ni service qatlami allaqachon tekshirgan (internal/ai emas,
// google.golang.org/api/idtoken orqali) - bu yerda faqat google_id bo'yicha
// user topiladi yoki yaratiladi.
func (r *UserRepo) GoogleLogin(req *m.GoogleAuthReq) (*m.TokenRes, error) {
	ctx := context.Background()

	var user_id string
	query := `SELECT id FROM users WHERE google_id = $1 AND deleted_at = 0`
	err := r.db.QueryRow(ctx, query, req.GoogleId).Scan(&user_id)

	if err != nil && err != pgx.ErrNoRows {
		return nil, fmt.Errorf("error while checking google user: %w", err)
	}

	if err == pgx.ErrNoRows {
		// email bo'yicha ham qidiramiz - oldin OTP bilan ro'yxatdan o'tgan bo'lishi mumkin
		if req.Email != "" {
			_ = r.db.QueryRow(ctx, `SELECT id FROM users WHERE email = $1 AND deleted_at = 0`, req.Email).Scan(&user_id)
		}
	}

	isNew := false
	if user_id == "" {
		isNew = true
		user_id = uuid.NewString()
		refCode := r.newReferralCode(ctx)
		insertQuery := `
		INSERT INTO users (id, google_id, email, name, referral_code)
		VALUES ($1, $2, $3, $4, $5)
		`
		_, err = r.db.Exec(ctx, insertQuery, user_id, req.GoogleId, req.Email, req.Name, refCode)
		if err != nil {
			return nil, fmt.Errorf("error while creating google user: %w", err)
		}
	} else {
		_, _ = r.db.Exec(ctx, `UPDATE users SET google_id = $1 WHERE id = $2 AND google_id IS NULL`, req.GoogleId, user_id)
	}

	if isNew {
		r.applyReferral(ctx, user_id, req.ReferralCode)
	}

	tkn := token.GenerateJWTToken(user_id, "user", r.rdb)
	return &m.TokenRes{
		AccessToken:  tkn.AccessToken,
		RefreshToken: tkn.RefreshToken,
		Role:         "user",
		Id:           user_id,
	}, nil
}

// newReferralCode - yangi user uchun noyob referral_code (3 urinishgacha,
// deyarli hech qachon collision bo'lmaydi - lekin himoya bo'lsin).
func (r *UserRepo) newReferralCode(ctx context.Context) string {
	for i := 0; i < 3; i++ {
		code := helper.GenerateReferralCode()
		var exists bool
		if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE referral_code = $1)`, code).Scan(&exists); err == nil && !exists {
			return code
		}
	}
	return uuid.New().String()[:8]
}

// applyReferral - registratsiyada referral_code yuborilgan bo'lsa, kodni
// egasini topib referred_by'ni qo'yadi va ikkalasiga ham tanga beradi
// (miqdorlar app_settings'dan). Xato bo'lsa ham registratsiyani buzmaydi -
// shuning uchun error qaytarmaydi, faqat log qiladi.
func (r *UserRepo) applyReferral(ctx context.Context, newUserId, referralCode string) {
	if referralCode == "" {
		return
	}

	var referrerId string
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE referral_code = $1 AND deleted_at = 0`, referralCode).Scan(&referrerId)
	if err != nil {
		if err != pgx.ErrNoRows {
			slog.Error("applyReferral: lookup failed", "err", err)
		}
		return
	}
	if referrerId == newUserId {
		return // o'zini o'zi taklif qilolmaydi
	}

	referrerCoins := 50
	referredCoins := 20
	if v, err := r.getAppSettingInt(ctx, "referral_reward_referrer_coins"); err == nil {
		referrerCoins = v
	}
	if v, err := r.getAppSettingInt(ctx, "referral_reward_referred_coins"); err == nil {
		referredCoins = v
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		slog.Error("applyReferral: begin tx failed", "err", err)
		return
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE users SET referred_by = $1 WHERE id = $2`, referrerId, newUserId); err != nil {
		slog.Error("applyReferral: set referred_by failed", "err", err)
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET coins = coins + $1 WHERE id = $2`, referrerCoins, referrerId); err != nil {
		slog.Error("applyReferral: reward referrer failed", "err", err)
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET coins = coins + $1 WHERE id = $2`, referredCoins, newUserId); err != nil {
		slog.Error("applyReferral: reward referred failed", "err", err)
		return
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO referral_rewards (referrer_id, referred_id, referrer_coins, referred_coins)
		VALUES ($1, $2, $3, $4)
	`, referrerId, newUserId, referrerCoins, referredCoins); err != nil {
		slog.Error("applyReferral: insert reward row failed", "err", err)
		return
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("applyReferral: commit failed", "err", err)
	}
}

func (r *UserRepo) getAppSettingInt(ctx context.Context, key string) (int, error) {
	var raw string
	if err := r.db.QueryRow(ctx, `SELECT value FROM app_settings WHERE key = $1`, key).Scan(&raw); err != nil {
		return 0, err
	}
	return strconv.Atoi(raw)
}

// GetReferralInfo - user'ning o'z referral_code'i, taklif qilganlar soni va
// ular evaziga olingan jami tanga (mobile profil "do'stlarni taklif qilish" ekrani).
func (r *UserRepo) GetReferralInfo(userId string) (*m.ReferralInfoRes, error) {
	ctx := context.Background()
	res := &m.ReferralInfoRes{}

	if err := r.db.QueryRow(ctx, `SELECT referral_code FROM users WHERE id = $1`, userId).Scan(&res.ReferralCode); err != nil {
		return nil, fmt.Errorf("error getting referral code: %w", err)
	}

	res.ReferrerReward = 50
	res.ReferredReward = 20
	if v, err := r.getAppSettingInt(ctx, "referral_reward_referrer_coins"); err == nil {
		res.ReferrerReward = v
	}
	if v, err := r.getAppSettingInt(ctx, "referral_reward_referred_coins"); err == nil {
		res.ReferredReward = v
	}

	rows, err := r.db.Query(ctx, `
		SELECT u.name, rr.created_at, rr.referrer_coins
		FROM referral_rewards rr
		JOIN users u ON u.id = rr.referred_id
		WHERE rr.referrer_id = $1
		ORDER BY rr.created_at DESC
	`, userId)
	if err != nil {
		return nil, fmt.Errorf("error getting referral history: %w", err)
	}
	defer rows.Close()

	res.Invited = []m.ReferralInvitedItem{}
	for rows.Next() {
		var item m.ReferralInvitedItem
		var createdAt time.Time
		if err := rows.Scan(&item.Name, &createdAt, &item.CoinsGiven); err != nil {
			return nil, fmt.Errorf("error scanning referral history: %w", err)
		}
		item.JoinedAt = createdAt.Format(time.RFC3339)
		res.Invited = append(res.Invited, item)
		res.InvitedCount++
		res.TotalCoinsEarned += item.CoinsGiven
	}

	return res, nil
}

func (r *UserRepo) GetById(req *m.ById) (*m.UserRes, error) {
	query := `
	SELECT
		id,
		name,
		phone_number,
		email,
		image_url,
		language,
		specialization,
		level,
		xp,
		coins,
		streak_count,
		created_at::text,
		updated_at::text
	FROM
		users
	WHERE
		deleted_at = 0
	AND
		id = $1
	`

	row := r.db.QueryRow(context.Background(), query, req.Id)

	var user m.UserRes
	err := row.Scan(
		&user.Id,
		&user.Name,
		&user.PhoneNumber,
		&user.Email,
		&user.ImageUrl,
		&user.Language,
		&user.Specialization,
		&user.Level,
		&user.XP,
		&user.Coins,
		&user.StreakCount,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, errs.ErrUserNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("error while getting user by id: %w", err)
	}

	return &user, nil
}

func (r *UserRepo) GetAll(req *m.UserGetListReq) (*m.PaginationResponse, *m.UserList, error) {
	query := `
	SELECT
		COUNT(id) OVER() AS total_count,
		id,
		name,
		phone_number,
		email,
		image_url,
		language,
		specialization,
		level,
		xp,
		coins,
		streak_count,
		created_at::text,
		updated_at::text
	FROM
		users
	WHERE
		deleted_at = 0
	`

	var args []interface{}

	if req.Name != "" && req.Name != "string" {
		query += fmt.Sprintf(" AND name ILIKE $%d", len(args)+1)
		args = append(args, "%"+req.Name+"%")
	}
	if req.PhoneNumber != "" && req.PhoneNumber != "string" {
		query += fmt.Sprintf(" AND phone_number ILIKE $%d", len(args)+1)
		args = append(args, "%"+req.PhoneNumber+"%")
	}

	query += " ORDER BY created_at DESC"

	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting users: %w", err)
	}
	defer rows.Close()

	users := m.UserList{}

	for rows.Next() {
		user := m.UserRes{}
		var totalCount int
		err := rows.Scan(
			&totalCount,
			&user.Id,
			&user.Name,
			&user.PhoneNumber,
			&user.Email,
			&user.ImageUrl,
			&user.Language,
			&user.Specialization,
			&user.Level,
			&user.XP,
			&user.Coins,
			&user.StreakCount,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("error while scanning user row: %w", err)
		}
		users.Users = append(users.Users, user)
		users.Count = totalCount
	}

	response := m.NewPaginationResponse(req.Filter, users.Count, users.Users)

	return &response, &users, nil
}

func (r *UserRepo) Update(req *m.UserUpdateReq) (*m.Void, error) {
	ctx := context.Background()
	query := `
	UPDATE users
	SET
	`

	var conditions []string
	var args []interface{}

	if req.Name != "" && req.Name != "string" {
		conditions = append(conditions, " name = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Name)
	}
	if req.PhoneNumber != "" && req.PhoneNumber != "string" {
		var exists bool
		checkQuery := `
			SELECT EXISTS(
				SELECT 1
				FROM users
				WHERE phone_number = $1
				  AND id <> $2
				  AND deleted_at = 0
			)
		`
		err := r.db.QueryRow(ctx, checkQuery, req.PhoneNumber, req.Id).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("error while checking phone number: %w", err)
		}
		if exists {
			return nil, errs.ErrUserExists
		}

		conditions = append(conditions, "phone_number = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.PhoneNumber)
	}
	if req.ImageUrl != "" && req.ImageUrl != "string" {
		conditions = append(conditions, " image_url = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.ImageUrl)
	}
	if req.Language != "" && req.Language != "string" {
		conditions = append(conditions, " language = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Language)
	}
	if req.Specialization != "" && req.Specialization != "string" {
		conditions = append(conditions, " specialization = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Specialization)
	}
	if req.Email != "" && req.Email != "string" {
		var exists bool
		checkQuery := `
			SELECT EXISTS(
				SELECT 1
				FROM users
				WHERE email = $1
				  AND id <> $2
				  AND deleted_at = 0
			)
		`
		err := r.db.QueryRow(ctx, checkQuery, req.Email, req.Id).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("error while checking email: %w", err)
		}
		if exists {
			return nil, errs.ErrUserExists
		}

		conditions = append(conditions, "email = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Email)
	}

	if len(conditions) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	conditions = append(conditions, " updated_at = now()")
	query += strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0"

	args = append(args, req.Id)

	res, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating user: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("user with id %s couldnt update", req.Id)
	}
	return &m.Void{}, nil
}

func (r *UserRepo) Delete(req *m.ById) (*m.Void, error) {
	query := `
	UPDATE users
	SET
		deleted_at = EXTRACT(EPOCH FROM NOW())
	WHERE
		id = $1
	AND
		deleted_at = 0
	`

	res, err := r.db.Exec(context.Background(), query, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting user: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("user with id %s couldnt delete", req.Id)
	}

	return &m.Void{}, nil
}

// ChangePassword - faqat admin uchun (mobil user'da parol yo'q, TZ 3-bo'lim: faqat OTP/Google).
func (r *UserRepo) ChangePassword(req *m.ChangePasswordReq) (*m.Void, error) {
	if req.Role != "admin" {
		return nil, errs.ErrForbidden.Wrap(fmt.Errorf("change password is only supported for admins"))
	}

	ctx := context.Background()
	var curPass string

	row := r.db.QueryRow(ctx, "SELECT password FROM admins WHERE id = $1", req.Id)
	if err := row.Scan(&curPass); err != nil {
		return nil, err
	}

	var valid bool
	if strings.HasPrefix(curPass, "$2") {
		valid = bcrypt.CompareHashAndPassword([]byte(curPass), []byte(req.OldPassword)) == nil
	} else if strings.HasPrefix(curPass, "pbkdf2_sha256$") {
		valid, _ = helper.CheckDjangoPBKDF2(req.OldPassword, curPass)
	}
	if !valid {
		return nil, errs.ErrInvalidCredentials
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error while hashing password: %w", err)
	}

	_, err = r.db.Exec(ctx, "UPDATE admins SET password = $1 WHERE id = $2", string(hashedPassword), req.Id)
	if err != nil {
		return nil, err
	}

	return &m.Void{}, nil
}

// RefreshPassword - faqat admin uchun (mobil user'da parol yo'q).
func (r *UserRepo) RefreshPassword(req *m.RefreshPasswordReq) (*m.RefreshPassword, error) {
	if req.Role != "admin" {
		return nil, errs.ErrForbidden.Wrap(fmt.Errorf("refresh password is only supported for admins"))
	}

	ctx := context.Background()
	password := helper.GeneratePassword()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error while hashing password: %w", err)
	}

	_, err = r.db.Exec(ctx, "UPDATE admins SET password = $1 WHERE id = $2", string(hashedPassword), req.UserId)
	if err != nil {
		return nil, err
	}

	return &m.RefreshPassword{
		UserId:   req.UserId,
		Password: password,
	}, nil
}

func (r *UserRepo) CreateActivity(req *m.UserActivityCreateReq) (*m.Void, error) {

	query := `
    INSERT INTO user_activities (user_id, day, activity)
    VALUES ($1, (now() AT TIME ZONE 'Asia/Tashkent')::date, $2)
    ON CONFLICT (user_id, day)
    DO UPDATE SET
		activity = user_activities.activity + EXCLUDED.activity,
        updated_at = now();
    `
	_, err := r.db.Exec(context.Background(), query, req.UserId, req.Activity)

	if err != nil {
		return nil, fmt.Errorf("error while creating/updating user activity: %w", err)
	}

	return &m.Void{}, nil
}

// AddPoints - keys yakunlanganda yoki promokod ishlatilganda XP/Coin qo'shish va
// levels jadvaliga qarab users.level'ni qayta hisoblash (BACKEND_STRUCTURE.md 6-bo'lim).
func (r *UserRepo) AddPoints(req *m.AddPointsReq) (*m.Void, error) {
	ctx := context.Background()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		UPDATE users
		SET xp = xp + $1, coins = coins + $2, updated_at = now()
		WHERE id = $3 AND deleted_at = 0
	`, req.XPDelta, req.CoinsDelta, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("error while adding points: %w", err)
	}

	// eng yuqori mos level'ni levels jadvalidan tanlab qo'yamiz (statik jadval)
	_, err = tx.Exec(ctx, `
		UPDATE users u
		SET level = COALESCE((
			SELECT l.level_number FROM levels l
			WHERE l.required_xp <= u.xp
			ORDER BY l.required_xp DESC
			LIMIT 1
		), u.level)
		WHERE u.id = $1
	`, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("error while recalculating level: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO point_transactions (id, user_id, kind, delta, reason, reference_id)
		VALUES ($1, $2, 'xp', $3, $4, $5), ($6, $2, 'coin', $7, $4, $5)
	`, uuid.NewString(), req.UserId, req.XPDelta, req.Reason, req.ReferenceId, uuid.NewString(), req.CoinsDelta)
	if err != nil {
		return nil, fmt.Errorf("error while logging point transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &m.Void{}, nil
}

func (r *UserRepo) GetActivities(req *m.UserActivityGetListReq) (*m.ActivityStatsResponse, error) {
	ctx := context.Background()

	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	if req.UserId == "" {
		return nil, fmt.Errorf("user_id required")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("type required")
	}

	res := &models.ActivityStatsResponse{
		UserID: req.UserId,
		Type:   req.Type,
		Items:  make([]models.ActivityStatItem, 0, 32),
	}

	switch req.Type {

	case "day":
		query := `
	SELECT
		COALESCE(NULLIF($2, '')::date, CURRENT_DATE) AS day,
		COALESCE(ua.activity, 0) AS value
	FROM (SELECT 1) x
	LEFT JOIN user_activities ua
	ON ua.user_id = $1
	AND ua.day = COALESCE(NULLIF($2, '')::date, CURRENT_DATE)

		`

		var day time.Time
		var value int64

		if err := r.db.QueryRow(ctx, query, req.UserId, req.Date).Scan(&day, &value); err != nil {
			return nil, fmt.Errorf("day query error: %w", err)
		}

		res.Items = append(res.Items, models.ActivityStatItem{
			Date:  day.Format("2006-01-02"),
			Value: value,
		})
		res.Total = value
		return res, nil

	case "week", "month", "range":
		var start interface{}
		var end interface{}

		if req.Type == "range" {
			if req.FromDate == "" || req.ToDate == "" {
				return nil, fmt.Errorf("from_date and to_date are required for range")
			}
			start = req.FromDate
			end = req.ToDate

			queryRange := `
			WITH r AS (
			  SELECT
				$1::uuid AS user_id,
				$2::date AS start_day,
				LEAST($3::date, CURRENT_DATE) AS end_day
			),
			series AS (
			  SELECT
				d.day::date AS day,
				COALESCE(ua.activity, 0) AS value
			  FROM r
			  JOIN LATERAL generate_series(r.start_day, r.end_day, interval '1 day') AS d(day)
				ON r.start_day <= r.end_day
			  LEFT JOIN user_activities ua
				ON ua.user_id = r.user_id
			   AND ua.day = d.day::date
			)
			SELECT day, value, SUM(value) OVER () AS total
			FROM series
			ORDER BY day DESC
			`

			rows, err := r.db.Query(ctx, queryRange, req.UserId, start, end)
			if err != nil {
				return nil, fmt.Errorf("range query error: %w", err)
			}
			defer rows.Close()

			items, total, err := scanKeyValueTotalRows(rows)
			if err != nil {
				return nil, fmt.Errorf("range scan error: %w", err)
			}
			res.Items = items
			res.Total = total
			return res, nil
		}

		var seriesQuery string
		if req.Type == "week" {
			seriesQuery = `
			WITH r AS (
			  SELECT $1::uuid AS user_id,
			         date_trunc('week', CURRENT_DATE)::date AS start_day,
			         CURRENT_DATE AS end_day
			),
			series AS (
			  SELECT
				d.day::date AS day,
				COALESCE(ua.activity, 0) AS value
			  FROM r
			  JOIN LATERAL generate_series(r.start_day, r.end_day, interval '1 day') AS d(day) ON TRUE
			  LEFT JOIN user_activities ua
				ON ua.user_id = r.user_id
			   AND ua.day = d.day::date
			)
			SELECT day, value, SUM(value) OVER () AS total
			FROM series
			ORDER BY day DESC
			`
		} else {
			seriesQuery = `
			WITH r AS (
			  SELECT $1::uuid AS user_id,
			         date_trunc('month', CURRENT_DATE)::date AS start_day,
			         CURRENT_DATE AS end_day
			),
			series AS (
			  SELECT
				d.day::date AS day,
				COALESCE(ua.activity, 0) AS value
			  FROM r
			  JOIN LATERAL generate_series(r.start_day, r.end_day, interval '1 day') AS d(day) ON TRUE
			  LEFT JOIN user_activities ua
				ON ua.user_id = r.user_id
			   AND ua.day = d.day::date
			)
			SELECT day, value, SUM(value) OVER () AS total
			FROM series
			ORDER BY day DESC
			`
		}

		rows, err := r.db.Query(ctx, seriesQuery, req.UserId)
		if err != nil {
			return nil, fmt.Errorf("%s query error: %w", req.Type, err)
		}
		defer rows.Close()

		items, total, err := scanKeyValueTotalRows(rows)
		if err != nil {
			return nil, fmt.Errorf("%s scan error: %w", req.Type, err)
		}
		res.Items = items
		res.Total = total
		return res, nil

	case "year":
		query := `
		WITH months AS (
		  SELECT generate_series(
				   date_trunc('year', CURRENT_DATE)::date,
				   date_trunc('month', CURRENT_DATE)::date,
				   interval '1 month'
				 )::date AS month_start
		),
		agg AS (
		  SELECT
			m.month_start AS day,
			COALESCE(SUM(ua.activity), 0) AS value
		  FROM months m
		  LEFT JOIN user_activities ua
			ON ua.user_id = $1
		   AND ua.day >= m.month_start
		   AND ua.day <  (m.month_start + interval '1 month')::date
		   AND ua.day <= CURRENT_DATE
		  GROUP BY m.month_start
		)
		SELECT day, value, SUM(value) OVER () AS total
		FROM agg
		ORDER BY day DESC
		`

		rows, err := r.db.Query(ctx, query, req.UserId)
		if err != nil {
			return nil, fmt.Errorf("year query error: %w", err)
		}
		defer rows.Close()

		items, total, err := scanKeyValueTotalRows(rows)
		if err != nil {
			return nil, fmt.Errorf("year scan error: %w", err)
		}
		res.Items = items
		res.Total = total
		return res, nil

	default:
		return nil, fmt.Errorf("unsupported type: %s", req.Type)
	}
}

func (r *UserRepo) GetRating(req *m.RatingRequest) (*m.RatingResponse, error) {
	ctx := context.Background()

	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 10
	}
	if req.Type == "" {
		req.Type = "total"
	}

	loc := time.FixedZone("UTC+5", 5*3600)
	rt := m.RatingType(req.Type)
	start, end := calcRange(rt, loc)

	res := &m.RatingResponse{
		Type:  rt,
		Limit: req.Limit,
		Items: make([]m.RatingUserItem, 0, req.Limit),
	}

	query := `
	WITH scores AS (
		SELECT
		  u.id AS user_id,
		  u.name AS name,
		  u.image_url AS avatar,
		  COALESCE(SUM(ua.activity), 0) AS activity
		FROM users u
		LEFT JOIN user_activities ua
		  ON ua.user_id = u.id
		 AND ua.day >= $1::date
		 AND ua.day <= $2::date
		GROUP BY u.id, u.name, u.image_url
	),
	ranked AS (
		SELECT
		  user_id,
		  name,
		  avatar,
		  activity,
		  ROW_NUMBER() OVER (
			ORDER BY activity DESC, name ASC, user_id ASC
		  ) AS rank
		FROM scores
	),
	topn AS (
		SELECT user_id, name, avatar, activity, rank
		FROM ranked
		ORDER BY rank
		LIMIT $3
	),
	me AS (
		SELECT user_id, name, avatar, activity, rank
		FROM ranked
		WHERE user_id = $4::uuid
	)
	SELECT 'top' AS kind, user_id, name, avatar, activity, rank FROM topn
	UNION ALL
	SELECT 'me'  AS kind, user_id, name, avatar, activity, rank FROM me;
	`

	rows, err := r.db.Query(ctx, query, start, end, req.Limit, req.UserId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var kind string
		var it m.RatingUserItem

		if err := rows.Scan(&kind, &it.UserID, &it.Name, &it.ImageUrl, &it.Activity, &it.Rank); err != nil {
			return nil, err
		}

		if kind == "me" {
			res.Me = &it
		} else {
			res.Items = append(res.Items, it)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

func (r *UserRepo) GetTop(req *m.RatingRequest) ([]m.RatingUserItem, error) {
	ctx := context.Background()

	qTopRating := `
	WITH scores AS (
	  SELECT
	    u.id AS user_id,
	    u.name AS name,
	    u.image_url AS avatar,
	    COALESCE(SUM(ua.activity), 0) AS activity
	  FROM users u
	  LEFT JOIN user_activities ua
	    ON ua.user_id = u.id
	   AND ua.day >= $1::date
	   AND ua.day <= $2::date
	  GROUP BY u.id, u.name, u.image_url
	),
	ranked AS (
	  SELECT
	    user_id, name, avatar, activity,
	    ROW_NUMBER() OVER (ORDER BY activity DESC, name ASC, user_id ASC) AS rank
	  FROM scores
	)
	SELECT user_id, name, avatar, activity, rank, (user_id = $3) as is_me
	FROM ranked
	ORDER BY rank
	LIMIT $4
	`

	loc := time.FixedZone("UTC+5", 5*3600)
	rt := m.RatingType(req.Type)
	start, end := calcRange(rt, loc)

	rows, err := r.db.Query(ctx, qTopRating, start, end, req.UserId, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("top rating query: %w", err)
	}
	defer rows.Close()

	items := make([]m.RatingUserItem, 0, req.Limit)
	for rows.Next() {
		var it m.RatingUserItem

		if err := rows.Scan(&it.UserID, &it.Name, &it.ImageUrl, &it.Activity, &it.Rank, &it.IsMe); err != nil {
			return nil, fmt.Errorf("top rating scan: %w", err)
		}

		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("top rating rows: %w", err)
	}

	slog.Debug("GetTop rating", "type", req.Type, "limit", req.Limit, "items_count", len(items))
	return items, nil
}

func (r *UserRepo) GetMe(req *m.RatingRequest) (*m.RatingUserItem, error) {
	ctx := context.Background()

	qMeRating := `
	WITH scores AS (
	  SELECT
	    u.id AS user_id,
	    u.name AS name,
	    u.image_url AS avatar,
	    COALESCE(SUM(ua.activity), 0) AS activity
	  FROM users u
	  LEFT JOIN user_activities ua
	    ON ua.user_id = u.id
	   AND ua.day >= $1::date
	   AND ua.day <= $2::date
	  GROUP BY u.id, u.name, u.image_url
	),
	ranked AS (
	  SELECT
	    user_id, name, avatar, activity,
	    ROW_NUMBER() OVER (ORDER BY activity DESC, name ASC, user_id ASC) AS rank
	  FROM scores
	)
	SELECT user_id, name, avatar, activity, rank
	FROM ranked
	WHERE user_id = $3::uuid
	`

	var it models.RatingUserItem

	loc := time.FixedZone("UTC+5", 5*3600)
	rt := m.RatingType(req.Type)
	start, end := calcRange(rt, loc)

	err := r.db.QueryRow(ctx, qMeRating, start, end, req.UserId).
		Scan(&it.UserID, &it.Name, &it.ImageUrl, &it.Activity, &it.Rank)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("me rating query: %w", err)
	}

	it.IsMe = true

	return &it, nil
}

func scanKeyValueTotalRows(rows pgx.Rows) (items []models.ActivityStatItem, total int64, err error) {
	for rows.Next() {
		var day time.Time
		var value int64
		var rowTotal *int64

		if err := rows.Scan(&day, &value, &rowTotal); err != nil {
			return nil, 0, err
		}
		if rowTotal != nil {
			total = *rowTotal
		}
		items = append(items, models.ActivityStatItem{
			Date:  day.Format("2006-01-02"),
			Value: value,
		})
	}
	return items, total, rows.Err()
}

func calcRange(t m.RatingType, loc *time.Location) (time.Time, time.Time) {
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	switch t {
	case m.RatingDay:
		return today, today

	case m.RatingWeek:
		wd := int(today.Weekday())
		if wd == 0 {
			wd = 7
		}
		start := today.AddDate(0, 0, -(wd - 1))
		return start, today

	case m.RatingMonth:
		start := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, loc)
		return start, today

	case m.RatingYear:
		start := time.Date(today.Year(), time.January, 1, 0, 0, 0, 0, loc)
		return start, today

	default:
		start := time.Date(1970, 1, 1, 0, 0, 0, 0, loc)
		return start, today
	}
}
