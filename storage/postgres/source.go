package postgres

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SourceRepo - AI practice manbalari (case_sources jadvali + source_links + source_chunks).
type SourceRepo struct {
	db *pgxpool.Pool
}

func NewSourceRepo(db *pgxpool.Pool) *SourceRepo { return &SourceRepo{db: db} }

func (r *SourceRepo) Create(req *m.SourceCreateReq, content, status string) (*m.ById, error) {
	ctx := context.Background()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	id := uuid.NewString()
	if _, err := tx.Exec(ctx, `
		INSERT INTO case_sources (id, name, content, type, language, author, url, file_url, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, id, req.Title, content, req.Type, req.Language, req.Author, req.Url, req.FileUrl, status); err != nil {
		return nil, fmt.Errorf("error while creating source: %w", err)
	}
	if err := insertLinks(ctx, tx, id, req.CategoryIds, req.TopicIds); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &m.ById{Id: id}, nil
}

func insertLinks(ctx context.Context, tx pgx.Tx, sourceId string, categoryIds, topicIds []string) error {
	for _, c := range categoryIds {
		if c == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO source_links (source_id, category_id) VALUES ($1, $2)`, sourceId, c); err != nil {
			return fmt.Errorf("error while linking source to category: %w", err)
		}
	}
	for _, t := range topicIds {
		if t == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `INSERT INTO source_links (source_id, topic_id) VALUES ($1, $2)`, sourceId, t); err != nil {
			return fmt.Errorf("error while linking source to topic: %w", err)
		}
	}
	return nil
}

const sourceSelect = `
	s.id, s.name, s.type, s.language, s.author, s.url, s.file_url, s.status, s.error, s.is_active,
	(SELECT COUNT(*) FROM source_chunks c WHERE c.source_id = s.id),
	COALESCE((SELECT array_agg(l.category_id::text) FROM source_links l WHERE l.source_id = s.id AND l.category_id IS NOT NULL), '{}'),
	COALESCE((SELECT array_agg(l.topic_id::text) FROM source_links l WHERE l.source_id = s.id AND l.topic_id IS NOT NULL), '{}'),
	s.created_at::text, s.updated_at::text
`

func scanSource(row pgx.Row, total *int) (*m.SourceRes, error) {
	s := &m.SourceRes{}
	dest := []any{&s.Id, &s.Title, &s.Type, &s.Language, &s.Author, &s.Url, &s.FileUrl, &s.Status, &s.Error, &s.IsActive,
		&s.ChunkCount, &s.CategoryIds, &s.TopicIds, &s.CreatedAt, &s.UpdatedAt}
	if total != nil {
		dest = append([]any{total}, dest...)
	}
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	return s, nil
}

func (r *SourceRepo) GetById(req *m.ById) (*m.SourceRes, error) {
	s, err := scanSource(r.db.QueryRow(context.Background(),
		`SELECT `+sourceSelect+` FROM case_sources s WHERE s.id = $1 AND s.deleted_at = 0`, req.Id), nil)
	if err == pgx.ErrNoRows {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting source: %w", err)
	}
	return s, nil
}

