package service

import (
	"context"
	"errors"
	"testing"

	"tibsphereai/internal/ai"
	m "tibsphereai/pkg/models"
	"tibsphereai/storage/mocks"
	st "tibsphereai/storage/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// testSimulationDB - SimulationService.hasActiveSubscription() xom SQL orqali
// to'g'ridan-to'g'ri storage.Db (*pgxpool.Pool) bilan ishlaydi, mock qilib
// bo'lmaydi - shu sababli mahalliy docker-compose Postgres'ga ulanamiz (docker
// stack ishlamasa test avtomatik skip qilinadi).
func testSimulationDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "host=localhost port=5473 user=postgres password=1111 dbname=tibsphereai sslmode=disable")
	if err != nil {
		t.Skip("skipping: local postgres (docker-compose) not reachable: " + err.Error())
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skip("skipping: local postgres (docker-compose) not reachable: " + err.Error())
	}
	t.Cleanup(pool.Close)
	return pool
}

type fakePatientAI struct {
	reply string
	err   error
}

func (f fakePatientAI) Reply(ctx context.Context, caseScenario map[string]any, expectedAnswer, question string) (*ai.PatientReplyRes, *ai.Usage, error) {
	if f.err != nil {
		return nil, nil, f.err
	}
	return &ai.PatientReplyRes{PatientReply: f.reply}, nil, nil
}

type fakeDebriefAI struct {
	res *m.DebriefReportCreateReq
	err error
}

func (f fakeDebriefAI) Generate(ctx context.Context, caseScenario map[string]any, events []m.SimulationEventReq) (*m.DebriefReportCreateReq, *ai.Usage, error) {
	return f.res, nil, f.err
}

func newSimulationTestService(t *testing.T, sm *mocks.SimulationMock, cm *mocks.CaseMock, um *mocks.UserMock, dm *mocks.DebriefMock, am *mocks.AppSettingMock, aiClient *ai.Client) *SimulationService {
	pool := testSimulationDB(t)
	pgStorage := &st.Storage{Db: pool, SimulationS: sm, CaseS: cm, UserS: um, DebriefS: dm, AppSettingS: am}
	appSettings := NewAppSettingService(pgStorage, 3)
	if aiClient == nil {
		aiClient = &ai.Client{}
	}
	return NewSimulationService(pgStorage, aiClient, appSettings)
}

// Start - hech qanday faol obunasi (real DB'da subscriptions yozuvi yo'q, tasodifiy
// UUID) va kunlik bepul urinishi ham bo'lmagan foydalanuvchi uchun ErrDailyLimitReached
// qaytishi va SimulationS.Start umuman chaqirilmasligi kerak.
func TestSimulationService_Start_NoSubscription_LimitReached(t *testing.T) {
	sm := mocks.NewSimulationMock()
	am := mocks.NewAppSettingMock()
	svc := newSimulationTestService(t, sm, mocks.NewCaseMock(), mocks.NewUserMock(), mocks.NewDebriefMock(), am, nil)

	userId := uuid.NewString()
	am.On("Get", m.AppSettingDailyFreeLimit).Return(&m.AppSettingRes{Value: "3"}, nil)
	am.On("GetTodayFreeAttemptsUsed", userId).Return(3, nil)

	req := &m.SimulationStartReq{UserId: userId, CaseId: "case-1"}
	got, err := svc.Start(context.Background(), req)

	assert.Nil(t, got)
	assert.Error(t, err)
	sm.AssertNotCalled(t, "Start", mock.Anything)
}

