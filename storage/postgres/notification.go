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

type NotificationRepo struct {
	db *pgxpool.Pool
}

func NewNotificationRepo(db *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{
		db: db,
	}
}

func (r *NotificationRepo) Create(req *m.NotificationCreateReq) (*m.ById, error) {
	ctx := context.Background()

	tz, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if p := recover(); p != nil {
			tz.Rollback(ctx)
			err = fmt.Errorf("panic occurred: %v", p)
		} else if err != nil {
			tz.Rollback(ctx)
		} else {
			err = tz.Commit(ctx)
		}
	}()

	id := uuid.NewString()

	query := `
	INSERT INTO master_notifications (
		id,
		title,
		message,
		type
	) VALUES ($1, $2, $3, $4)
	`

	_, err = tz.Exec(ctx, query,
		id,
		req.Title,
		req.Message,
		req.Type,
	)

	if err != nil {
		return nil, fmt.Errorf("error while creating notification: %w", err)
	}

	var user_ids []string

	if req.UserId != "" && req.UserId != "string" {
		user_ids = append(user_ids, req.UserId)
	} else if req.Type == "all" {
		user_query := `
		SELECT
			id
		FROM
			users
		WHERE
			deleted_at = 0
		`

		row, err2 := tz.Query(ctx, user_query)
		if err2 != nil {
			err = err2
			return nil, fmt.Errorf("error while getting users: %w", err2)
		}
		defer row.Close()

		for row.Next() {
			var user_id string
			if scanErr := row.Scan(&user_id); scanErr != nil {
				err = scanErr
				return nil, fmt.Errorf("error while scanning user ids: %w", scanErr)
			}
			user_ids = append(user_ids, user_id)
		}
	}

	if len(user_ids) > 0 {
		valueStrings := []string{}
		valueArgs := []interface{}{}
		for i, user_id := range user_ids {
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3))
			valueArgs = append(valueArgs, uuid.NewString())
			valueArgs = append(valueArgs, id)
			valueArgs = append(valueArgs, user_id)
		}
		user_insert_query := fmt.Sprintf(`
		INSERT INTO user_notifications (
			id,
			master_notification_id,
			user_id
		) VALUES %s
		`, strings.Join(valueStrings, ","))
		_, err = tz.Exec(ctx, user_insert_query, valueArgs...)
		if err != nil {
			return nil, fmt.Errorf("error while inserting user notifications: %w", err)
		}
	}

	return &m.ById{Id: id}, nil
}
func (r *NotificationRepo) GetById(req *m.ById) (*m.MasterNotificationRes, error) {

	query := `
	SELECT
		n.id,
		n.title,
		n.message,
		n.type,
		n.created_at::text,
		n.updated_at::text
	FROM
		master_notifications n
	WHERE
		n.id = $1
	`

	row := r.db.QueryRow(context.Background(), query, req.Id)

	var titleJSON []byte
	var messageJSON []byte

	notification := m.MasterNotificationRes{}
	err := row.Scan(
		&notification.Id,
		&titleJSON,
		&messageJSON,
		&notification.Type,
		&notification.CreatedAt,
		&notification.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("notification not found with id: %s", req.Id)
	}

	if err != nil {
		return nil, fmt.Errorf("error while getting notification by id: %w", err)
	}

	json.Unmarshal(titleJSON, &notification.Title)
	json.Unmarshal(messageJSON, &notification.Message)

	return &notification, nil
}
func (r *NotificationRepo) GetAll(req *m.NotificationGetListReq) (*m.PaginationResponse, *m.MasterNotificationList, error) {
	query := `
	SELECT
		COUNT(n.id) OVER() AS total_count,
		n.id,
		n.title,
		n.message,
		n.type,
		n.created_at::text,
		n.updated_at::text
	FROM
		master_notifications n
	WHERE
		1=1
	`

	var args []interface{}
	if req.Title != "" && req.Title != "string" {
		query += fmt.Sprintf(" AND n.title::text ILIKE $%d", len(args)+1)
		args = append(args, "%"+req.Title+"%")
	}

	query += " ORDER BY n.created_at DESC"

	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting notification: %w", err)
	}
	defer rows.Close()

	notifications := m.MasterNotificationList{}

	for rows.Next() {

		notification := m.MasterNotificationRes{}
		var titleJSON []byte
		var messageJSON []byte
		var totalCount int

		err := rows.Scan(
			&totalCount,
			&notification.Id,
			&titleJSON,
			&messageJSON,
			&notification.Type,
			&notification.CreatedAt,
			&notification.UpdatedAt,
		)

		if err != nil {
			return nil, nil, fmt.Errorf("error while scanning notification row: %w", err)
		}

		json.Unmarshal(titleJSON, &notification.Title)
		json.Unmarshal(messageJSON, &notification.Message)

		notifications.Notifications = append(notifications.Notifications, &notification)
		notifications.Count = totalCount
	}

	response := m.NewPaginationResponse(req.Filter, notifications.Count, notifications.Notifications)

	return &response, &notifications, nil
}

