package postgres

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	m "tibsphereai/pkg/models"

	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FaqRepo struct {
	db *pgxpool.Pool
}

func NewFaqRepo(db *pgxpool.Pool) *FaqRepo {
	return &FaqRepo{
		db: db,
	}
}

func (r *FaqRepo) Create(req *m.FaqCreateReq) (*m.ById, error) {
	id := uuid.NewString()
	query := `
	INSERT INTO faqs (
		id,
		question,
		answer,
		order_num
	) VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.Exec(context.Background(), query,
		id,
		req.Question,
		req.Answer,
		req.OrderNum,
	)

	if err != nil {
		return nil, fmt.Errorf("error while creating faq: %w", err)
	}

	return &m.ById{Id: id}, nil
}
func (r *FaqRepo) GetById(req *m.ById) (*m.FaqRes, error) {
	query := `
	SELECT 
		id,
		question,
		answer,
		order_num,
		created_at::text,
		updated_at::text
	FROM
		faqs
	WHERE
		deleted_at = 0
	AND
		id = $1
	`

	row := r.db.QueryRow(context.Background(), query, req.Id)

	var questionJSON []byte
	var answerJSON []byte

	faq := m.FaqRes{}
	err := row.Scan(
		&faq.Id,
		&questionJSON,
		&answerJSON,
		&faq.OrderNum,
		&faq.CreatedAt,
		&faq.UpdatedAt,
	)

	json.Unmarshal(questionJSON, &faq.Question)
	json.Unmarshal(answerJSON, &faq.Answer)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("faq not found with id: %s", req.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("error while getting faq by id: %w", err)
	}

	return &faq, nil
}
func (r *FaqRepo) GetAll(req *m.FaqGetListReq) (*m.PaginationResponse, *m.FaqList, error) {
	query := `
	SELECT 
		COUNT(id) OVER() AS total_count,
		id,
		question,
		answer,
		order_num,
		created_at::text,
		updated_at::text
	FROM
		faqs
	WHERE
		deleted_at = 0
	`

	var args []interface{}
	if req.Question != "" && req.Question != "string" {
		query += fmt.Sprintf(" AND question::text ILIKE $%d", len(args)+1)
		args = append(args, "%"+req.Question+"%")
	}

	query += " ORDER BY order_num ASC"

	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting faq: %w", err)
	}
	defer rows.Close()

	faqs := m.FaqList{}

	for rows.Next() {

		faq := m.FaqRes{}

		var questionJSON []byte
		var answerJSON []byte
		var totalCount int

		err := rows.Scan(
			&totalCount,
			&faq.Id,
			&questionJSON,
			&answerJSON,
			&faq.OrderNum,
			&faq.CreatedAt,
			&faq.UpdatedAt,
		)

		json.Unmarshal(questionJSON, &faq.Question)
		json.Unmarshal(answerJSON, &faq.Answer)

		if err != nil {
			return nil, nil, fmt.Errorf("error while scanning faq row: %w", err)
		}

		faqs.Faqs = append(faqs.Faqs, &faq)
		faqs.Count = totalCount
	}

	response := m.NewPaginationResponse(req.Filter, faqs.Count, faqs.Faqs)

	return &response, &faqs, nil
}
func (r *FaqRepo) Update(req *m.FaqUpdateReq) (*m.Void, error) {
	query := `
	UPDATE faqs
	SET
	`

	var conditions []string
	var args []interface{}

	if req.Question != "" && req.Question != "string" {
		conditions = append(conditions, " question = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Question)
	}

	conditions = append(conditions, " answer = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.Answer)

	conditions = append(conditions, " order_num = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.OrderNum)

	if len(conditions) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	conditions = append(conditions, " updated_at = now()")
	query += strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0"

	args = append(args, req.Id)

	res, err := r.db.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating faq: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("faq with id %s couldnt update", req.Id)
	}
	return &m.Void{}, nil
}
func (r *FaqRepo) Delete(req *m.ById) (*m.Void, error) {
	query := `
	UPDATE faqs
	SET
		deleted_at = EXTRACT(EPOCH FROM NOW())
	WHERE
		id = $1
	AND
		deleted_at = 0
	`

	res, err := r.db.Exec(context.Background(), query, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting faq: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("faq with id %s couldnt delete", req.Id)
	}

	return &m.Void{}, nil
}

func (r *FaqRepo) GetByIdMobile(req *m.ByIdMobile) (*m.FaqMobileRes, error) {
	query := `
	SELECT 
		id,
		question->> $1,
		answer->> $1,
		order_num,
		created_at::text,
		updated_at::text
	FROM
		faqs
	WHERE
		deleted_at = 0
	AND
		id = $2
	`

	row := r.db.QueryRow(context.Background(), query, req.Language, req.Id)

	faq := m.FaqMobileRes{}

	err := row.Scan(
		&faq.Id,
		&faq.Question,
		&faq.Answer,
		&faq.OrderNum,
		&faq.CreatedAt,
		&faq.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("faq not found with id: %s", req.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("error while getting faq by id: %w", err)
	}

	return &faq, nil
}
func (r *FaqRepo) GetAllMobile(req *m.FaqGetListMobileReq) (*m.PaginationResponse, *m.FaqMobileList, error) {
	query := `
	SELECT 
		COUNT(id) OVER() AS total_count,
		id,
		question->> $1,
		answer->> $1,
		order_num,
		created_at::text,
		updated_at::text
	FROM
		faqs
	WHERE
		deleted_at = 0
	`

	var args []interface{}
	args = append(args, req.Language)

	if req.Question != "" && req.Question != "string" {
		query += fmt.Sprintf(" AND question::text ILIKE $%d", len(args)+1)
		args = append(args, "%"+req.Question+"%")
	}

	query += " ORDER BY order_num ASC"

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting faq: %w", err)
	}
	defer rows.Close()

	faqs := m.FaqMobileList{}

	for rows.Next() {

		faq := m.FaqMobileRes{}

		var totalCount int

		err := rows.Scan(
			&totalCount,
			&faq.Id,
			&faq.Question,
			&faq.Answer,
			&faq.OrderNum,
			&faq.CreatedAt,
			&faq.UpdatedAt,
		)

		if err != nil {
			return nil, nil, fmt.Errorf("error while scanning faq row: %w", err)
		}

		faqs.Faqs = append(faqs.Faqs, &faq)
		faqs.Count = totalCount
	}

	response := m.NewPaginationResponse(m.Pagination{Limit: faqs.Count}, faqs.Count, faqs.Faqs)

	return &response, &faqs, nil
}
