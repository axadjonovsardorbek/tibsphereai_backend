package service

import (
	"context"
	"hash/fnv"
	"math/rand"
	"os"
	"testing"

	"tibsphereai/internal/ai"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- sof (DB'siz) funksiyalar ---

func TestPickChunks(t *testing.T) {
	pool := []m.SourceChunk{{Id: "1"}, {Id: "2"}, {Id: "3"}, {Id: "4"}, {Id: "5"}, {Id: "6"}}
	got := PickChunks(pool, 4, rand.New(rand.NewSource(1)))
	require.Len(t, got, 4)
	assert.Equal(t, "1", got[0].Id) // eng mos ikkitasi doim olinadi
	assert.Equal(t, "2", got[1].Id)
	ids := map[string]bool{}
	for _, c := range got {
		assert.False(t, ids[c.Id], "dublikat chunk")
		ids[c.Id] = true
	}
	assert.Len(t, PickChunks(pool[:3], 4, rand.New(rand.NewSource(1))), 3)
}

func TestBuildSourceRefs(t *testing.T) {
	chunks := []m.SourceChunk{
		{Id: "c1", SourceId: "s1", SourceName: "Kitob", SourceType: "book", Author: "A", Locator: "p. 5", Url: "u"},
		{Id: "c2", SourceId: "s1", SourceName: "Kitob", SourceType: "book", Locator: "p. 9"},
		{Id: "c3", SourceId: "s2", SourceName: "Sayt", SourceType: "website", Url: "https://x.uz"},
	}
	// AI 3 va 1 ni ishlatgan (+ 99 va takror 3 - e'tiborsiz): tartib AI aytganicha, manba bo'yicha dublikatsiz.
	refs, ids := BuildSourceRefs(chunks, []int{3, 1, 99, 3})
	require.Len(t, refs, 2)
	assert.Equal(t, "s2", refs[0].SourceId)
	assert.Equal(t, "https://x.uz", refs[0].Url) // website - havola
	assert.Equal(t, "s1", refs[1].SourceId)
	assert.Empty(t, refs[1].Url) // book uchun havola yo'q
	assert.Equal(t, "p. 5", refs[1].Locator)
	assert.Equal(t, []string{"c3", "c1"}, ids)

	// AI used_sources'ni umuman aytmasa (nil) - promptga kiritilgan hammasi (manba bo'yicha birlashtirilgan).
	refs, _ = BuildSourceRefs(chunks, nil)
	assert.Len(t, refs, 2)
	// AI aniq "hech qaysi" desa ([]) yoki faqat noto'g'ri raqamlar - manba yo'q.
	refs, ids = BuildSourceRefs(chunks, []int{})
	assert.Empty(t, refs)
	assert.Empty(t, ids)
	refs, _ = BuildSourceRefs(chunks, []int{99})
	assert.Empty(t, refs)
	// chunk umuman yo'q - bo'sh ro'yxat (manba to'qib chiqarilmaydi).
	refs, ids = BuildSourceRefs(nil, []int{1, 2})
	assert.Empty(t, refs)
	assert.Empty(t, ids)
}

func TestShuffleTest_KeepsCorrectAnswer(t *testing.T) {
	for seed := int64(0); seed < 20; seed++ {
		tg := &m.PracticeTestGen{Options: []string{"a", "b", "c", "d"}, CorrectIndex: 2}
		ShuffleTest(tg, rand.New(rand.NewSource(seed)))
		assert.Equal(t, "c", tg.Options[tg.CorrectIndex])
		assert.ElementsMatch(t, []string{"a", "b", "c", "d"}, tg.Options)
	}
}

func TestScoreAndXP(t *testing.T) {
	s, x := ScoreAndXP(nil)
	assert.Equal(t, 0, s)
	assert.Equal(t, 0, x)
	s, x = ScoreAndXP([]int{100, 0, 50})
	assert.Equal(t, 50, s)
	assert.Equal(t, 8, x) // 150/100*5 = 7.5 -> 8
}

func TestOptionIndex(t *testing.T) {
	assert.Equal(t, 0, optionIndex("a"))
	assert.Equal(t, 3, optionIndex(" D "))
	assert.Equal(t, -1, optionIndex("E"))
	assert.Equal(t, -1, optionIndex("AB"))
	assert.Equal(t, -1, optionIndex(""))
}

func TestItemRes_HidesAnswerUntilAnswered(t *testing.T) {
	payload := []byte(`{"question":"Q?","options":["x","y","z","w"],"correct_index":1,"explanation":"because"}`)
	it := &m.PracticeItemRow{Id: "i1", Kind: "test", Payload: payload, Sources: []m.SourceRef{{SourceId: "s", Title: "T", Type: "book"}}}
	r, err := itemRes(it)
	require.NoError(t, err)
	assert.False(t, r.Answered)
	assert.Empty(t, r.Explanation)
	assert.Empty(t, r.CorrectKey)
	assert.Empty(t, r.CorrectAnswer)
	assert.Empty(t, r.Sources) // manbalar ham javobdan keyin
	assert.Len(t, r.Options, 4)

	score, ok := 100, true
	it.Score, it.IsCorrect = &score, &ok
	it.UserAnswer = []byte(`{"key":"B"}`)
	it.Evaluation = []byte(`{"score":100,"verdict":"correct"}`)
	r, err = itemRes(it)
	require.NoError(t, err)
	assert.True(t, r.Answered)
	assert.Equal(t, "B", r.CorrectKey)
	assert.Equal(t, "y", r.CorrectAnswer)
	assert.Equal(t, "because", r.Explanation)
	assert.Len(t, r.Sources, 1)

	// manbasiz element - eslatma yozuvi
	it.Sources = nil
	r, _ = itemRes(it)
	assert.Equal(t, noSourcesNote, r.SourcesNote)
}

// --- integratsiya (haqiqiy Postgres; yo'q bo'lsa skip) ---

func practiceTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TIB_TEST_DB")
	if dsn == "" {
		dsn = "host=localhost port=5473 user=postgres password=1111 dbname=tibsphereai sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skip("skipping: postgres not reachable: " + err.Error())
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skip("skipping: postgres not reachable: " + err.Error())
	}
	var has bool
	_ = pool.QueryRow(ctx, `SELECT to_regclass('practice_items') IS NOT NULL`).Scan(&has)
	if !has {
		pool.Close()
		t.Skip("skipping: migration 00024 not applied")
	}
	t.Cleanup(pool.Close)
	return pool
}

type fakePracticeAI struct {
	verdictScore int
	failEval     bool
	notMedical   bool
	tokens       int // har chaqiruvdagi jami token (0 = 150)
	calls        int
}

func (f *fakePracticeAI) usage() *ai.Usage {
	t := f.tokens
	if t == 0 {
		t = 150
	}
	return &ai.Usage{PromptTokens: t * 2 / 3, OutputTokens: t - t*2/3, TotalTokens: t}
}

func (f *fakePracticeAI) GenerateCase(ctx context.Context, in *m.PracticeGenInput) (*m.PracticeCaseGen, []int, *ai.Usage, error) {
	f.calls++
	if f.notMedical {
		return nil, nil, f.usage(), ai.ErrTopicNotMedical
	}
	used := []int{}
	if len(in.Chunks) > 0 {
		used = []int{1}
	}
	return &m.PracticeCaseGen{
		ChiefComplaint: "Ko'kragim og'riyapti", Patient: map[string]any{"yosh": 55}, ExpectedAnswer: "STEMI",
		Explanation: "ST elevatsiya", Summary: "STEMI case " + string(rune('0'+f.calls)),
	}, used, f.usage(), nil
}

func (f *fakePracticeAI) GenerateTest(ctx context.Context, in *m.PracticeGenInput) (*m.PracticeTestGen, []int, *ai.Usage, error) {
	f.calls++
	if f.notMedical {
		return nil, nil, f.usage(), ai.ErrTopicNotMedical
	}
	used := []int{}
	if len(in.Chunks) > 0 {
		used = []int{1}
	}
	return &m.PracticeTestGen{
		Question: "Qaysi biri to'g'ri?", Options: []string{"To'g'ri", "Xato 1", "Xato 2", "Xato 3"}, CorrectIndex: 0,
		Explanation: "Chunki", Summary: "savol " + string(rune('0'+f.calls)),
	}, used, f.usage(), nil
}

func (f *fakePracticeAI) EvaluateCase(ctx context.Context, lang, complaint string, patient map[string]any, expected, answer string) (*m.PracticeEval, *ai.Usage, error) {
	if f.failEval {
		return nil, nil, assert.AnError
	}
	return &m.PracticeEval{Score: f.verdictScore, Feedback: "izoh"}, f.usage(), nil
}

// fakeEmbedder - deterministik 8 o'lchamli vektor (matn xeshidan).
type fakeEmbedder struct{ fail bool }

func (f fakeEmbedder) Embed(ctx context.Context, texts []string, query bool) ([][]float32, *ai.Usage, error) {
	if f.fail {
		return nil, nil, assert.AnError
	}
	out := make([][]float32, len(texts))
	for i, t := range texts {
		h := fnv.New64a()
		h.Write([]byte(t))
		r := rand.New(rand.NewSource(int64(h.Sum64())))
		v := make([]float32, 8)
		for j := range v {
			v[j] = r.Float32() + 0.01
		}
		out[i] = v
	}
	return out, &ai.Usage{}, nil
}

type practiceEnv struct {
	pool     *pgxpool.Pool
	store    *st.Storage
	prac     *PracticeService
	flash    *FlashcardService
	cardAI   *fakeFlashcardAI
	settings *AppSettingService
	src      *SourceService
	aiFake   *fakePracticeAI
	userId   string
	catId    string
	topic1   string
	topic2   string
}

func newPracticeEnv(t *testing.T) *practiceEnv {
	pool := practiceTestDB(t)
	ctx := context.Background()
	store := &st.Storage{
		Db: pool, SourceS: st.NewSourceRepo(pool), PracticeS: st.NewPracticeRepo(pool),
		CategoryS: st.NewCategoryRepo(pool), TopicS: st.NewTopicRepo(pool), UserS: st.NewUserRepo(pool, nil),
		AIUsageLogS: st.NewAIUsageLogRepo(pool), AppSettingS: st.NewAppSettingRepo(pool), FlashcardS: st.NewFlashcardRepo(pool),
	}
	settings := NewAppSettingService(store, 3)
	fa := &fakePracticeAI{verdictScore: 90}
	ff := &fakeFlashcardAI{}
	client := &ai.Client{Practice: fa, Flashcards: ff, Embedder: fakeEmbedder{}}

	env := &practiceEnv{pool: pool, store: store, aiFake: fa, userId: uuid.NewString(), catId: uuid.NewString(), topic1: uuid.NewString(), topic2: uuid.NewString()}
	env.prac = NewPracticeService(store, client, settings)
	env.settings = settings
	env.prac.prefetch = false
	env.prac.asyncFlashcards = false
	env.flash = NewFlashcardService(store, client, settings)
	env.cardAI = ff
	env.src = NewSourceService(store, client)
	env.src.async = false

	_, err := pool.Exec(ctx, `INSERT INTO users (id, name) VALUES ($1, 'practice test')`, env.userId)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO categories (id, name) VALUES ($1, '{"uz":"Kardiologiya"}')`, env.catId)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO topics (id, category_id, name, ai_hint) VALUES ($1, $3, '{"uz":"OKS"}', 'faqat STEMI'), ($2, $3, '{"uz":"Aritmiya"}', '')`, env.topic1, env.topic2, env.catId)
	require.NoError(t, err)
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM flashcard_reviews WHERE user_id = $1`, env.userId)
		_, _ = pool.Exec(bg, `DELETE FROM flashcards WHERE user_id = $1`, env.userId)
		_, _ = pool.Exec(bg, `DELETE FROM flashcard_decks WHERE user_id = $1`, env.userId)
		_, _ = pool.Exec(bg, `DELETE FROM practice_sessions WHERE user_id = $1`, env.userId)
		_, _ = pool.Exec(bg, `DELETE FROM case_sources WHERE name LIKE 'ptest-%'`)
		_, _ = pool.Exec(bg, `DELETE FROM topics WHERE category_id = $1`, env.catId)
		_, _ = pool.Exec(bg, `DELETE FROM categories WHERE id = $1`, env.catId)
		_, _ = pool.Exec(bg, `DELETE FROM point_transactions WHERE user_id = $1`, env.userId)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id = $1`, env.userId)
	})
	return env
}