func (r *SourceRepo) GetAll(req *m.SourceGetListReq) (*m.PaginationResponse, *m.SourceList, error) {
	query := `SELECT COUNT(*) OVER() AS total_count, ` + sourceSelect + ` FROM case_sources s WHERE s.deleted_at = 0`
	var args []interface{}
	add := func(cond string, v interface{}) {
		args = append(args, v)
		query += strings.ReplaceAll(cond, "$#", "$"+strconv.Itoa(len(args)))
	}
	if req.Type != "" && req.Type != "string" {
		add(" AND s.type = $#", req.Type)
	}
	if req.Status != "" && req.Status != "string" {
		add(" AND s.status = $#", req.Status)
	}
	if req.Name != "" && req.Name != "string" {
		add(" AND s.name ILIKE $#", "%"+req.Name+"%")
	}
	if req.CategoryId != "" && req.CategoryId != "string" {
		add(" AND EXISTS (SELECT 1 FROM source_links l WHERE l.source_id = s.id AND l.category_id = $#)", req.CategoryId)
	}
	if req.TopicId != "" && req.TopicId != "string" {
		add(" AND EXISTS (SELECT 1 FROM source_links l WHERE l.source_id = s.id AND l.topic_id = $#)", req.TopicId)
	}
	query += " ORDER BY s.created_at DESC"
	args = append(args, req.Filter.Limit, req.Filter.Page)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("error while getting sources: %w", err)
	}
	defer rows.Close()
	list := m.SourceList{}
	for rows.Next() {
		var total int
		s, err := scanSource(rows, &total)
		if err != nil {
			return nil, nil, fmt.Errorf("error while scanning source: %w", err)
		}
		list.Sources = append(list.Sources, s)
		list.Count = total
	}
	resp := m.NewPaginationResponse(req.Filter, list.Count, list.Sources)
	return &resp, &list, nil
}

func (r *SourceRepo) Update(req *m.SourceUpdateReq) (*m.Void, error) {
	ctx := context.Background()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var conds []string
	var args []interface{}
	set := func(col string, v interface{}) {
		args = append(args, v)
		conds = append(conds, col+" = $"+strconv.Itoa(len(args)))
	}
	if req.Title != "" && req.Title != "string" {
		set("name", req.Title)
	}
	if req.Language != "" {
		set("language", req.Language)
	}
	if req.Author != "" {
		set("author", req.Author)
	}
	if req.Url != "" {
		set("url", req.Url)
	}
	if req.IsActive != nil {
		set("is_active", *req.IsActive)
	}
	conds = append(conds, "updated_at = now()")
	args = append(args, req.Id)
	res, err := tx.Exec(ctx, "UPDATE case_sources SET "+strings.Join(conds, ", ")+" WHERE id = $"+strconv.Itoa(len(args))+" AND deleted_at = 0", args...)
	if err != nil {
		return nil, fmt.Errorf("error while updating source: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, errs.ErrNotFound
	}
	// nil = o'zgartirilmaydi; bo'sh ro'yxat = bog'lanishlar tozalanadi.
	if req.CategoryIds != nil || req.TopicIds != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM source_links WHERE source_id = $1`, req.Id); err != nil {
			return nil, err
		}
		if err := insertLinks(ctx, tx, req.Id, req.CategoryIds, req.TopicIds); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &m.Void{}, nil
}

func (r *SourceRepo) Delete(req *m.ById) (*m.Void, error) {
	ctx := context.Background()
	res, err := r.db.Exec(ctx, `UPDATE case_sources SET deleted_at = EXTRACT(EPOCH FROM NOW()) WHERE id = $1 AND deleted_at = 0`, req.Id)
	if err != nil {
		return nil, fmt.Errorf("error while deleting source: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, errs.ErrNotFound
	}
	_, _ = r.db.Exec(ctx, `DELETE FROM source_chunks WHERE source_id = $1`, req.Id)
	return &m.Void{}, nil
}

func (r *SourceRepo) GetRaw(id string) (typ, url, fileUrl, content string, err error) {
	err = r.db.QueryRow(context.Background(),
		`SELECT type, url, file_url, content FROM case_sources WHERE id = $1 AND deleted_at = 0`, id).Scan(&typ, &url, &fileUrl, &content)
	if err == pgx.ErrNoRows {
		err = errs.ErrNotFound
	}
	return
}

func (r *SourceRepo) SetStatus(id, status, errText string) error {
	if len(errText) > 1000 {
		errText = errText[:1000]
	}
	_, err := r.db.Exec(context.Background(),
		`UPDATE case_sources SET status = $1, error = $2, updated_at = now() WHERE id = $3`, status, errText, id)
	return err
}

func (r *SourceRepo) SetContent(id, content string) error {
	_, err := r.db.Exec(context.Background(), `UPDATE case_sources SET content = $1, updated_at = now() WHERE id = $2`, content, id)
	return err
}

func (r *SourceRepo) ReplaceChunks(sourceId string, chunks []m.SourceChunk) error {
	ctx := context.Background()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM source_chunks WHERE source_id = $1`, sourceId); err != nil {
		return err
	}
	batch := &pgx.Batch{}
	for i, c := range chunks {
		var emb interface{}
		if len(c.Embedding) > 0 {
			emb = c.Embedding
		}
		batch.Queue(`INSERT INTO source_chunks (source_id, idx, locator, content, tokens, embedding) VALUES ($1, $2, $3, $4, $5, $6)`,
			sourceId, i, c.Locator, c.Content, c.Tokens, emb)
	}
	br := tx.SendBatch(ctx, batch)
	for range chunks {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return fmt.Errorf("error while inserting chunk: %w", err)
		}
	}
	if err := br.Close(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// scopeWhere - chunk doirasi: faol, tayyor, o'chirilmagan manbalar; manba yo'q link'li
// (umumiy) YOKI tanlangan mavzu(lar)ga YOKI shu bo'limga bog'langan.
// topicIds bo'sh bo'lsa (aralash) - bo'limdagi barcha mavzularga bog'langan manbalar ham kiradi.
const scopeWhere = `
	s.deleted_at = 0 AND s.is_active AND s.status = 'ready'
	AND (
		NOT EXISTS (SELECT 1 FROM source_links l WHERE l.source_id = s.id)
		OR EXISTS (
			SELECT 1 FROM source_links l WHERE l.source_id = s.id AND (
				l.category_id = $1::uuid
				OR l.topic_id = ANY($2::uuid[])
				OR (cardinality($2::uuid[]) = 0 AND l.topic_id IN (SELECT id FROM topics WHERE category_id = $1::uuid AND deleted_at = 0))
			)
		)
	)
`

// Normalize - vektorni birlik uzunlikka keltiradi (dot product = cosine).
func Normalize(v []float32) []float32 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return v
	}
	n := float32(math.Sqrt(sum))
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = x / n
	}
	return out
}

