package postgres

import (
	"fmt"
	"strconv"
	"strings"
	m "tibsphereai/pkg/models"

	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TariffRepo struct {
	db *pgxpool.Pool
}

func NewTariffRepo(db *pgxpool.Pool) *TariffRepo {
	return &TariffRepo{
		db: db,
	}
}

func (r *TariffRepo) Create(req *m.TariffCreateReq) (*m.ById, error) {
	id := uuid.NewString()
	kind := req.Kind
	if kind == "" {
		kind = "subscription"
	}
	query := `
	INSERT INTO tariffs (
		id,
		name,
		kind,
		duration,
		coins,
		price,
		description
	) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(context.Background(), query,
		id,
		req.Name,
		kind,
		req.Duration,
		req.Coins,
		req.Price,
		req.Description,
	)

	if err != nil {
		return nil, fmt.Errorf("error while creating tariff: %w", err)
	}

	return &m.ById{Id: id}, nil
}
func (r *TariffRepo) GetById(req *m.ById) (*m.TariffRes, error) {
	query := `
	SELECT
		id,
		name,
		kind,
		duration,
		coins,
		price,
		description,
		created_at::text,
		updated_at::text
	FROM
		tariffs
	WHERE
		deleted_at = 0
	AND
		id = $1
	`

	row := r.db.QueryRow(context.Background(), query, req.Id)

	tariff := m.TariffRes{}
	err := row.Scan(
		&tariff.Id,
		&tariff.Name,
		&tariff.Kind,
		&tariff.Duration,
		&tariff.Coins,
		&tariff.Price,
		&tariff.Description,
		&tariff.CreatedAt,
		&tariff.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("tariff not found with id: %s", req.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("error while getting tariff by id: %w", err)
	}

	return &tariff, nil
}
func (r *TariffRepo) GetAll(req *m.TariffGetListReq) (*m.PaginationResponse, *m.TariffList, error) {
	query := `
	SELECT
		COUNT(id) OVER() AS total_count,
		id,
		name,
		kind,
		duration,
		coins,
		price,
		description,
		created_at::text,
		updated_at::text
	FROM
		tariffs
	WHERE
		deleted_at = 0
	`

	var args []interface{}
	if req.Duration != 0 {
		query += fmt.Sprintf(" AND duration = $%d", len(args)+1)
		args = append(args, req.Duration)
	}
	if req.Kind != "" {
		query += fmt.Sprintf(" AND kind = $%d", len(args)+1)
		args = append(args, req.Kind)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting tariff: %w", err)
	}
	defer rows.Close()

	tariffs := m.TariffList{}

	for rows.Next() {

		tariff := m.TariffRes{}

		var totalCount int

		err := rows.Scan(
			&totalCount,
			&tariff.Id,
			&tariff.Name,
			&tariff.Kind,
			&tariff.Duration,
			&tariff.Coins,
			&tariff.Price,
			&tariff.Description,
			&tariff.CreatedAt,
			&tariff.UpdatedAt,
		)

		if err != nil {
			return nil, nil, fmt.Errorf("error while scanning tariff row: %w", err)
		}

		tariffs.Tariffs = append(tariffs.Tariffs, &tariff)
		tariffs.Count = totalCount
	}

	response := m.NewPaginationResponse(m.Pagination{Limit: tariffs.Count}, tariffs.Count, tariffs.Tariffs)

	return &response, &tariffs, nil
}
func (r *TariffRepo) Update(req *m.TariffUpdateReq) (*m.Void, error) {
	query := `
	UPDATE tariffs
	SET
	`

	var conditions []string
	var args []interface{}

	if req.Name != "" && req.Name != "string" {
		conditions = append(conditions, " name = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Name)
	}

	if req.Kind != "" {
		conditions = append(conditions, " kind = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Kind)
	}

	// Faqat kind'ga tegishli maydon yoziladi: subscription - duration, coin_package - coins.
	// Ikkinchisi o'zgarmaydi (0 bilan ustiga yozilmaydi).
	if req.Kind == "coin_package" {
		conditions = append(conditions, " coins = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Coins)
	} else {
		conditions = append(conditions, " duration = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Duration)
	}

	conditions = append(conditions, " price = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.Price)

	conditions = append(conditions, " description = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.Description)

	if len(conditions) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	conditions = append(conditions, " updated_at = now()")
	query += strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0"

	args = append(args, req.Id)

	res, err := r.db.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating tariff: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("tariff with id %s couldnt update", req.Id)
	}
	return &m.Void{}, nil
}
func (r *TariffRepo) Delete(req *m.ById) (*m.Void, error) {
	query := `
	UPDATE tariffs
	SET
		deleted_at = EXTRACT(EPOCH FROM NOW())
	WHERE
		id = $1
	AND
		deleted_at = 0
	`

	res, err := r.db.Exec(context.Background(), query, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting tariff: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("tariff with id %s couldnt delete", req.Id)
	}

	return &m.Void{}, nil
}
