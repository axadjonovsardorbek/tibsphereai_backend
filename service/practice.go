package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"tibsphereai/internal/ai"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
	"time"
)

const (
	// practiceSearchPool - RAG'da dastlab olinadigan chunklar soni; ulardan practicePromptChunks tasi
	// tanlanadi (eng mosi + tasodifiy) - shunda keyingi case'lar bir xil bo'lib qolmaydi.
	practiceSearchPool   = 12
	practicePromptChunks = 4
	practiceGenAttempts  = 3
	maxAnswerChars       = 4000

	noSourcesNote = "Manba topilmadi, javob umumiy tibbiy bilimga asoslangan."
)

// PracticeService - AI real-time case/test (new-feature.md): sessiya, generatsiya (RAG),
// javobni baholash, yakunlash (XP) va shikoyat.
type PracticeService struct {
	storage  st.Storage
	ai       *ai.Client
	settings *AppSettingService
	biller   *Biller
	locks    sync.Map // sessionId -> *sync.Mutex (bir vaqtda bitta generatsiya)
	// prefetch - javobdan keyin keyingi elementni fonda generatsiya qilish (testlarda o'chiriladi).
	prefetch bool
}

func NewPracticeService(storage *st.Storage, aiClient *ai.Client, settings *AppSettingService) *PracticeService {
	return &PracticeService{storage: *storage, ai: aiClient, settings: settings, biller: NewBiller(storage.Db, settings), prefetch: true}
}

// Wallet - foydalanuvchi hamyoni (kunlik bepul tangalar, balans, tarif) - "Limit" belgisi uchun.
func (s *PracticeService) Wallet(ctx context.Context, userId string) (Wallet, error) {
	return s.biller.Wallet(ctx, userId)
}

// sanitizeTopic - erkin mavzu matnini tozalaydi: boshqaruv belgilari/yangi qatorlar olib tashlanadi,
// uzunlik cheklanadi (prompt injection yuzasini kamaytirish).
func sanitizeTopic(t string) string {
	var b strings.Builder
	for _, r := range t {
		switch {
		case r == '\n' || r == '\r' || r == '\t':
			b.WriteRune(' ')
		case r < 32 || r == 127 || r == '<' || r == '>' || r == '`':
			// tashlab yuboriladi
		default:
			b.WriteRune(r)
		}
	}
	out := strings.Join(strings.Fields(b.String()), " ")
	if r := []rune(out); len(r) > 200 {
		out = string(r[:200])
	}
	return out
}

