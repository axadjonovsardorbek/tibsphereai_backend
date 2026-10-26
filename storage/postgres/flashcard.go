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

// FlashcardRepo - AI fleshkartalar (decks, cards, reviews).
type FlashcardRepo struct {
	db *pgxpool.Pool
}

func NewFlashcardRepo(db *pgxpool.Pool) *FlashcardRepo { return &FlashcardRepo{db: db} }

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (r *FlashcardRepo) CreateDeck(d *m.FlashcardDeckRow) (string, error) {
	var id string
	err := r.db.QueryRow(context.Background(), `
		INSERT INTO flashcard_decks (user_id, title, kind, category_id, topic_id, free_topic, language)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id::text
	`, d.UserId, d.Title, d.Kind, nullStr(d.CategoryId), nullStr(d.TopicId), d.FreeTopic, d.Language).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("error while creating flashcard deck: %w", err)
	}
	return id, nil
}

const deckCols = `d.id::text, d.user_id::text, d.title, d.kind, COALESCE(d.category_id::text, ''), COALESCE(d.topic_id::text, ''),
	d.free_topic, d.language, d.created_at,
	(SELECT COUNT(*) FROM flashcards c WHERE c.deck_id = d.id AND c.deleted_at = 0),
	(SELECT COUNT(*) FROM flashcards c WHERE c.deck_id = d.id AND c.deleted_at = 0 AND c.status = 'active' AND c.due_at <= now())`

func scanDeck(row pgx.Row) (*m.FlashcardDeckRow, error) {
	d := &m.FlashcardDeckRow{}
	err := row.Scan(&d.Id, &d.UserId, &d.Title, &d.Kind, &d.CategoryId, &d.TopicId, &d.FreeTopic, &d.Language, &d.CreatedAt, &d.CardCount, &d.DueCount)
	return d, err
}

func (r *FlashcardRepo) GetDeck(id string) (*m.FlashcardDeckRow, error) {
	d, err := scanDeck(r.db.QueryRow(context.Background(), `SELECT `+deckCols+` FROM flashcard_decks d WHERE d.id = $1 AND d.deleted_at = 0`, id))
	if err == pgx.ErrNoRows {
		return nil, errs.ErrNotFound
	}
	return d, err
}

