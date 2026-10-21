package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	m "tibsphereai/pkg/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrDebriefNotFound - sessiya uchun debrief qatori yo'q.
var ErrDebriefNotFound = errors.New("debrief report not found")

type DebriefRepo struct {
	db *pgxpool.Pool
}

func NewDebriefRepo(db *pgxpool.Pool) *DebriefRepo {
	return &DebriefRepo{db: db}
}

// Create - internal/ai.DebriefAI (hozircha stub) tomonidan tayyorlangan hisobotni saqlaydi.
func (r *DebriefRepo) Create(req *m.DebriefReportCreateReq) (*m.Void, error) {
	correct, _ := json.Marshal(req.CorrectSteps)
	incorrect, _ := json.Marshal(req.IncorrectSteps)
	weak, _ := json.Marshal(req.WeakTopics)

	_, err := r.db.Exec(context.Background(), `
		INSERT INTO debrief_reports (session_id, correct_steps, incorrect_steps, guideline_notes, weak_topics)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (session_id) DO UPDATE SET
			correct_steps = EXCLUDED.correct_steps,
			incorrect_steps = EXCLUDED.incorrect_steps,
			guideline_notes = EXCLUDED.guideline_notes,
			weak_topics = EXCLUDED.weak_topics
	`, req.SessionId, string(correct), string(incorrect), req.GuidelineNotes, string(weak))
	if err != nil {
		return nil, fmt.Errorf("error while creating debrief report: %w", err)
	}
	return &m.Void{}, nil
}

func (r *DebriefRepo) GetBySessionId(sessionId string) (*m.DebriefReportRes, error) {
	row := r.db.QueryRow(context.Background(), `
		SELECT d.session_id, d.correct_steps, d.incorrect_steps, s.final_score, s.xp_earned,
			s.coins_earned, d.guideline_notes, d.weak_topics, d.created_at::text
		FROM debrief_reports d
		JOIN simulation_sessions s ON s.id = d.session_id
		WHERE d.session_id = $1
	`, sessionId)

	var correctJSON, incorrectJSON, weakJSON []byte
	d := m.DebriefReportRes{}
	err := row.Scan(
		&d.SessionId, &correctJSON, &incorrectJSON, &d.FinalScore, &d.XPEarned,
		&d.CoinsEarned, &d.GuidelineNotes, &weakJSON, &d.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("%w for session: %s", ErrDebriefNotFound, sessionId)
	}
	if err != nil {
		return nil, fmt.Errorf("error while getting debrief report: %w", err)
	}

	json.Unmarshal(correctJSON, &d.CorrectSteps)
	json.Unmarshal(incorrectJSON, &d.IncorrectSteps)
	json.Unmarshal(weakJSON, &d.WeakTopics)

	return &d, nil
}
