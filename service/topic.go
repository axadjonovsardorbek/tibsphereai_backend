package service

import (
	"context"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

type TopicService struct {
	storage st.Storage
}

func NewTopicService(storage *st.Storage) *TopicService {
	return &TopicService{storage: *storage}
}

func (s *TopicService) Create(ctx context.Context, req *m.TopicCreateReq) (*m.ById, error) {
	return s.storage.TopicS.Create(req)
}
func (s *TopicService) GetById(ctx context.Context, req *m.ById) (*m.TopicRes, error) {
	return s.storage.TopicS.GetById(req)
}
func (s *TopicService) GetAll(ctx context.Context, req *m.TopicGetListReq) (*m.PaginationResponse, *m.TopicList, error) {
	return s.storage.TopicS.GetAll(req)
}
func (s *TopicService) Update(ctx context.Context, req *m.TopicUpdateReq) (*m.Void, error) {
	return s.storage.TopicS.Update(req)
}
func (s *TopicService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.TopicS.Delete(req)
}
func (s *TopicService) GetAllMobile(ctx context.Context, req *m.TopicGetListMobileReq) (*m.PaginationResponse, *m.TopicMobileList, error) {
	return s.storage.TopicS.GetAllMobile(req)
}
