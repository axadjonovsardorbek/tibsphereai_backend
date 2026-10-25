package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CaseSourceRepo struct {
	db *pgxpool.Pool
}

func NewCaseSourceRepo(db *pgxpool.Pool) *CaseSourceRepo {
	return &CaseSourceRepo{db: db}
}

func (r *CaseSourceRepo) Create(req *m.CaseSourceCreateReq) (*m.ById, error) {
	id := uuid.NewString()
	_, err := r.db.Exec(context.Background(), `
		INSERT INTO case_sources (id, name, content) VALUES ($1, $2, $3)
	`, id, req.Name, req.Content)
	if err != nil {
		return nil, fmt.Errorf("error while creating case source: %w", err)
	}
	return &m.ById{Id: id}, nil
}

func (r *CaseSourceRepo) GetById(req *m.ById) (*m.CaseSourceRes, error) {
	s := m.CaseSourceRes{}
	err := r.db.QueryRow(context.Background(), `
		SELECT id, name, content, created_at::text, updated_at::text
		FROM case_sources WHERE id = $1 AND deleted_at = 0
	`, req.Id).Scan(&s.Id, &s.Name, &s.Content, &s.CreatedAt, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting case source: %w", err)
	}
	return &s, nil
}

func (r *CaseSourceRepo) GetAll(req *m.CaseSourceGetListReq) (*m.PaginationResponse, *m.CaseSourceList, error) {
	query := `
	SELECT COUNT(id) OVER() AS total_count, id, name, content, created_at::text, updated_at::text
	FROM case_sources WHERE deleted_at = 0
	`
	var args []interface{}
	if req.Name != "" && req.Name != "string" {
		query += fmt.Sprintf(" AND name ILIKE $%d", len(args)+1)
		args = append(args, "%"+req.Name+"%")
	}
	query += " ORDER BY created_at DESC"
	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting case sources: %w", err)
	}
	defer rows.Close()

	list := m.CaseSourceList{}
	for rows.Next() {
		s := &m.CaseSourceRes{}
		var total int
		if err := rows.Scan(&total, &s.Id, &s.Name, &s.Content, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, nil, fmt.Errorf("error while scanning case source: %w", err)
		}
		list.Sources = append(list.Sources, s)
		list.Count = total
	}
	response := m.NewPaginationResponse(req.Filter, list.Count, list.Sources)
	return &response, &list, nil
}

func (r *CaseSourceRepo) Update(req *m.CaseSourceUpdateReq) (*m.Void, error) {
	var conditions []string
	var args []interface{}
	if req.Name != "" && req.Name != "string" {
		conditions = append(conditions, "name = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Name)
	}
	if req.Content != "" && req.Content != "string" {
		conditions = append(conditions, "content = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Content)
	}
	conditions = append(conditions, "updated_at = now()")
	query := "UPDATE case_sources SET " + strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0"
	args = append(args, req.Id)

	res, err := r.db.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating case source: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("case source with id %s couldnt update", req.Id)
	}
	return &m.Void{}, nil
}

func (r *CaseSourceRepo) Delete(req *m.ById) (*m.Void, error) {
	res, err := r.db.Exec(context.Background(), `
		UPDATE case_sources SET deleted_at = EXTRACT(EPOCH FROM NOW()) WHERE id = $1 AND deleted_at = 0
	`, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting case source: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("case source with id %s couldnt delete", req.Id)
	}
	return &m.Void{}, nil
}

// GetContents - generatsiya uchun manba matnlari: ids bo'sh bo'lsa barcha
// manbalar, aks holda faqat tanlanganlari.
func (r *CaseSourceRepo) GetContents(ids []string) ([]string, error) {
	query := `SELECT content FROM case_sources WHERE deleted_at = 0 AND is_active AND status = 'ready'`
	var args []interface{}
	if len(ids) > 0 {
		query += ` AND id = ANY($1::uuid[])`
		args = append(args, ids)
	}
	query += ` ORDER BY created_at`
	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while getting case source contents: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}
