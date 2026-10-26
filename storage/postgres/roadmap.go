package postgres

import (
	"context"
	"fmt"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RoadmapRepo - AI yo'l xaritasi uchun xom ma'lumotlar va kesh.
type RoadmapRepo struct {
	db *pgxpool.Pool
}

func NewRoadmapRepo(db *pgxpool.Pool) *RoadmapRepo { return &RoadmapRepo{db: db} }

// Performance - foydalanuvchining javob berilgan, mavzusi ma'lum practice elementlari (eng yangilari birinchi).
func (r *RoadmapRepo) Performance(userId string, limit int) ([]m.PerfRow, error) {
	rows, err := r.db.Query(context.Background(), `
		SELECT pi.topic_id::text, COALESCE(pi.score, 0), COALESCE(pi.is_correct, false), pi.answered_at
		FROM practice_items pi JOIN practice_sessions ps ON ps.id = pi.session_id
		WHERE ps.user_id = $1 AND pi.answered_at IS NOT NULL AND pi.topic_id IS NOT NULL
		ORDER BY pi.answered_at DESC LIMIT $2
	`, userId, limit)
	if err != nil {
		return nil, fmt.Errorf("error while getting performance: %w", err)
	}
	defer rows.Close()
	var out []m.PerfRow
	for rows.Next() {
		var p m.PerfRow
		if err := rows.Scan(&p.TopicId, &p.Score, &p.IsCorrect, &p.AnsweredAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

// Catalog - faol bo'lim va mavzular (mobil ro'yxat bilan bir xil shartlar).
func (r *RoadmapRepo) Catalog() ([]m.CatalogTopic, error) {
	rows, err := r.db.Query(context.Background(), `
		SELECT c.id::text, c.name, COALESCE(c.order_num, 0), t.id::text, t.name, COALESCE(t.order_num, 0)
		FROM topics t JOIN categories c ON c.id = t.category_id
		WHERE t.deleted_at = 0 AND c.deleted_at = 0 AND t.is_active AND c.is_active
		ORDER BY c.order_num, t.order_num
	`)
	if err != nil {
		return nil, fmt.Errorf("error while getting catalog: %w", err)
	}
	defer rows.Close()
	var out []m.CatalogTopic
	for rows.Next() {
		var ct m.CatalogTopic
		if err := rows.Scan(&ct.CategoryId, &ct.CategoryName, &ct.CategoryOrder, &ct.TopicId, &ct.TopicName, &ct.TopicOrder); err != nil {
			return nil, err
		}
		out = append(out, ct)
	}
	return out, nil
}

func (r *RoadmapRepo) CardsByTopic(userId string) (map[string]m.TopicCards, error) {
	rows, err := r.db.Query(context.Background(), `
		SELECT topic_id::text, COUNT(*), COUNT(*) FILTER (WHERE due_at <= now())
		FROM flashcards WHERE user_id = $1 AND deleted_at = 0 AND status = 'active' AND topic_id IS NOT NULL
		GROUP BY topic_id
	`, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]m.TopicCards{}
	for rows.Next() {
		var id string
		var tc m.TopicCards
		if err := rows.Scan(&id, &tc.Total, &tc.Due); err != nil {
			return nil, err
		}
		out[id] = tc
	}
	return out, nil
}

// DueCards - foydalanuvchining hozir takrorlashga tayyor jami kartalari.
func (r *RoadmapRepo) DueCards(userId string) (int, error) {
	var n int
	err := r.db.QueryRow(context.Background(), `
		SELECT COUNT(*) FROM flashcards WHERE user_id = $1 AND deleted_at = 0 AND status = 'active' AND due_at <= now()
	`, userId).Scan(&n)
	return n, err
}

func (r *RoadmapRepo) GetCache(userId string) ([]byte, string, time.Time, error) {
	var payload []byte
	var lang string
	var at time.Time
	err := r.db.QueryRow(context.Background(), `SELECT payload, language, generated_at FROM user_roadmaps WHERE user_id = $1`, userId).Scan(&payload, &lang, &at)
	if err == pgx.ErrNoRows {
		return nil, "", time.Time{}, nil
	}
	return payload, lang, at, err
}

func (r *RoadmapRepo) SaveCache(userId string, payload []byte, lang string) error {
	_, err := r.db.Exec(context.Background(), `
		INSERT INTO user_roadmaps (user_id, payload, language, generated_at) VALUES ($1, $2, $3, now())
		ON CONFLICT (user_id) DO UPDATE SET payload = EXCLUDED.payload, language = EXCLUDED.language, generated_at = now()
	`, userId, payload, lang)
	return err
}

// --- Foydalanuvchilar fikri ---

type FeedbackRepo struct {
	db *pgxpool.Pool
}

func NewFeedbackRepo(db *pgxpool.Pool) *FeedbackRepo { return &FeedbackRepo{db: db} }

func (r *FeedbackRepo) Create(req *m.FeedbackCreateReq) (*m.ById, error) {
	var rating interface{}
	if req.Rating >= 1 && req.Rating <= 5 {
		rating = req.Rating
	}
	var id string
	err := r.db.QueryRow(context.Background(), `
		INSERT INTO user_feedback (user_id, rating, kind, message, context, app_version) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text
	`, req.UserId, rating, req.Kind, req.Message, req.Context, req.AppVersion).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("error while creating feedback: %w", err)
	}
	return &m.ById{Id: id}, nil
}

func (r *FeedbackRepo) GetAll(req *m.FeedbackListReq) (*m.PaginationResponse, []*m.FeedbackRes, *m.FeedbackSummary, error) {
	query := `
		SELECT COUNT(*) OVER(), f.id::text, f.user_id::text, COALESCE(u.name, ''), f.rating, f.kind, f.message, f.context, f.app_version, f.status, f.created_at::text
		FROM user_feedback f LEFT JOIN users u ON u.id = f.user_id WHERE true`
	var args []interface{}
	if req.Kind != "" {
		args = append(args, req.Kind)
		query += fmt.Sprintf(" AND f.kind = $%d", len(args))
	}
	if req.Status != "" {
		args = append(args, req.Status)
		query += fmt.Sprintf(" AND f.status = $%d", len(args))
	}
	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" ORDER BY f.created_at DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()
	out := []*m.FeedbackRes{}
	total := 0
	for rows.Next() {
		f := &m.FeedbackRes{}
		if err := rows.Scan(&total, &f.Id, &f.UserId, &f.UserName, &f.Rating, &f.Kind, &f.Message, &f.Context, &f.AppVersion, &f.Status, &f.CreatedAt); err != nil {
			return nil, nil, nil, err
		}
		out = append(out, f)
	}
	sum := &m.FeedbackSummary{}
	_ = r.db.QueryRow(context.Background(), `SELECT COUNT(*), COALESCE(AVG(rating), 0), COUNT(*) FILTER (WHERE status = 'new') FROM user_feedback`).Scan(&sum.Total, &sum.AvgRating, &sum.New)
	resp := m.NewPaginationResponse(req.Filter, total, out)
	return &resp, out, sum, nil
}

func (r *FeedbackRepo) SetStatus(id, status string) error {
	res, err := r.db.Exec(context.Background(), `UPDATE user_feedback SET status = $2 WHERE id = $1`, id, status)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errs.ErrNotFound
	}
	return nil
}
