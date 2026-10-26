package service

import (
	"context"
	"testing"
	"time"

	"tibsphereai/internal/ai"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var rmNow = time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)

func catalogOf(ids ...string) []m.CatalogTopic {
	var out []m.CatalogTopic
	for i, id := range ids {
		out = append(out, m.CatalogTopic{CategoryId: "c1", CategoryName: map[string]string{"uz": "Kardio"}, CategoryOrder: 1, TopicId: id, TopicName: map[string]string{"uz": "Mavzu " + id, "ru": "Тема " + id}, TopicOrder: i})
	}
	return out
}

func perfN(topic string, n, score int, ageDays float64, correct bool) []m.PerfRow {
	var out []m.PerfRow
	for i := 0; i < n; i++ {
		out = append(out, m.PerfRow{TopicId: topic, Score: score, IsCorrect: correct, AnsweredAt: rmNow.Add(-time.Duration(ageDays*24) * time.Hour)})
	}
	return out
}

func TestBuildMap_Levels(t *testing.T) {
	cat := catalogOf("t0", "t1", "t2", "t3", "t4")
	var perf []m.PerfRow
	perf = append(perf, perfN("t1", 2, 90, 1, true)...)  // learning (<3)
	perf = append(perf, perfN("t2", 5, 30, 1, false)...) // weak
	perf = append(perf, perfN("t3", 5, 60, 1, true)...)  // medium
	perf = append(perf, perfN("t4", 5, 95, 1, true)...)  // strong
	mp := BuildMap(cat, perf, map[string]m.TopicCards{"t2": {Total: 4, Due: 2}}, rmNow)
	got := map[string]string{}
	for _, tm := range mp {
		got[tm.TopicId] = tm.Level
	}
	assert.Equal(t, map[string]string{"t0": "unexplored", "t1": "learning", "t2": "weak", "t3": "medium", "t4": "strong"}, got)
	assert.Equal(t, 4, mp[2].Cards)
	assert.Equal(t, 2, mp[2].CardsDue)
	assert.InDelta(t, 0.0, mp[2].Accuracy, 0.001)
	assert.Nil(t, mp[0].LastPracticed)
	assert.NotNil(t, mp[4].LastPracticed)
}

func TestBuildMap_RecencyWeighting(t *testing.T) {
	cat := catalogOf("t1")
	// 5 ta eski past ball (60 kun oldin) va 5 ta yangi yuqori ball: mastery yangisiga yaqin
	perf := append(perfN("t1", 5, 95, 0, true), perfN("t1", 5, 20, 60, false)...)
	mp := BuildMap(cat, perf, nil, rmNow)
	assert.Greater(t, mp[0].Mastery, 85)
	// teskarisi: yangi past
	perf = append(perfN("t1", 5, 20, 0, false), perfN("t1", 5, 95, 60, true)...)
	mp = BuildMap(cat, perf, nil, rmNow)
	assert.Less(t, mp[0].Mastery, 35)
}

func TestPlanPath_PriorityAndActions(t *testing.T) {
	cat := catalogOf("weak", "learn", "unex", "med", "strongFresh", "strongStale")
	var perf []m.PerfRow
	perf = append(perf, perfN("weak", 5, 20, 2, false)...)
	perf = append(perf, perfN("learn", 1, 90, 2, true)...)
	perf = append(perf, perfN("med", 5, 60, 2, true)...)
	perf = append(perf, perfN("strongFresh", 5, 95, 2, true)...)
	perf = append(perf, perfN("strongStale", 5, 95, 40, true)...)
	mp := BuildMap(cat, perf, nil, rmNow)
	path := PlanPath(mp, 7, "uz", rmNow)

	require.LessOrEqual(t, len(path), maxPathSteps)
	// 1-qadam: bugungi takrorlash (7 ta karta)
	assert.Equal(t, m.RoadmapActionFlashcardReview, path[0].Action)
	assert.Equal(t, 7, path[0].Count)
	// keyingisi: eng zaif mavzu - oson test (mastery 20 < 35), keyin shu mavzu uchun deck (kartalari yo'q)
	assert.Equal(t, "weak", path[1].TopicId)
	assert.Equal(t, m.RoadmapActionTest, path[1].Action)
	assert.Equal(t, "easy", path[1].Difficulty)
	assert.Equal(t, m.RoadmapActionDeck, path[2].Action)
	assert.Equal(t, "weak", path[2].TopicId)
	// yangi mavzular va o'rtacha ham kiradi, yangi (tez-tez mashq qilingan) kuchli kirmaydi
	ids := map[string]string{}
	for _, s := range path {
		if s.TopicId != "" && s.Action != m.RoadmapActionDeck {
			ids[s.TopicId] = s.Action
		}
	}
	assert.Equal(t, m.RoadmapActionTest, ids["learn"])
	assert.Equal(t, m.RoadmapActionTest, ids["unex"])
	assert.Equal(t, m.RoadmapActionCase, ids["med"])
	_, hasFresh := ids["strongFresh"]
	assert.False(t, hasFresh, "yaqinda mashq qilingan kuchli mavzu kerak emas")
	for i, s := range path {
		assert.Equal(t, i+1, s.Order)
		assert.NotEmpty(t, s.Reason)
	}
}

