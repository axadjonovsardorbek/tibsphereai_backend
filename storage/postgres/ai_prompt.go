package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	m "tibsphereai/pkg/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AIPromptRepo - "AI Promptlari va Model parametrlari" (BACKEND_STRUCTURE.md 7-bo'lim).
type AIPromptRepo struct {
	db *pgxpool.Pool
}

func NewAIPromptRepo(db *pgxpool.Pool) *AIPromptRepo {
	return &AIPromptRepo{db: db}
}

func (r *AIPromptRepo) GetByKey(key string) (*m.AIPromptRes, error) {
	row := r.db.QueryRow(context.Background(), `
		SELECT key, template, model_params, updated_at::text FROM ai_prompts WHERE key = $1
	`, key)

	var paramsJSON []byte
	p := m.AIPromptRes{}
	err := row.Scan(&p.Key, &p.Template, &paramsJSON, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting ai prompt: %w", err)
	}
	json.Unmarshal(paramsJSON, &p.ModelParams)
	return &p, nil
}

func (r *AIPromptRepo) GetAll() ([]m.AIPromptRes, error) {
	rows, err := r.db.Query(context.Background(), `SELECT key, template, model_params, updated_at::text FROM ai_prompts ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("error while getting ai prompts: %w", err)
	}
	defer rows.Close()

	var list []m.AIPromptRes
	for rows.Next() {
		var paramsJSON []byte
		p := m.AIPromptRes{}
		if err := rows.Scan(&p.Key, &p.Template, &paramsJSON, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error while scanning ai prompt: %w", err)
		}
		json.Unmarshal(paramsJSON, &p.ModelParams)
		list = append(list, p)
	}
	return list, nil
}

func (r *AIPromptRepo) Upsert(req *m.AIPromptUpsertReq) (*m.Void, error) {
	params, _ := json.Marshal(req.ModelParams)
	_, err := r.db.Exec(context.Background(), `
		INSERT INTO ai_prompts (key, template, model_params, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (key) DO UPDATE SET template = EXCLUDED.template, model_params = EXCLUDED.model_params, updated_at = now()
	`, req.Key, req.Template, string(params))
	if err != nil {
		return nil, fmt.Errorf("error while upserting ai prompt: %w", err)
	}
	return &m.Void{}, nil
}

type AIUsageLogRepo struct {
	db *pgxpool.Pool
}

func NewAIUsageLogRepo(db *pgxpool.Pool) *AIUsageLogRepo {
	return &AIUsageLogRepo{db: db}
}

func (r *AIUsageLogRepo) Log(req *m.AIUsageLogCreateReq) error {
	var userId, caseId, sessionId interface{}
	if req.UserId != "" {
		userId = req.UserId
	}
	if req.CaseId != "" {
		caseId = req.CaseId
	}
	if req.SessionId != "" {
		sessionId = req.SessionId
	}
	// user_tier - chaqiruv paytidagi holat: faol obuna -> premium; simulyatsiya sessiyasi tanga
	// hisobidan (coin_billed) yoki bugungi bepul tangalar tugagan -> coin; aks holda free.
	_, err := r.db.Exec(context.Background(), `
		INSERT INTO ai_usage_logs (id, feature, tokens_used, cost_usd, user_id, case_id,
			session_kind, session_id, model, prompt_tokens, completion_tokens, user_tier)
		VALUES (gen_random_uuid(), $1, $2, $3, $4::uuid, $5, $6::varchar, $7::uuid, $8, $9, $10,
			CASE
				WHEN $4::uuid IS NULL THEN ''
				WHEN EXISTS (SELECT 1 FROM subscriptions s WHERE s.user_id = $4::uuid AND s.status = 'active' AND s.ended_at > now()) THEN 'premium'
				WHEN $6::varchar = 'simulation' THEN COALESCE((SELECT CASE WHEN ss.coin_billed THEN 'coin' ELSE 'free' END
					FROM simulation_sessions ss WHERE ss.id = $7::uuid), 'free')
				WHEN COALESCE((SELECT d.granted - d.used FROM user_daily_coins d
					WHERE d.user_id = $4::uuid AND d.day = (now() AT TIME ZONE 'Asia/Tashkent')::date), 0) > 0 THEN 'free'
				ELSE 'coin'
			END)
	`, req.Feature, req.Tokens, req.CostUSD, userId, caseId, req.SessionKind, sessionId, req.Model, req.PromptTokens, req.CompletionTokens)
	if err != nil {
		return fmt.Errorf("error while logging ai usage: %w", err)
	}
	return nil
}

func (r *AIUsageLogRepo) GetSummary() (*m.AIUsageSummaryRes, error) {
	row := r.db.QueryRow(context.Background(), `
		SELECT COALESCE(SUM(tokens_used), 0), COALESCE(SUM(cost_usd), 0) FROM ai_usage_logs
	`)
	s := m.AIUsageSummaryRes{}
	if err := row.Scan(&s.TotalTokens, &s.TotalCost); err != nil {
		return nil, fmt.Errorf("error while getting ai usage summary: %w", err)
	}
	return &s, nil
}
