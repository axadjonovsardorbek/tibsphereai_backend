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

type AboutRepo struct {
	db *pgxpool.Pool
}

func NewAboutRepo(db *pgxpool.Pool) *AboutRepo {
	return &AboutRepo{
		db: db,
	}
}

func (r *AboutRepo) Create(req *m.AboutCreateReq) (*m.ById, error) {
	id := uuid.NewString()
	query := `
	INSERT INTO abouts (
		id,
		title,
		description,
		link_url,
		order_num
	) VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(context.Background(), query,
		id,
		req.Title,
		req.Description,
		req.LinkUrl,
		req.OrderNum,
	)

	if err != nil {
		return nil, fmt.Errorf("error while creating about: %w", err)
	}

	return &m.ById{Id: id}, nil
}
func (r *AboutRepo) GetById(req *m.ById) (*m.AboutRes, error) {
	query := `
	SELECT 
		id,
		title,
		description,
		link_url,
		order_num,
		created_at::text,
		updated_at::text
	FROM
		abouts
	WHERE
		deleted_at = 0
	AND
		id = $1
	`

	row := r.db.QueryRow(context.Background(), query, req.Id)

	var titleJSON []byte
	var descriptionJSON []byte

	about := m.AboutRes{}
	err := row.Scan(
		&about.Id,
		&titleJSON,
		&descriptionJSON,
		&about.LinkUrl,
		&about.OrderNum,
		&about.CreatedAt,
		&about.UpdatedAt,
	)

	json.Unmarshal(titleJSON, &about.Title)
	json.Unmarshal(descriptionJSON, &about.Description)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("about not found with id: %s", req.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("error while getting about by id: %w", err)
	}

	return &about, nil
}
func (r *AboutRepo) GetAll(req *m.AboutGetListReq) (*m.PaginationResponse, *m.AboutList, error) {
	query := `
	SELECT 
		COUNT(id) OVER() AS total_count,
		id,
		title,
		description,
		link_url,
		order_num,
		created_at::text,
		updated_at::text
	FROM
		abouts
	WHERE
		deleted_at = 0
	`

	var args []interface{}
	if req.Title != "" && req.Title != "string" {
		query += fmt.Sprintf(" AND title::text ILIKE $%d", len(args)+1)
		args = append(args, "%"+req.Title+"%")
	}

	query += " ORDER BY order_num ASC"

	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting about: %w", err)
	}
	defer rows.Close()

	abouts := m.AboutList{}

	for rows.Next() {

		about := m.AboutRes{}

		var titleJSON []byte
		var descriptionJSON []byte
		var totalCount int

		err := rows.Scan(
			&totalCount,
			&about.Id,
			&titleJSON,
			&descriptionJSON,
			&about.LinkUrl,
			&about.OrderNum,
			&about.CreatedAt,
			&about.UpdatedAt,
		)

		json.Unmarshal(titleJSON, &about.Title)
		json.Unmarshal(descriptionJSON, &about.Description)

		if err != nil {
			return nil, nil, fmt.Errorf("error while scanning about row: %w", err)
		}

		abouts.Abouts = append(abouts.Abouts, &about)
		abouts.Count = totalCount
	}

	response := m.NewPaginationResponse(req.Filter, abouts.Count, abouts.Abouts)

	return &response, &abouts, nil
}
func (r *AboutRepo) Update(req *m.AboutUpdateReq) (*m.Void, error) {
	query := `
	UPDATE abouts
	SET
	`

	var conditions []string
	var args []interface{}

	if req.Title != "" && req.Title != "string" {
		conditions = append(conditions, " title = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Title)
	}

	conditions = append(conditions, " description = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.Description)

	conditions = append(conditions, " link_url = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.LinkUrl)

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
		return nil, fmt.Errorf("error while updating about: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("about with id %s couldnt update", req.Id)
	}
	return &m.Void{}, nil
}
func (r *AboutRepo) Delete(req *m.ById) (*m.Void, error) {
	query := `
	UPDATE abouts
	SET
		deleted_at = EXTRACT(EPOCH FROM NOW())
	WHERE
		id = $1
	AND
		deleted_at = 0
	`

	res, err := r.db.Exec(context.Background(), query, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting about: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("about with id %s couldnt delete", req.Id)
	}

	return &m.Void{}, nil
}

func (r *AboutRepo) GetByIdMobile(req *m.ByIdMobile) (*m.AboutMobileRes, error) {
	query := `
	SELECT 
		id,
		title->> $1,
		description->> $1,
		link_url,
		order_num,
		created_at::text,
		updated_at::text
	FROM
		abouts
	WHERE
		deleted_at = 0
	AND
		id = $2
	`

	row := r.db.QueryRow(context.Background(), query, req.Language, req.Id)

	about := m.AboutMobileRes{}
	err := row.Scan(
		&about.Id,
		&about.Title,
		&about.Description,
		&about.LinkUrl,
		&about.OrderNum,
		&about.CreatedAt,
		&about.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("about not found with id: %s", req.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("error while getting about by id: %w", err)
	}

	return &about, nil
}
func (r *AboutRepo) GetAllMobile(req *m.AboutGetListMobileReq) (*m.PaginationResponse, *m.AboutMobileList, error) {
	query := `
	SELECT 
		COUNT(id) OVER() AS total_count,
		id,
		title->> $1,
		description->> $1,
		link_url,
		order_num,
		created_at::text,
		updated_at::text
	FROM
		abouts
	WHERE
		deleted_at = 0
	`

	var args []interface{}
	args = append(args, req.Language)

	if req.Title != "" && req.Title != "string" {
		query += fmt.Sprintf(" AND title::text ILIKE $%d", len(args)+1)
		args = append(args, "%"+req.Title+"%")
	}

	query += " ORDER BY order_num ASC"

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting about: %w", err)
	}
	defer rows.Close()

	abouts := m.AboutMobileList{}

	for rows.Next() {

		about := m.AboutMobileRes{}

		var totalCount int

		err := rows.Scan(
			&totalCount,
			&about.Id,
			&about.Title,
			&about.Description,
			&about.LinkUrl,
			&about.OrderNum,
			&about.CreatedAt,
			&about.UpdatedAt,
		)

		if err != nil {
			return nil, nil, fmt.Errorf("error while scanning about row: %w", err)
		}

		abouts.Abouts = append(abouts.Abouts, &about)
		abouts.Count = totalCount
	}

	response := m.NewPaginationResponse(m.Pagination{Limit: abouts.Count}, abouts.Count, abouts.Abouts)

	return &response, &abouts, nil
}