// Start - obunasi yo'q, lekin bepul urinish qolgan bo'lsa - Start chaqiriladi va
// urinish sarflanadi (ConsumeFreeAttempt).
func TestSimulationService_Start_NoSubscription_HasFreeAttempt(t *testing.T) {
	sm := mocks.NewSimulationMock()
	am := mocks.NewAppSettingMock()
	svc := newSimulationTestService(t, sm, mocks.NewCaseMock(), mocks.NewUserMock(), mocks.NewDebriefMock(), am, nil)

	userId := uuid.NewString()
	am.On("Get", m.AppSettingDailyFreeLimit).Return(&m.AppSettingRes{Value: "3"}, nil)
	am.On("GetTodayFreeAttemptsUsed", userId).Return(1, nil)
	am.On("IncrTodayFreeAttempts", userId).Return(nil)

	req := &m.SimulationStartReq{UserId: userId, CaseId: "case-1"}
	want := &m.SimulationStartRes{SessionId: "sess-1"}
	sm.On("Start", req).Return(want, nil)

	got, err := svc.Start(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	am.AssertExpectations(t)
}

func TestSimulationService_Start_StorageError(t *testing.T) {
	sm := mocks.NewSimulationMock()
	am := mocks.NewAppSettingMock()
	svc := newSimulationTestService(t, sm, mocks.NewCaseMock(), mocks.NewUserMock(), mocks.NewDebriefMock(), am, nil)

	userId := uuid.NewString()
	am.On("Get", m.AppSettingDailyFreeLimit).Return(&m.AppSettingRes{Value: "3"}, nil)
	am.On("GetTodayFreeAttemptsUsed", userId).Return(0, nil)

	req := &m.SimulationStartReq{UserId: userId, CaseId: "case-1"}
	wantErr := errors.New("case not found")
	sm.On("Start", req).Return(nil, wantErr)

	got, err := svc.Start(context.Background(), req)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
	am.AssertNotCalled(t, "IncrTodayFreeAttempts", mock.Anything)
}

// AddEvent - "question" turida bemor AI javobi qo'shilishi kerak (Anamnez).
func TestSimulationService_AddEvent_Question_AttachesAIReply(t *testing.T) {
	sm := mocks.NewSimulationMock()
	cm := mocks.NewCaseMock()
	aiClient := &ai.Client{Patient: fakePatientAI{reply: "Boshim og'riyapti"}}
	svc := newSimulationTestService(t, sm, cm, mocks.NewUserMock(), mocks.NewDebriefMock(), mocks.NewAppSettingMock(), aiClient)

	req := &m.SimulationEventReq{SessionId: "sess-1", Type: "question", Payload: map[string]any{"question": "Qayerda og'riq bor?"}}
	eventRes := &m.SimulationEventRes{Response: map[string]any{}}
	sm.On("AddEvent", req, true, 0).Return(eventRes, nil)

	caseId := "case-1"
	sm.On("GetById", &m.ById{Id: "sess-1"}).Return(&m.SimulationSessionRes{CaseId: &caseId}, nil)
	cm.On("GetById", &m.ById{Id: "case-1"}).Return(&m.CaseRes{}, nil)

	got, err := svc.AddEvent(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Boshim og'riyapti", got.Response["patient_reply"])
}

// AddEvent - "question" bo'lmagan turlarda ActionEvaluatorAI orqali klinik
// baho olinadi va health_delta/is_correct shu bahoga mos qo'yiladi.
func TestSimulationService_AddEvent_NonQuestion_AppliesAIEval(t *testing.T) {
	sm := mocks.NewSimulationMock()
	cm := mocks.NewCaseMock()
	aiClient := &ai.Client{Action: fakeActionEvalAI{res: &ai.ActionEvalRes{IsCorrect: false, HealthDelta: -15, Feedback: "Vital ko'rsatkichlar baholanmadi"}}}
	svc := newSimulationTestService(t, sm, cm, mocks.NewUserMock(), mocks.NewDebriefMock(), mocks.NewAppSettingMock(), aiClient)

	req := &m.SimulationEventReq{SessionId: "sess-1", Type: "lab", Payload: map[string]any{"name": "Troponin"}}
	eventRes := &m.SimulationEventRes{Response: map[string]any{}}
	sm.On("AddEvent", req, false, -15).Return(eventRes, nil)

	caseId := "case-1"
	sm.On("GetById", &m.ById{Id: "sess-1"}).Return(&m.SimulationSessionRes{CaseId: &caseId}, nil)
	cm.On("GetById", &m.ById{Id: "case-1"}).Return(&m.CaseRes{}, nil)

	got, err := svc.AddEvent(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Vital ko'rsatkichlar baholanmadi", got.Response["feedback"])
}

func TestSimulationService_AddEvent_StorageError(t *testing.T) {
	sm := mocks.NewSimulationMock()
	svc := newSimulationTestService(t, sm, mocks.NewCaseMock(), mocks.NewUserMock(), mocks.NewDebriefMock(), mocks.NewAppSettingMock(), nil)

	req := &m.SimulationEventReq{SessionId: "sess-1", Type: "lab"}
	wantErr := errors.New("session not found")
	sm.On("GetById", &m.ById{Id: "sess-1"}).Return(&m.SimulationSessionRes{}, nil)
	sm.On("AddEvent", req, true, 0).Return(nil, wantErr)

	got, err := svc.AddEvent(context.Background(), req)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}

// fakeActionEvalAI - ai.ActionEvaluatorAI ning test uchun soxta implementatsiyasi.
type fakeActionEvalAI struct {
	res *ai.ActionEvalRes
	err error
}

func (f fakeActionEvalAI) Evaluate(ctx context.Context, caseScenario map[string]any, expectedAnswer, eventType string, payload map[string]any) (*ai.ActionEvalRes, *ai.Usage, error) {
	return f.res, nil, f.err
}

// Finish - sessiya yakunlanganda mukofot case.difficulty bo'yicha bazaviy
// XP/Coin'ga (app_settings, sozlanmagan bo'lsa medium default: xp=100,coin=10)
// health_percent nisbatan mutanosib hisoblanadi, balansga qo'shiladi va faoliyat
// (streak) loglanadi (BACKEND_STRUCTURE.md 6-bo'lim).
func TestSimulationService_Finish_AwardsPointsAndLogsActivity(t *testing.T) {
	sm := mocks.NewSimulationMock()
	um := mocks.NewUserMock()
	cm := mocks.NewCaseMock()
	am := mocks.NewAppSettingMock()
	svc := newSimulationTestService(t, sm, cm, um, mocks.NewDebriefMock(), am, nil)

	userId := "user-1"
	caseId := "case-1"
	health := 50
	sm.On("GetById", &m.ById{Id: "sess-1"}).Return(&m.SimulationSessionRes{UserId: &userId, CaseId: &caseId, HealthPercent: &health}, nil)

	difficulty := "medium"
	cm.On("GetById", &m.ById{Id: "case-1"}).Return(&m.CaseRes{Difficulty: &difficulty}, nil)
	am.On("Get", mock.Anything).Return(nil, errors.New("not set")).Maybe() // -> medium default (xp=100, coin=10)

	// kamida bitta klinik harakat bo'lishi kerak (aks holda mukofot 0 - pastdagi
	// TestSimulationService_Finish_NoActions_NoReward'ga qarang).
	sm.On("GetEvents", "sess-1").Return([]m.SimulationEventReq{{Type: "medication"}}, nil)

	req := &m.SimulationFinishReq{SessionId: "sess-1", Reason: "manual"}
	// xp = 100 * 50/100 = 50, coins = 10 * 50/100 = 5
	finishRes := &m.SimulationFinishRes{SessionId: "sess-1", XPEarned: 50, CoinsEarned: 5}
	sm.On("Finish", req, 50, 5).Return(finishRes, nil)

	um.On("AddPoints", &m.AddPointsReq{UserId: "user-1", XPDelta: 50, CoinsDelta: 5, Reason: "simulation_finish", ReferenceId: "sess-1"}).
		Return(&m.Void{}, nil).Maybe()
	um.On("CreateActivity", &m.UserActivityCreateReq{UserId: "user-1", Activity: 1}).Return(&m.Void{}, nil).Maybe()

	got, err := svc.Finish(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "sess-1", got.SessionId)
	assert.Equal(t, 50, got.XPEarned)
	assert.Equal(t, 5, got.CoinsEarned)
	assert.False(t, got.DebriefReady) // TODO: WS/poll orqali xabar qilinadi, hozircha doim false
}

// Finish - hech qanday klinik harakat qilinmagan bo'lsa (faqat anamnez "question"
// yoki umuman hech narsa), xp/coins 0 bo'lishi kerak - health_percent boshlang'ich
// 100'dan boshlangani uchun aks holda darhol "finish" bosib, hech narsa qilmasdan
// to'liq mukofot olib ketish mumkin bo'lardi.
func TestSimulationService_Finish_NoActions_NoReward(t *testing.T) {
	sm := mocks.NewSimulationMock()
	am := mocks.NewAppSettingMock()
	svc := newSimulationTestService(t, sm, mocks.NewCaseMock(), mocks.NewUserMock(), mocks.NewDebriefMock(), am, nil)

	health := 100
	sm.On("GetById", &m.ById{Id: "sess-1"}).Return(&m.SimulationSessionRes{HealthPercent: &health}, nil)
	am.On("Get", mock.Anything).Return(nil, errors.New("not set")).Maybe()
	sm.On("GetEvents", "sess-1").Return([]m.SimulationEventReq{{Type: "question"}}, nil)

	req := &m.SimulationFinishReq{SessionId: "sess-1", Reason: "manual"}
	sm.On("Finish", req, 0, 0).Return(&m.SimulationFinishRes{SessionId: "sess-1"}, nil)

	got, err := svc.Finish(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 0, got.XPEarned)
	assert.Equal(t, 0, got.CoinsEarned)
}

// Finish - health_zero (health 0'ga tushib avtomatik tugagan) sabab bo'lsa,
// tanga umuman berilmaydi (coins=0), XP baribir health_percent nisbatida hisoblanadi.
func TestSimulationService_Finish_HealthZero_NoCoins(t *testing.T) {
	sm := mocks.NewSimulationMock()
	am := mocks.NewAppSettingMock()
	svc := newSimulationTestService(t, sm, mocks.NewCaseMock(), mocks.NewUserMock(), mocks.NewDebriefMock(), am, nil)

	health := 0
	sm.On("GetById", &m.ById{Id: "sess-1"}).Return(&m.SimulationSessionRes{HealthPercent: &health}, nil)
	am.On("Get", mock.Anything).Return(nil, errors.New("not set")).Maybe()
	sm.On("GetEvents", "sess-1").Return([]m.SimulationEventReq{{Type: "medication"}}, nil)

	req := &m.SimulationFinishReq{SessionId: "sess-1", Reason: "health_zero"}
	sm.On("Finish", req, 0, 0).Return(&m.SimulationFinishRes{SessionId: "sess-1"}, nil)

	got, err := svc.Finish(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 0, got.CoinsEarned)
}

func TestSimulationService_Finish_StorageError(t *testing.T) {
	sm := mocks.NewSimulationMock()
	am := mocks.NewAppSettingMock()
	svc := newSimulationTestService(t, sm, mocks.NewCaseMock(), mocks.NewUserMock(), mocks.NewDebriefMock(), am, nil)

	health := 100
	sm.On("GetById", &m.ById{Id: "sess-1"}).Return(&m.SimulationSessionRes{HealthPercent: &health}, nil)
	am.On("Get", mock.Anything).Return(nil, errors.New("not set")).Maybe()
	sm.On("GetEvents", "sess-1").Return([]m.SimulationEventReq{{Type: "exam"}}, nil)

	req := &m.SimulationFinishReq{SessionId: "sess-1"}
	wantErr := errors.New("already finished")
	sm.On("Finish", req, 100, 10).Return(nil, wantErr)

	got, err := svc.Finish(context.Background(), req)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}

func TestSimulationService_GetById(t *testing.T) {
	sm := mocks.NewSimulationMock()
	svc := newSimulationTestService(t, sm, mocks.NewCaseMock(), mocks.NewUserMock(), mocks.NewDebriefMock(), mocks.NewAppSettingMock(), nil)

	req := &m.ById{Id: "sess-1"}
	want := &m.SimulationSessionRes{}
	sm.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestSimulationService_GetOngoing(t *testing.T) {
	sm := mocks.NewSimulationMock()
	svc := newSimulationTestService(t, sm, mocks.NewCaseMock(), mocks.NewUserMock(), mocks.NewDebriefMock(), mocks.NewAppSettingMock(), nil)

	req := &m.ById{Id: "user-1"}
	wantPag := &m.PaginationResponse{Total: 1}
	wantList := &m.SimulationSessionList{}
	sm.On("GetOngoing", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetOngoing(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Equal(t, wantList, list)
}

func TestSimulationService_GetCompleted(t *testing.T) {
	sm := mocks.NewSimulationMock()
	svc := newSimulationTestService(t, sm, mocks.NewCaseMock(), mocks.NewUserMock(), mocks.NewDebriefMock(), mocks.NewAppSettingMock(), nil)

	req := &m.ById{Id: "user-1"}
	wantPag := &m.PaginationResponse{Total: 1}
	wantList := &m.SimulationSessionList{}
	sm.On("GetCompleted", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetCompleted(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Equal(t, wantList, list)
}