func (e *practiceEnv) addSource(t *testing.T, title, typ, text string, cats, topics []string) string {
	t.Helper()
	res, err := e.src.Create(context.Background(), &m.SourceCreateReq{Title: title, Type: typ, Content: text, Url: "https://example.uz/p", CategoryIds: cats, TopicIds: topics})
	require.NoError(t, err)
	return res.Id
}

func TestSource_IngestAndScope(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	long := ""
	for i := 0; i < 80; i++ {
		long += "Miokard infarkti belgilari: ko'krak og'rig'i, EKGda ST elevatsiya, troponin oshishi. "
	}
	linkedCat := e.addSource(t, "ptest-cat", "text", long, []string{e.catId}, nil)
	linkedTopic2 := e.addSource(t, "ptest-topic2", "text", long, nil, []string{e.topic2})
	general := e.addSource(t, "ptest-general", "text", long, nil, nil)
	otherCatId := uuid.NewString()
	_, err := e.pool.Exec(ctx, `INSERT INTO categories (id, name) VALUES ($1, '{"uz":"Boshqa"}')`, otherCatId)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = e.pool.Exec(context.Background(), `DELETE FROM categories WHERE id = $1`, otherCatId) })
	e.addSource(t, "ptest-other", "text", long, []string{otherCatId}, nil)

	res, err := e.store.SourceS.GetById(&m.ById{Id: linkedCat})
	require.NoError(t, err)
	assert.Equal(t, m.SourceStatusReady, *res.Status)
	assert.Greater(t, *res.ChunkCount, 0)
	assert.Equal(t, []string{e.catId}, res.CategoryIds)

	// topic1 tanlangan: bo'lim-link'li va umumiy manbalar kiradi, topic2-link'li kirmaydi.
	chunks, err := e.store.SourceS.SearchChunks(e.catId, []string{e.topic1}, nil, 500)
	require.NoError(t, err)
	names := map[string]bool{}
	for _, c := range chunks {
		names[c.SourceName] = true
	}
	assert.True(t, names["ptest-cat"])
	assert.True(t, names["ptest-general"])
	assert.False(t, names["ptest-topic2"])
	assert.False(t, names["ptest-other"], "boshqa bo'limga bog'langan manba kirmasligi kerak")

	// aralash (mavzu yo'q): bo'limdagi barcha mavzularga bog'langan manbalar ham kiradi.
	chunks, err = e.store.SourceS.SearchChunks(e.catId, nil, nil, 500)
	require.NoError(t, err)
	names = map[string]bool{}
	for _, c := range chunks {
		names[c.SourceName] = true
	}
	assert.True(t, names["ptest-topic2"])

	// faol emas / tayyor emas manbalar chiqmaydi
	inactive := false
	_, err = e.store.SourceS.Update(&m.SourceUpdateReq{Id: general, IsActive: &inactive})
	require.NoError(t, err)
	chunks, _ = e.store.SourceS.SearchChunks(e.catId, []string{e.topic1}, nil, 500)
	for _, c := range chunks {
		assert.NotEqual(t, "ptest-general", c.SourceName)
	}

	// embedding bo'yicha tartiblash: so'rov = chunkning o'zi bo'lsa, eng mos birinchi keladi.
	emb, _, _ := fakeEmbedder{}.Embed(ctx, []string{long[:200]}, true)
	chunks, err = e.store.SourceS.SearchChunks(e.catId, nil, st.Normalize(emb[0]), 3)
	require.NoError(t, err)
	assert.Len(t, chunks, 3)

	// o'chirilgan manba chunklari ham chiqmaydi
	_, err = e.src.Delete(ctx, &m.ById{Id: linkedCat})
	require.NoError(t, err)
	chunks, _ = e.store.SourceS.SearchChunks(e.catId, nil, nil, 500)
	for _, c := range chunks {
		assert.NotEqual(t, "ptest-cat", c.SourceName)
	}
	_ = linkedTopic2
}