func (s *PracticeService) lock(sessionId string) func() {
	v, _ := s.locks.LoadOrStore(sessionId, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func validDifficulty(d string) bool { return d == "easy" || d == "medium" || d == "hard" }

func (s *PracticeService) Start(ctx context.Context, req *m.PracticeStartReq) (*m.PracticeSessionRes, error) {
	if req.Mode != m.PracticeModeCase && req.Mode != m.PracticeModeTest {
		return nil, errors.New("mode must be case or test")
	}
	if req.Difficulty == "" {
		req.Difficulty = "medium"
	}
	if !validDifficulty(req.Difficulty) {
		return nil, errors.New("difficulty must be easy, medium or hard")
	}
	freeTopic := sanitizeTopic(req.TopicText)
	if freeTopic != "" {
		if req.CategoryId != "" || len(req.TopicIds) > 0 {
			return nil, errors.New("topic_text cannot be combined with category_id/topic_ids")
		}
		if len([]rune(freeTopic)) < 3 {
			return nil, errors.New("topic_text is too short")
		}
	} else {
		if req.CategoryId == "" {
			return nil, errors.New("category_id or topic_text is required")
		}
		cat, err := s.storage.CategoryS.GetById(&m.ById{Id: req.CategoryId})
		if err != nil {
			return nil, err
		}
		if cat.IsActive != nil && !*cat.IsActive {
			return nil, errors.New("category is not active")
		}
	}
	for _, tid := range req.TopicIds {
		t, err := s.storage.TopicS.GetById(&m.ById{Id: tid})
		if err != nil {
			return nil, err
		}
		if t.CategoryId == nil || *t.CategoryId != req.CategoryId {
			return nil, fmt.Errorf("topic %s does not belong to the category", tid)
		}
		if t.IsActive != nil && !*t.IsActive {
			return nil, fmt.Errorf("topic %s is not active", tid)
		}
	}

	total := 0 // case rejimida cheksiz (PracticeMaxItems bilan chegaralanadi)
	if req.Mode == m.PracticeModeTest {
		total = req.Count
		if total <= 0 {
			total = m.PracticeDefaultTestCount
		}
		if total > m.PracticeMaxItems {
			total = m.PracticeMaxItems
		}
	}
	lang := req.Language
	if lang != "ru" && lang != "en" {
		lang = "uz"
	}
	// Pay-as-you-go: obunachi bo'lmagan foydalanuvchi uchun hamyonda (bepul kunlik + balans) tanga bo'lishi
	// kerak; sessiya davomida AI token sarfiga qarab yechiladi.
	billed := false
	if !s.biller.HasSubscription(ctx, req.UserId) {
		if err := s.biller.RequireFunds(ctx, req.UserId); err != nil {
			return nil, err
		}
		billed = s.settings.CoinsPer1kTokens(ctx) > 0
	}
	id, err := s.storage.PracticeS.CreateSession(&m.PracticeSessionRow{
		UserId: req.UserId, Mode: req.Mode, CategoryId: req.CategoryId, TopicIds: req.TopicIds,
		Difficulty: req.Difficulty, TotalItems: total, Language: lang, FreeTopic: freeTopic, CoinBilled: billed,
	})
	if err != nil {
		return nil, err
	}
	return s.GetSession(ctx, req.UserId, id)
}

// ownedSession - sessiya mavjud va shu foydalanuvchiniki ekanini tekshiradi.
func (s *PracticeService) ownedSession(userId, id string) (*m.PracticeSessionRow, error) {
	sess, err := s.storage.PracticeS.GetSession(id)
	if err != nil {
		return nil, err
	}
	if sess.UserId != userId {
		return nil, errs.ErrNotFound // boshqa foydalanuvchi sessiyasi mavjudligini oshkor qilmaymiz
	}
	return sess, nil
}

func (s *PracticeService) sessionRes(sess *m.PracticeSessionRow) (*m.PracticeSessionRes, error) {
	gen, ans, cor, err := s.storage.PracticeS.SessionCounts(sess.Id)
	if err != nil {
		return nil, err
	}
	return &m.PracticeSessionRes{
		Id: sess.Id, Mode: sess.Mode, CategoryId: sess.CategoryId, TopicIds: nonNil(sess.TopicIds),
		Difficulty: sess.Difficulty, TotalItems: sess.TotalItems, Generated: gen, Answered: ans, Correct: cor,
		Status: sess.Status, Score: sess.Score, XPEarned: sess.XPEarned, StartedAt: sess.StartedAt, FinishedAt: sess.FinishedAt,
		FreeTopic: sess.FreeTopic, TokensUsed: sess.PromptTokens + sess.CompletionTokens, CoinsSpent: sess.CoinsCharged,
		FlashcardsStatus: sess.FlashcardsStatus, FlashcardsCreated: sess.FlashcardsCreated,
	}, nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// GetSession - sessiya + barcha elementlari (javob berilmaganlarning to'g'ri javobi yashirin).
func (s *PracticeService) GetSession(ctx context.Context, userId, id string) (*m.PracticeSessionRes, error) {
	sess, err := s.ownedSession(userId, id)
	if err != nil {
		return nil, err
	}
	res, err := s.sessionRes(sess)
	if err != nil {
		return nil, err
	}
	items, err := s.storage.PracticeS.GetItems(id)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		r, err := itemRes(it)
		if err != nil {
			return nil, err
		}
		res.Items = append(res.Items, r)
	}
	return res, nil
}

func (s *PracticeService) ListSessions(ctx context.Context, req *m.PracticeListReq) (*m.PaginationResponse, *m.PracticeSessionList, error) {
	resp, rows, err := s.storage.PracticeS.ListSessions(req)
	if err != nil {
		return nil, nil, err
	}
	list := &m.PracticeSessionList{Sessions: []*m.PracticeSessionRes{}}
	for _, r := range rows {
		res, err := s.sessionRes(r)
		if err != nil {
			return nil, nil, err
		}
		list.Sessions = append(list.Sessions, res)
	}
	list.Count = resp.Total
	resp.Data = list.Sessions
	return resp, list, nil
}

// Next - keyingi case/savolni AI orqali generatsiya qiladi. Javob berilmagan element bo'lsa
// o'shani qaytaradi (takroriy chaqiruv yangi AI so'rovini ishlatmaydi).
func (s *PracticeService) Next(ctx context.Context, userId, sessionId string) (*m.PracticeItemRes, error) {
	sess, err := s.ownedSession(userId, sessionId)
	if err != nil {
		return nil, err
	}
	if sess.Status != m.PracticeStatusActive {
		return nil, errs.ErrSessionNotActive
	}
	unlock := s.lock(sessionId)
	defer unlock()

	if pending, err := s.storage.PracticeS.PendingItem(sessionId); err != nil {
		return nil, err
	} else if pending != nil {
		return itemRes(pending)
	}

	generated, _, _, err := s.storage.PracticeS.SessionCounts(sessionId)
	if err != nil {
		return nil, err
	}
	limit := m.PracticeMaxItems
	if sess.TotalItems > 0 {
		limit = sess.TotalItems
	}
	if generated >= limit {
		return nil, errs.ErrPracticeLimitReached
	}

	if sess.CoinBilled {
		if err := s.biller.RequireFunds(ctx, userId); err != nil {
			return nil, err
		}
	}
	in, err := s.buildInput(ctx, sess)
	if err != nil {
		return nil, err
	}

	row := &m.PracticeItemRow{SessionId: sessionId, Kind: sess.Mode, PromptVersion: ai.PracticePromptVersion, Model: ""}
	var (
		used   []int
		usage  *ai.Usage
		genErr error
	)
	for attempt := 0; attempt < practiceGenAttempts; attempt++ {
		var u *ai.Usage
		if sess.Mode == m.PracticeModeCase {
			var c *m.PracticeCaseGen
			c, used, u, genErr = s.ai.Practice.GenerateCase(ctx, in)
			if genErr == nil {
				c.Difficulty = sess.Difficulty
				row.Payload, _ = json.Marshal(c)
				row.Summary = c.Summary
			}
		} else {
			var t *m.PracticeTestGen
			t, used, u, genErr = s.ai.Practice.GenerateTest(ctx, in)
			if genErr == nil {
				t.Difficulty = sess.Difficulty
				ShuffleTest(t, rand.New(rand.NewSource(rand.Int63())))
				row.Payload, _ = json.Marshal(t)
				row.Summary = t.Summary
			}
		}
		usage = addUsage(usage, u)
		if genErr == nil || errors.Is(genErr, ai.ErrTopicNotMedical) {
			break
		}
		fmt.Println("practice generation failed (attempt", attempt+1, "):", genErr)
	}
	s.recordUsage(ctx, sessionId, userId, "practice_generate", usage)
	if errors.Is(genErr, ai.ErrTopicNotMedical) {
		// Tibbiy bo'lmagan erkin mavzu: sessiya yopiladi (keraksiz sessiya qolmasin).
		_ = s.storage.PracticeS.Finish(sessionId, m.PracticeStatusAbandoned, 0, 0)
		return nil, errs.ErrTopicNotMedical
	}
	if genErr != nil {
		return nil, errs.ErrPracticeGenFailed.Wrap(genErr)
	}
	if usage != nil {
		row.TokensUsed = usage.TotalTokens
	}
	row.Sources, row.SourceChunkIds = BuildSourceRefs(in.Chunks, used)

	if _, err := s.storage.PracticeS.AddItem(row); err != nil {
		return nil, err
	}
	pending, err := s.storage.PracticeS.PendingItem(sessionId)
	if err != nil || pending == nil {
		return nil, fmt.Errorf("practice item not saved: %v", err)
	}
	return itemRes(pending)
}

// buildInput - generatsiya kirishi: nomlar, mavzu yo'nalishlari, oldingi xulosalar va RAG chunklari.
func (s *PracticeService) buildInput(ctx context.Context, sess *m.PracticeSessionRow) (*m.PracticeGenInput, error) {
	in := &m.PracticeGenInput{Mode: sess.Mode, Language: sess.Language, Difficulty: sess.Difficulty}
	var err error
	if sess.FreeTopic != "" {
		in.FreeTopic = true
		in.CategoryName = "Erkin mavzu"
		in.TopicNames = []string{sess.FreeTopic}
		if in.Previous, err = s.storage.PracticeS.Summaries(sess.Id); err != nil {
			return nil, err
		}
		in.Chunks = s.retrieve(ctx, sess, in)
		return in, nil
	}
	cat, err := s.storage.CategoryS.GetById(&m.ById{Id: sess.CategoryId})
	if err != nil {
		return nil, err
	}
	in.CategoryName = pickName(cat.Name, sess.Language)

	topicIds := sess.TopicIds
	if len(topicIds) == 0 { // aralash: bo'limdagi mavzular
		_, list, err := s.storage.TopicS.GetAll(&m.TopicGetListReq{CategoryId: sess.CategoryId, Filter: m.Pagination{Page: 0, Limit: 100}})
		if err == nil && list != nil {
			for _, t := range list.Topics {
				if t.Id != nil {
					in.TopicNames = append(in.TopicNames, pickName(t.Name, sess.Language))
					if t.AiHint != nil {
						in.TopicHints = append(in.TopicHints, *t.AiHint)
					}
				}
			}
		}
	} else {
		for _, tid := range topicIds {
			t, err := s.storage.TopicS.GetById(&m.ById{Id: tid})
			if err != nil {
				return nil, err
			}
			in.TopicNames = append(in.TopicNames, pickName(t.Name, sess.Language))
			if t.AiHint != nil {
				in.TopicHints = append(in.TopicHints, *t.AiHint)
			}
		}
	}
	if in.Previous, err = s.storage.PracticeS.Summaries(sess.Id); err != nil {
		return nil, err
	}
	in.Chunks = s.retrieve(ctx, sess, in)
	return in, nil
}

func pickName(name map[string]string, lang string) string {
	if v := name[lang]; v != "" {
		return v
	}
	return pickTopicName(name)
}

// retrieve - RAG: so'rov embedding'i bo'yicha eng mos chunklar, ulardan bir qismi tasodifiy
// (xilma-xillik uchun). Embedding ishlamasa yoki manba bo'lmasa - tasodifiy/bo'sh.
func (s *PracticeService) retrieve(ctx context.Context, sess *m.PracticeSessionRow, in *m.PracticeGenInput) []m.SourceChunk {
	if s.storage.SourceS == nil {
		return nil
	}
	var qEmb []float32
	if s.ai != nil && s.ai.Embedder != nil {
		q := in.CategoryName + ". " + strings.Join(in.TopicNames, ", ") + ". " + strings.Join(in.TopicHints, " ")
		if in.FreeTopic {
			q = strings.Join(in.TopicNames, ", ")
		}
		if embs, _, err := s.ai.Embedder.Embed(ctx, []string{q}, true); err == nil && len(embs) == 1 {
			qEmb = st.Normalize(embs[0])
		}
	}
	pool, err := s.storage.SourceS.SearchChunks(sess.CategoryId, sess.TopicIds, qEmb, practiceSearchPool)
	if err != nil {
		fmt.Println("practice chunk search failed:", err)
		return nil
	}
	return PickChunks(pool, practicePromptChunks, rand.New(rand.NewSource(rand.Int63())))
}

// PickChunks - pool tartiblangan (eng mosi birinchi): birinchi ikkitasi doim olinadi,
// qolgan o'rinlar poolning qolgan qismidan tasodifiy to'ldiriladi.
func PickChunks(pool []m.SourceChunk, k int, rng *rand.Rand) []m.SourceChunk {
	if len(pool) <= k {
		return pool
	}
	keep := k / 2
	out := append([]m.SourceChunk{}, pool[:keep]...)
	rest := append([]m.SourceChunk{}, pool[keep:]...)
	rng.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
	return append(out, rest[:k-keep]...)
}

// BuildSourceRefs - foydalanuvchiga ko'rsatiladigan manbalar: FAQAT promptga kiritilgan chunklardan
// (AI o'ylab topa olmaydi). used - AI ishlatganini aytgan [n] raqamlar (1-based); noto'g'rilari
// tashlanadi. used == nil (AI aytmagan) bo'lsa kiritilgan chunklarning hammasi, bo'sh (non-nil) bo'lsa hech biri. Manba bo'yicha dublikatlar
// birlashtiriladi, tartib - promptdagi tartib (mosligi bo'yicha).
func BuildSourceRefs(chunks []m.SourceChunk, used []int) ([]m.SourceRef, []string) {
	var picked []m.SourceChunk
	if used == nil {
		// AI used_sources'ni umuman qaytarmagan - kiritilgan chunklarning hammasi hisoblanadi.
		picked = chunks
	} else {
		// AI aniq aytgan: faqat shular (bo'sh bo'lsa - hech qaysi manba ishlatilmagan, "Manba topilmadi").
		seenIdx := map[int]bool{}
		for _, n := range used {
			if n >= 1 && n <= len(chunks) && !seenIdx[n] {
				seenIdx[n] = true
				picked = append(picked, chunks[n-1])
			}
		}
	}
	refs := []m.SourceRef{}
	seen := map[string]bool{}
	ids := []string{}
	for _, c := range picked {
		ids = append(ids, c.Id)
		if seen[c.SourceId] {
			continue
		}
		seen[c.SourceId] = true
		ref := m.SourceRef{SourceId: c.SourceId, Type: c.SourceType, Title: c.SourceName, Author: c.Author, Locator: c.Locator}
		if c.SourceType == m.SourceTypeWebsite {
			ref.Url = c.Url
		}
		refs = append(refs, ref)
	}
	return refs, ids
}

// ShuffleTest - variantlarni aralashtiradi (AI to'g'ri javobni ko'pincha bir xil joyga qo'yadi)
// va correct_index'ni moslab yangilaydi.
func ShuffleTest(t *m.PracticeTestGen, rng *rand.Rand) {
	correct := t.Options[t.CorrectIndex]
	rng.Shuffle(len(t.Options), func(i, j int) { t.Options[i], t.Options[j] = t.Options[j], t.Options[i] })
	for i, o := range t.Options {
		if o == correct {
			t.CorrectIndex = i
			return
		}
	}
}

func addUsage(a, b *ai.Usage) *ai.Usage {
	if b == nil {
		return a
	}
	if a == nil {
		c := *b
		return &c
	}
	return &ai.Usage{
		PromptTokens: a.PromptTokens + b.PromptTokens, OutputTokens: a.OutputTokens + b.OutputTokens,
		TotalTokens: a.TotalTokens + b.TotalTokens, CostUSD: a.CostUSD + b.CostUSD,
	}
}

func (s *PracticeService) recordUsage(ctx context.Context, sessionId, userId, feature string, usage *ai.Usage) {
	if usage == nil {
		return
	}
	logAIUsage(&s.storage, feature, usage, userId, "")
	if err := s.storage.PracticeS.AddTokens(sessionId, usage.PromptTokens, usage.OutputTokens); err != nil {
		fmt.Println("failed to add practice tokens:", err)
	}
	s.settle(ctx, sessionId)
}

// settle - sessiyada shu paytgacha sarflangan tokenlarning tangadagi qiymatini (ceil(tokens/1000*tarif))
// hisoblab, hali yechilmagan farqni hamyondan yechadi (avval bugungi bepul tangalar, keyin balans;
// balansdan ortig'i yechilmaydi - keyingi chaqiruvdan oldin RequireFunds to'xtatadi).
func (s *PracticeService) settle(ctx context.Context, sessionId string) {
	sess, err := s.storage.PracticeS.GetSession(sessionId)
	if err != nil || !sess.CoinBilled {
		return
	}
	rate := s.settings.CoinsPer1kTokens(ctx)
	delta := CoinsForTokens(sess.PromptTokens+sess.CompletionTokens, rate) - sess.CoinsCharged
	if delta <= 0 {
		return
	}
	charged, err := s.biller.Charge(ctx, sess.UserId, delta, "practice_ai_usage", "practice_session", sessionId)
	if err != nil {
		fmt.Println("failed to charge practice usage:", err)
		return
	}
	if charged > 0 {
		if err := s.storage.PracticeS.AddCharged(sessionId, charged); err != nil {
			fmt.Println("failed to record practice coins_charged:", err)
		}
	}
}

// PracticeAnswerRes - javob natijasi; test rejimida oxirgi savol javoblangach sessiya avtomatik yakunlanadi.
type PracticeAnswerRes struct {
	Item            *m.PracticeItemRes    `json:"item"`
	SessionFinished bool                  `json:"session_finished"`
	Session         *m.PracticeSessionRes `json:"session"`
}

func (s *PracticeService) Answer(ctx context.Context, req *m.PracticeAnswerReq) (*PracticeAnswerRes, error) {
	answer := strings.TrimSpace(req.Answer)
	if answer == "" {
		return nil, errors.New("answer is required")
	}
	if len([]rune(answer)) > maxAnswerChars {
		answer = string([]rune(answer)[:maxAnswerChars])
	}
	it, err := s.storage.PracticeS.GetItem(req.ItemId)
	if err != nil {
		return nil, err
	}
	if it.UserId != req.UserId {
		return nil, errs.ErrNotFound
	}
	sess, err := s.storage.PracticeS.GetSession(it.SessionId)
	if err != nil {
		return nil, err
	}
	if sess.Status != m.PracticeStatusActive {
		return nil, errs.ErrSessionNotActive
	}
	if it.Score != nil || it.IsCorrect != nil {
		return nil, errs.ErrPracticeAnswered
	}

	var (
		isCorrect bool
		score     int
		evalJSON  []byte
		ansJSON   []byte
		tokens    int
	)
	if it.Kind == m.PracticeModeTest {
		var t m.PracticeTestGen
		if err := json.Unmarshal(it.Payload, &t); err != nil {
			return nil, err
		}
		idx := optionIndex(answer)
		if idx < 0 || idx >= len(t.Options) {
			return nil, errors.New("answer must be one of A, B, C, D")
		}
		isCorrect = idx == t.CorrectIndex
		if isCorrect {
			score = 100
		}
		verdict := "wrong"
		if isCorrect {
			verdict = "correct"
		}
		evalJSON, _ = json.Marshal(m.PracticeEval{Score: score, Verdict: verdict})
		ansJSON, _ = json.Marshal(map[string]any{"key": optionKey(idx)})
	} else {
		var c m.PracticeCaseGen
		if err := json.Unmarshal(it.Payload, &c); err != nil {
			return nil, err
		}
		if sess.CoinBilled {
			if err := s.biller.RequireFunds(ctx, req.UserId); err != nil {
				return nil, err
			}
		}
		var (
			ev    *m.PracticeEval
			usage *ai.Usage
			eErr  error
		)
		for attempt := 0; attempt < 2; attempt++ {
			var u *ai.Usage
			ev, u, eErr = s.ai.Practice.EvaluateCase(ctx, sess.Language, c.ChiefComplaint, c.Patient, c.ExpectedAnswer, answer)
			usage = addUsage(usage, u)
			if eErr == nil {
				break
			}
		}
		s.recordUsage(ctx, sess.Id, req.UserId, "practice_evaluate", usage)
		if eErr != nil {
			fmt.Println("practice evaluation failed:", eErr)
			return nil, errs.ErrPracticeGenFailed.Wrap(eErr) // element javobsiz qoladi - qayta yuborish mumkin
		}
		if usage != nil {
			tokens = usage.TotalTokens
		}
		ai.NormalizeEval(ev)
		score, isCorrect = ev.Score, ev.Verdict == "correct"
		evalJSON, _ = json.Marshal(ev)
		ansJSON, _ = json.Marshal(map[string]any{"text": answer})
	}

	if err := s.storage.PracticeS.SetAnswer(it.Id, ansJSON, evalJSON, isCorrect, score, tokens); err != nil {
		if errors.Is(err, errs.ErrConflict) || err == errs.ErrConflict {
			return nil, errs.ErrPracticeAnswered
		}
		return nil, err
	}

	touchStreak(ctx, s.storage.Db, req.UserId)

	updated, err := s.storage.PracticeS.GetItem(it.Id)
	if err != nil {
		return nil, err
	}
	itemOut, err := itemRes(updated)
	if err != nil {
		return nil, err
	}
	res := &PracticeAnswerRes{Item: itemOut}

	// Test rejimida oxirgi savol javoblangach sessiya o'zi yakunlanadi.
	if sess.Mode == m.PracticeModeTest && sess.TotalItems > 0 {
		if _, answered, _, cErr := s.storage.PracticeS.SessionCounts(sess.Id); cErr == nil && answered >= sess.TotalItems {
			if fin, fErr := s.finish(sess, m.PracticeStatusFinished); fErr == nil {
				res.SessionFinished, res.Session = true, fin
			}
		}
	}
	if res.Session == nil {
		if cur, err := s.storage.PracticeS.GetSession(sess.Id); err == nil {
			res.Session, _ = s.sessionRes(cur)
		}
	}
	// Prefetch: keyingi case/savol fonda oldindan tayyorlanadi (latency kamayadi). Next idempotent -
	// foydalanuvchi "Keyingisi" bosganda tayyor elementni oladi; limit tugagan bo'lsa xato e'tiborsiz.
	if !res.SessionFinished && s.prefetch {
		go func(userId, sessionId string) {
			bg, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			_, _ = s.Next(bg, userId, sessionId)
		}(req.UserId, sess.Id)
	}
	return res, nil
}

func optionKey(i int) string { return string(rune('A' + i)) }

func optionIndex(a string) int {
	a = strings.ToUpper(strings.TrimSpace(a))
	if len(a) != 1 || a[0] < 'A' || a[0] > 'D' {
		return -1
	}
	return int(a[0] - 'A')
}

// Finish - sessiyani yakunlaydi (user to'xtatsa yoki limit tugasa).
func (s *PracticeService) Finish(ctx context.Context, userId, sessionId string) (*m.PracticeSessionRes, error) {
	sess, err := s.ownedSession(userId, sessionId)
	if err != nil {
		return nil, err
	}
	if sess.Status != m.PracticeStatusActive {
		return nil, errs.ErrSessionNotActive
	}
	_, answered, _, err := s.storage.PracticeS.SessionCounts(sessionId)
	if err != nil {
		return nil, err
	}
	status := m.PracticeStatusFinished
	if answered == 0 {
		status = m.PracticeStatusAbandoned
	}
	if _, err := s.finish(sess, status); err != nil {
		return nil, err
	}
	return s.GetSession(ctx, userId, sessionId)
}

// ScoreAndXP - sessiya bahosi: javob berilgan elementlar ballarining o'rtachasi (0..100);
// XP = jami ball/100 * PracticeXPPerCorrect (to'liq to'g'ri javob = 5 XP).
func ScoreAndXP(scores []int) (score, xp int) {
	if len(scores) == 0 {
		return 0, 0
	}
	sum := 0
	for _, v := range scores {
		sum += v
	}
	return int(math.Round(float64(sum) / float64(len(scores)))), int(math.Round(float64(sum) / 100 * float64(m.PracticeXPPerCorrect)))
}

func (s *PracticeService) finish(sess *m.PracticeSessionRow, status string) (*m.PracticeSessionRes, error) {
	items, err := s.storage.PracticeS.GetItems(sess.Id)
	if err != nil {
		return nil, err
	}
	var scores []int
	for _, it := range items {
		if it.Score != nil {
			scores = append(scores, *it.Score)
		}
	}
	score, xp := ScoreAndXP(scores)
	// Qiyinroq daraja ko'proq XP beradi (admin sozlaydigan medium/hard koeffitsientlari; easy = 1x).
	if xp > 0 && s.settings != nil {
		xp = int(float64(xp)*s.settings.DifficultyMultiplier(context.Background(), sess.Difficulty) + 0.5)
	}
	if err := s.storage.PracticeS.Finish(sess.Id, status, score, xp); err != nil {
		return nil, err
	}
	if xp > 0 {
		if _, err := s.storage.UserS.AddPoints(&m.AddPointsReq{
			UserId: sess.UserId, XPDelta: xp, Reason: "practice_finish", ReferenceId: sess.Id,
		}); err != nil {
			fmt.Println("failed to add practice XP:", err)
		}
	}
	cur, err := s.storage.PracticeS.GetSession(sess.Id)
	if err != nil {
		return nil, err
	}
	return s.sessionRes(cur)
}

func (s *PracticeService) Report(ctx context.Context, req *m.PracticeReportReq) (*m.Void, error) {
	it, err := s.storage.PracticeS.GetItem(req.ItemId)
	if err != nil {
		return nil, err
	}
	if it.UserId != req.UserId {
		return nil, errs.ErrNotFound
	}
	reason := strings.TrimSpace(req.Reason)
	if len([]rune(reason)) > 1000 {
		reason = string([]rune(reason)[:1000])
	}
	if err := s.storage.PracticeS.Report(it.Id, reason); err != nil {
		return nil, err
	}
	return &m.Void{}, nil
}

func (s *PracticeService) Stats(ctx context.Context) (*m.PracticeStats, error) {
	return s.storage.PracticeS.Stats()
}

// itemRes - elementni userga ko'rinadigan shaklga keltiradi. Javob berilmaguncha to'g'ri
// javob, izoh va manbalar YASHIRIN (javob oldindan oshkor bo'lmasligi uchun).
func itemRes(it *m.PracticeItemRow) (*m.PracticeItemRes, error) {
	answered := it.Score != nil || it.IsCorrect != nil
	r := &m.PracticeItemRes{
		Id: it.Id, SessionId: it.SessionId, Idx: it.Idx, Kind: it.Kind, Answered: answered,
		Sources: []m.SourceRef{}, Reported: it.Reported,
	}
	var correctKey, correctText, explanation string
	if it.Kind == m.PracticeModeCase {
		var c m.PracticeCaseGen
		if err := json.Unmarshal(it.Payload, &c); err != nil {
			return nil, err
		}
		r.ChiefComplaint, r.Patient = c.ChiefComplaint, c.Patient
		correctText, explanation = c.ExpectedAnswer, c.Explanation
	} else {
		var t m.PracticeTestGen
		if err := json.Unmarshal(it.Payload, &t); err != nil {
			return nil, err
		}
		r.Question = t.Question
		for i, o := range t.Options {
			r.Options = append(r.Options, m.PracticeOption{Key: optionKey(i), Text: o})
		}
		explanation = t.Explanation
		if t.CorrectIndex >= 0 && t.CorrectIndex < len(t.Options) {
			correctKey, correctText = optionKey(t.CorrectIndex), t.Options[t.CorrectIndex]
		}
	}
	if !answered {
		return r, nil
	}
	r.IsCorrect, r.Score = it.IsCorrect, it.Score
	r.CorrectAnswer, r.CorrectKey, r.Explanation = correctText, correctKey, explanation
	if len(it.UserAnswer) > 0 {
		var a map[string]any
		if json.Unmarshal(it.UserAnswer, &a) == nil {
			if v, ok := a["key"]; ok {
				r.UserAnswer = v
			} else {
				r.UserAnswer = a["text"]
			}
		}
	}
	if len(it.Evaluation) > 0 {
		var ev m.PracticeEval
		if json.Unmarshal(it.Evaluation, &ev) == nil {
			r.Verdict, r.Feedback = ev.Verdict, ev.Feedback
		}
	}
	if len(it.Sources) > 0 {
		r.Sources = it.Sources
	} else {
		r.SourcesNote = noSourcesNote
	}
	return r, nil
}