func TestPlanPath_StaleStrongAndLanguage(t *testing.T) {
	mp := BuildMap(catalogOf("s"), perfN("s", 5, 95, 40, true), nil, rmNow)
	path := PlanPath(mp, 0, "ru", rmNow)
	require.Len(t, path, 1)
	assert.Equal(t, "hard", path[0].Difficulty)
	assert.Equal(t, "Тема s", path[0].TopicName)
	assert.Contains(t, path[0].Reason, "дн.")

	// hammasi yangi va kuchli: umumiy tavsiya
	mp = BuildMap(catalogOf("s"), perfN("s", 5, 95, 1, true), nil, rmNow)
	path = PlanPath(mp, 0, "en", rmNow)
	require.Len(t, path, 1)
	assert.Equal(t, m.RoadmapActionTest, path[0].Action)
	assert.Contains(t, path[0].Reason, "Great results")
}

// --- integratsiya ---

type fakeRoadmapAI struct {
	fail  bool
	calls int
}

func (f *fakeRoadmapAI) Advise(ctx context.Context, lang string, summary, steps []string) (string, []string, *ai.Usage, error) {
	f.calls++
	if f.fail {
		return "", nil, &ai.Usage{PromptTokens: 100, OutputTokens: 50, TotalTokens: 150}, assert.AnError
	}
	var rs []string
	for range steps {
		rs = append(rs, "AI izoh")
	}
	return "AI xulosa", rs, &ai.Usage{PromptTokens: 700, OutputTokens: 300, TotalTokens: 1000}, nil
}

func TestRoadmap_GetRefreshCache(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	e.setSetting(t, "coins_per_1k_tokens", "0")
	fr := &fakeRoadmapAI{}
	svc := NewRoadmapService(e.store, &ai.Client{Roadmap: fr}, e.settings)

	// natija yo'q: xarita bor (barchasi unexplored), AI so'rovi esa rad etiladi (token sarflanmaydi)
	res, err := svc.Get(ctx, e.userId, "uz")
	require.NoError(t, err)
	assert.False(t, res.IsAI)
	assert.NotEmpty(t, res.Path)
	_, err = svc.Refresh(ctx, e.userId, "uz")
	assert.Error(t, err)
	assert.Equal(t, 0, fr.calls)

	// practice natijasi: topic1 bo'yicha 3 ta xato javob
	sess, err := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "test", CategoryId: e.catId, TopicIds: []string{e.topic1}, Count: 3})
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		it, _ := e.prac.Next(ctx, e.userId, sess.Id)
		wrong := ""
		for _, o := range it.Options {
			if o.Text != "To'g'ri" {
				wrong = o.Key
			}
		}
		_, err = e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: it.Id, Answer: wrong})
		require.NoError(t, err)
	}

	res, err = svc.Get(ctx, e.userId, "uz")
	require.NoError(t, err)
	var weak *m.TopicMastery
	for i := range res.Map {
		if res.Map[i].TopicId == e.topic1 {
			weak = &res.Map[i]
		}
	}
	require.NotNil(t, weak)
	assert.Equal(t, m.MasteryWeak, weak.Level)
	assert.Equal(t, 3, weak.Answered)
	assert.Equal(t, 0, weak.Mastery)
	assert.Equal(t, 1, res.Weak)
	// fleshkartalar yaratilgan (sessiya xatolari) - takrorlash qadami birinchi
	assert.Equal(t, m.RoadmapActionFlashcardReview, res.Path[0].Action)

	// AI yangilash: izohlar va xulosa keshlanadi
	ref, err := svc.Refresh(ctx, e.userId, "uz")
	require.NoError(t, err)
	assert.True(t, ref.IsAI)
	assert.Equal(t, "AI xulosa", ref.Summary)
	assert.Equal(t, "AI izoh", ref.Path[0].Reason)
	assert.Equal(t, 1, fr.calls)

	cached, err := svc.Get(ctx, e.userId, "uz")
	require.NoError(t, err)
	assert.True(t, cached.IsAI)
	assert.Equal(t, "AI xulosa", cached.Summary)
	assert.Equal(t, 1, fr.calls, "kesh: AI qayta chaqirilmaydi")

	// boshqa til: kesh mos emas -> shablon izohlari
	other, _ := svc.Get(ctx, e.userId, "ru")
	assert.False(t, other.IsAI)

	// AI xato: xato qaytadi, eski kesh saqlanadi
	fr.fail = true
	_, err = svc.Refresh(ctx, e.userId, "uz")
	var ae *errs.AppError
	require.ErrorAs(t, err, &ae)
	assert.Equal(t, "practice_generation_failed", ae.Slug)
	cached, _ = svc.Get(ctx, e.userId, "uz")
	assert.True(t, cached.IsAI)
}

