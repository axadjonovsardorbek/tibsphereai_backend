package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"tibsphereai/internal/ai"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"
	models "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
	"time"
)

const existingFrontsForPrompt = 40

// FlashcardService - AI fleshkartalar: deck buyurtmasi, kunlik takrorlash (bepul), kartalar boshqaruvi.
// Xatolar asosidagi kartalar sessiya yakunida PracticeService tomonidan yaratiladi.
type FlashcardService struct {
	storage  st.Storage
	ai       *ai.Client
	settings *AppSettingService
	biller   *Biller
	now      func() time.Time
}

func NewFlashcardService(storage *st.Storage, aiClient *ai.Client, settings *AppSettingService) *FlashcardService {
	return &FlashcardService{storage: *storage, ai: aiClient, settings: settings, biller: NewBiller(storage.Db, settings), now: time.Now}
}

func cardRes(c *m.FlashcardRow, now time.Time) *m.FlashcardRes {
	srcs := c.Sources
	if srcs == nil {
		srcs = []m.SourceRef{}
	}
	return &m.FlashcardRes{
		Id: c.Id, DeckId: c.DeckId, Front: c.Front, Back: c.Back, Sources: srcs, Status: c.Status,
		Interval: c.IntervalDays, Reps: c.Repetitions, Lapses: c.Lapses, DueAt: c.DueAt.Format(time.RFC3339),
		IsNew: c.LastReviewedAt == nil, CreatedAt: c.CreatedAt.Format(time.RFC3339),
		NextIntervals: PreviewIntervals(SRS{Ease: c.Ease, Interval: c.IntervalDays, Reps: c.Repetitions, Lapses: c.Lapses}, now),
	}
}

func deckRes(d *m.FlashcardDeckRow) *m.FlashcardDeckRes {
	return &m.FlashcardDeckRes{
		Id: d.Id, Title: d.Title, Kind: d.Kind, CategoryId: d.CategoryId, TopicId: d.TopicId, FreeTopic: d.FreeTopic,
		CardCount: d.CardCount, DueCount: d.DueCount, CreatedAt: d.CreatedAt.Format(time.RFC3339),
	}
}

func (s *FlashcardService) ownedCard(userId, id string) (*m.FlashcardRow, error) {
	c, err := s.storage.FlashcardS.GetCard(id)
	if err != nil {
		return nil, err
	}
	if c.UserId != userId {
		return nil, errs.ErrNotFound
	}
	return c, nil
}

func (s *FlashcardService) ownedDeck(userId, id string) (*m.FlashcardDeckRow, error) {
	d, err := s.storage.FlashcardS.GetDeck(id)
	if err != nil {
		return nil, err
	}
	if d.UserId != userId {
		return nil, errs.ErrNotFound
	}
	return d, nil
}

// Due - bugungi takrorlashlar. BEPUL: tanga yechilmaydi (TZ_MVP monetizatsiya).
func (s *FlashcardService) Due(ctx context.Context, userId, deckId string, limit int) (*m.FlashcardDueRes, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if deckId != "" {
		if _, err := s.ownedDeck(userId, deckId); err != nil {
			return nil, err
		}
	}
	rows, total, err := s.storage.FlashcardS.Due(userId, deckId, limit)
	if err != nil {
		return nil, err
	}
	now := s.now()
	res := &m.FlashcardDueRes{Cards: []*m.FlashcardRes{}, DueTotal: total}
	for _, c := range rows {
		res.Cards = append(res.Cards, cardRes(c, now))
	}
	return res, nil
}

// Review - kartani baholash (again|hard|good|easy). BEPUL. Streak'ni yangilaydi.
func (s *FlashcardService) Review(ctx context.Context, req *m.FlashcardReviewReq) (*m.FlashcardReviewRes, error) {
	c, err := s.ownedCard(req.UserId, req.CardId)
	if err != nil {
		return nil, err
	}
	if c.Status != "active" {
		return nil, errors.New("card is suspended")
	}
	before := c.IntervalDays
	now := s.now()
	next, due, err := Schedule(SRS{Ease: c.Ease, Interval: c.IntervalDays, Reps: c.Repetitions, Lapses: c.Lapses}, req.Rating, now)
	if err != nil {
		return nil, err
	}
	c.Ease, c.IntervalDays, c.Repetitions, c.Lapses, c.DueAt = next.Ease, next.Interval, next.Reps, next.Lapses, due
	if err := s.storage.FlashcardS.SaveReview(c, req.Rating, before); err != nil {
		return nil, err
	}
	touchStreak(ctx, s.storage.Db, req.UserId)
	var streak int
	_ = s.storage.Db.QueryRow(ctx, `SELECT streak_count FROM users WHERE id = $1`, req.UserId).Scan(&streak)
	t := now
	c.LastReviewedAt = &t
	return &m.FlashcardReviewRes{Card: cardRes(c, now), NextInterval: next.Interval, DueAt: due.Format(time.RFC3339), StreakCount: streak}, nil
}

