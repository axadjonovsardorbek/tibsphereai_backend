package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PracticeRepo - AI practice sessiyalari va elementlari (new-feature.md 5-bo'lim).
type PracticeRepo struct {
	db *pgxpool.Pool
}

func NewPracticeRepo(db *pgxpool.Pool) *PracticeRepo { return &PracticeRepo{db: db} }

func (r *PracticeRepo) CreateSession(s *m.PracticeSessionRow) (string, error) {
	var id string
	topics := s.TopicIds
	if topics == nil {
		topics = []string{}
	}
	var cat interface{}
	if s.CategoryId != "" {
		cat = s.CategoryId
	}
	err := r.db.QueryRow(context.Background(), `
		INSERT INTO practice_sessions (user_id, mode, category_id, topic_ids, difficulty, total_items, language, free_topic, coin_billed)
		VALUES ($1, $2, $3, $4::uuid[], $5, $6, $7, $8, $9) RETURNING id::text
	`, s.UserId, s.Mode, cat, topics, s.Difficulty, s.TotalItems, s.Language, s.FreeTopic, s.CoinBilled).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("error while creating practice session: %w", err)
	}
	return id, nil
}

const sessionCols = `id::text, user_id::text, mode, COALESCE(category_id::text, ''), topic_ids::text[], difficulty, total_items,
	status, score, xp_earned, language, started_at::text, finished_at::text,
	free_topic, coin_billed, coins_charged, prompt_tokens, completion_tokens, flashcards_status, flashcards_created`

func scanSession(row pgx.Row) (*m.PracticeSessionRow, error) {
	s := &m.PracticeSessionRow{}
	err := row.Scan(&s.Id, &s.UserId, &s.Mode, &s.CategoryId, &s.TopicIds, &s.Difficulty, &s.TotalItems,
		&s.Status, &s.Score, &s.XPEarned, &s.Language, &s.StartedAt, &s.FinishedAt,
		&s.FreeTopic, &s.CoinBilled, &s.CoinsCharged, &s.PromptTokens, &s.CompletionTokens, &s.FlashcardsStatus, &s.FlashcardsCreated)
	return s, err
}

func (r *PracticeRepo) GetSession(id string) (*m.PracticeSessionRow, error) {
	s, err := scanSession(r.db.QueryRow(context.Background(), `SELECT `+sessionCols+` FROM practice_sessions WHERE id = $1`, id))
	if err == pgx.ErrNoRows {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting practice session: %w", err)
	}
	return s, nil
}

func (r *PracticeRepo) ListSessions(req *m.PracticeListReq) (*m.PaginationResponse, []*m.PracticeSessionRow, error) {
	rows, err := r.db.Query(context.Background(), `
		SELECT COUNT(*) OVER(), `+sessionCols+` FROM practice_sessions
		WHERE user_id = $1 ORDER BY started_at DESC LIMIT $2 OFFSET $3
	`, req.UserId, req.Filter.Limit, req.Filter.Page)
	if err != nil {
		return nil, nil, fmt.Errorf("error while listing practice sessions: %w", err)
	}
	defer rows.Close()
	var out []*m.PracticeSessionRow
	total := 0
	for rows.Next() {
		s := &m.PracticeSessionRow{}
		if err := rows.Scan(&total, &s.Id, &s.UserId, &s.Mode, &s.CategoryId, &s.TopicIds, &s.Difficulty, &s.TotalItems,
			&s.Status, &s.Score, &s.XPEarned, &s.Language, &s.StartedAt, &s.FinishedAt,
			&s.FreeTopic, &s.CoinBilled, &s.CoinsCharged, &s.PromptTokens, &s.CompletionTokens, &s.FlashcardsStatus, &s.FlashcardsCreated); err != nil {
			return nil, nil, err
		}
		out = append(out, s)
	}
	resp := m.NewPaginationResponse(req.Filter, total, out)
	return &resp, out, nil
}

func (r *PracticeRepo) SessionCounts(sessionId string) (generated, answered, correct int, err error) {
	err = r.db.QueryRow(context.Background(), `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE answered_at IS NOT NULL), COUNT(*) FILTER (WHERE is_correct)
		FROM practice_items WHERE session_id = $1
	`, sessionId).Scan(&generated, &answered, &correct)
	return
}