func TestSource_EmbeddingFailure_StillReady(t *testing.T) {
	e := newPracticeEnv(t)
	e.src.ai = &ai.Client{Embedder: fakeEmbedder{fail: true}}
	id := e.addSource(t, "ptest-noembed", "text", "Yetarlicha uzun klinik matn bo'lib, embedding xato bersa ham chunk saqlanadi va manba tayyor bo'ladi.", nil, nil)
	res, err := e.store.SourceS.GetById(&m.ById{Id: id})
	require.NoError(t, err)
	assert.Equal(t, m.SourceStatusReady, *res.Status)
	assert.Contains(t, *res.Error, "embedding")
	assert.Equal(t, 1, *res.ChunkCount)
}

func TestSource_Validation(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	for name, req := range map[string]*m.SourceCreateReq{
		"no title":       {Type: "text", Content: "x"},
		"bad type":       {Title: "t", Type: "video"},
		"website no url": {Title: "t", Type: "website", Url: "ftp://x"},
		"book no file":   {Title: "t", Type: "book"},
		"text empty":     {Title: "t", Type: "text", Content: "  "},
		"bad ext":        {Title: "t", Type: "book", FileName: "a.exe", FileData: []byte("x")},
	} {
		_, err := e.src.Create(ctx, req)
		assert.Error(t, err, name)
	}
}

