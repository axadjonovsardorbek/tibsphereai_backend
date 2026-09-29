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

// Start - kirish huquqi 2 bosqichda tekshiriladi: 1) faol Premium obuna (cheksiz),
// 2) kunlik bepul urinish (appSettings). Ikkalasi ham yo'q bo'lsa ErrDailyLimitReached
// (bo'lim (category) tanga evaziga sotib olish - coin bilan narxlash - olib tashlangan).
func (s *SimulationService) Start(ctx context.Context, req *m.SimulationStartReq) (*m.SimulationStartRes, error) {
	hasSub := s.hasActiveSubscription(req.UserId)

	if !hasSub {
		ok, err := s.appSettings.HasFreeAttemptLeft(ctx, req.UserId)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errs.ErrDailyLimitReached
		}
	}

	res, err := s.storage.SimulationS.Start(req)
	if err != nil {
		return nil, err
	}

	if !hasSub {
		if err := s.appSettings.ConsumeFreeAttempt(ctx, req.UserId); err != nil {
			fmt.Println("failed to consume free attempt:", err)
		}
	}

	return res, nil
}

// friendlyAIError - AI provayder (Gemini) xatosining xom texnik matni
// (masalan "gemini API error (status 503): This model is currently
// experiencing high demand...") foydalanuvchiga TO'G'RIDAN-TO'G'RI
// ko'rsatilmaydi - inglizcha, tushunarsiz va ichki infratuzilma
// tafsilotlarini oshkor qiladi. To'liq matn serverda logga yoziladi
// (debug uchun), foydalanuvchiga esa umumiy, o'zbekcha xabar qaytadi.
func friendlyAIError(err error) string {
	fmt.Println("AI error:", err)
	return "AI xizmati hozir band yoki vaqtincha ishlamayapti. Birozdan so'ng qayta urinib ko'ring."
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
	if sErr != nil {
		return nil, sErr
	}
	if session == nil {
		return nil, errs.ErrSessionNotFound
	}
	// health_percent 0'ga tushib avtomatik yakunlangan (yoki boshqa sabab bilan
	// yakunlangan) sessiyaga yana buyruq/savol yuborilishini to'sadi - "o'lgan"
	// bemorga cheksiz muolaja qilib bo'lmaydi.
	if session.Status != nil && *session.Status != "active" {
		return nil, errs.ErrSessionNotActive
	}

	var caseRes *m.CaseRes
	if session.CaseId != nil {
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

	expectedAnswer := ""
	if caseRes != nil && caseRes.ExpectedAnswer != nil {
		expectedAnswer = *caseRes.ExpectedAnswer
	}

	if caseRes != nil {
		if req.Type == "question" {
			question, _ := req.Payload["question"].(string)
			reply, usage, aiErr := s.ai.Patient.Reply(ctx, caseRes.Scenario, expectedAnswer, question)
			logAIUsage(&s.storage, "chat", usage, userId, caseId)
			if aiErr != nil {
				responseExtra["patient_reply_error"] = friendlyAIError(aiErr)
			} else if reply.IsAction {
				// Shifokor chatda qaror/harakat bayon qildi ("dori beraman", "tekshiruv
				// qilamiz") - bu bemorga emas, platformaga aytilgan deb hisoblanadi va
				// ExpectedAnswer'ga solishtirib to'g'ri/xato deb baholanadi.
				isCorrect, healthDelta = reply.IsCorrect, reply.HealthDelta
				responseExtra["feedback"] = reply.Feedback
			} else {
				responseExtra["patient_reply"] = reply.PatientReply
			}
		} else {
			eval, usage, aiErr := s.ai.Action.Evaluate(ctx, caseRes.Scenario, expectedAnswer, req.Type, req.Payload)
			logAIUsage(&s.storage, "action_eval", usage, userId, caseId)
			if aiErr == nil && eval != nil {
				isCorrect, healthDelta = eval.IsCorrect, eval.HealthDelta
				responseExtra["feedback"] = eval.Feedback
			} else if aiErr != nil {
				responseExtra["feedback_error"] = friendlyAIError(aiErr)
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

	// health_percent 0'ga tushdi - bemor "o'ldi", sessiya frontend "Yakunlash"
	// bosishini kutmasdan server tomonidan avtomatik yakunlanadi (aks holda
	// HP 0%da ham cheksiz buyruq berish mumkin bo'lib qolardi).
	if res.HealthPercent <= 0 {
		finishRes, fErr := s.Finish(ctx, &m.SimulationFinishReq{SessionId: req.SessionId, Reason: "health_zero"})
		if fErr != nil {
			fmt.Println("failed to auto-finish simulation on health_zero:", fErr)
		} else {
			res.SessionEnded = true
			res.FinishResult = finishRes
		}
	}

	return res, nil
}

// defaultLevelXPReward/defaultLevelCoinReward - levels jadvalida hali hech qanday
// yozuv bo'lmasa (yoki foydalanuvchi XP'siga mos level topilmasa) ishlatiladigan
// bazaviy mukofot (Level 1'ning odatiy qiymatiga mos, ko'ring migrations/00013).
const (
	defaultLevelXPReward   = 50
	defaultLevelCoinReward = 5
)

// incorrectStepPenaltyPct - DebriefAI (Finish ichida sinxron generatsiya
// qilinadi) topgan har bir klinik xato (incorrect_steps) uchun yakuniy
// foizdan ayiriladigan jarima. Sabab: health_percent faqat ActionEvaluatorAI'ning
// REAL VAQTDAGI (har harakatdan keyingi) bahosiga qarab o'zgaradi; DebriefAI esa
// sessiya tugagach BUTUN tarixni ko'rib chiqib, real vaqtda yetarlicha
// jazolanmagan (yoki umuman e'tibordan chetda qolgan) xatolarni topishi mumkin -
// masalan "gipotenziyada nitroglikerin berish xavfi" kabi. Bu jarima bo'lmasa,
// bunday xato final_score/XP/tangaga umuman ta'sir qilmay qolardi.
const incorrectStepPenaltyPct = 10

// Finish - sessiyani yakunlaydi, XP/Coin beradi va AI debriefing hisobotini
// generatsiya qiladi. Mukofot ikki omilga qarab hisoblanadi:
// 1) foydalanuvchining O'Z DARAJASI (levels.xp_reward/coin_reward, [[LevelRes]]) -
// qancha yuqori level, shuncha ko'p bazaviy mukofot;
// 2) case.difficulty (easy|medium|hard) koeffitsienti (AppSettingService.DifficultyMultiplier,
// masalan hard=2x) - shu bazaga qo'llanadi.
// Ikkalasi ham finalScore'ga (qobiliyat/to'g'ri bajarish darajasi) mutanosib
// beriladi - masalan Level 3 foydalanuvchi hard case'ni 80% finalScore bilan
// tugatsa: xp = round(level3_xp_reward * hard_multiplier * 80/100). finalScore -
// health_percent'dan DebriefAI xatolari uchun jarima ayirilgan qiymat (pastda,
// incorrectStepPenaltyPct). Muvaffaqiyatsiz (health_zero) bo'lsa tanga
// berilmaydi (coins=0), faqat XP hisoblanadi (BACKEND_STRUCTURE.md 6-bo'lim).
// Hech qanday klinik harakat qilinmagan bo'lsa (0 ta action event - faqat "question"
// chat hisobga olinmaydi), xp/coins ham 0 - aks holda darhol "finish" bosib, hech
// narsa qilmasdan to'liq mukofot olish mumkin bo'lardi (health_percent 100'dan boshlanadi).
func (s *SimulationService) Finish(ctx context.Context, req *m.SimulationFinishReq) (*m.SimulationFinishRes, error) {
	session, err := s.storage.SimulationS.GetById(&m.ById{Id: req.SessionId})
	if err != nil {
		return nil, err
	}
	if session == nil || session.HealthPercent == nil {
		return nil, fmt.Errorf("active session not found: %s", req.SessionId)
	}
	healthPercent := *session.HealthPercent

	difficulty := "medium"
	var caseRes *m.CaseRes
	if session.CaseId != nil {
		if cr, cErr := s.storage.CaseS.GetById(&m.ById{Id: *session.CaseId}); cErr == nil && cr != nil {
			caseRes = cr
			if cr.Difficulty != nil && *cr.Difficulty != "" {
				difficulty = *cr.Difficulty
			}
		}
	}

	xpBase, coinBase := defaultLevelXPReward, defaultLevelCoinReward
	if session.UserId != nil {
		if user, uErr := s.storage.UserS.GetById(&m.ById{Id: *session.UserId}); uErr == nil && user != nil && user.XP != nil {
			if lvl, lErr := s.storage.LevelS.GetLevelForXP(*user.XP); lErr == nil && lvl != nil {
				if lvl.XPReward != nil {
					xpBase = *lvl.XPReward
				}
				if lvl.CoinReward != nil {
					coinBase = *lvl.CoinReward
				}
			}
		}
	}
	mult := s.appSettings.DifficultyMultiplier(ctx, difficulty)
	xpBase = int(float64(xpBase)*mult + 0.5)
	coinBase = int(float64(coinBase)*mult + 0.5)

	// actionCount - "question" (anamnez chat) hisobga olinmaydi, chunki u
	// health_percent'ga ta'sir qilmaydi (AddEvent). health_percent boshlang'ich
	// 100'dan boshlanadi va faqat klinik harakatlar (ko'rik/tahlil/dori/protsedura)
	// orqali o'zgaradi - shu sabab hech qanday klinik harakat qilmasdan (yoki faqat
	// savol berib) darhol "finish" bosilsa, health_percent 100'da qolib, hech narsa
	// qilmagan holda to'liq mukofot olib ketmasligi uchun bu tekshiruv kerak.
	events, eErr := s.storage.SimulationS.GetEvents(req.SessionId)
	actionCount := 0
	if eErr == nil {
		for _, e := range events {
			if e.Type != "question" {
				actionCount++
			}
		}
	}

	// DebriefAI hisobotini XP/coin hisoblashdan OLDIN, sinxron generatsiya
	// qilamiz (avval "go s.generateDebrief(...)" bilan fonda, natija reward'ga
	// umuman ta'sir qilmasdi) - shunda incorrect_steps darhol jarima sifatida
	// hisobga olinadi. AI mavjud bo'lmasa (Debrief=nil, mock/GEMINI_API_KEY
	// yo'q) yoki xato bersa, jarimasiz (finalScore=healthPercent) davom etadi.
	var debrief *m.DebriefReportCreateReq
	if s.ai != nil && s.ai.Debrief != nil && caseRes != nil && eErr == nil {
		d, usage, dErr := s.ai.Debrief.Generate(ctx, caseRes.Scenario, events)
		if dErr != nil {
			fmt.Println("debrief AI generation failed:", dErr)
		} else {
			debrief = d
		}
		userId := ""
		if session.UserId != nil {
			userId = *session.UserId
		}
		if session.CaseId != nil {
			logAIUsage(&s.storage, "debrief", usage, userId, *session.CaseId)
		}
	}

	finalScore := healthPercent
	if debrief != nil {
		finalScore -= len(debrief.IncorrectSteps) * incorrectStepPenaltyPct
		if finalScore < 0 {
			finalScore = 0
		}
	}

	xp, coins := 0, 0
	if actionCount > 0 {
		xp = xpBase * finalScore / 100
		if req.Reason != "health_zero" {
			coins = coinBase * finalScore / 100
		}
	}

	res, err := s.storage.SimulationS.Finish(req, finalScore, xp, coins)
	if err != nil {
		return nil, err
	}

	if session.UserId != nil {
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

	debriefReady := false
	if debrief != nil {
		debrief.SessionId = req.SessionId
		if _, dcErr := s.storage.DebriefS.Create(debrief); dcErr != nil {
			fmt.Println("failed to save debrief report:", dcErr)
		} else {
			debriefReady = true
		}
	}

	res.DebriefReady = debriefReady
	return res, nil
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