func (s *FlashcardService) Stats(ctx context.Context, userId string) (*m.FlashcardStatsRes, error) {
	return s.storage.FlashcardS.Stats(userId)
}

func (s *FlashcardService) AdminStats(ctx context.Context) (*m.FlashcardAdminStats, error) {
	return s.storage.FlashcardS.AdminStats()
}

func (s *FlashcardService) ListDecks(ctx context.Context, userId string) ([]*m.FlashcardDeckRes, error) {
	rows, err := s.storage.FlashcardS.ListDecks(userId)
	if err != nil {
		return nil, err
	}
	out := []*m.FlashcardDeckRes{}
	for _, d := range rows {
		out = append(out, deckRes(d))
	}
	return out, nil
}

func (s *FlashcardService) DeleteDeck(ctx context.Context, userId, id string) error {
	return s.storage.FlashcardS.DeleteDeck(userId, id)
}

func (s *FlashcardService) ListCards(ctx context.Context, userId, deckId string, limit, offset int) ([]*m.FlashcardRes, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if deckId != "" {
		if _, err := s.ownedDeck(userId, deckId); err != nil {
			return nil, 0, err
		}
	}
	rows, total, err := s.storage.FlashcardS.ListCards(userId, deckId, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	now := s.now()
	out := []*m.FlashcardRes{}
	for _, c := range rows {
		out = append(out, cardRes(c, now))
	}
	return out, total, nil
}

// CreateCard - qo'lda karta qo'shish (bepul).
func (s *FlashcardService) CreateCard(ctx context.Context, req *m.FlashcardCreateReq) (*m.FlashcardRes, error) {
	front, back := strings.TrimSpace(req.Front), strings.TrimSpace(req.Back)
	if front == "" || back == "" {
		return nil, errors.New("front and back are required")
	}
	if len([]rune(front)) > 300 || len([]rune(back)) > 700 {
		return nil, errors.New("front (max 300) or back (max 700) is too long")
	}
	if req.DeckId != "" {
		if _, err := s.ownedDeck(req.UserId, req.DeckId); err != nil {
			return nil, err
		}
	}
	n, err := s.storage.FlashcardS.AddCards([]*m.FlashcardRow{{UserId: req.UserId, DeckId: req.DeckId, Front: front, Back: back, Language: "uz"}})
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, errs.ErrConflict // bunday old tomonli karta allaqachon bor
	}
	rows, _, err := s.storage.FlashcardS.ListCards(req.UserId, req.DeckId, 1, 0)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return cardRes(rows[0], s.now()), nil
}

func (s *FlashcardService) UpdateCard(ctx context.Context, req *m.FlashcardUpdateReq) error {
	if req.Status != nil && *req.Status != "active" && *req.Status != "suspended" {
		return errors.New("status must be active or suspended")
	}
	for _, p := range []*string{req.Front, req.Back} {
		if p != nil && strings.TrimSpace(*p) == "" {
			return errors.New("front/back cannot be empty")
		}
	}
	return s.storage.FlashcardS.UpdateCard(req.UserId, req.Id, req.Front, req.Back, req.Status)
}

func (s *FlashcardService) DeleteCard(ctx context.Context, userId, id string) error {
	return s.storage.FlashcardS.DeleteCard(userId, id)
}

// DeckCreateRes - deck buyurtmasi natijasi: yaratilgan deck, kartalar soni va narxi.
type DeckCreateRes struct {
	Deck       *models.FlashcardDeckRes `json:"deck"`
	Created    int                      `json:"created"`
	TokensUsed int                      `json:"tokens_used"`
	CoinsSpent int                      `json:"coins_spent"`
}

