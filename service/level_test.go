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

func newLevelTestService(lm *mocks.LevelMock) *LevelService {
	return NewLevelService(&st.Storage{LevelS: lm})
}

func TestLevelService_Create(t *testing.T) {
	lm := mocks.NewLevelMock()
	svc := newLevelTestService(lm)

	req := &m.LevelCreateReq{}
	want := &m.ById{Id: "level-1"}
	lm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestLevelService_GetAll(t *testing.T) {
	lm := mocks.NewLevelMock()
	svc := newLevelTestService(lm)

	want := []m.LevelRes{{}, {}}
	lm.On("GetAll").Return(want, nil)

	got, err := svc.GetAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestLevelService_GetAll_Error(t *testing.T) {
	lm := mocks.NewLevelMock()
	svc := newLevelTestService(lm)

	wantErr := errors.New("db down")
	lm.On("GetAll").Return(nil, wantErr)

	got, err := svc.GetAll(context.Background())
	assert.Nil(t, got)
	assert.ErrorIs(t, err, wantErr)
}

func TestLevelService_Update(t *testing.T) {
	lm := mocks.NewLevelMock()
	svc := newLevelTestService(lm)

	req := &m.LevelUpdateReq{}
	lm.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
}

func TestLevelService_Delete(t *testing.T) {
	lm := mocks.NewLevelMock()
	svc := newLevelTestService(lm)

	req := &m.ById{Id: "level-1"}
	lm.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}
