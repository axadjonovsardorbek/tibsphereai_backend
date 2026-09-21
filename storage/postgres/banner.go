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

type BannerRepo struct {
	db *pgxpool.Pool
}

func NewBannerRepo(db *pgxpool.Pool) *BannerRepo {
	return &BannerRepo{
		db: db,
	}
}

func (r *BannerRepo) Create(req *m.BannerCreateReq) (*m.ById, error) {
	id := uuid.NewString()
	query := `
	INSERT INTO banners (
		id,
		title,
		description,
		image_url,
		link_url,
		order_num
	) VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(context.Background(), query,
		id,
		req.Title,
		req.Description,
		req.ImageUrl,
		req.LinkUrl,
		req.OrderNum,
	)

	if err != nil {
		return nil, fmt.Errorf("error while creating banner: %w", err)
	}

	return &m.ById{Id: id}, nil
}
func (r *BannerRepo) GetById(req *m.ById) (*m.BannerRes, error) {
	query := `
	SELECT 
		id,
		title,
		description,
		image_url,
		link_url,
		order_num,
		created_at::text,
		updated_at::text
	FROM
		banners
	WHERE
		deleted_at = 0
	AND
		id = $1
	`

	row := r.db.QueryRow(context.Background(), query, req.Id)

	var titleJSON []byte
	var descriptionJSON []byte
	var imageUrlJSON []byte

	banner := m.BannerRes{}
	err := row.Scan(
		&banner.Id,
		&titleJSON,
		&descriptionJSON,
		&imageUrlJSON,
		&banner.LinkUrl,
		&banner.OrderNum,
		&banner.CreatedAt,
		&banner.UpdatedAt,
	)

	json.Unmarshal(titleJSON, &banner.Title)
	json.Unmarshal(descriptionJSON, &banner.Description)
	json.Unmarshal(imageUrlJSON, &banner.ImageUrl)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("banner not found with id: %s", req.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("error while getting banner by id: %w", err)
	}

	return &banner, nil
}
func (r *BannerRepo) GetAll(req *m.BannerGetListReq) (*m.PaginationResponse, *m.BannerList, error) {
	query := `
	SELECT 
		COUNT(id) OVER() AS total_count,
		id,
		title,
		description,
		image_url,
		link_url,
		order_num,
		created_at::text,
		updated_at::text
	FROM
		banners
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
		return nil, nil, fmt.Errorf("error while getting banner: %w", err)
	}
	defer rows.Close()

	banners := m.BannerList{}

	for rows.Next() {

		banner := m.BannerRes{}

		var titleJSON []byte
		var descriptionJSON []byte
		var imageUrlJSON []byte
		var totalCount int

		err := rows.Scan(
			&totalCount,
			&banner.Id,
			&titleJSON,
			&descriptionJSON,
			&imageUrlJSON,
			&banner.LinkUrl,
			&banner.OrderNum,
			&banner.CreatedAt,
			&banner.UpdatedAt,
		)

		json.Unmarshal(titleJSON, &banner.Title)
		json.Unmarshal(descriptionJSON, &banner.Description)
		json.Unmarshal(imageUrlJSON, &banner.ImageUrl)

		if err != nil {
			return nil, nil, fmt.Errorf("error while scanning banner row: %w", err)
		}

		banners.Banners = append(banners.Banners, &banner)
		banners.Count = totalCount
	}

	response := m.NewPaginationResponse(req.Filter, banners.Count, banners.Banners)

	return &response, &banners, nil
}
func (r *BannerRepo) Update(req *m.BannerUpdateReq) (*m.Void, error) {
	query := `
	UPDATE banners
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

	conditions = append(conditions, " image_url = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.ImageUrl)

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
		return nil, fmt.Errorf("error while updating banner: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("banner with id %s couldnt update", req.Id)
	}
	return &m.Void{}, nil
}
func (r *BannerRepo) Delete(req *m.ById) (*m.Void, error) {
	query := `
	UPDATE banners
	SET
		deleted_at = EXTRACT(EPOCH FROM NOW())
	WHERE
		id = $1
	AND
		deleted_at = 0
	`

	res, err := r.db.Exec(context.Background(), query, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting banner: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("banner with id %s couldnt delete", req.Id)
	}

	return &m.Void{}, nil
}

func (r *BannerRepo) GetByIdMobile(req *m.ByIdMobile) (*m.BannerMobileRes, error) {
	query := `
	SELECT 
		id,
		title->> $1,
		description->> $1,
		image_url->> $1,
		link_url,
		order_num,
		created_at::text,
		updated_at::text
	FROM
		banners
	WHERE
		deleted_at = 0
	AND
		id = $2
	`

	row := r.db.QueryRow(context.Background(), query, req.Language, req.Id)

	banner := m.BannerMobileRes{}
	err := row.Scan(
		&banner.Id,
		&banner.Title,
		&banner.Description,
		&banner.ImageUrl,
		&banner.LinkUrl,
		&banner.OrderNum,
		&banner.CreatedAt,
		&banner.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("banner not found with id: %s", req.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("error while getting banner by id: %w", err)
	}

	return &banner, nil
}
func (r *BannerRepo) GetAllMobile(req *m.BannerGetListMobileReq) (*m.PaginationResponse, *m.BannerMobileList, error) {
	query := `
	SELECT 
		COUNT(id) OVER() AS total_count,
		id,
		title->> $1,
		description->> $1,
		image_url->> $1,
		link_url,
		order_num,
		created_at::text,
		updated_at::text
	FROM
		banners
	WHERE
		deleted_at = 0
	`

	var args []interface{}
	args = append(args, req.Language)

	query += " ORDER BY order_num ASC"

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting banner: %w", err)
	}
	defer rows.Close()

	banners := m.BannerMobileList{}

	for rows.Next() {

		banner := m.BannerMobileRes{}

		var totalCount int

		err := rows.Scan(
			&totalCount,
			&banner.Id,
			&banner.Title,
			&banner.Description,
			&banner.ImageUrl,
			&banner.LinkUrl,
			&banner.OrderNum,
			&banner.CreatedAt,
			&banner.UpdatedAt,
		)

		if err != nil {
			return nil, nil, fmt.Errorf("error while scanning banner row: %w", err)
		}

		banners.Banners = append(banners.Banners, &banner)
		banners.Count = totalCount
	}

	response := m.NewPaginationResponse(m.Pagination{Limit: banners.Count}, banners.Count, banners.Banners)

	return &response, &banners, nil
}