// CreateDeck - mavzu bo'yicha mustaqil fleshkarta to'plami (TZ_MVP: AI token sarfi hisobidan tanga yechiladi).
func (s *FlashcardService) CreateDeck(ctx context.Context, req *m.FlashcardDeckCreateReq) (*DeckCreateRes, error) {
	count := req.Count
	if count <= 0 {
		count = 10
	}
	count = min(max(count, m.FlashcardMinDeckCards), m.FlashcardMaxDeckCards)
	lang := req.Language
	if lang != "ru" && lang != "en" {
		lang = "uz"
	}
	if !s.biller.HasSubscription(ctx, req.UserId) {
		if err := s.biller.RequireFunds(ctx, req.UserId); err != nil {
			return nil, err
		}
	}

	in := &m.FlashcardGenInput{Mode: m.FlashcardKindTopic, Language: lang, Count: count}
	deck := &m.FlashcardDeckRow{UserId: req.UserId, Kind: m.FlashcardKindTopic, Language: lang}
	var scopeCat string
	var scopeTopics []string
	free := sanitizeTopic(req.TopicText)
	switch {
	case free != "":
		if req.CategoryId != "" || req.TopicId != "" {
			return nil, errors.New("topic_text cannot be combined with category_id/topic_id")
		}
		if len([]rune(free)) < 3 {
			return nil, errors.New("topic_text is too short")
		}
		in.FreeTopic, in.TopicNames, deck.FreeTopic = true, []string{free}, free
		deck.Title = free
	case req.CategoryId != "":
		cat, err := s.storage.CategoryS.GetById(&m.ById{Id: req.CategoryId})
		if err != nil {
			return nil, err
		}
		if cat.IsActive != nil && !*cat.IsActive {
			return nil, errors.New("category is not active")
		}
		in.CategoryName, deck.CategoryId, scopeCat = pickName(cat.Name, lang), req.CategoryId, req.CategoryId
		deck.Title = in.CategoryName
		if req.TopicId != "" {
			t, err := s.storage.TopicS.GetById(&m.ById{Id: req.TopicId})
			if err != nil {
				return nil, err
			}
			if t.CategoryId == nil || *t.CategoryId != req.CategoryId {
				return nil, errors.New("topic does not belong to the category")
			}
			if t.IsActive != nil && !*t.IsActive {
				return nil, errors.New("topic is not active")
			}
			in.TopicNames = []string{pickName(t.Name, lang)}
			if t.AiHint != nil {
				in.TopicHints = []string{*t.AiHint}
			}
			deck.TopicId, scopeTopics = req.TopicId, []string{req.TopicId}
			deck.Title = in.TopicNames[0]
		}
	default:
		return nil, errors.New("category_id or topic_text is required")
	}
	if t := strings.TrimSpace(req.Title); t != "" {
		deck.Title = truncateRunes(t, 200)
	}

	query := in.CategoryName + ". " + strings.Join(in.TopicNames, ", ")
	if in.FreeTopic {
		query = strings.Join(in.TopicNames, ", ")
	}
	in.Chunks = ragChunks(ctx, &s.storage, s.ai, scopeCat, scopeTopics, query)
	in.Existing, _ = s.storage.FlashcardS.ExistingFronts(req.UserId, existingFrontsForPrompt)

	var (
		gen    []m.FlashcardGen
		usage  *ai.Usage
		genErr error
	)
	for attempt := 0; attempt < 3; attempt++ {
		var u *ai.Usage
		gen, u, genErr = s.ai.Flashcards.GenerateFlashcards(ctx, in)
		usage = addUsage(usage, u)
		if genErr == nil || errors.Is(genErr, ai.ErrTopicNotMedical) {
			break
		}
		fmt.Println("flashcard generation failed (attempt", attempt+1, "):", genErr)
	}
	res := &DeckCreateRes{}
	if usage != nil {
		logAIUsage(&s.storage, "flashcard_deck", usage, req.UserId, "")
		res.TokensUsed = usage.TotalTokens
		// Manba ishlatilgan bo'lsa ham, token narxi har doim yechiladi (AI resursi sarflangan).
		if coins := CoinsForTokens(usage.PromptTokens+usage.OutputTokens, s.settings.CoinsPer1kTokens(ctx)); coins > 0 {
			charged, cErr := s.biller.Charge(ctx, req.UserId, coins, "flashcard_deck", "flashcard_deck", "")
			if cErr != nil {
				fmt.Println("failed to charge flashcard deck:", cErr)
			}
			res.CoinsSpent = charged
		}
	}
	if errors.Is(genErr, ai.ErrTopicNotMedical) {
		return nil, errs.ErrTopicNotMedical
	}
	if genErr != nil {
		return nil, errs.ErrPracticeGenFailed.Wrap(genErr)
	}

	deckId, err := s.storage.FlashcardS.CreateDeck(deck)
	if err != nil {
		return nil, err
	}
	rows := make([]*m.FlashcardRow, 0, len(gen))
	for _, g := range gen {
		refs, _ := BuildSourceRefs(in.Chunks, g.UsedSources)
		rows = append(rows, &m.FlashcardRow{
			UserId: req.UserId, DeckId: deckId, Front: g.Front, Back: g.Back, Sources: refs,
			CategoryId: deck.CategoryId, TopicId: deck.TopicId, Language: lang,
		})
	}
	n, err := s.storage.FlashcardS.AddCards(rows)
	if err != nil {
		return nil, err
	}
	res.Created = n
	d, err := s.storage.FlashcardS.GetDeck(deckId)
	if err != nil {
		return nil, err
	}
	res.Deck = deckRes(d)
	return res, nil
}
