package postgres

import (
	"context"
	"fmt"
	"strings"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PromoCodeRepo struct {
	db *pgxpool.Pool
}

func NewPromoCodeRepo(db *pgxpool.Pool) *PromoCodeRepo {
	return &PromoCodeRepo{db: db}
}

// Create - Hamkor (yoki Super Admin) OLDINDAN tanga qiymati bilan promokod yaratadi
// (TZ 5-6-bo'lim: to'lovga bog'liq avtomatik chiqarilmaydi).
func (r *PromoCodeRepo) Create(req *m.PromoCodeCreateReq) (*m.ById, error) {
	code := strings.ToUpper(uuid.NewString()[:8])
	id := uuid.NewString()

	scope := req.Scope
	if scope == "" {
		scope = "limited"
		if req.MaxUses == 0 {
			req.MaxUses = 1
		}
	}
	perUser := req.MaxUsesPerUser
	if perUser < 1 {
		perUser = 1
	}
	var targetUser, maxUses interface{}
	switch scope {
	case "user":
		if req.TargetUserId == "" {
			return nil, errs.ErrPromoCodeScopeInvalid
		}
		targetUser = req.TargetUserId
		maxUses = perUser
	case "users":
		seen := map[string]bool{}
		ids := req.TargetUserIds[:0:0]
		for _, u := range req.TargetUserIds {
			if u != "" && !seen[u] {
				seen[u] = true
				ids = append(ids, u)
			}
		}
		if len(ids) == 0 {
			return nil, errs.ErrPromoCodeScopeInvalid
		}
		req.TargetUserIds = ids
		maxUses = len(ids) * perUser
	case "limited":
		if req.MaxUses < 1 {
			return nil, errs.ErrPromoCodeScopeInvalid
		}
		maxUses = req.MaxUses
	case "all":
		// cheklanmagan: max_uses NULL
	default:
		return nil, errs.ErrPromoCodeScopeInvalid
	}

	// partner ixtiyoriy: bo'sh bo'lsa promokod hamkorsiz (NULL) yaratiladi.
	var partnerId interface{}
	if req.PartnerId != "" && req.PartnerId != "string" {
		partnerId = req.PartnerId
	}

	var createdBy interface{}
	if req.CreatedBy != "" {
		createdBy = req.CreatedBy
	}
	var expiresAt interface{}
	if req.ExpiresAt != "" {
		expiresAt = req.ExpiresAt
	}

	ctx := context.Background()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("error while starting promo create tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO promo_codes (id, code, partner_id, value, created_by, status, issued_at, expires_at, scope, target_user_id, max_uses, max_uses_per_user)
		VALUES ($1, $2, $3, $4, $5, 'active', now(), $6, $7, $8, $9, $10)
	`, id, code, partnerId, req.Value, createdBy, expiresAt, scope, targetUser, maxUses, perUser)
	if err != nil {
		return nil, fmt.Errorf("error while creating promo code: %w", err)
	}
	if scope == "users" {
		for _, u := range req.TargetUserIds {
			if _, err := tx.Exec(ctx, `INSERT INTO promo_code_targets (promo_code_id, user_id) VALUES ($1, $2)`, id, u); err != nil {
				return nil, fmt.Errorf("error while saving promo code targets: %w", err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("error while committing promo create: %w", err)
	}

	return &m.ById{Id: id}, nil
}

func (r *PromoCodeRepo) GetById(req *m.ById) (*m.PromoCodeRes, error) {
	query := `
	SELECT
		pc.id, pc.code, pc.partner_id, p.name, pc.value,
		pc.created_by, pc.used_by, pc.status, pc.issued_at::text, pc.expires_at::text, pc.used_at::text,
		pc.scope, pc.target_user_id, pc.max_uses, pc.max_uses_per_user, pc.used_count,
		COALESCE((SELECT array_agg(t.user_id::text) FROM promo_code_targets t WHERE t.promo_code_id = pc.id), '{}')
	FROM promo_codes pc
	LEFT JOIN partners p ON p.id = pc.partner_id
	WHERE pc.id = $1
	`

	row := r.db.QueryRow(context.Background(), query, req.Id)

	pc := m.PromoCodeRes{}
	err := row.Scan(
		&pc.Id, &pc.Code, &pc.PartnerId, &pc.PartnerName, &pc.Value,
		&pc.CreatedBy, &pc.UsedBy, &pc.Status, &pc.IssuedAt, &pc.ExpiresAt, &pc.UsedAt,
		&pc.Scope, &pc.TargetUserId, &pc.MaxUses, &pc.MaxUsesPerUser, &pc.UsedCount, &pc.TargetUserIds,
	)

	if err == pgx.ErrNoRows {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting promo code by id: %w", err)
	}

	return &pc, nil
}

func (r *PromoCodeRepo) GetAll(req *m.PromoCodeGetListReq) (*m.PaginationResponse, *m.PromoCodeList, error) {
	return r.list(req, false)
}

func (r *PromoCodeRepo) GetMy(req *m.PromoCodeGetListReq) (*m.PaginationResponse, *m.PromoCodeList, error) {
	return r.list(req, true)
}

func (r *PromoCodeRepo) list(req *m.PromoCodeGetListReq, forceUsedByUser bool) (*m.PaginationResponse, *m.PromoCodeList, error) {
	query := `
	SELECT
		COUNT(pc.id) OVER() AS total_count,
		pc.id, pc.code, pc.partner_id, p.name, pc.value,
		pc.created_by, pc.used_by, pc.status, pc.issued_at::text, pc.expires_at::text, pc.used_at::text,
		pc.scope, pc.target_user_id, pc.max_uses, pc.max_uses_per_user, pc.used_count,
		COALESCE((SELECT array_agg(t.user_id::text) FROM promo_code_targets t WHERE t.promo_code_id = pc.id), '{}')
	FROM promo_codes pc
	LEFT JOIN partners p ON p.id = pc.partner_id
	WHERE 1=1
	`

	var args []interface{}
	if (forceUsedByUser || (req.UserId != "" && req.UserId != "string")) && req.UserId != "" {
		args = append(args, req.UserId)
		query += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM promo_code_redemptions r WHERE r.promo_code_id = pc.id AND r.user_id = $%d)", len(args))
	}
	if req.PartnerId != "" && req.PartnerId != "string" {
		args = append(args, req.PartnerId)
		query += fmt.Sprintf(" AND pc.partner_id = $%d", len(args))
	}
	if req.Status != "" && req.Status != "string" {
		args = append(args, req.Status)
		query += fmt.Sprintf(" AND pc.status = $%d", len(args))
	}

	query += " ORDER BY pc.issued_at DESC"
	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting promo codes: %w", err)
	}
	defer rows.Close()

	list := m.PromoCodeList{}
	for rows.Next() {
		pc := &m.PromoCodeRes{}
		var totalCount int
		err := rows.Scan(
			&totalCount,
			&pc.Id, &pc.Code, &pc.PartnerId, &pc.PartnerName, &pc.Value,
			&pc.CreatedBy, &pc.UsedBy, &pc.Status, &pc.IssuedAt, &pc.ExpiresAt, &pc.UsedAt,
			&pc.Scope, &pc.TargetUserId, &pc.MaxUses, &pc.MaxUsesPerUser, &pc.UsedCount, &pc.TargetUserIds,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("error while scanning promo code: %w", err)
		}
		list.PromoCodes = append(list.PromoCodes, pc)
		list.Count = totalCount
	}

	response := m.NewPaginationResponse(req.Filter, list.Count, list.PromoCodes)
	return &response, &list, nil
}

// Redeem - foydalanuvchi mobil ilovada kodni kiritganda (TZ 2.3-bo'lim). Qamrovga qarab:
// "user" - faqat target_user_id; "limited" - max_uses ta foydalanuvchigacha; "all" - hamma.
// Har bir foydalanuvchi bitta kodni ko'pi bilan max_uses_per_user marta (default 1) ishlatadi. Hammasi bitta tranzaksiyada
// (qator qulflanadi), shuning uchun parallel so'rovlar limitdan oshib keta olmaydi.
// Tanga miqdorini qaytaradi (service qatlami shu qiymat bilan UserS.AddPoints chaqiradi).
func (r *PromoCodeRepo) Redeem(req *m.PromoCodeRedeemReq) (int, error) {
	ctx := context.Background()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("error while starting promo redeem tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var (
		id, status, scope string
		targetUser        *string
		maxUses           *int
		usedCount, value  int
		perUser           int
		expired           bool
	)
	err = tx.QueryRow(ctx, `
		SELECT id, status, scope, target_user_id, max_uses, used_count, value, max_uses_per_user,
		       (expires_at IS NOT NULL AND expires_at <= now())
		FROM promo_codes WHERE code = $1 FOR UPDATE
	`, req.Code).Scan(&id, &status, &scope, &targetUser, &maxUses, &usedCount, &value, &perUser, &expired)
	if err == pgx.ErrNoRows {
		return 0, errs.ErrPromoCodeInvalid
	}
	if err != nil {
		return 0, fmt.Errorf("error while looking up promo code: %w", err)
	}

	if scope == "user" && (targetUser == nil || *targetUser != req.UserId) {
		return 0, errs.ErrPromoCodeNotForYou
	}
	if scope == "users" {
		var allowed bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM promo_code_targets WHERE promo_code_id = $1 AND user_id = $2)
		`, id, req.UserId).Scan(&allowed); err != nil {
			return 0, fmt.Errorf("error while checking promo targets: %w", err)
		}
		if !allowed {
			return 0, errs.ErrPromoCodeNotForYou
		}
	}
	var userUses int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM promo_code_redemptions WHERE promo_code_id = $1 AND user_id = $2
	`, id, req.UserId).Scan(&userUses); err != nil {
		return 0, fmt.Errorf("error while checking promo redemption: %w", err)
	}
	if userUses >= perUser {
		return 0, errs.ErrPromoCodeAlreadyRedeemed
	}
	if status == "expired" || expired {
		return 0, errs.ErrPromoCodeExpired
	}
	if status == "used" || (maxUses != nil && usedCount >= *maxUses) {
		return 0, errs.ErrPromoCodeUsed
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO promo_code_redemptions (promo_code_id, user_id, coins) VALUES ($1, $2, $3)
	`, id, req.UserId, value); err != nil {
		return 0, fmt.Errorf("error while saving promo redemption: %w", err)
	}
	// status faqat limit tugaganda "used" bo'ladi ("all" kodlari hech qachon "used" bo'lmaydi).
	if _, err := tx.Exec(ctx, `
		UPDATE promo_codes
		SET used_count = used_count + 1, used_by = $2, used_at = now(),
		    status = CASE WHEN max_uses IS NOT NULL AND used_count + 1 >= max_uses THEN 'used' ELSE status END
		WHERE id = $1
	`, id, req.UserId); err != nil {
		return 0, fmt.Errorf("error while redeeming promo code: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("error while committing promo redeem: %w", err)
	}
	return value, nil
}

func (r *PromoCodeRepo) Delete(req *m.ById) (*m.Void, error) {
	_, err := r.db.Exec(context.Background(), `DELETE FROM promo_codes WHERE id = $1`, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting promo code: %w", err)
	}
	return &m.Void{}, nil
}