func (r *SourceRepo) SearchChunks(categoryId string, topicIds []string, queryEmb []float32, limit int) ([]m.SourceChunk, error) {
	if topicIds == nil {
		topicIds = []string{}
	}
	var (
		query string
		args  = []interface{}{categoryId, topicIds}
	)
	cols := `c.id, c.source_id, c.idx, c.locator, c.content, c.tokens, s.name, s.type, s.author, s.url`
	if len(queryEmb) > 0 {
		args = append(args, queryEmb, limit)
		// Embeddingi bor chunklar - dot product bo'yicha; embeddingsizlar oxirida.
		query = `SELECT ` + cols + ` FROM source_chunks c JOIN case_sources s ON s.id = c.source_id
			WHERE ` + scopeWhere + `
			ORDER BY CASE WHEN c.embedding IS NULL OR cardinality(c.embedding) <> cardinality($3::real[]) THEN -1
				ELSE (SELECT COALESCE(SUM(a * b), 0) FROM unnest(c.embedding, $3::real[]) AS t(a, b)) END DESC
			LIMIT $4`
	} else {
		args = append(args, limit)
		query = `SELECT ` + cols + ` FROM source_chunks c JOIN case_sources s ON s.id = c.source_id
			WHERE ` + scopeWhere + ` ORDER BY random() LIMIT $3`
	}
	rows, err := r.db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("error while searching chunks: %w", err)
	}
	defer rows.Close()
	var out []m.SourceChunk
	for rows.Next() {
		var c m.SourceChunk
		if err := rows.Scan(&c.Id, &c.SourceId, &c.Idx, &c.Locator, &c.Content, &c.Tokens, &c.SourceName, &c.SourceType, &c.Author, &c.Url); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}
