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

func newFavoriteTestService(fm *mocks.FavoriteMock) *FavoriteService {
	return NewFavoriteService(&st.Storage{FavoriteS: fm})
}

func TestFavoriteService_Toggle(t *testing.T) {
	fm := mocks.NewFavoriteMock()
	svc := newFavoriteTestService(fm)

	req := &m.FavoriteToggleReq{}
	want := &m.FavoriteToggleRes{}
	fm.On("Toggle", req).Return(want, nil)

	got, err := svc.Toggle(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestFavoriteService_Toggle_Error(t *testing.T) {
	fm := mocks.NewFavoriteMock()
	svc := newFavoriteTestService(fm)

	req := &m.FavoriteToggleReq{}
	wantErr := errors.New("db error")
	fm.On("Toggle", req).Return(nil, wantErr)

	_, err := svc.Toggle(context.Background(), req)
	assert.ErrorIs(t, err, wantErr)
}

func TestFavoriteService_GetAll(t *testing.T) {
	fm := mocks.NewFavoriteMock()
	svc := newFavoriteTestService(fm)

	req := &m.FavoriteGetListReq{}
	wantPag := &m.PaginationResponse{Total: 2}
	wantList := &m.CaseMobileList{}
	fm.On("GetAll", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetAll(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Equal(t, wantList, list)
}