func (r *PracticeRepo) AddItem(it *m.PracticeItemRow) (string, error) {
	srcs, _ := json.Marshal(it.Sources)
	if it.Sources == nil {
		srcs = []byte("[]")
	}
	chunkIds := it.SourceChunkIds
	if chunkIds == nil {
		chunkIds = []string{}
	}
	var id string
	err := r.db.QueryRow(context.Background(), `
		INSERT INTO practice_items (session_id, idx, kind, payload, sources, source_chunk_ids, summary, prompt_version, model, tokens_used)
		VALUES ($1, (SELECT COALESCE(MAX(idx), -1) + 1 FROM practice_items WHERE session_id = $1), $2, $3, $4, $5::uuid[], $6, $7, $8, $9)
		RETURNING id::text
	`, it.SessionId, it.Kind, it.Payload, srcs, chunkIds, it.Summary, it.PromptVersion, it.Model, it.TokensUsed).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("error while adding practice item: %w", err)
	}
	return id, nil
}

const itemCols = `i.id::text, i.session_id::text, s.user_id::text, i.idx, i.kind, i.payload, i.sources, i.source_chunk_ids::text[],
	i.summary, i.user_answer, i.evaluation, i.is_correct, i.score, i.reported, i.prompt_version, i.model, i.tokens_used`

func scanItem(row pgx.Row) (*m.PracticeItemRow, error) {
	it := &m.PracticeItemRow{}
	var srcs []byte
	err := row.Scan(&it.Id, &it.SessionId, &it.UserId, &it.Idx, &it.Kind, &it.Payload, &srcs, &it.SourceChunkIds,
		&it.Summary, &it.UserAnswer, &it.Evaluation, &it.IsCorrect, &it.Score, &it.Reported, &it.PromptVersion, &it.Model, &it.TokensUsed)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(srcs, &it.Sources)
	return it, nil
}

func (r *PracticeRepo) GetItem(id string) (*m.PracticeItemRow, error) {
	it, err := scanItem(r.db.QueryRow(context.Background(),
		`SELECT `+itemCols+` FROM practice_items i JOIN practice_sessions s ON s.id = i.session_id WHERE i.id = $1`, id))
	if err == pgx.ErrNoRows {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting practice item: %w", err)
	}
	return it, nil
}

func (r *PracticeRepo) GetItems(sessionId string) ([]*m.PracticeItemRow, error) {
	rows, err := r.db.Query(context.Background(),
		`SELECT `+itemCols+` FROM practice_items i JOIN practice_sessions s ON s.id = i.session_id WHERE i.session_id = $1 ORDER BY i.idx`, sessionId)
	if err != nil {
		return nil, fmt.Errorf("error while getting practice items: %w", err)
	}
	defer rows.Close()
	var out []*m.PracticeItemRow
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, nil
}

func (r *PracticeRepo) PendingItem(sessionId string) (*m.PracticeItemRow, error) {
	it, err := scanItem(r.db.QueryRow(context.Background(),
		`SELECT `+itemCols+` FROM practice_items i JOIN practice_sessions s ON s.id = i.session_id
		 WHERE i.session_id = $1 AND i.answered_at IS NULL ORDER BY i.idx DESC LIMIT 1`, sessionId))
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return it, err
}