func TestRoadmap_Refresh_Billing(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	e.setSetting(t, "coins_per_1k_tokens", "1")
	e.setSetting(t, "daily_free_coins", "3")
	fr := &fakeRoadmapAI{}
	svc := NewRoadmapService(e.store, &ai.Client{Roadmap: fr}, e.settings)

	sess, _ := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "test", CategoryId: e.catId, TopicIds: []string{e.topic1}, Count: 1})
	it, _ := e.prac.Next(ctx, e.userId, sess.Id)
	_, err := e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: it.Id, Answer: "A"})
	require.NoError(t, err)

	w0, _ := e.biller().Wallet(ctx, e.userId)
	ref, err := svc.Refresh(ctx, e.userId, "uz")
	require.NoError(t, err)
	assert.Equal(t, 1, ref.CoinsSpent) // 1000 token = 1 tanga
	w1, _ := e.biller().Wallet(ctx, e.userId)
	assert.Equal(t, w0.FreeCoinsLeft-1, w1.FreeCoinsLeft)

	// hamyon bo'sh: AI ishlamaydi
	_, _ = e.biller().Charge(ctx, e.userId, 100, "test", "t", "x")
	calls := fr.calls
	_, err = svc.Refresh(ctx, e.userId, "uz")
	assert.ErrorIs(t, err, errs.ErrInsufficientCoins)
	assert.Equal(t, calls, fr.calls)
}

func TestFeedback(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	svc := NewFeedbackService(e.store)
	_, err := svc.Create(ctx, &m.FeedbackCreateReq{UserId: e.userId})
	assert.Error(t, err, "baho yoki xabar kerak")
	_, err = svc.Create(ctx, &m.FeedbackCreateReq{UserId: e.userId, Rating: 9})
	assert.Error(t, err)
	_, err = svc.Create(ctx, &m.FeedbackCreateReq{UserId: e.userId, Kind: "spam", Message: "x"})
	assert.Error(t, err)
	res, err := svc.Create(ctx, &m.FeedbackCreateReq{UserId: e.userId, Rating: 4, Kind: "idea", Message: "  Yaxshi ilova  ", Context: "flashcards"})
	require.NoError(t, err)
	_, err = svc.Create(ctx, &m.FeedbackCreateReq{UserId: e.userId, Message: "Faqat xabar"})
	require.NoError(t, err)

	page, sum, err := svc.GetAll(ctx, &m.FeedbackListReq{Kind: "idea", Filter: m.Pagination{Limit: 10}})
	require.NoError(t, err)
	assert.Equal(t, 1, page.Total)
	assert.GreaterOrEqual(t, sum.Total, 2)
	require.NoError(t, svc.SetStatus(ctx, res.Id, "seen"))
	assert.Error(t, svc.SetStatus(ctx, res.Id, "weird"))
	assert.ErrorIs(t, svc.SetStatus(ctx, "00000000-0000-0000-0000-000000000001", "seen"), errs.ErrNotFound)
	t.Cleanup(func() {
		_, _ = e.pool.Exec(context.Background(), `DELETE FROM user_feedback WHERE user_id = $1`, e.userId)
	})
}
