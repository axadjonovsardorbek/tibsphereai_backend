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

func newStudyPlanTestService(sm *mocks.StudyPlanMock) *StudyPlanService {
	return NewStudyPlanService(&st.Storage{StudyPlanS: sm}, nil)
}

func TestStudyPlanService_Upsert(t *testing.T) {
	sm := mocks.NewStudyPlanMock()
	svc := newStudyPlanTestService(sm)

	req := &m.StudyPlanUpsertReq{}
	sm.On("Upsert", req).Return(&m.Void{}, nil)

	_, err := svc.Upsert(context.Background(), req)
	require.NoError(t, err)
}

func TestStudyPlanService_GetByUserId(t *testing.T) {
	sm := mocks.NewStudyPlanMock()
	svc := newStudyPlanTestService(sm)

	want := &m.StudyPlanRes{}
	sm.On("GetByUserId", "user-1").Return(want, nil)

	got, err := svc.GetByUserId(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestStudyPlanService_GetByUserId_Error(t *testing.T) {
	sm := mocks.NewStudyPlanMock()
	svc := newStudyPlanTestService(sm)

	wantErr := errors.New("not found")
	sm.On("GetByUserId", "user-1").Return(nil, wantErr)

	got, err := svc.GetByUserId(context.Background(), "user-1")
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}