func (r *FlashcardRepo) ListDecks(userId string) ([]*m.FlashcardDeckRow, error) {
	rows, err := r.db.Query(context.Background(), `SELECT `+deckCols+` FROM flashcard_decks d WHERE d.user_id = $1 AND d.deleted_at = 0 ORDER BY d.created_at DESC`, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*m.FlashcardDeckRow
	for rows.Next() {
		d, err := scanDeck(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func (r *FlashcardRepo) DeleteDeck(userId, id string) error {
	ctx := context.Background()
	res, err := r.db.Exec(ctx, `UPDATE flashcard_decks SET deleted_at = EXTRACT(EPOCH FROM NOW()) WHERE id = $1 AND user_id = $2 AND deleted_at = 0`, id, userId)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errs.ErrNotFound
	}
	_, err = r.db.Exec(ctx, `UPDATE flashcards SET deleted_at = EXTRACT(EPOCH FROM NOW()) WHERE deck_id = $1 AND user_id = $2 AND deleted_at = 0`, id, userId)
	return err
}

func (r *FlashcardRepo) AddCards(cards []*m.FlashcardRow) (int, error) {
	n := 0
	for _, c := range cards {
		srcs, _ := json.Marshal(c.Sources)
		if c.Sources == nil {
			srcs = []byte("[]")
		}
		res, err := r.db.Exec(context.Background(), `
			INSERT INTO flashcards (user_id, deck_id, front, back, source_refs, origin_session_id, category_id, topic_id, language)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9
			WHERE NOT EXISTS (SELECT 1 FROM flashcards x WHERE x.user_id = $1 AND x.deleted_at = 0 AND lower(x.front) = lower($3))
		`, c.UserId, nullStr(c.DeckId), c.Front, c.Back, srcs, nullStr(c.OriginSessionId), nullStr(c.CategoryId), nullStr(c.TopicId), c.Language)
		if err != nil {
			return n, fmt.Errorf("error while adding flashcard: %w", err)
		}
		n += int(res.RowsAffected())
	}
	return n, nil
}

const cardCols = `c.id::text, c.user_id::text, COALESCE(c.deck_id::text, ''), c.front, c.back, c.source_refs,
	COALESCE(c.origin_session_id::text, ''), COALESCE(c.category_id::text, ''), COALESCE(c.topic_id::text, ''), c.language,
	c.ease, c.interval_days, c.repetitions, c.lapses, c.due_at, c.last_reviewed_at, c.status, c.created_at`

func scanCard(row pgx.Row) (*m.FlashcardRow, error) {
	c := &m.FlashcardRow{}
	var srcs []byte
	var ease float32
	err := row.Scan(&c.Id, &c.UserId, &c.DeckId, &c.Front, &c.Back, &srcs, &c.OriginSessionId, &c.CategoryId, &c.TopicId, &c.Language,
		&ease, &c.IntervalDays, &c.Repetitions, &c.Lapses, &c.DueAt, &c.LastReviewedAt, &c.Status, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	c.Ease = float64(ease)
	_ = json.Unmarshal(srcs, &c.Sources)
	return c, nil
}

func (r *FlashcardRepo) GetCard(id string) (*m.FlashcardRow, error) {
	c, err := scanCard(r.db.QueryRow(context.Background(), `SELECT `+cardCols+` FROM flashcards c WHERE c.id = $1 AND c.deleted_at = 0`, id))
	if err == pgx.ErrNoRows {
		return nil, errs.ErrNotFound
	}
	return c, err
}

func (r *FlashcardRepo) ListCards(userId, deckId string, limit, offset int) ([]*m.FlashcardRow, int, error) {
	rows, err := r.db.Query(context.Background(), `
		SELECT COUNT(*) OVER(), `+cardCols+` FROM flashcards c
		WHERE c.user_id = $1 AND c.deleted_at = 0 AND ($2 = '' OR c.deck_id::text = $2)
		ORDER BY c.created_at DESC LIMIT $3 OFFSET $4
	`, userId, deckId, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*m.FlashcardRow
	total := 0
	for rows.Next() {
		c := &m.FlashcardRow{}
		var srcs []byte
		var ease float32
		if err := rows.Scan(&total, &c.Id, &c.UserId, &c.DeckId, &c.Front, &c.Back, &srcs, &c.OriginSessionId, &c.CategoryId, &c.TopicId, &c.Language,
			&ease, &c.IntervalDays, &c.Repetitions, &c.Lapses, &c.DueAt, &c.LastReviewedAt, &c.Status, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		c.Ease = float64(ease)
		_ = json.Unmarshal(srcs, &c.Sources)
		out = append(out, c)
	}
	return out, total, nil
}

func (r *FlashcardRepo) UpdateCard(userId, id string, front, back, status *string) error {
	res, err := r.db.Exec(context.Background(), `
		UPDATE flashcards SET front = COALESCE($3, front), back = COALESCE($4, back), status = COALESCE($5, status)
		WHERE id = $1 AND user_id = $2 AND deleted_at = 0
	`, id, userId, front, back, status)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errs.ErrNotFound
	}
	return nil
}

func (r *FlashcardRepo) DeleteCard(userId, id string) error {
	res, err := r.db.Exec(context.Background(), `UPDATE flashcards SET deleted_at = EXTRACT(EPOCH FROM NOW()) WHERE id = $1 AND user_id = $2 AND deleted_at = 0`, id, userId)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return errs.ErrNotFound
	}
	return nil
}

func (r *FlashcardRepo) Due(userId, deckId string, limit int) ([]*m.FlashcardRow, int, error) {
	// Avval muddati o'tgan (takrorlanayotgan) kartalar, keyin hali ko'rilmagan yangilari.
	rows, err := r.db.Query(context.Background(), `
		SELECT COUNT(*) OVER(), `+cardCols+` FROM flashcards c
		WHERE c.user_id = $1 AND c.deleted_at = 0 AND c.status = 'active' AND c.due_at <= now()
		  AND ($2 = '' OR c.deck_id::text = $2)
		ORDER BY (c.last_reviewed_at IS NULL), c.due_at ASC LIMIT $3
	`, userId, deckId, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*m.FlashcardRow
	total := 0
	for rows.Next() {
		c := &m.FlashcardRow{}
		var srcs []byte
		var ease float32
		if err := rows.Scan(&total, &c.Id, &c.UserId, &c.DeckId, &c.Front, &c.Back, &srcs, &c.OriginSessionId, &c.CategoryId, &c.TopicId, &c.Language,
			&ease, &c.IntervalDays, &c.Repetitions, &c.Lapses, &c.DueAt, &c.LastReviewedAt, &c.Status, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		c.Ease = float64(ease)
		_ = json.Unmarshal(srcs, &c.Sources)
		out = append(out, c)
	}
	return out, total, nil
}

func (r *FlashcardRepo) SaveReview(c *m.FlashcardRow, rating string, intervalBefore int) error {
	ctx := context.Background()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE flashcards SET ease = $2, interval_days = $3, repetitions = $4, lapses = $5, due_at = $6, last_reviewed_at = now()
		WHERE id = $1
	`, c.Id, float32(c.Ease), c.IntervalDays, c.Repetitions, c.Lapses, c.DueAt); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO flashcard_reviews (card_id, user_id, rating, interval_before, interval_after) VALUES ($1, $2, $3, $4, $5)
	`, c.Id, c.UserId, rating, intervalBefore, c.IntervalDays); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

const endOfTashkentDay = `((date_trunc('day', now() AT TIME ZONE 'Asia/Tashkent') + interval '1 day') AT TIME ZONE 'Asia/Tashkent')`
const startOfTashkentDay = `(date_trunc('day', now() AT TIME ZONE 'Asia/Tashkent') AT TIME ZONE 'Asia/Tashkent')`

func (r *FlashcardRepo) Stats(userId string) (*m.FlashcardStatsRes, error) {
	st := &m.FlashcardStatsRes{}
	err := r.db.QueryRow(context.Background(), `
		SELECT
		  COUNT(*) FILTER (WHERE true),
		  COUNT(*) FILTER (WHERE status = 'active' AND due_at <= now()),
		  COUNT(*) FILTER (WHERE status = 'active' AND due_at < `+endOfTashkentDay+`),
		  COUNT(*) FILTER (WHERE last_reviewed_at IS NULL),
		  COUNT(*) FILTER (WHERE interval_days >= 21),
		  (SELECT COUNT(*) FROM flashcard_decks WHERE user_id = $1 AND deleted_at = 0)
		FROM flashcards WHERE user_id = $1 AND deleted_at = 0
	`, userId).Scan(&st.Total, &st.DueNow, &st.DueToday, &st.New, &st.Learned, &st.Decks)
	if err != nil {
		return nil, fmt.Errorf("error while getting flashcard stats: %w", err)
	}
	_ = r.db.QueryRow(context.Background(), `
		SELECT COUNT(*) FILTER (WHERE reviewed_at >= `+startOfTashkentDay+`),
		       COALESCE(AVG(CASE WHEN rating IN ('good','easy') THEN 1.0 ELSE 0.0 END) FILTER (WHERE reviewed_at >= now() - interval '30 days'), 0)
		FROM flashcard_reviews WHERE user_id = $1
	`, userId).Scan(&st.ReviewsToday, &st.Retention30d)
	_ = r.db.QueryRow(context.Background(), `SELECT streak_count, longest_streak FROM users WHERE id = $1`, userId).Scan(&st.StreakCount, &st.LongestStreak)
	return st, nil
}

func (r *FlashcardRepo) ExistingFronts(userId string, limit int) ([]string, error) {
	rows, err := r.db.Query(context.Background(), `SELECT front FROM flashcards WHERE user_id = $1 AND deleted_at = 0 ORDER BY created_at DESC LIMIT $2`, userId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func (r *FlashcardRepo) AdminStats() (*m.FlashcardAdminStats, error) {
	ctx := context.Background()
	st := &m.FlashcardAdminStats{WeakTopics: []m.FlashcardWeakTopic{}}
	err := r.db.QueryRow(ctx, `
		SELECT
		  (SELECT COUNT(*) FROM flashcards WHERE deleted_at = 0),
		  (SELECT COUNT(*) FROM flashcard_decks WHERE deleted_at = 0),
		  (SELECT COUNT(DISTINCT user_id) FROM flashcards WHERE deleted_at = 0),
		  (SELECT COUNT(*) FROM flashcards WHERE deleted_at = 0 AND created_at >= `+startOfTashkentDay+`),
		  (SELECT COUNT(*) FROM flashcard_reviews WHERE reviewed_at >= `+startOfTashkentDay+`),
		  (SELECT COUNT(DISTINCT user_id) FROM flashcard_reviews WHERE reviewed_at >= `+startOfTashkentDay+`),
		  COALESCE((SELECT AVG(CASE WHEN rating IN ('good','easy') THEN 1.0 ELSE 0.0 END) FROM flashcard_reviews WHERE reviewed_at >= now() - interval '30 days'), 0),
		  COALESCE((SELECT AVG(CASE WHEN interval_days >= 21 THEN 1.0 ELSE 0.0 END) FROM flashcards WHERE deleted_at = 0 AND status = 'active'), 0)
	`).Scan(&st.TotalCards, &st.TotalDecks, &st.UsersWithCards, &st.GeneratedToday, &st.ReviewsToday, &st.ActiveReviewers, &st.Retention30d, &st.MasteryRate)
	if err != nil {
		return nil, fmt.Errorf("error while getting flashcard admin stats: %w", err)
	}

	// Eng ko'p xato qilinayotgan mavzular: practice javoblaridagi xato ulushi + shu mavzudan yaratilgan kartalar.
	rows, err := r.db.Query(ctx, `
		SELECT t.id::text, t.name,
		       COUNT(*) FILTER (WHERE pi.answered_at IS NOT NULL) AS answered,
		       COALESCE(AVG(CASE WHEN pi.is_correct THEN 0.0 ELSE 1.0 END) FILTER (WHERE pi.answered_at IS NOT NULL), 0) AS wrong_rate,
		       (SELECT COUNT(*) FROM flashcards f WHERE f.topic_id = t.id AND f.deleted_at = 0 AND f.origin_session_id IS NOT NULL) AS cards
		FROM practice_items pi JOIN topics t ON t.id = pi.topic_id
		GROUP BY t.id, t.name
		HAVING COUNT(*) FILTER (WHERE pi.answered_at IS NOT NULL) >= 3
		ORDER BY wrong_rate DESC, answered DESC LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		w := m.FlashcardWeakTopic{}
		if err := rows.Scan(&w.TopicId, &w.Name, &w.Answered, &w.WrongRate, &w.CardCount); err != nil {
			return nil, err
		}
		st.WeakTopics = append(st.WeakTopics, w)
	}
	return st, nil
}
