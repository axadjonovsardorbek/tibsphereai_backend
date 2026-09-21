package service

import (
	"context"
	"fmt"
	"tibsphereai/internal/ai"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// SimulationService - Simulyatsiya dvigateli (BACKEND_STRUCTURE.md 4-bo'lim):
// kirish huquqi (limit/premium), qadamlar logi, yakunlash + AI debrief + XP/Coin.
type SimulationService struct {
	storage     st.Storage
	ai          *ai.Client
	appSettings *AppSettingService
}

func NewSimulationService(storage *st.Storage, aiClient *ai.Client, appSettings *AppSettingService) *SimulationService {
	return &SimulationService{storage: *storage, ai: aiClient, appSettings: appSettings}
}

// hasActiveSubscription - Premium obuna faolligini tekshiradi.
func (s *SimulationService) hasActiveSubscription(userId string) bool {
	var active bool
	_ = s.storage.Db.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM subscriptions WHERE user_id = $1 AND status = 'active' AND ended_at > now())
	`, userId).Scan(&active)
	return active
}

// Start - kirish huquqi 3 bosqichda tekshiriladi: 1) faol Premium obuna (cheksiz),
// 2) kunlik bepul urinish (appSettings), 3) shular tugagan bo'lsa - foydalanuvchi
// shu case'ning bo'limidan (category) tanga evaziga oldindan sotib olgan case-huquqi
// (category_unlocks, ko'ring [[CategoryUnlockService]]). Uchalasi ham yo'q bo'lsa
// ErrDailyLimitReached.
func (s *SimulationService) Start(ctx context.Context, req *m.SimulationStartReq) (*m.SimulationStartRes, error) {
	hasSub := s.hasActiveSubscription(req.UserId)
	needsCategoryUnlock := false
	var categoryId string

	if !hasSub {
		ok, err := s.appSettings.HasFreeAttemptLeft(ctx, req.UserId)
		if err != nil {
			return nil, err
		}
		if !ok {
			cid, cErr := s.storage.CaseS.GetCategoryId(req.CaseId)
			if cErr != nil {
				return nil, cErr
			}
			remaining, rErr := s.storage.CategoryUnlockS.GetRemaining(req.UserId, cid)
			if rErr != nil {
				return nil, rErr
			}
			if remaining <= 0 {
				return nil, errs.ErrDailyLimitReached
			}
			categoryId = cid
			needsCategoryUnlock = true
		}
	}

	res, err := s.storage.SimulationS.Start(req)
	if err != nil {
		return nil, err
	}

	if !hasSub {
		if needsCategoryUnlock {
			if _, err := s.storage.CategoryUnlockS.ConsumeOne(req.UserId, categoryId); err != nil {
				fmt.Println("failed to consume category unlock:", err)
			}
		} else if err := s.appSettings.ConsumeFreeAttempt(ctx, req.UserId); err != nil {
			fmt.Println("failed to consume free attempt:", err)
		}
	}

	return res, nil
}

// AddEvent - "question" turi uchun bemor AI (internal/ai.PatientAI) javobini
// qo'shadi (BACKEND_STRUCTURE.md 4-bo'lim: Anamnez); qolgan turlar (ko'rik,
// tahlil, instrumental, dori, protsedura) uchun ActionEvaluatorAI orqali
// harakat case scenario/expected_steps'ga solishtirib real vaqtda baholanadi
// - shundan keyingina Sog'liq shkalasi (health_percent) o'zgaradi.
func (s *SimulationService) AddEvent(ctx context.Context, req *m.SimulationEventReq) (*m.SimulationEventRes, error) {
	isCorrect, healthDelta := true, 0
	responseExtra := map[string]any{}

	session, sErr := s.storage.SimulationS.GetById(&m.ById{Id: req.SessionId})
	var caseRes *m.CaseRes
	if sErr == nil && session != nil && session.CaseId != nil {
		caseRes, _ = s.storage.CaseS.GetById(&m.ById{Id: *session.CaseId})
	}
	userId := ""
	if session != nil && session.UserId != nil {
		userId = *session.UserId
	}
	caseId := ""
	if session != nil && session.CaseId != nil {
		caseId = *session.CaseId
	}

	if caseRes != nil {
		if req.Type == "question" {
			question, _ := req.Payload["question"].(string)
			reply, usage, aiErr := s.ai.Patient.Reply(ctx, caseRes.Scenario, question)
			logAIUsage(&s.storage, "chat", usage, userId, caseId)
			if aiErr == nil {
				responseExtra["patient_reply"] = reply
			} else {
				responseExtra["patient_reply_error"] = aiErr.Error()
			}
		} else {
			eval, usage, aiErr := s.ai.Action.Evaluate(ctx, caseRes.Scenario, req.Type, req.Payload)
			logAIUsage(&s.storage, "action_eval", usage, userId, caseId)
			if aiErr == nil && eval != nil {
				isCorrect, healthDelta = eval.IsCorrect, eval.HealthDelta
				responseExtra["feedback"] = eval.Feedback
			} else if aiErr != nil {
				responseExtra["feedback_error"] = aiErr.Error()
			}
		}
	}

	res, err := s.storage.SimulationS.AddEvent(req, isCorrect, healthDelta)
	if err != nil {
		return nil, err
	}
	for k, v := range responseExtra {
		res.Response[k] = v
	}
	return res, nil
}

// Finish - sessiyani yakunlaydi, XP/Coin beradi (BACKEND_STRUCTURE.md 6-bo'lim:
// muvaffaqiyatsiz bo'lsa tanga berilmaydi, faqat XP/urinish hisoblanadi) va
// AI debriefing hisobotini generatsiya qiladi.
func (s *SimulationService) Finish(ctx context.Context, req *m.SimulationFinishReq) (*m.SimulationFinishRes, error) {
	res, err := s.storage.SimulationS.Finish(req)
	if err != nil {
		return nil, err
	}

	session, err := s.storage.SimulationS.GetById(&m.ById{Id: req.SessionId})
	if err == nil && session != nil && session.UserId != nil {
		if _, err := s.storage.UserS.AddPoints(&m.AddPointsReq{
			UserId:      *session.UserId,
			XPDelta:     res.XPEarned,
			CoinsDelta:  res.CoinsEarned,
			Reason:      "simulation_finish",
			ReferenceId: req.SessionId,
		}); err != nil {
			fmt.Println("failed to add points:", err)
		}

		if _, err := s.storage.UserS.CreateActivity(&m.UserActivityCreateReq{UserId: *session.UserId, Activity: 1}); err != nil {
			fmt.Println("failed to log activity:", err)
		}
	}

	go s.generateDebrief(req.SessionId)

	res.DebriefReady = false // TODO: WebSocket/poll orqali tayyor bo'lgani xabar qilinadi
	return res, nil
}

func (s *SimulationService) generateDebrief(sessionId string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("debrief generation panic:", r)
		}
	}()

	ctx := context.Background()

	session, err := s.storage.SimulationS.GetById(&m.ById{Id: sessionId})
	if err != nil || session == nil || session.CaseId == nil {
		return
	}

	caseRes, err := s.storage.CaseS.GetById(&m.ById{Id: *session.CaseId})
	if err != nil {
		return
	}

	events, err := s.storage.SimulationS.GetEvents(sessionId)
	if err != nil {
		return
	}

	report, usage, err := s.ai.Debrief.Generate(ctx, caseRes.Scenario, events)
	userId := ""
	if session.UserId != nil {
		userId = *session.UserId
	}
	logAIUsage(&s.storage, "debrief", usage, userId, *session.CaseId)
	if err != nil {
		fmt.Println("debrief AI generation failed:", err)
		return
	}
	report.SessionId = sessionId

	if _, err := s.storage.DebriefS.Create(report); err != nil {
		fmt.Println("failed to save debrief report:", err)
	}
}

func (s *SimulationService) GetById(ctx context.Context, req *m.ById) (*m.SimulationSessionRes, error) {
	return s.storage.SimulationS.GetById(req)
}
func (s *SimulationService) GetOngoing(ctx context.Context, req *m.ById) (*m.PaginationResponse, *m.SimulationSessionList, error) {
	return s.storage.SimulationS.GetOngoing(req)
}
func (s *SimulationService) GetCompleted(ctx context.Context, req *m.ById) (*m.PaginationResponse, *m.SimulationSessionList, error) {
	return s.storage.SimulationS.GetCompleted(req)
}
