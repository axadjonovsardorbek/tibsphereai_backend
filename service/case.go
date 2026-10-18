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

// Create - qo'lda yaratish. Admin faqat mavzu, qiyinlik, rasm, shikoyat va to'g'ri
// javobni kiritadi; qolgan maydonlar (sarlavha, vitals, scenario...) bo'sh bo'lsa
// AI to'ldiradi. Admin kiritganlar AI tomonidan o'zgartirilmaydi. AI ishlamasa
// case baribir minimal ko'rinishda saqlanadi.
func (s *CaseService) Create(ctx context.Context, req *m.CaseCreateReq) (*m.ById, error) {
	if len(req.Title) == 0 && s.ai != nil && s.ai.Enabled {
		s.enrichWithAI(ctx, req)
	}
	// title ustuni NOT NULL: AI ishlamagan/o'chiq bo'lsa shikoyatdan vaqtinchalik
	// sarlavha olamiz (admin keyin tahrirlaydi).
	if len(req.Title) == 0 {
		t := []rune(pickTopicName(req.ChiefComplaint))
		if len(t) > 60 {
			t = t[:60]
		}
		if len(t) == 0 {
			t = []rune("Yangi case")
		}
		req.Title = map[string]string{"uz": string(t)}
	}
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}
	return s.storage.CaseS.Create(req)
}

// enrichWithAI - req'dagi bo'sh maydonlarni AI generatsiya qilgan qiymatlar bilan
// to'ldiradi; admin kiritgan maydonlarga tegmaydi. Xatolik bo'lsa req o'zgarmaydi.
func (s *CaseService) enrichWithAI(ctx context.Context, req *m.CaseCreateReq) {
	gen := &m.CaseGenReq{
		TopicId:        req.TopicId,
		Difficulty:     req.Difficulty,
		ChiefComplaint: pickTopicName(req.ChiefComplaint),
		ExpectedAnswer: req.ExpectedAnswer,
		CoverImageUrl:  req.CoverImageUrl,
	}
	if gen.ChiefComplaint == "" && gen.ExpectedAnswer == "" {
		return
	}
	if err := s.prepareGen(gen); err != nil {
		return
	}
	out, usage, err := s.ai.CaseGen.Generate(ctx, gen)
	logAIUsage(&s.storage, "case_generation", usage, "", "")
	if err != nil || out == nil {
		return
	}
	req.Title, req.Subtitle = out.Title, out.Subtitle
	if len(req.ChiefComplaint) == 0 {
		req.ChiefComplaint = out.ChiefComplaint
	} else {
		for k, v := range out.ChiefComplaint { // faqat yetishmayotgan tillar
			if req.ChiefComplaint[k] == "" {
				req.ChiefComplaint[k] = v
			}
		}
	}
	if req.PatientAge == 0 {
		req.PatientAge = out.PatientAge
	}
	if req.PatientGender == "" {
		req.PatientGender = out.PatientGender
	}
	if req.ExpectedDurationMinutes == 0 {
		req.ExpectedDurationMinutes = out.ExpectedDurationMinutes
	}
	if req.InitialVitals == (m.CaseVitals{}) {
		req.InitialVitals = out.InitialVitals
	}
	if req.VisualState == "" {
		req.VisualState = out.VisualState
	}
	if len(req.Scenario) == 0 {
		req.Scenario = out.Scenario
	}
}

// prepareGen - topic_id bo'yicha mavzu nomini va manbalarni to'ldiradi.
func (s *CaseService) prepareGen(genReq *m.CaseGenReq) error {
	if genReq.TopicId != "" {
		topic, err := s.storage.TopicS.GetById(&m.ById{Id: genReq.TopicId})
		if err != nil {
			return err
		}
		if genReq.Topic == "" && topic != nil {
			genReq.Topic = pickTopicName(topic.Name)
		}
	}
	if s.storage.CaseSourceS != nil {
		contents, err := s.storage.CaseSourceS.GetContents(genReq.SourceIds)
		if err != nil {
			return err
		}
		genReq.Sources = contents
	}
	return nil
}

// GenerateWithAI - "AI yordamida avtomatik keys yaratish" (BACKEND_STRUCTURE.md 7-bo'lim):
// admin mavzu va darajani kiritadi, Gemini AI (yoki GEMINI_API_KEY yo'q bo'lsa xatolik)
// to'liq ssenariyni tayyorlaydi, admin ko'rib chiqib alohida Publish bilan tasdiqlaydi.
func (s *CaseService) GenerateWithAI(ctx context.Context, genReq *m.CaseGenReq) (*m.ById, error) {
	if err := s.prepareGen(genReq); err != nil {
		return nil, err
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
