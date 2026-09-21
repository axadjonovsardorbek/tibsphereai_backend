package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"tibsphereai/api/token"
	m "tibsphereai/pkg/models"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AdminRepo struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

func NewAdminRepo(db *pgxpool.Pool, rdb *redis.Client) *AdminRepo {
	return &AdminRepo{
		db:  db,
		rdb: rdb,
	}
}

func (r *AdminRepo) Login(req *m.LoginReq) (*m.TokenRes, error) {

	var id string
	var password string

	query := `
	SELECT
		id,
		password
	FROM
		admins
	WHERE
		login = $1 AND deleted_at = 0
	`

	row := r.db.QueryRow(context.Background(), query, req.Login)

	err := row.Scan(
		&id,
		&password,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("invalid phone: %w", err)
	}

	if err != nil {
		return nil, fmt.Errorf("error while login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(password), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	tkn := token.GenerateJWTToken(id, "admin", r.rdb)

	return &m.TokenRes{
		AccessToken:  tkn.AccessToken,
		RefreshToken: tkn.RefreshToken,
		Role:         "admin",
		Id:           id,
	}, nil
}

func (r *AdminRepo) GetById(req *m.ById) (*m.AdminRes, error) {
	query := `
	SELECT
		a.id,
		a.login,
		a.role_id,
		r.name,
		a.created_at::text,
		a.updated_at::text
	FROM
		admins a
	LEFT JOIN roles r ON r.id = a.role_id
	WHERE
		a.id = $1 AND a.deleted_at = 0
	`
	admin := m.AdminRes{}

	err := r.db.QueryRow(context.Background(), query, req.Id).Scan(
		&admin.Id,
		&admin.Login,
		&admin.RoleId,
		&admin.RoleName,
		&admin.CreatedAt,
		&admin.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting admin by id: %w", err)
	}

	return &admin, nil
}

// Create - yangi admin/xodim qo'shish (superadmin panel: adminlar ro'yxati).
func (r *AdminRepo) Create(req *m.AdminCreateReq) (*m.ById, error) {
	if req.Login == "" || req.Password == "" {
		return nil, fmt.Errorf("login and password are required")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error while hashing password: %w", err)
	}

	var roleId interface{}
	if req.RoleId != "" && req.RoleId != "string" {
		roleId = req.RoleId
	}

	id := uuid.NewString()
	_, err = r.db.Exec(context.Background(), `
		INSERT INTO admins (id, login, password, role_id) VALUES ($1, $2, $3, $4)
	`, id, req.Login, string(hashed), roleId)
	if err != nil {
		return nil, fmt.Errorf("error while creating admin: %w", err)
	}

	return &m.ById{Id: id}, nil
}

// GetAll - adminlar ro'yxati (login bo'yicha qidiruv + pagination).
func (r *AdminRepo) GetAll(req *m.AdminGetListReq) (*m.PaginationResponse, *m.AdminList, error) {
	query := `
	SELECT COUNT(a.id) OVER() AS total_count, a.id, a.login, a.role_id, r.name, a.created_at::text, a.updated_at::text
	FROM admins a
	LEFT JOIN roles r ON r.id = a.role_id
	WHERE a.deleted_at = 0
	`
	var args []interface{}
	if req.Login != "" && req.Login != "string" {
		query += fmt.Sprintf(" AND a.login ILIKE $%d", len(args)+1)
		args = append(args, "%"+req.Login+"%")
	}
	query += " ORDER BY a.created_at DESC"
	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting admins: %w", err)
	}
	defer rows.Close()

	list := m.AdminList{}
	for rows.Next() {
		a := &m.AdminRes{}
		var total int
		if err := rows.Scan(&total, &a.Id, &a.Login, &a.RoleId, &a.RoleName, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, nil, fmt.Errorf("error while scanning admin: %w", err)
		}
		list.Admins = append(list.Admins, a)
		list.Count = total
	}

	response := m.NewPaginationResponse(req.Filter, list.Count, list.Admins)
	return &response, &list, nil
}

// Update - login/role_id va (ixtiyoriy) parolni yangilaydi. Bo'sh maydonlar o'zgartirilmaydi.
func (r *AdminRepo) Update(req *m.AdminUpdateReq) (*m.Void, error) {
	var conditions []string
	var args []interface{}

	if req.Login != "" && req.Login != "string" {
		conditions = append(conditions, "login = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Login)
	}
	if req.RoleId != "" && req.RoleId != "string" {
		conditions = append(conditions, "role_id = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.RoleId)
	}
	if req.Password != "" && req.Password != "string" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("error while hashing password: %w", err)
		}
		conditions = append(conditions, "password = $"+strconv.Itoa(len(args)+1))
		args = append(args, string(hashed))
	}
	if len(conditions) == 0 {
		return &m.Void{}, nil
	}
	conditions = append(conditions, "updated_at = now()")

	query := "UPDATE admins SET " + strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0"
	args = append(args, req.Id)

	res, err := r.db.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating admin: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("admin with id %s not found", req.Id)
	}
	return &m.Void{}, nil
}

// Delete - soft delete (boshqa jadvallar bilan bir xil pattern).
func (r *AdminRepo) Delete(req *m.ById) (*m.Void, error) {
	res, err := r.db.Exec(context.Background(), `
		UPDATE admins SET deleted_at = EXTRACT(EPOCH FROM NOW())
		WHERE id = $1 AND deleted_at = 0
	`, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting admin: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("admin with id %s not found", req.Id)
	}
	return &m.Void{}, nil
}
