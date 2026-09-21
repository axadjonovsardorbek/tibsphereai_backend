package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	m "tibsphereai/pkg/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SimulationRepo struct {
	db *pgxpool.Pool
}

func NewSimulationRepo(db *pgxpool.Pool) *SimulationRepo {
	return &SimulationRepo{db: db}
}

// Start - kirish huquqi (kunlik limit/premium) service qatlamida AppSettingI
// orqali oldindan tekshiriladi (BACKEND_STRUCTURE.md 3,6-bo'lim).
func (r *SimulationRepo) Start(req *m.SimulationStartReq) (*m.SimulationStartRes, error) {
	ctx := context.Background()

	var vitalsJSON, visualState string
	var timeLimit int
	err := r.db.QueryRow(ctx, `
		SELECT initial_vitals::text, visual_state, expected_duration_minutes * 60
		FROM cases WHERE id = $1 AND deleted_at = 0 AND status = 'published'
	`, req.CaseId).Scan(&vitalsJSON, &visualState, &timeLimit)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("case not found: %s", req.CaseId)
	}
	if err != nil {
		return nil, fmt.Errorf("error while loading case: %w", err)
	}

	// req.DurationMinutes - home ekranda foydalanuvchi tanlagan davomiylik
	// (masalan 5/10/15 daqiqa) case'ning o'zidagi expected_duration_minutes'ni
	// ustidan bosib o'tadi (timer shu qiymatga qarab ishlaydi).
	if req.DurationMinutes > 0 {
		timeLimit = req.DurationMinutes * 60
	}

	id := uuid.NewString()
	_, err = r.db.Exec(ctx, `
		INSERT INTO simulation_sessions (id, user_id, case_id, status, health_percent, time_limit_seconds, started_at)
		VALUES ($1, $2, $3, 'active', 100, $4, now())
	`, id, req.UserId, req.CaseId, timeLimit)
	if err != nil {
		return nil, fmt.Errorf("error while starting simulation: %w", err)
	}

	var vitals m.CaseVitals
	json.Unmarshal([]byte(vitalsJSON), &vitals)

	return &m.SimulationStartRes{
		SessionId:       id,
		CaseId:          req.CaseId,
		HealthPercent:   100,
		TimeLimitSecond: timeLimit,
		InitialVitals:   vitals,
		VisualState:     visualState,
		WSUrl:           fmt.Sprintf("/mobile/simulation/%s/ws", id),
	}, nil
}

