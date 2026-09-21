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

// TopicRepo - Bo'lim (categories) ichidagi Mavzular (TZ 6-bo'lim: Bo'lim -> Mavzu -> Keys).
type TopicRepo struct {
	db *pgxpool.Pool
}

func NewTopicRepo(db *pgxpool.Pool) *TopicRepo {
	return &TopicRepo{db: db}
}

func (r *TopicRepo) Create(req *m.TopicCreateReq) (*m.ById, error) {
	id := uuid.NewString()
	_, err := r.db.Exec(context.Background(), `
		INSERT INTO topics (id, category_id, name, order_num) VALUES ($1, $2, $3, $4)
	`, id, req.CategoryId, req.Name, req.OrderNum)
	if err != nil {
		return nil, fmt.Errorf("error while creating topic: %w", err)
	}
	return &m.ById{Id: id}, nil
}

func (r *TopicRepo) GetById(req *m.ById) (*m.TopicRes, error) {
	row := r.db.QueryRow(context.Background(), `
		SELECT id, category_id, name, order_num, created_at::text, updated_at::text
		FROM topics WHERE id = $1 AND deleted_at = 0
	`, req.Id)

	var nameJSON []byte
	t := m.TopicRes{}
	err := row.Scan(&t.Id, &t.CategoryId, &nameJSON, &t.OrderNum, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting topic by id: %w", err)
	}
	json.Unmarshal(nameJSON, &t.Name)
	return &t, nil
}

func (r *TopicRepo) GetAll(req *m.TopicGetListReq) (*m.PaginationResponse, *m.TopicList, error) {
	query := `
	SELECT COUNT(id) OVER() AS total_count, id, category_id, name, order_num, created_at::text, updated_at::text
	FROM topics WHERE deleted_at = 0
	`
	var args []interface{}
	if req.CategoryId != "" && req.CategoryId != "string" {
		args = append(args, req.CategoryId)
		query += fmt.Sprintf(" AND category_id = $%d", len(args))
	}
	query += " ORDER BY order_num ASC"
	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting topics: %w", err)
	}
	defer rows.Close()

	list := m.TopicList{}
	for rows.Next() {
		t := &m.TopicRes{}
		var nameJSON []byte
		var total int
		if err := rows.Scan(&total, &t.Id, &t.CategoryId, &nameJSON, &t.OrderNum, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, nil, fmt.Errorf("error while scanning topic: %w", err)
		}
		json.Unmarshal(nameJSON, &t.Name)
		list.Topics = append(list.Topics, t)
		list.Count = total
	}

	response := m.NewPaginationResponse(req.Filter, list.Count, list.Topics)
	return &response, &list, nil
}

func (r *TopicRepo) Update(req *m.TopicUpdateReq) (*m.Void, error) {
	var conditions []string
	var args []interface{}

	if req.CategoryId != "" && req.CategoryId != "string" {
		conditions = append(conditions, "category_id = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.CategoryId)
	}
	if req.Name != nil {
		conditions = append(conditions, "name = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Name)
	}
	conditions = append(conditions, "order_num = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.OrderNum)

	conditions = append(conditions, "updated_at = now()")
	query := "UPDATE topics SET " + strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0"
	args = append(args, req.Id)

	res, err := r.db.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating topic: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("topic with id %s couldnt update", req.Id)
	}
	return &m.Void{}, nil
}

func (r *TopicRepo) Delete(req *m.ById) (*m.Void, error) {
	res, err := r.db.Exec(context.Background(), `
		UPDATE topics SET deleted_at = EXTRACT(EPOCH FROM NOW()) WHERE id = $1 AND deleted_at = 0
	`, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting topic: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("topic with id %s couldnt delete", req.Id)
	}
	return &m.Void{}, nil
}

func (r *TopicRepo) GetAllMobile(req *m.TopicGetListMobileReq) (*m.PaginationResponse, *m.TopicMobileList, error) {
	query := `
	SELECT COUNT(id) OVER() AS total_count, id, category_id, name ->> $1, order_num
	FROM topics WHERE deleted_at = 0
	`
	args := []interface{}{req.Language}
	if req.CategoryId != "" && req.CategoryId != "string" {
		args = append(args, req.CategoryId)
		query += fmt.Sprintf(" AND category_id = $%d", len(args))
	}
	query += " ORDER BY order_num ASC"
	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting topics: %w", err)
	}
	defer rows.Close()

	list := m.TopicMobileList{}
	for rows.Next() {
		t := &m.TopicMobileRes{}
		var total int
		if err := rows.Scan(&total, &t.Id, &t.CategoryId, &t.Name, &t.OrderNum); err != nil {
			return nil, nil, fmt.Errorf("error while scanning topic: %w", err)
		}
		list.Topics = append(list.Topics, t)
		list.Count = total
	}

	response := m.NewPaginationResponse(req.Filter, list.Count, list.Topics)
	return &response, &list, nil
}
