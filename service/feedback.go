package service

import (
	"context"
	"errors"
	"strings"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

// FeedbackService - foydalanuvchilar fikri (TZ_MVP: "Userlarning fikrini olish").
type FeedbackService struct {
	storage st.Storage
}

func NewFeedbackService(storage *st.Storage) *FeedbackService {
	return &FeedbackService{storage: *storage}
}

var feedbackKinds = map[string]bool{"general": true, "bug": true, "idea": true, "content": true, "nps": true}

func (s *FeedbackService) Create(ctx context.Context, req *m.FeedbackCreateReq) (*m.ById, error) {
	if req.Kind == "" {
		req.Kind = "general"
	}
	if !feedbackKinds[req.Kind] {
		return nil, errors.New("kind must be general|bug|idea|content|nps")
	}
	if req.Rating < 0 || req.Rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}
	req.Message = truncateRunes(strings.TrimSpace(req.Message), 2000)
	req.Context = truncateRunes(strings.TrimSpace(req.Context), 100)
	req.AppVersion = truncateRunes(strings.TrimSpace(req.AppVersion), 40)
	if req.Rating == 0 && req.Message == "" {
		return nil, errors.New("rating or message is required")
	}
	return s.storage.FeedbackS.Create(req)
}

func (s *FeedbackService) GetAll(ctx context.Context, req *m.FeedbackListReq) (*m.PaginationResponse, *m.FeedbackSummary, error) {
	resp, _, sum, err := s.storage.FeedbackS.GetAll(req)
	return resp, sum, err
}

func (s *FeedbackService) SetStatus(ctx context.Context, id, status string) error {
	if status != "new" && status != "seen" && status != "done" {
		return errors.New("status must be new|seen|done")
	}
	return s.storage.FeedbackS.SetStatus(id, status)
}