func TestPractice_TestMode_FullFlow(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	e.addSource(t, "ptest-kitob", "text", "Klinik qo'llanma matni, yetarlicha uzun bo'lishi uchun yozilgan jumla.", nil, nil)
	_, _ = e.pool.Exec(ctx, `UPDATE case_sources SET type='book' WHERE name='ptest-kitob'`)

	sess, err := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "test", CategoryId: e.catId, TopicIds: []string{e.topic1}, Difficulty: "easy", Count: 2, Language: "uz"})
	require.NoError(t, err)
	assert.Equal(t, 2, sess.TotalItems)

	// noto'g'ri mavzu (boshqa bo'limniki) rad etiladi
	_, err = e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "test", CategoryId: e.catId, TopicIds: []string{uuid.NewString()}})
	assert.Error(t, err)

	item, err := e.prac.Next(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	assert.Len(t, item.Options, 4)
	assert.Empty(t, item.CorrectKey, "to'g'ri javob javobdan oldin yashirin")
	assert.Empty(t, item.Explanation)

	// takroriy next - xuddi shu element, yangi AI so'rovi yo'q
	calls := e.aiFake.calls
	again, err := e.prac.Next(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	assert.Equal(t, item.Id, again.Id)
	assert.Equal(t, calls, e.aiFake.calls)

	// boshqa foydalanuvchi sessiyaga/elementga tega olmaydi
	_, err = e.prac.Next(ctx, uuid.NewString(), sess.Id)
	assert.ErrorIs(t, err, errs.ErrNotFound)
	_, err = e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: uuid.NewString(), ItemId: item.Id, Answer: "A"})
	assert.ErrorIs(t, err, errs.ErrNotFound)

	// noto'g'ri format
	_, err = e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: item.Id, Answer: "Z"})
	assert.Error(t, err)

	// to'g'ri variantni topamiz (aralashtirilgan): "To'g'ri" matni
	correctKey := ""
	for _, o := range item.Options {
		if o.Text == "To'g'ri" {
			correctKey = o.Key
		}
	}
	require.NotEmpty(t, correctKey)
	ans, err := e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: item.Id, Answer: correctKey})
	require.NoError(t, err)
	assert.True(t, *ans.Item.IsCorrect)
	assert.Equal(t, correctKey, ans.Item.CorrectKey)
	assert.Equal(t, "Chunki", ans.Item.Explanation)
	assert.NotEmpty(t, ans.Item.Sources, "manba bog'langan: izoh ostida manbalar")
	assert.Equal(t, "ptest-kitob", ans.Item.Sources[0].Title)
	assert.False(t, ans.SessionFinished)

	// ikki marta javob berib bo'lmaydi
	_, err = e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: item.Id, Answer: correctKey})
	assert.ErrorIs(t, err, errs.ErrPracticeAnswered)

	// ikkinchi savol: noto'g'ri javob, oxirgi savol - sessiya avtomatik yakunlanadi
	item2, err := e.prac.Next(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	assert.NotEqual(t, item.Id, item2.Id)
	wrong := "A"
	for _, o := range item2.Options {
		if o.Text != "To'g'ri" {
			wrong = o.Key
			break
		}
	}
	ans2, err := e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: item2.Id, Answer: wrong})
	require.NoError(t, err)
	assert.False(t, *ans2.Item.IsCorrect)
	assert.True(t, ans2.SessionFinished)
	assert.Equal(t, 50, ans2.Session.Score)
	assert.Equal(t, 5, ans2.Session.XPEarned) // 100/100*5 = 5
	assert.Equal(t, m.PracticeStatusFinished, ans2.Session.Status)

	// yakunlangan sessiyada next/finish yo'q
	_, err = e.prac.Next(ctx, e.userId, sess.Id)
	assert.ErrorIs(t, err, errs.ErrSessionNotActive)
	_, err = e.prac.Finish(ctx, e.userId, sess.Id)
	assert.ErrorIs(t, err, errs.ErrSessionNotActive)

	// tarix va detal
	det, err := e.prac.GetSession(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	assert.Len(t, det.Items, 2)
	assert.Equal(t, 2, det.Answered)
	assert.Equal(t, 1, det.Correct)
	_, list, err := e.prac.ListSessions(ctx, &m.PracticeListReq{UserId: e.userId, Filter: m.Pagination{Limit: 10, Page: 0}})
	require.NoError(t, err)
	assert.Equal(t, 1, list.Count)

	// shikoyat
	_, err = e.prac.Report(ctx, &m.PracticeReportReq{UserId: e.userId, ItemId: item.Id, Reason: "noto'g'ri"})
	require.NoError(t, err)
	stats, err := e.store.PracticeS.Stats()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, stats.Reports, 1)
	assert.GreaterOrEqual(t, stats.Sessions, 1)
}

