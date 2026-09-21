package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	m "tibsphereai/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoleRepo struct {
	db *pgxpool.Pool
}

func NewRoleRepo(db *pgxpool.Pool) *RoleRepo {
	return &RoleRepo{db: db}
}

func (r *RoleRepo) Create(req *m.RoleCreateReq) (*m.ById, error) {
	id := uuid.NewString()
	_, err := r.db.Exec(context.Background(), `
		INSERT INTO roles (id, name, slug) VALUES ($1, $2, $3)
	`, id, req.Name, req.Slug)
	if err != nil {
		return nil, fmt.Errorf("error while creating role: %w", err)
	}
	return &m.ById{Id: id}, nil
}

func (r *RoleRepo) GetById(req *m.ById) (*m.RoleRes, error) {
	row := r.db.QueryRow(context.Background(), `
		SELECT id, name, slug, is_system, created_at::text, updated_at::text
		FROM roles WHERE id = $1 AND deleted_at = 0
	`, req.Id)

	role := m.RoleRes{}
	err := row.Scan(&role.Id, &role.Name, &role.Slug, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("role not found with id: %s", req.Id)
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting role by id: %w", err)
	}
	return &role, nil
}

func (r *RoleRepo) GetAll(req *m.RoleGetListReq) (*m.PaginationResponse, *m.RoleList, error) {
	query := `
	SELECT COUNT(id) OVER() AS total_count, id, name, slug, is_system, created_at::text, updated_at::text
	FROM roles WHERE deleted_at = 0
	`
	var args []interface{}
	if req.Name != "" && req.Name != "string" {
		query += fmt.Sprintf(" AND name ILIKE $%d", len(args)+1)
		args = append(args, "%"+req.Name+"%")
	}
	query += " ORDER BY is_system DESC, name ASC"
	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting roles: %w", err)
	}
	defer rows.Close()

	list := m.RoleList{}
	for rows.Next() {
		role := &m.RoleRes{}
		var total int
		if err := rows.Scan(&total, &role.Id, &role.Name, &role.Slug, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, nil, fmt.Errorf("error while scanning role: %w", err)
		}
		list.Roles = append(list.Roles, role)
		list.Count = total
	}

	response := m.NewPaginationResponse(req.Filter, list.Count, list.Roles)
	return &response, &list, nil
}

func (r *RoleRepo) Update(req *m.RoleUpdateReq) (*m.Void, error) {
	var conditions []string
	var args []interface{}

	if req.Name != "" && req.Name != "string" {
		conditions = append(conditions, "name = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Name)
	}
	if req.Slug != "" && req.Slug != "string" {
		conditions = append(conditions, "slug = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Slug)
	}
	if len(conditions) == 0 {
		return &m.Void{}, nil
	}
	conditions = append(conditions, "updated_at = now()")

	// is_system rollar (superadmin) qayta nomlanmaydi - Casbin/policy shu slug'ga tayanadi.
	query := "UPDATE roles SET " + strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0 AND is_system = false"
	args = append(args, req.Id)

	res, err := r.db.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating role: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("role with id %s not found or is a protected system role", req.Id)
	}
	return &m.Void{}, nil
}

func (r *RoleRepo) Delete(req *m.ById) (*m.Void, error) {
	res, err := r.db.Exec(context.Background(), `
		UPDATE roles SET deleted_at = EXTRACT(EPOCH FROM NOW())
		WHERE id = $1 AND deleted_at = 0 AND is_system = false
	`, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting role: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("role with id %s not found or is a protected system role", req.Id)
	}
	return &m.Void{}, nil
}
