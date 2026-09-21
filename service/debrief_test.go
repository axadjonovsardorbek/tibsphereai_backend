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

func newDebriefTestService(dm *mocks.DebriefMock) *DebriefService {
	return NewDebriefService(&st.Storage{DebriefS: dm})
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
