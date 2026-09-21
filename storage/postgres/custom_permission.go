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

type CustomPermissionRepo struct {
	db *pgxpool.Pool
}

func NewCustomPermissionRepo(db *pgxpool.Pool) *CustomPermissionRepo {
	return &CustomPermissionRepo{db: db}
}

func (r *CustomPermissionRepo) Create(req *m.CustomPermissionCreateReq) (*m.ById, error) {
	id := uuid.NewString()
	attrs, _ := json.Marshal(req.Attributes)

	var parentId interface{}
	if req.ParentId != "" && req.ParentId != "string" {
		parentId = req.ParentId
	}

	_, err := r.db.Exec(context.Background(), `
		INSERT INTO custom_permissions (id, title, parent_id, attributes, order_num)
		VALUES ($1, $2, $3, $4, $5)
	`, id, req.Title, parentId, string(attrs), req.OrderNum)
	if err != nil {
		return nil, fmt.Errorf("error while creating custom permission: %w", err)
	}
	return &m.ById{Id: id}, nil
}

func (r *CustomPermissionRepo) Update(req *m.CustomPermissionUpdateReq) (*m.Void, error) {
	var conditions []string
	var args []interface{}

	if req.Title != "" && req.Title != "string" {
		conditions = append(conditions, "title = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Title)
	}
	if req.Attributes != nil {
		attrs, _ := json.Marshal(req.Attributes)
		conditions = append(conditions, "attributes = $"+strconv.Itoa(len(args)+1))
		args = append(args, string(attrs))
	}
	conditions = append(conditions, "order_num = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.OrderNum)
	conditions = append(conditions, "updated_at = now()")

	query := "UPDATE custom_permissions SET " + strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0"
	args = append(args, req.Id)

	res, err := r.db.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating custom permission: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("custom permission with id %s couldnt update", req.Id)
	}
	return &m.Void{}, nil
}

func (r *CustomPermissionRepo) Delete(req *m.ById) (*m.Void, error) {
	res, err := r.db.Exec(context.Background(), `
		UPDATE custom_permissions SET deleted_at = EXTRACT(EPOCH FROM NOW()) WHERE id = $1 AND deleted_at = 0
	`, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting custom permission: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("custom permission with id %s couldnt delete", req.Id)
	}
	return &m.Void{}, nil
}

func scanPermissionRows(rows pgx.Rows) (*m.CustomPermissionList, error) {
	list := m.CustomPermissionList{}
	for rows.Next() {
		p := &m.CustomPermissionRes{}
		var attrsJSON []byte
		if err := rows.Scan(&p.Id, &p.Title, &p.ParentId, &attrsJSON, &p.OrderNum, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error while scanning custom permission: %w", err)
		}
		json.Unmarshal(attrsJSON, &p.Attributes)
		list.Modules = append(list.Modules, p)
		list.Count++
	}
	return &list, nil
}

const customPermissionSelect = `
	SELECT id, title, parent_id, attributes, order_num, created_at::text, updated_at::text
	FROM custom_permissions
`

func (r *CustomPermissionRepo) GetRoots() (*m.CustomPermissionList, error) {
	rows, err := r.db.Query(context.Background(), customPermissionSelect+`
		WHERE parent_id IS NULL AND deleted_at = 0 ORDER BY order_num ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("error while getting root custom permissions: %w", err)
	}
	defer rows.Close()
	return scanPermissionRows(rows)
}

func (r *CustomPermissionRepo) GetChildren(parentId string) (*m.CustomPermissionList, error) {
	rows, err := r.db.Query(context.Background(), customPermissionSelect+`
		WHERE parent_id = $1 AND deleted_at = 0 ORDER BY order_num ASC
	`, parentId)
	if err != nil {
		return nil, fmt.Errorf("error while getting child custom permissions: %w", err)
	}
	defer rows.Close()
	return scanPermissionRows(rows)
}

func (r *CustomPermissionRepo) HasChildren(id string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM custom_permissions WHERE parent_id = $1 AND deleted_at = 0)
	`, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("error while checking children: %w", err)
	}
	return exists, nil
}

// GetAccesses - root (parentId bo'sh) yoki child (parentId berilgan) modullarni, shu rol
// uchun hozirgi ruxsatlar bilan birga qaytaradi (role_accesses LEFT JOIN - hali
// belgilanmagan bo'lsa hammasi false, U-Code'dagi "No" ekvivalenti).
func (r *CustomPermissionRepo) GetAccesses(roleId, parentId string) (*m.RoleAccessList, error) {
	query := `
	SELECT cp.id, cp.title, cp.attributes,
		EXISTS(SELECT 1 FROM custom_permissions c2 WHERE c2.parent_id = cp.id AND c2.deleted_at = 0) AS has_children,
		COALESCE(ra.can_read, false), COALESCE(ra.can_write, false),
		COALESCE(ra.can_update, false), COALESCE(ra.can_delete, false)
	FROM custom_permissions cp
	LEFT JOIN role_accesses ra ON ra.custom_permission_id = cp.id AND ra.role_id = $1
	WHERE cp.deleted_at = 0 AND `

	var args []interface{}
	args = append(args, roleId)
	if parentId != "" && parentId != "string" {
		query += "cp.parent_id = $2"
		args = append(args, parentId)
	} else {
		query += "cp.parent_id IS NULL"
	}
	query += " ORDER BY cp.order_num ASC"

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while getting accesses: %w", err)
	}
	defer rows.Close()

	list := m.RoleAccessList{}
	for rows.Next() {
		item := m.RoleAccessItem{}
		var attrsJSON []byte
		if err := rows.Scan(&item.CustomPermissionId, &item.Title, &attrsJSON, &item.HasChildren,
			&item.CanRead, &item.CanWrite, &item.CanUpdate, &item.CanDelete); err != nil {
			return nil, fmt.Errorf("error while scanning access: %w", err)
		}
		json.Unmarshal(attrsJSON, &item.Attributes)
		list.Items = append(list.Items, item)
	}
	return &list, nil
}

func (r *CustomPermissionRepo) UpsertAccess(req *m.RoleAccessUpsertReq) (*m.Void, error) {
	_, err := r.db.Exec(context.Background(), `
		INSERT INTO role_accesses (id, role_id, custom_permission_id, can_read, can_write, can_update, can_delete)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (role_id, custom_permission_id)
		DO UPDATE SET can_read = $4, can_write = $5, can_update = $6, can_delete = $7, updated_at = now()
	`, uuid.NewString(), req.RoleId, req.CustomPermissionId, req.CanRead, req.CanWrite, req.CanUpdate, req.CanDelete)
	if err != nil {
		return nil, fmt.Errorf("error while upserting access: %w", err)
	}
	return &m.Void{}, nil
}