func TestPractice_TestMode_LimitReached(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	sess, err := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "test", CategoryId: e.catId, Count: 1})
	require.NoError(t, err)
	it, err := e.prac.Next(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	_ = it
	// javobsiz element bor - takroriy next o'sha elementni qaytaradi (limit emas)
	again, err := e.prac.Next(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	assert.Equal(t, it.Id, again.Id)
	// manba yo'q: "manba topilmadi" eslatmasi
	ans, err := e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: it.Id, Answer: "A"})
	require.NoError(t, err)
	assert.Empty(t, ans.Item.Sources)
	assert.Equal(t, noSourcesNote, ans.Item.SourcesNote)
	assert.True(t, ans.SessionFinished) // count=1 - birinchi javobdayoq yakun
}

func TestPractice_CaseMode_Evaluation(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	sess, err := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "case", CategoryId: e.catId, Difficulty: "hard"})
	require.NoError(t, err)
	assert.Equal(t, 0, sess.TotalItems)

	item, err := e.prac.Next(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	assert.Equal(t, "Ko'kragim og'riyapti", item.ChiefComplaint)
	assert.Empty(t, item.CorrectAnswer)

	// bo'sh javob rad etiladi
	_, err = e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: item.Id, Answer: "   "})
	assert.Error(t, err)

	// AI baholash xatosi: element javobsiz qoladi, qayta yuborish mumkin
	e.aiFake.failEval = true
	_, err = e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: item.Id, Answer: "STEMI, aspirin"})
	assert.Error(t, err)
	e.aiFake.failEval = false

	e.aiFake.verdictScore = 60
	ans, err := e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: item.Id, Answer: "STEMI, aspirin"})
	require.NoError(t, err)
	assert.Equal(t, "partial", ans.Item.Verdict)
	assert.False(t, *ans.Item.IsCorrect) // partial = to'g'ri emas
	assert.Equal(t, 60, *ans.Item.Score)
	assert.Equal(t, "STEMI", ans.Item.CorrectAnswer)
	assert.False(t, ans.SessionFinished) // case rejimi cheksiz

	// foydalanuvchi to'xtatadi: ball 60, XP = 60/100*5 = 3, qiyinlik hard (2x) -> 6
	fin, err := e.prac.Finish(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	assert.Equal(t, 60, fin.Score)
	assert.Equal(t, 6, fin.XPEarned)
	assert.Equal(t, m.PracticeStatusFinished, fin.Status)

	// takrorlanmaslik: ikkinchi generatsiyada oldingi xulosa promptga beriladi
	sums, err := e.store.PracticeS.Summaries(sess.Id)
	require.NoError(t, err)
	assert.Len(t, sums, 1)
}

