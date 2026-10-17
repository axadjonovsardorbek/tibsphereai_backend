package service

import (
	"context"
	"errors"
	"fmt"
	"tibsphereai/internal/ai"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
)

type CaseService struct {
	storage st.Storage
	ai      *ai.Client
}

func NewCaseService(storage *st.Storage, aiClient *ai.Client) *CaseService {
	return &CaseService{storage: *storage, ai: aiClient}
}

func (s *CaseService) Create(ctx context.Context, req *m.CaseCreateReq) (*m.ById, error) {
	return s.storage.CaseS.Create(req)
}

// GenerateWithAI - "AI yordamida avtomatik keys yaratish" (BACKEND_STRUCTURE.md 7-bo'lim):
// admin mavzu va darajani kiritadi, Gemini AI (yoki GEMINI_API_KEY yo'q bo'lsa xatolik)
// to'liq ssenariyni tayyorlaydi, admin ko'rib chiqib alohida Publish bilan tasdiqlaydi.
func (s *CaseService) GenerateWithAI(ctx context.Context, genReq *m.CaseGenReq) (*m.ById, error) {
	if genReq.TopicId != "" {
		topic, err := s.storage.TopicS.GetById(&m.ById{Id: genReq.TopicId})
		if err != nil {
			return nil, err
		}
		if genReq.Topic == "" && topic != nil {
			genReq.Topic = pickTopicName(topic.Name)
		}
	}
	if s.storage.CaseSourceS != nil {
		contents, err := s.storage.CaseSourceS.GetContents(genReq.SourceIds)
		if err != nil {
			return nil, err
		}
		genReq.Sources = contents
	}
	req, usage, err := s.ai.CaseGen.Generate(ctx, genReq)
	logAIUsage(&s.storage, "case_generation", usage, "", "")
	if err != nil {
		return nil, err
	}
	req.IsAIGenerated = true
	if genReq.TopicId != "" {
		req.TopicId = genReq.TopicId
	}
	return s.storage.CaseS.Create(req)
}

// maxBatchCases - bitta ommaviy so'rovda yaratiladigan case'lar soni chegarasi
// (generatsiya ketma-ket, sinxron bajariladi - HTTP so'rov uzoq osilib qolmasligi uchun).
const maxBatchCases = 20

func pickTopicName(name map[string]string) string {
	for _, k := range []string{"uz", "ru", "en"} {
		if v := name[k]; v != "" {
			return v
		}
	}
	for _, v := range name {
		if v != "" {
			return v
		}
	}
	return ""
}

// GenerateBatch - tanlangan mavzular (yoki butun bo'lim mavzulari) uchun ketma-ket
// AI case qoralamalarini yaratadi. Bitta mavzudagi xato qolganlarini to'xtatmaydi.
func (s *CaseService) GenerateBatch(ctx context.Context, req *m.CaseBatchGenReq) (*m.CaseBatchGenRes, error) {
	count := req.CountPerTopic
	if count <= 0 {
		count = 1
	}
	topicIds := req.TopicIds
	if len(topicIds) == 0 {
		if req.CategoryId == "" {
			return nil, errors.New("category_id or topic_ids is required")
		}
		_, list, err := s.storage.TopicS.GetAll(&m.TopicGetListReq{CategoryId: req.CategoryId, Filter: m.Pagination{Page: 0, Limit: 500}})
		if err != nil {
			return nil, err
		}
		for _, t := range list.Topics {
			if t.Id != nil {
				topicIds = append(topicIds, *t.Id)
			}
		}
	}
	if len(topicIds) == 0 {
		return nil, errors.New("no topics found")
	}
	if len(topicIds)*count > maxBatchCases {
		return nil, fmt.Errorf("too many cases requested (%d), max %d per request", len(topicIds)*count, maxBatchCases)
	}

	res := &m.CaseBatchGenRes{}
	for _, tid := range topicIds {
		for i := 0; i < count; i++ {
			item := m.CaseBatchGenItem{TopicId: tid}
			created, err := s.GenerateWithAI(ctx, &m.CaseGenReq{
				TopicId: tid, Difficulty: req.Difficulty, SourceIds: req.SourceIds,
			})
			if err != nil {
				item.Error = err.Error()
				res.Failed++
			} else {
				item.CaseId = created.Id
				res.Created++
			}
			res.Items = append(res.Items, item)
		}
	}
	return res, nil
}

func (s *CaseService) GetById(ctx context.Context, req *m.ById) (*m.CaseRes, error) {
	return s.storage.CaseS.GetById(req)
}
func (s *CaseService) GetAll(ctx context.Context, req *m.CaseGetListReq) (*m.PaginationResponse, *m.CaseList, error) {
	return s.storage.CaseS.GetAll(req)
}
func (s *CaseService) Update(ctx context.Context, req *m.CaseUpdateReq) (*m.Void, error) {
	return s.storage.CaseS.Update(req)
}
func (s *CaseService) Delete(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.CaseS.Delete(req)
}
func (s *CaseService) Publish(ctx context.Context, req *m.ById) (*m.Void, error) {
	return s.storage.CaseS.Publish(req)
}
func (s *CaseService) GetByIdMobile(ctx context.Context, req *m.ByIdMobile) (*m.CaseMobileRes, error) {
	return s.storage.CaseS.GetByIdMobile(req)
}
func (s *CaseService) GetAllMobile(ctx context.Context, req *m.CaseGetListMobileReq) (*m.PaginationResponse, *m.CaseMobileList, error) {
	return s.storage.CaseS.GetAllMobile(req)
}
func (s *CaseService) GetRandom(ctx context.Context, req *m.CaseRandomReq) (*m.CaseMobileRes, error) {
	return s.storage.CaseS.GetRandom(req)
}