// AddEvent - to'g'ri/noto'g'ri baholash (AI orqali, [[service/simulation.go]])
// service qatlamida hisoblanib shu yerga uzatiladi; bu yerda faqat log yozamiz
// va health_percent'ni HealthDelta bo'yicha yangilaymiz.
func (r *SimulationRepo) AddEvent(req *m.SimulationEventReq, isCorrect bool, healthDelta int) (*m.SimulationEventRes, error) {
	ctx := context.Background()

	payload, _ := json.Marshal(req.Payload)

	_, err := r.db.Exec(ctx, `
		INSERT INTO simulation_events (id, session_id, type, payload, is_correct, health_delta)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.NewString(), req.SessionId, req.Type, string(payload), isCorrect, healthDelta)
	if err != nil {
		return nil, fmt.Errorf("error while logging simulation event: %w", err)
	}

	var healthPercent int
	err = r.db.QueryRow(ctx, `
		UPDATE simulation_sessions
		SET health_percent = GREATEST(0, LEAST(100, health_percent + $1))
		WHERE id = $2
		RETURNING health_percent
	`, healthDelta, req.SessionId).Scan(&healthPercent)
	if err != nil {
		return nil, fmt.Errorf("error while updating health: %w", err)
	}

	return &m.SimulationEventRes{
		IsCorrect:     isCorrect,
		HealthDelta:   healthDelta,
		HealthPercent: healthPercent,
		Response:      map[string]any{}, // TODO(ai): Gemini AI javobi shu yerga
	}, nil
}

// Finish - TODO: yakuniy ball/AI debriefingni internal/ai.DebriefAI orqali hisoblash;
// hozircha health_percent'ni final_score sifatida ishlatamiz.
func (r *SimulationRepo) Finish(req *m.SimulationFinishReq) (*m.SimulationFinishRes, error) {
	ctx := context.Background()

	var healthPercent int
	status := "completed"
	if req.Reason == "health_zero" {
		status = "failed"
	}

	err := r.db.QueryRow(ctx, `
		UPDATE simulation_sessions
		SET status = $1, ended_at = now(), final_score = health_percent
		WHERE id = $2 AND status = 'active'
		RETURNING health_percent
	`, status, req.SessionId).Scan(&healthPercent)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("active session not found: %s", req.SessionId)
	}
	if err != nil {
		return nil, fmt.Errorf("error while finishing simulation: %w", err)
	}

	xp := healthPercent
	coins := 0
	if status == "completed" {
		coins = healthPercent / 10
	}

	_, err = r.db.Exec(ctx, `
		UPDATE simulation_sessions SET xp_earned = $1, coins_earned = $2 WHERE id = $3
	`, xp, coins, req.SessionId)
	if err != nil {
		return nil, fmt.Errorf("error while saving rewards: %w", err)
	}

	return &m.SimulationFinishRes{
		SessionId:    req.SessionId,
		FinalScore:   healthPercent,
		XPEarned:     xp,
		CoinsEarned:  coins,
		DebriefReady: false, // TODO(ai): DebriefAI orqali hisobot tayyor bo'lganda true
	}, nil
}

func (r *SimulationRepo) GetById(req *m.ById) (*m.SimulationSessionRes, error) {
	row := r.db.QueryRow(context.Background(), `
		SELECT s.id, s.user_id, s.case_id, c.title ->> 'uz', s.status, s.health_percent,
			s.final_score, s.xp_earned, s.coins_earned, s.started_at::text, s.ended_at::text
		FROM simulation_sessions s
		JOIN cases c ON c.id = s.case_id
		WHERE s.id = $1
	`, req.Id)

	s := m.SimulationSessionRes{}
	err := row.Scan(
		&s.Id, &s.UserId, &s.CaseId, &s.CaseTitle, &s.Status, &s.HealthPercent,
		&s.FinalScore, &s.XPEarned, &s.CoinsEarned, &s.StartedAt, &s.EndedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("simulation session not found: %s", req.Id)
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting simulation session: %w", err)
	}
	return &s, nil
}

func (r *SimulationRepo) GetEvents(sessionId string) ([]m.SimulationEventReq, error) {
	rows, err := r.db.Query(context.Background(), `
		SELECT type, payload FROM simulation_events WHERE session_id = $1 ORDER BY created_at ASC
	`, sessionId)
	if err != nil {
		return nil, fmt.Errorf("error while getting simulation events: %w", err)
	}
	defer rows.Close()

	var events []m.SimulationEventReq
	for rows.Next() {
		var e m.SimulationEventReq
		var payloadJSON []byte
		if err := rows.Scan(&e.Type, &payloadJSON); err != nil {
			return nil, fmt.Errorf("error while scanning simulation event: %w", err)
		}
		json.Unmarshal(payloadJSON, &e.Payload)
		e.SessionId = sessionId
		events = append(events, e)
	}
	return events, nil
}

func (r *SimulationRepo) GetOngoing(req *m.ById) (*m.PaginationResponse, *m.SimulationSessionList, error) {
	return r.listByStatus(req.Id, "active")
}

func (r *SimulationRepo) GetCompleted(req *m.ById) (*m.PaginationResponse, *m.SimulationSessionList, error) {
	return r.listByStatus(req.Id, "completed")
}

func (r *SimulationRepo) listByStatus(userId, status string) (*m.PaginationResponse, *m.SimulationSessionList, error) {
	rows, err := r.db.Query(context.Background(), `
		SELECT s.id, s.user_id, s.case_id, c.title ->> 'uz', s.status, s.health_percent,
			s.final_score, s.xp_earned, s.coins_earned, s.started_at::text, s.ended_at::text
		FROM simulation_sessions s
		JOIN cases c ON c.id = s.case_id
		WHERE s.user_id = $1 AND s.status = $2
		ORDER BY s.started_at DESC
	`, userId, status)
	if err != nil {
		return nil, nil, fmt.Errorf("error while listing simulation sessions: %w", err)
	}
	defer rows.Close()

	list := m.SimulationSessionList{}
	for rows.Next() {
		s := &m.SimulationSessionRes{}
		if err := rows.Scan(
			&s.Id, &s.UserId, &s.CaseId, &s.CaseTitle, &s.Status, &s.HealthPercent,
			&s.FinalScore, &s.XPEarned, &s.CoinsEarned, &s.StartedAt, &s.EndedAt,
		); err != nil {
			return nil, nil, fmt.Errorf("error while scanning simulation session: %w", err)
		}
		list.Sessions = append(list.Sessions, s)
		list.Count++
	}

	response := m.NewPaginationResponse(m.Pagination{Limit: list.Count}, list.Count, list.Sessions)
	return &response, &list, nil
}
