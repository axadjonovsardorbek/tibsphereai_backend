package service

import (
	"context"
	"errors"
	"testing"

	m "tibsphereai/pkg/models"
	"tibsphereai/storage/mocks"
	st "tibsphereai/storage/postgres"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAIPromptTestService(pm *mocks.AIPromptMock, lm *mocks.AIUsageLogMock) *AIPromptService {
	return NewAIPromptService(&st.Storage{AIPromptS: pm, AIUsageLogS: lm}, nil)
}

func TestAIPromptService_GetAll(t *testing.T) {
	pm := mocks.NewAIPromptMock()
	svc := newAIPromptTestService(pm, mocks.NewAIUsageLogMock())

	want := []m.AIPromptRes{{Key: "patient_persona"}}
	pm.On("GetAll").Return(want, nil)

	got, err := svc.GetAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestAIPromptService_GetAll_Error(t *testing.T) {
	pm := mocks.NewAIPromptMock()
	svc := newAIPromptTestService(pm, mocks.NewAIUsageLogMock())

	wantErr := errors.New("db down")
	pm.On("GetAll").Return(nil, wantErr)

	got, err := svc.GetAll(context.Background())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}

func TestAIPromptService_Upsert(t *testing.T) {
	pm := mocks.NewAIPromptMock()
	svc := newAIPromptTestService(pm, mocks.NewAIUsageLogMock())

	req := &m.AIPromptUpsertReq{}
	pm.On("Upsert", req).Return(&m.Void{}, nil)

	_, err := svc.Upsert(context.Background(), req)
	require.NoError(t, err)
}

func TestAIPromptService_GetUsageSummary(t *testing.T) {
	lm := mocks.NewAIUsageLogMock()
	svc := newAIPromptTestService(mocks.NewAIPromptMock(), lm)

	want := &m.AIUsageSummaryRes{}
	lm.On("GetSummary").Return(want, nil)

	got, err := svc.GetUsageSummary(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}
