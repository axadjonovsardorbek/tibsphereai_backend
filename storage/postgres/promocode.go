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

	var createdBy interface{}
	if req.CreatedBy != "" {
		createdBy = req.CreatedBy
	}
	var expiresAt interface{}
	if req.ExpiresAt != "" {
		expiresAt = req.ExpiresAt
	}

	_, err := r.db.Exec(context.Background(), `
		INSERT INTO promo_codes (id, code, partner_id, value, created_by, status, issued_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, 'active', now(), $6)
	`, id, code, req.PartnerId, req.Value, createdBy, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("error while creating promo code: %w", err)
	}

	return &m.ById{Id: id}, nil
}

func (r *PromoCodeRepo) GetById(req *m.ById) (*m.PromoCodeRes, error) {
	query := `
	SELECT
		pc.id, pc.code, pc.partner_id, p.name, pc.value,
		pc.created_by, pc.used_by, pc.status, pc.issued_at::text, pc.expires_at::text, pc.used_at::text
	FROM promo_codes pc
	JOIN partners p ON p.id = pc.partner_id
	WHERE pc.id = $1
	`

	row := r.db.QueryRow(context.Background(), query, req.Id)

	pc := m.PromoCodeRes{}
	err := row.Scan(
		&pc.Id, &pc.Code, &pc.PartnerId, &pc.PartnerName, &pc.Value,
		&pc.CreatedBy, &pc.UsedBy, &pc.Status, &pc.IssuedAt, &pc.ExpiresAt, &pc.UsedAt,
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
		pc.created_by, pc.used_by, pc.status, pc.issued_at::text, pc.expires_at::text, pc.used_at::text
	FROM promo_codes pc
	JOIN partners p ON p.id = pc.partner_id
	WHERE 1=1
	`

	var args []interface{}
	if (forceUsedByUser || (req.UserId != "" && req.UserId != "string")) && req.UserId != "" {
		args = append(args, req.UserId)
		query += fmt.Sprintf(" AND pc.used_by = $%d", len(args))
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

// Redeem - foydalanuvchi mobil ilovada kodni kiritganda (TZ 2.3-bo'lim): "active" va
// muddati o'tmagan bo'lsa "used" qiladi, tanga miqdorini qaytaradi (service qatlami
// shu qiymat bilan UserS.AddPoints chaqiradi - status avtomatik, admin qo'lda o'zgartirmaydi).
func (r *PromoCodeRepo) Redeem(req *m.PromoCodeRedeemReq) (int, error) {
	var status string
	var expiresAt *string
	var value int

	err := r.db.QueryRow(context.Background(), `
		SELECT status, expires_at::text, value FROM promo_codes WHERE code = $1
	`, req.Code).Scan(&status, &expiresAt, &value)
	if err == pgx.ErrNoRows {
		return 0, errs.ErrPromoCodeInvalid
	}
	if err != nil {
		return 0, fmt.Errorf("error while looking up promo code: %w", err)
	}

	if status == "used" {
		return 0, errs.ErrPromoCodeUsed
	}
	if status == "expired" {
		return 0, errs.ErrPromoCodeExpired
	}

	res, err := r.db.Exec(context.Background(), `
		UPDATE promo_codes
		SET status = 'used', used_by = $2, used_at = now()
		WHERE code = $1 AND status = 'active'
		  AND (expires_at IS NULL OR expires_at > now())
	`, req.Code, req.UserId)
	if err != nil {
		return 0, fmt.Errorf("error while redeeming promo code: %w", err)
	}
	if res.RowsAffected() == 0 {
		return 0, errs.ErrPromoCodeExpired
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
