package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	m "tibsphereai/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryRepo struct {
	db *pgxpool.Pool
}

func NewCategoryRepo(db *pgxpool.Pool) *CategoryRepo {
	return &CategoryRepo{db: db}
}

func (r *CategoryRepo) Create(req *m.CategoryCreateReq) (*m.ById, error) {
	audience := req.Audience
	if audience == "" {
		audience = m.AudienceDoctor
	}
	id := uuid.NewString()
	_, err := r.db.Exec(context.Background(), `
		INSERT INTO categories (id, name, icon_url, order_num, coins_per_case, audience) VALUES ($1, $2, $3, $4, $5, $6)
	`, id, req.Name, req.IconUrl, req.OrderNum, req.CoinsPerCase, audience)
	if err != nil {
		return nil, fmt.Errorf("error while creating category: %w", err)
	}
	return &m.ById{Id: id}, nil
}

func (r *CategoryRepo) GetById(req *m.ById) (*m.CategoryRes, error) {
	row := r.db.QueryRow(context.Background(), `
		SELECT id, name, icon_url, order_num, coins_per_case, audience, created_at::text, updated_at::text
		FROM categories WHERE id = $1 AND deleted_at = 0
	`, req.Id)

	var nameJSON []byte
	c := m.CategoryRes{}
	err := row.Scan(&c.Id, &nameJSON, &c.IconUrl, &c.OrderNum, &c.CoinsPerCase, &c.Audience, &c.CreatedAt, &c.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("category not found with id: %s", req.Id)
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting category by id: %w", err)
	}
	json.Unmarshal(nameJSON, &c.Name)
	return &c, nil
}

func (r *CategoryRepo) GetAll(req *m.CategoryGetListReq) (*m.PaginationResponse, *m.CategoryList, error) {
	query := `
	SELECT COUNT(id) OVER() AS total_count, id, name, icon_url, order_num, coins_per_case, audience, created_at::text, updated_at::text
	FROM categories WHERE deleted_at = 0
	`
	var args []interface{}
	if req.Name != "" && req.Name != "string" {
		query += fmt.Sprintf(" AND name::text ILIKE $%d", len(args)+1)
		args = append(args, "%"+req.Name+"%")
	}
	if req.Audience != "" {
		query += fmt.Sprintf(" AND audience = $%d", len(args)+1)
		args = append(args, req.Audience)
	}
	query += " ORDER BY order_num ASC"
	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting categories: %w", err)
	}
	defer rows.Close()

	list := m.CategoryList{}
	for rows.Next() {
		c := &m.CategoryRes{}
		var nameJSON []byte
		var total int
		if err := rows.Scan(&total, &c.Id, &nameJSON, &c.IconUrl, &c.OrderNum, &c.CoinsPerCase, &c.Audience, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, nil, fmt.Errorf("error while scanning category: %w", err)
		}
		json.Unmarshal(nameJSON, &c.Name)
		list.Categories = append(list.Categories, c)
		list.Count = total
	}

	response := m.NewPaginationResponse(req.Filter, list.Count, list.Categories)
	return &response, &list, nil
}

func (r *CategoryRepo) Update(req *m.CategoryUpdateReq) (*m.Void, error) {
	var conditions []string
	var args []interface{}

	if req.Name != nil {
		conditions = append(conditions, "name = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Name)
	}
	if req.IconUrl != "" && req.IconUrl != "string" {
		conditions = append(conditions, "icon_url = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.IconUrl)
	}
	conditions = append(conditions, "order_num = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.OrderNum)

	conditions = append(conditions, "coins_per_case = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.CoinsPerCase)

	if req.Audience != "" {
		conditions = append(conditions, "audience = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Audience)
	}

	conditions = append(conditions, "updated_at = now()")
	query := "UPDATE categories SET " + strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0"
	args = append(args, req.Id)

	res, err := r.db.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating category: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("category with id %s couldnt update", req.Id)
	}
	return &m.Void{}, nil
}

func (r *CategoryRepo) Delete(req *m.ById) (*m.Void, error) {
	res, err := r.db.Exec(context.Background(), `
		UPDATE categories SET deleted_at = EXTRACT(EPOCH FROM NOW()) WHERE id = $1 AND deleted_at = 0
	`, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting category: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("category with id %s couldnt delete", req.Id)
	}
	return &m.Void{}, nil
}

// GetByIdMobile - Diqqat: case'lar categories'ga emas, topics orqali bog'langan
// (cases.topic_id -> topics.category_id), shu sababli case-soni hisoblanganda
// topics orqali JOIN qilinadi (avvalgi versiyada bevosita cases.category_id
// ishlatilgan edi - bunday ustun endi mavjud emas, xato edi).
func (r *CategoryRepo) GetByIdMobile(req *m.ByIdMobile) (*m.CategoryMobileRes, error) {
	row := r.db.QueryRow(context.Background(), `
		SELECT c.id, c.name ->> $1, c.icon_url, c.order_num, c.coins_per_case, c.audience,
			(SELECT COUNT(*) FROM cases ca JOIN topics t ON t.id = ca.topic_id
				WHERE t.category_id = c.id AND ca.deleted_at = 0 AND ca.status = 'published')
		FROM categories c WHERE c.id = $2 AND c.deleted_at = 0
	`, req.Language, req.Id)

	c := m.CategoryMobileRes{}
	err := row.Scan(&c.Id, &c.Name, &c.IconUrl, &c.OrderNum, &c.CoinsPerCase, &c.Audience, &c.CasesCount)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("category not found with id: %s", req.Id)
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting category by id: %w", err)
	}
	return &c, nil
}

func (r *CategoryRepo) GetAllMobile(req *m.CategoryGetListMobileReq) (*m.PaginationResponse, *m.CategoryMobileList, error) {
	query := `
	SELECT COUNT(c.id) OVER() AS total_count, c.id, c.name ->> $1, c.icon_url, c.order_num, c.coins_per_case, c.audience,
		(SELECT COUNT(*) FROM cases ca JOIN topics t ON t.id = ca.topic_id
			WHERE t.category_id = c.id AND ca.deleted_at = 0 AND ca.status = 'published'),
		COALESCE((SELECT cu.cases_remaining FROM category_unlocks cu
			WHERE cu.category_id = c.id AND cu.user_id = $2), 0)
	FROM categories c
	WHERE c.deleted_at = 0
	  AND (NULLIF($3, '') IS NULL OR c.audience = $3)
	ORDER BY c.order_num ASC
	`
	var userId interface{}
	if req.UserId != "" {
		userId = req.UserId
	}
	rows, err := r.db.Query(context.Background(), query, req.Language, userId, req.Audience)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting categories: %w", err)
	}
	defer rows.Close()

	list := m.CategoryMobileList{}
	for rows.Next() {
		c := &m.CategoryMobileRes{}
		var total int
		if err := rows.Scan(&total, &c.Id, &c.Name, &c.IconUrl, &c.OrderNum, &c.CoinsPerCase, &c.Audience, &c.CasesCount, &c.CasesRemaining); err != nil {
			return nil, nil, fmt.Errorf("error while scanning category: %w", err)
		}
		list.Categories = append(list.Categories, c)
		list.Count = total
	}

	response := m.NewPaginationResponse(m.Pagination{Limit: list.Count}, list.Count, list.Categories)
	return &response, &list, nil
}
