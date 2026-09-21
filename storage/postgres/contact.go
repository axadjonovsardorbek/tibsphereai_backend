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

type ContactRepo struct {
	db *pgxpool.Pool
}

func NewContactRepo(db *pgxpool.Pool) *ContactRepo {
	return &ContactRepo{
		db: db,
	}
}

func (r *ContactRepo) Create(req *m.ContactCreateReq) (*m.ById, error) {
	id := uuid.NewString()
	query := `
	INSERT INTO contacts (
		id,
		name,
		phone_number,
		link_url
	) VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.Exec(context.Background(), query,
		id,
		req.Name,
		req.PhoneNumber,
		req.LinkUrl,
	)

	if err != nil {
		return nil, fmt.Errorf("error while creating contact: %w", err)
	}

	return &m.ById{Id: id}, nil
}
func (r *ContactRepo) GetById(req *m.ById) (*m.ContactRes, error) {
	query := `
	SELECT 
		id,
		name,
		phone_number,
		link_url,
		created_at::text,
		updated_at::text
	FROM
		contacts
	WHERE
		deleted_at = 0
	AND
		id = $1
	`

	row := r.db.QueryRow(context.Background(), query, req.Id)

	contact := m.ContactRes{}
	err := row.Scan(
		&contact.Id,
		&contact.Name,
		&contact.PhoneNumber,
		&contact.LinkUrl,
		&contact.CreatedAt,
		&contact.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("contact not found with id: %s", req.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("error while getting contact by id: %w", err)
	}

	return &contact, nil
}
func (r *ContactRepo) GetAll(req *m.ContactGetListReq) (*m.PaginationResponse, *m.ContactList, error) {
	query := `
	SELECT 
		COUNT(id) OVER() AS total_count,
		id,
		name,
		phone_number,
		link_url,
		created_at::text,
		updated_at::text
	FROM
		contacts
	WHERE
		deleted_at = 0
	`

	var args []interface{}
	if req.Name != "" && req.Name != "string" {
		query += fmt.Sprintf(" AND name ILIKE $%d", len(args)+1)
		args = append(args, "%"+req.Name+"%")
	}
	if req.PhoneNumber != "" && req.PhoneNumber != "string" {
		query += fmt.Sprintf(" AND phone_number ILIKE $%d", len(args)+1)
		args = append(args, "%"+req.PhoneNumber+"%")
	}

	query += " ORDER BY created_at DESC"

	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting contact: %w", err)
	}
	defer rows.Close()

	contacts := m.ContactList{}

	for rows.Next() {

		contact := m.ContactRes{}

		var totalCount int

		err := rows.Scan(
			&totalCount,
			&contact.Id,
			&contact.Name,
			&contact.PhoneNumber,
			&contact.LinkUrl,
			&contact.CreatedAt,
			&contact.UpdatedAt,
		)

		if err != nil {
			return nil, nil, fmt.Errorf("error while scanning c row: %w", err)
		}

		contacts.Contacts = append(contacts.Contacts, &contact)
		contacts.Count = totalCount
	}

	response := m.NewPaginationResponse(req.Filter, contacts.Count, contacts.Contacts)

	return &response, &contacts, nil
}
func (r *ContactRepo) Update(req *m.ContactUpdateReq) (*m.Void, error) {
	query := `
	UPDATE contacts
	SET
	`

	var conditions []string
	var args []interface{}

	if req.Name != "" && req.Name != "string" {
		conditions = append(conditions, " name = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Name)
	}

	conditions = append(conditions, " phone_number = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.PhoneNumber)

	conditions = append(conditions, " link_url = $"+strconv.Itoa(len(args)+1))
	args = append(args, req.LinkUrl)

	if len(conditions) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	conditions = append(conditions, " updated_at = now()")
	query += strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0"

	args = append(args, req.Id)

	res, err := r.db.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating contact: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("contact with id %s couldnt update", req.Id)
	}
	return &m.Void{}, nil
}
func (r *ContactRepo) Delete(req *m.ById) (*m.Void, error) {
	query := `
	UPDATE contacts
	SET
		deleted_at = EXTRACT(EPOCH FROM NOW())
	WHERE
		id = $1
	AND
		deleted_at = 0
	`

	res, err := r.db.Exec(context.Background(), query, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting contact: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("contact with id %s couldnt delete", req.Id)
	}

	return &m.Void{}, nil
}