func (r *PracticeRepo) Summaries(sessionId string) ([]string, error) {
	rows, err := r.db.Query(context.Background(),
		`SELECT summary FROM practice_items WHERE session_id = $1 AND summary <> '' ORDER BY idx`, sessionId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

func (r *PracticeRepo) SetAnswer(itemId string, answer, eval []byte, isCorrect bool, score int, tokens int) error {
	res, err := r.db.Exec(context.Background(), `
		UPDATE practice_items SET user_answer = $2, evaluation = $3, is_correct = $4, score = $5,
			tokens_used = tokens_used + $6, answered_at = now()
		WHERE id = $1 AND answered_at IS NULL
	`, itemId, answer, eval, isCorrect, score, tokens)
	if err != nil {
		return fmt.Errorf("error while saving practice answer: %w", err)
	}
	if res.RowsAffected() == 0 {
		return errs.ErrConflict // allaqachon javob berilgan
	}
	return nil
}

func (r *PracticeRepo) AddTokens(sessionId string, prompt, completion int) error {
	_, err := r.db.Exec(context.Background(),
		`UPDATE practice_sessions SET prompt_tokens = prompt_tokens + $1, completion_tokens = completion_tokens + $2 WHERE id = $3`,
		prompt, completion, sessionId)
	return err
}

func (r *PracticeRepo) Finish(sessionId, status string, score, xp int) error {
	res, err := r.db.Exec(context.Background(), `
		UPDATE practice_sessions SET status = $2, score = $3, xp_earned = $4, finished_at = now()
		WHERE id = $1 AND status = 'active'
	`, sessionId, status, score, xp)
	if err != nil {
		return fmt.Errorf("error while finishing practice session: %w", err)
	}
	if res.RowsAffected() == 0 {
		return errs.ErrSessionNotActive
	}
	return nil
}

func (r *PracticeRepo) SetBilling(sessionId string, billed bool) error {
	_, err := r.db.Exec(context.Background(), `UPDATE practice_sessions SET coin_billed = $2 WHERE id = $1`, sessionId, billed)
	return err
}

func (r *PracticeRepo) AddCharged(sessionId string, coins int) error {
	_, err := r.db.Exec(context.Background(), `UPDATE practice_sessions SET coins_charged = coins_charged + $2 WHERE id = $1`, sessionId, coins)
	return err
}

func (r *PracticeRepo) SetFlashcards(sessionId, status string, created int) error {
	_, err := r.db.Exec(context.Background(),
		`UPDATE practice_sessions SET flashcards_status = $2, flashcards_created = $3 WHERE id = $1`, sessionId, status, created)
	return err
}

func (r *PracticeRepo) Report(itemId, reason string) error {
	_, err := r.db.Exec(context.Background(),
		`UPDATE practice_items SET reported = TRUE, report_reason = $2 WHERE id = $1`, itemId, reason)
	return err
}

func (r *PracticeRepo) Stats() (*m.PracticeStats, error) {
	ctx := context.Background()
	st := &m.PracticeStats{ByCategory: []m.PracticeStatByGroup{}, ByTopic: []m.PracticeStatByGroup{}, RecentReports: []m.PracticeReportRow{}}
	if err := r.db.QueryRow(ctx, `
		SELECT (SELECT COUNT(*) FROM practice_sessions),
		       (SELECT COUNT(*) FROM practice_sessions WHERE status = 'active'),
		       (SELECT COUNT(*) FROM practice_items),
		       (SELECT COUNT(*) FROM practice_items WHERE answered_at IS NOT NULL),
		       COALESCE((SELECT AVG(CASE WHEN is_correct THEN 1.0 ELSE 0.0 END) FROM practice_items WHERE answered_at IS NOT NULL), 0),
		       COALESCE((SELECT SUM(prompt_tokens + completion_tokens) FROM practice_sessions), 0),
		       (SELECT COUNT(*) FROM practice_items WHERE reported)
	`).Scan(&st.Sessions, &st.ActiveSession, &st.Items, &st.Answered, &st.CorrectRate, &st.Tokens, &st.Reports); err != nil {
		return nil, fmt.Errorf("error while getting practice stats: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT c.id::text, c.name, COUNT(DISTINCT ps.id), COUNT(pi.id)
		FROM practice_sessions ps JOIN categories c ON c.id = ps.category_id -- erkin mavzu sessiyalari (category_id NULL) kirmaydi
		LEFT JOIN practice_items pi ON pi.session_id = ps.id
		GROUP BY c.id, c.name ORDER BY COUNT(DISTINCT ps.id) DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		g := m.PracticeStatByGroup{}
		if err := rows.Scan(&g.Id, &g.Name, &g.Sessions, &g.Items); err != nil {
			rows.Close()
			return nil, err
		}
		st.ByCategory = append(st.ByCategory, g)
	}
	rows.Close()

	rows, err = r.db.Query(ctx, `
		SELECT t.id::text, t.name, COUNT(DISTINCT ps.id), COUNT(pi.id)
		FROM practice_sessions ps CROSS JOIN LATERAL unnest(ps.topic_ids) AS tid
		JOIN topics t ON t.id = tid
		LEFT JOIN practice_items pi ON pi.session_id = ps.id
		GROUP BY t.id, t.name ORDER BY COUNT(DISTINCT ps.id) DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		g := m.PracticeStatByGroup{}
		if err := rows.Scan(&g.Id, &g.Name, &g.Sessions, &g.Items); err != nil {
			rows.Close()
			return nil, err
		}
		st.ByTopic = append(st.ByTopic, g)
	}
	rows.Close()

	rows, err = r.db.Query(ctx, `
		SELECT id::text, session_id::text, kind, report_reason, created_at::text
		FROM practice_items WHERE reported ORDER BY created_at DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		rp := m.PracticeReportRow{}
		if err := rows.Scan(&rp.ItemId, &rp.SessionId, &rp.Kind, &rp.Reason, &rp.CreatedAt); err != nil {
			return nil, err
		}
		st.RecentReports = append(st.RecentReports, rp)
	}
	return st, nil
}