func (r *NotificationRepo) Update(req *m.MasterNotificationUpdateReq) (*m.Void, error) {
	query := `
	UPDATE master_notifications
	SET
	`

	var conditions []string
	var args []interface{}

	if req.Title != "" && req.Title != "string" {
		conditions = append(conditions, " title = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Title)
	}
	if req.Message != "" && req.Message != "string" {
		conditions = append(conditions, " message = $"+strconv.Itoa(len(args)+1))
		args = append(args, req.Message)
	}

	if len(conditions) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	conditions = append(conditions, " updated_at = now()")
	query += strings.Join(conditions, ", ")
	query += " WHERE id = $" + strconv.Itoa(len(args)+1)

	args = append(args, req.Id)

	res, err := r.db.Exec(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating notification: %w", err)
	}

	if res.RowsAffected() == 0 {
		return nil, fmt.Errorf("notification with id %s couldnt update", req.Id)
	}
	return &m.Void{}, nil
}

func (r *NotificationRepo) Delete(req *m.ById) (*m.Void, error) {
	ctx := context.Background()

	query := `
	DELETE FROM master_notifications
	WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting notification: %w", err)
	}

	user_query := `
	DELETE FROM user_notifications
	WHERE master_notification_id = $1
	`
	_, err = r.db.Exec(ctx, user_query, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting user notifications: %w", err)
	}

	return &m.Void{}, nil
}

func (r *NotificationRepo) UserNotifications(req *m.NotificationGetListReq) (*m.PaginationResponse, *m.UserNotificationList, error) {
	query := `
	SELECT
		COUNT(n.id) OVER() AS total_count,
		n.id,
		n.is_read,
		m.id,
		m.title->> $1,
		m.message->> $1,
		n.created_at::text,
		n.updated_at::text
	FROM
		user_notifications n
	JOIN
		master_notifications m ON n.master_notification_id = m.id
	WHERE
		n.user_id = $2
	`

	var args []interface{}
	args = append(args, req.Language, req.UserId)

	if req.IsRead != "" && req.IsRead != "string" {
		if req.IsRead == "true" {
			query += fmt.Sprintf(" AND n.is_read = $%d", len(args)+1)
			args = append(args, true)
		} else if req.IsRead == "false" {
			query += fmt.Sprintf(" AND n.is_read = $%d", len(args)+1)
			args = append(args, false)
		}
	}

	query += " ORDER BY n.created_at DESC"

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting notification: %w", err)
	}
	defer rows.Close()

	notifications := m.UserNotificationList{}

	for rows.Next() {

		notification := m.UserNotificationRes{}
		var totalCount int
		var master_notification_id string

		err := rows.Scan(
			&totalCount,
			&notification.Id,
			&notification.IsRead,
			&master_notification_id,
			&notification.Title,
			&notification.Message,
			&notification.CreatedAt,
			&notification.UpdatedAt,
		)

		if err != nil {
			return nil, nil, fmt.Errorf("error while scanning notification row: %w", err)
		}

		notifications.Notifications = append(notifications.Notifications, &notification)
		notifications.Count = totalCount
	}

	response := m.NewPaginationResponse(m.Pagination{Limit: notifications.Count}, notifications.Count, notifications.Notifications)

	return &response, &notifications, nil
}
func (r *NotificationRepo) MarkAsRead(req *m.NotificationMarkReadReq) (*m.Void, error) {

	query := `
	UPDATE user_notifications
	SET
		is_read = true,
		updated_at = now()
	WHERE id = $1 AND user_id = $2
	`

	_, err := r.db.Exec(context.Background(), query, req.Id, req.UserId)
	if err != nil {
		return nil, fmt.Errorf("error while marking notification as read: %w", err)
	}

	return &m.Void{}, nil
}
