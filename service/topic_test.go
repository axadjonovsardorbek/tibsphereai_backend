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

func newTopicTestService(tm *mocks.TopicMock) *TopicService {
	return NewTopicService(&st.Storage{TopicS: tm})
}

func TestTopicService_Create(t *testing.T) {
	tm := mocks.NewTopicMock()
	svc := newTopicTestService(tm)

	req := &m.TopicCreateReq{}
	want := &m.ById{Id: "topic-1"}
	tm.On("Create", req).Return(want, nil)

	got, err := svc.Create(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestTopicService_Create_Error(t *testing.T) {
	tm := mocks.NewTopicMock()
	svc := newTopicTestService(tm)

	req := &m.TopicCreateReq{}
	wantErr := errors.New("db error")
	tm.On("Create", req).Return(nil, wantErr)

	_, err := svc.Create(context.Background(), req)
	assert.ErrorIs(t, err, wantErr)
}

func TestTopicService_GetById(t *testing.T) {
	tm := mocks.NewTopicMock()
	svc := newTopicTestService(tm)

	req := &m.ById{Id: "topic-1"}
	want := &m.TopicRes{}
	tm.On("GetById", req).Return(want, nil)

	got, err := svc.GetById(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestTopicService_GetAll(t *testing.T) {
	tm := mocks.NewTopicMock()
	svc := newTopicTestService(tm)

	req := &m.TopicGetListReq{}
	wantPag := &m.PaginationResponse{Total: 2}
	wantList := &m.TopicList{}
	tm.On("GetAll", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetAll(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Equal(t, wantList, list)
}

func TestTopicService_Update(t *testing.T) {
	tm := mocks.NewTopicMock()
	svc := newTopicTestService(tm)

	req := &m.TopicUpdateReq{}
	tm.On("Update", req).Return(&m.Void{}, nil)

	_, err := svc.Update(context.Background(), req)
	require.NoError(t, err)
}

func TestTopicService_Delete(t *testing.T) {
	tm := mocks.NewTopicMock()
	svc := newTopicTestService(tm)

	req := &m.ById{Id: "topic-1"}
	tm.On("Delete", req).Return(&m.Void{}, nil)

	_, err := svc.Delete(context.Background(), req)
	require.NoError(t, err)
}

func TestTopicService_GetAllMobile(t *testing.T) {
	tm := mocks.NewTopicMock()
	svc := newTopicTestService(tm)

	req := &m.TopicGetListMobileReq{}
	wantPag := &m.PaginationResponse{Total: 1}
	wantList := &m.TopicMobileList{}
	tm.On("GetAllMobile", req).Return(wantPag, wantList, nil)

	pag, list, err := svc.GetAllMobile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, wantPag, pag)
	assert.Equal(t, wantList, list)
}
