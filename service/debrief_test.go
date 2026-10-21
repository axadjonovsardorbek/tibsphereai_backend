package service

import (
	"context"
	"errors"
	"testing"

	"tibsphereai/internal/ai"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"
	"tibsphereai/storage/mocks"
	st "tibsphereai/storage/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDebriefTestService(dm *mocks.DebriefMock) *DebriefService {
	return NewDebriefService(&st.Storage{DebriefS: dm}, nil)
}

func TestDebriefService_GetBySessionId(t *testing.T) {
	dm := mocks.NewDebriefMock()
	svc := newDebriefTestService(dm)

	want := &m.DebriefReportRes{}
	dm.On("GetBySessionId", "session-1").Return(want, nil)

	got, err := svc.GetBySessionId(context.Background(), "session-1")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDebriefService_GetBySessionId_Error(t *testing.T) {
	dm := mocks.NewDebriefMock()
	svc := newDebriefTestService(dm)

	wantErr := errors.New("not found")
	dm.On("GetBySessionId", "session-1").Return(nil, wantErr)

	got, err := svc.GetBySessionId(context.Background(), "session-1")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}

func strp(s string) *string { return &s }

func newDebriefSelfHealService(dm *mocks.DebriefMock, sm *mocks.SimulationMock, cm *mocks.CaseMock, d ai.DebriefAI) *DebriefService {
	return NewDebriefService(&st.Storage{DebriefS: dm, SimulationS: sm, CaseS: cm}, &ai.Client{Debrief: d})
}

// Sessiya hali faol - debrief_not_ready.
func TestDebriefService_ActiveSession_NotReady(t *testing.T) {
	dm, sm := mocks.NewDebriefMock(), mocks.NewSimulationMock()
	svc := newDebriefSelfHealService(dm, sm, mocks.NewCaseMock(), nil)
	dm.On("GetBySessionId", "s1").Return(nil, st.ErrDebriefNotFound)
	sm.On("GetById", &m.ById{Id: "s1"}).Return(&m.SimulationSessionRes{Status: strp("active")}, nil)

	_, err := svc.GetBySessionId(context.Background(), "s1")
	assert.ErrorIs(t, err, errs.ErrDebriefNotReady)
}

// Sessiya yopilgan, hisobot yo'q, AI xato - debrief_failed.
func TestDebriefService_EndedNoReport_AIFails(t *testing.T) {
	dm, sm, cm := mocks.NewDebriefMock(), mocks.NewSimulationMock(), mocks.NewCaseMock()
	svc := newDebriefSelfHealService(dm, sm, cm, fakeDebriefAI{err: errors.New("gemini 401")})
	dm.On("GetBySessionId", "s1").Return(nil, st.ErrDebriefNotFound)
	sm.On("GetById", &m.ById{Id: "s1"}).Return(&m.SimulationSessionRes{Status: strp("failed"), CaseId: strp("c1")}, nil)
	cm.On("GetById", &m.ById{Id: "c1"}).Return(&m.CaseRes{}, nil)
	sm.On("GetEvents", "s1").Return([]m.SimulationEventReq{{Type: "question"}}, nil)

	_, err := svc.GetBySessionId(context.Background(), "s1")
	assert.ErrorIs(t, err, errs.ErrDebriefFailed)
	dm.AssertNotCalled(t, "Create")
}

// Sessiya yopilgan, hisobot yo'q, AI ishlaydi - qayta generatsiya qilinib 200 qaytadi.
func TestDebriefService_EndedNoReport_Regenerates(t *testing.T) {
	dm, sm, cm := mocks.NewDebriefMock(), mocks.NewSimulationMock(), mocks.NewCaseMock()
	report := &m.DebriefReportCreateReq{CorrectSteps: []string{"a"}}
	svc := newDebriefSelfHealService(dm, sm, cm, fakeDebriefAI{res: report})
	want := &m.DebriefReportRes{}
	dm.On("GetBySessionId", "s1").Return(nil, st.ErrDebriefNotFound).Once()
	dm.On("GetBySessionId", "s1").Return(want, nil).Once()
	sm.On("GetById", &m.ById{Id: "s1"}).Return(&m.SimulationSessionRes{Status: strp("failed"), CaseId: strp("c1")}, nil)
	cm.On("GetById", &m.ById{Id: "c1"}).Return(&m.CaseRes{}, nil)
	sm.On("GetEvents", "s1").Return([]m.SimulationEventReq{{Type: "question"}}, nil)
	dm.On("Create", report).Return(&m.Void{}, nil)

	got, err := svc.GetBySessionId(context.Background(), "s1")
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, "s1", report.SessionId)
}
