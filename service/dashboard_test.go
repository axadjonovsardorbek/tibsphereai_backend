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

func newDashboardTestService(dm *mocks.DashBoardMock) *DashboardService {
	return NewDashboardService(&st.Storage{DashboardS: dm}, nil)
}

func TestDashboardService_GetDashboard(t *testing.T) {
	dm := mocks.NewDashBoardMock()
	svc := newDashboardTestService(dm)

	req := &m.GetDashboardReq{Type: "weekly"}
	want := &m.DashboardRes{}
	dm.On("GetDashboard", req).Return(want, nil)

	got, err := svc.GetDashboard(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestDashboardService_GetDashboard_Error(t *testing.T) {
	dm := mocks.NewDashBoardMock()
	svc := newDashboardTestService(dm)

	req := &m.GetDashboardReq{Type: "weekly"}
	wantErr := errors.New("db down")
	dm.On("GetDashboard", req).Return(nil, wantErr)

	got, err := svc.GetDashboard(context.Background(), req)
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}