func TestPractice_Finish_NoAnswers_Abandoned(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	sess, err := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "case", CategoryId: e.catId})
	require.NoError(t, err)
	fin, err := e.prac.Finish(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	assert.Equal(t, m.PracticeStatusAbandoned, fin.Status)
	assert.Equal(t, 0, fin.XPEarned)
}

// --- Pay-as-you-go, erkin mavzu, streak, XP koeffitsienti ---

func (e *practiceEnv) setSetting(t *testing.T, key, value string) {
	t.Helper()
	var old *string
	var v string
	if err := e.pool.QueryRow(context.Background(), `SELECT value FROM app_settings WHERE key = $1`, key).Scan(&v); err == nil {
		old = &v
	}
	_, err := e.pool.Exec(context.Background(), `INSERT INTO app_settings (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, key, value)
	require.NoError(t, err)
	t.Cleanup(func() {
		if old == nil {
			_, _ = e.pool.Exec(context.Background(), `DELETE FROM app_settings WHERE key = $1`, key)
		} else {
			_, _ = e.pool.Exec(context.Background(), `UPDATE app_settings SET value = $2 WHERE key = $1`, key, *old)
		}
	})
}

func TestCoinsForTokens(t *testing.T) {
	assert.Equal(t, 0, CoinsForTokens(0, 1))
	assert.Equal(t, 0, CoinsForTokens(500, 0))
	assert.Equal(t, 1, CoinsForTokens(1, 1))
	assert.Equal(t, 1, CoinsForTokens(1000, 1))
	assert.Equal(t, 2, CoinsForTokens(1001, 1))
	assert.Equal(t, 3, CoinsForTokens(2000, 1.5))
}

func TestBiller_DailyFreeCoins_ChargeOrder(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	e.setSetting(t, "daily_free_coins", "10")
	b := NewBiller(e.pool, e.settings)

	w, err := b.Wallet(ctx, e.userId)
	require.NoError(t, err)
	assert.Equal(t, 10, w.DailyFreeCoins)
	assert.Equal(t, 10, w.FreeCoinsLeft)
	assert.Equal(t, 0, w.Coins)

	// bepul tangalar birinchi sarflanadi
	n, err := b.Charge(ctx, e.userId, 4, "test", "t", "1")
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	w, _ = b.Wallet(ctx, e.userId)
	assert.Equal(t, 6, w.FreeCoinsLeft)

	// balans bor: bepuldan keyin balansdan yechiladi, balansdan ortig'i yechilmaydi
	_, err = e.pool.Exec(ctx, `UPDATE users SET coins = 3 WHERE id = $1`, e.userId)
	require.NoError(t, err)
	n, err = b.Charge(ctx, e.userId, 20, "test", "t", "2")
	require.NoError(t, err)
	assert.Equal(t, 9, n) // 6 bepul + 3 balans
	w, _ = b.Wallet(ctx, e.userId)
	assert.Equal(t, 0, w.FreeCoinsLeft)
	assert.Equal(t, 0, w.Coins)
	assert.Equal(t, 10, w.DailyFreeCoins, "bugungi bepul tangalar qayta berilmaydi")
	require.ErrorIs(t, b.RequireFunds(ctx, e.userId), errs.ErrInsufficientCoins)

	// obunachidan yechilmaydi
	tid := uuid.NewString()
	_, err = e.pool.Exec(ctx, `INSERT INTO tariffs (id, name, duration, price) VALUES ($1, 'ptest-tariff', 1, 0)`, tid)
	require.NoError(t, err)
	_, err = e.pool.Exec(ctx, `INSERT INTO subscriptions (user_id, tariff_id, status, ended_at) VALUES ($1, $2, 'active', now() + interval '1 day')`, e.userId, tid)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), `DELETE FROM subscriptions WHERE user_id = $1`, e.userId)
		_, _ = e.pool.Exec(context.Background(), `DELETE FROM tariffs WHERE id = $1`, tid)
	})
	assert.True(t, b.HasSubscription(ctx, e.userId))
	n, err = b.Charge(ctx, e.userId, 5, "test", "t", "3")
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	require.NoError(t, b.RequireFunds(ctx, e.userId))
}

func TestPractice_Billing_StopsWhenWalletEmpty(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	e.setSetting(t, "coins_per_1k_tokens", "1")
	e.setSetting(t, "daily_free_coins", "2")
	e.aiFake.tokens = 1500 // har generatsiya = 2 tanga

	sess, err := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "test", CategoryId: e.catId, Count: 5})
	require.NoError(t, err)

	it, err := e.prac.Next(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	got, err := e.prac.GetSession(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	assert.Equal(t, 1500, got.TokensUsed)
	assert.Equal(t, 2, got.CoinsSpent)

	// test javobi AI chaqirmaydi - hamyon bo'sh bo'lsa ham javob berish mumkin
	_, err = e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: it.Id, Answer: "A"})
	require.NoError(t, err)
	// hamyon bo'sh: keyingi generatsiya to'xtatiladi
	_, err = e.prac.Next(ctx, e.userId, sess.Id)
	assert.ErrorIs(t, err, errs.ErrInsufficientCoins)

	// yangi sessiya ham boshlanmaydi
	_, err = e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "case", CategoryId: e.catId})
	assert.ErrorIs(t, err, errs.ErrInsufficientCoins)

	// tanga sotib olgach davom etadi
	_, err = e.pool.Exec(ctx, `UPDATE users SET coins = 5 WHERE id = $1`, e.userId)
	require.NoError(t, err)
	_, err = e.prac.Next(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	var coins int
	_ = e.pool.QueryRow(ctx, `SELECT coins FROM users WHERE id = $1`, e.userId).Scan(&coins)
	// jami 3000 token = 3 tanga (kumulyativ ceil), 2 tasi avval bepuldan yechilgan -> 1 tanga balansdan
	assert.Equal(t, 4, coins)
}

func TestPractice_Billing_RateZero_IsFree(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	e.setSetting(t, "coins_per_1k_tokens", "0")
	e.setSetting(t, "daily_free_coins", "0")
	sess, err := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "test", CategoryId: e.catId, Count: 2})
	require.NoError(t, err)
	_, err = e.prac.Next(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	got, _ := e.prac.GetSession(ctx, e.userId, sess.Id)
	assert.Equal(t, 0, got.CoinsSpent)
}

func TestPractice_FreeTopic(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	e.setSetting(t, "coins_per_1k_tokens", "0")

	_, err := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "case", TopicText: "ab"})
	assert.Error(t, err, "juda qisqa")
	_, err = e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "case", TopicText: "Qandli diabet", CategoryId: e.catId})
	assert.Error(t, err, "category bilan aralashtirib bo'lmaydi")
	_, err = e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "case"})
	assert.Error(t, err, "na category, na topic_text")

	sess, err := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "case", TopicText: "  Qandli\n diabet <b>2</b> tur  "})
	require.NoError(t, err)
	assert.Equal(t, "Qandli diabet b2/b tur", sess.FreeTopic) // < > va yangi qatorlar tozalanadi
	assert.Empty(t, sess.CategoryId)
	it, err := e.prac.Next(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	assert.NotEmpty(t, it.ChiefComplaint)

	// tibbiy bo'lmagan mavzu: AI rad etadi, sessiya yopiladi
	e.aiFake.notMedical = true
	s2, err := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "test", TopicText: "Bitcoin narxi"})
	require.NoError(t, err)
	_, err = e.prac.Next(ctx, e.userId, s2.Id)
	assert.ErrorIs(t, err, errs.ErrTopicNotMedical)
	d, _ := e.prac.GetSession(ctx, e.userId, s2.Id)
	assert.Equal(t, m.PracticeStatusAbandoned, d.Status)
}

func TestStreak(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	read := func() (int, int) {
		var c, l int
		_ = e.pool.QueryRow(ctx, `SELECT streak_count, longest_streak FROM users WHERE id = $1`, e.userId).Scan(&c, &l)
		return c, l
	}
	touchStreak(ctx, e.pool, e.userId)
	c, l := read()
	assert.Equal(t, 1, c)
	assert.Equal(t, 1, l)
	touchStreak(ctx, e.pool, e.userId) // bir kunda takror - o'zgarmaydi
	c, _ = read()
	assert.Equal(t, 1, c)

	// kecha faol bo'lgan, streak 3 -> bugun 4
	_, err := e.pool.Exec(ctx, `UPDATE users SET streak_count = 3, longest_streak = 3, streak_last_day = (now() AT TIME ZONE 'Asia/Tashkent')::date - 1 WHERE id = $1`, e.userId)
	require.NoError(t, err)
	touchStreak(ctx, e.pool, e.userId)
	c, l = read()
	assert.Equal(t, 4, c)
	assert.Equal(t, 4, l)

	// 2 kun tanaffus - qaytadan 1, longest saqlanadi
	_, err = e.pool.Exec(ctx, `UPDATE users SET streak_last_day = (now() AT TIME ZONE 'Asia/Tashkent')::date - 2 WHERE id = $1`, e.userId)
	require.NoError(t, err)
	touchStreak(ctx, e.pool, e.userId)
	c, l = read()
	assert.Equal(t, 1, c)
	assert.Equal(t, 4, l)
}

func TestPractice_XP_DifficultyMultiplier(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	e.setSetting(t, "coins_per_1k_tokens", "0")
	e.setSetting(t, "hard_reward_multiplier", "2")
	e.aiFake.verdictScore = 100
	sess, err := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "case", CategoryId: e.catId, Difficulty: "hard"})
	require.NoError(t, err)
	it, _ := e.prac.Next(ctx, e.userId, sess.Id)
	_, err = e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: it.Id, Answer: "STEMI"})
	require.NoError(t, err)
	fin, err := e.prac.Finish(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	assert.Equal(t, 10, fin.XPEarned) // 5 XP * 2.0
	var streak int
	_ = e.pool.QueryRow(ctx, `SELECT streak_count FROM users WHERE id = $1`, e.userId).Scan(&streak)
	assert.Equal(t, 1, streak, "javob berish streak'ni boshlaydi")
}
