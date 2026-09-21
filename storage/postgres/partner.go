package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PartnerRepo struct {
	db *pgxpool.Pool
}

func NewPartnerRepo(db *pgxpool.Pool) *PartnerRepo {
	return &PartnerRepo{db: db}
}

func (r *PartnerRepo) Create(req *m.PartnerCreateReq) (*m.ById, error) {
	id := uuid.NewString()
	desc, _ := json.Marshal(req.Description)
	_, err := r.db.Exec(context.Background(), `
		INSERT INTO partners (id, name, image_url, link_url, description) VALUES ($1, $2, $3, $4, $5)
	`, id, req.Name, req.ImageUrl, req.LinkUrl, string(desc))
	if err != nil {
		return nil, fmt.Errorf("error while creating partner: %w", err)
	}
	return &m.ById{Id: id}, nil
}

func (r *PartnerRepo) GetById(req *m.ById) (*m.PartnerRes, error) {
	row := r.db.QueryRow(context.Background(), `
		SELECT id, name, image_url, link_url, description, created_at::text, updated_at::text
		FROM partners WHERE id = $1 AND deleted_at = 0
	`, req.Id)

	p := m.PartnerRes{}
	var descJSON []byte
	err := row.Scan(&p.Id, &p.Name, &p.ImageUrl, &p.LinkUrl, &descJSON, &p.CreatedAt, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting partner by id: %w", err)
	}
	json.Unmarshal(descJSON, &p.Description)
	return &p, nil
}

func (r *PartnerRepo) GetAll(req *m.PartnerGetListReq) (*m.PaginationResponse, *m.PartnerList, error) {
	query := `
	SELECT COUNT(id) OVER() AS total_count, id, name, image_url, link_url, description, created_at::text, updated_at::text
	FROM partners WHERE deleted_at = 0
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
		return nil, nil, fmt.Errorf("error while getting partners: %w", err)
	}
	defer rows.Close()

	list := m.PartnerList{}
	for rows.Next() {
		p := &m.PartnerRes{}
		var descJSON []byte
		var total int
		if err := rows.Scan(&total, &p.Id, &p.Name, &p.ImageUrl, &p.LinkUrl, &descJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, nil, fmt.Errorf("error while scanning partner: %w", err)
		}
		json.Unmarshal(descJSON, &p.Description)
		list.Partners = append(list.Partners, p)
		list.Count = total
	}

	response := m.NewPaginationResponse(req.Filter, list.Count, list.Partners)
	return &response, &list, nil
}

func (r *PartnerRepo) Update(req *m.PartnerUpdateReq) (*m.Void, error) {
	var conditions []string
	var args []interface{}

	if req.Name != "" && req.Name != "string" {
		conditions = append(conditions, "name = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Name)
	}
	if req.ImageUrl != "" && req.ImageUrl != "string" {
		conditions = append(conditions, "image_url = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.ImageUrl)
	}
	if req.LinkUrl != "" && req.LinkUrl != "string" {
		conditions = append(conditions, "link_url = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.LinkUrl)
	}
	if req.Description != nil {
		conditions = append(conditions, "description = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Description)
	}

	conditions = append(conditions, "updated_at = now()")
	query := "UPDATE partners SET " + strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0"
	args = append(args, req.Id)

	res, err := r.db.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating partner: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("partner with id %s couldnt update", req.Id)
	}
	return &m.Void{}, nil
}

func (r *PartnerRepo) Delete(req *m.ById) (*m.Void, error) {
	res, err := r.db.Exec(context.Background(), `
		UPDATE partners SET deleted_at = EXTRACT(EPOCH FROM NOW()) WHERE id = $1 AND deleted_at = 0
	`, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting partner: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("partner with id %s couldnt delete", req.Id)
	}
	return &m.Void{}, nil
}

// GetAllMobile - Home ekranidagi "Hamkor kurslari" banneri (TZ 2.1-bo'lim).
func (r *PartnerRepo) GetAllMobile() (*m.PartnerMobileList, error) {
	rows, err := r.db.Query(context.Background(), `
		SELECT id, name, image_url, link_url FROM partners WHERE deleted_at = 0 ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("error while getting partners: %w", err)
	}
	defer rows.Close()

	list := m.PartnerMobileList{}
	for rows.Next() {
		p := &m.PartnerMobileRes{}
		if err := rows.Scan(&p.Id, &p.Name, &p.ImageUrl, &p.LinkUrl); err != nil {
			return nil, fmt.Errorf("error while scanning partner: %w", err)
		}
		list.Partners = append(list.Partners, p)
		list.Count++
	}
	return &list, nil
}
