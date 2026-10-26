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

// --- SRS (sof) ---

func TestSchedule_Again(t *testing.T) {
	now := time.Date(2026, 10, 26, 10, 0, 0, 0, time.UTC)
	n, due, err := Schedule(SRS{Ease: 2.5, Interval: 7, Reps: 3}, "again", now)
	require.NoError(t, err)
	assert.Equal(t, 0, n.Reps)
	assert.Equal(t, 1, n.Lapses)
	assert.Equal(t, 0, n.Interval)
	assert.InDelta(t, 2.3, n.Ease, 0.001)
	assert.Equal(t, now.Add(10*time.Minute), due)
}

func TestSchedule_Good_Ladder_1_3_7_21(t *testing.T) {
	now := time.Date(2026, 10, 26, 10, 0, 0, 0, time.UTC)
	c := SRS{Ease: 2.5}
	var got []int
	for i := 0; i < 5; i++ {
		var due time.Time
		c, due, _ = Schedule(c, "good", now)
		assert.Equal(t, now.Add(time.Duration(c.Interval)*24*time.Hour), due)
		got = append(got, c.Interval)
	}
	// TZ: 3 -> 7 -> 21 kun, keyin prev*ease (21*2.5 = 52.5 -> 53, 53*2.5 = 132.5 -> 133)
	assert.Equal(t, []int{3, 7, 21, 53, 133}, got)
}

func TestSchedule_Hard_Easy(t *testing.T) {
	now := time.Now()
	n, _, _ := Schedule(SRS{Ease: 2.5}, "hard", now)
	assert.Equal(t, 1, n.Interval)
	assert.InDelta(t, 2.35, n.Ease, 0.001)
	n, _, _ = Schedule(SRS{Ease: 2.5, Interval: 10, Reps: 3}, "hard", now)
	assert.Equal(t, 12, n.Interval)

	n, _, _ = Schedule(SRS{Ease: 2.5}, "easy", now)
	assert.Equal(t, 7, n.Interval)
	assert.InDelta(t, 2.65, n.Ease, 0.001)
	n, _, _ = Schedule(n, "easy", now)
	assert.Equal(t, 14, n.Interval)
	n, _, _ = Schedule(n, "easy", now)
	assert.Greater(t, n.Interval, 14)
}

func TestSchedule_EaseBounds_And_Invalid(t *testing.T) {
	now := time.Now()
	c := SRS{Ease: 1.35}
	c, _, _ = Schedule(c, "again", now)
	assert.InDelta(t, 1.3, c.Ease, 0.001) // minimum
	c = SRS{Ease: 2.95}
	c, _, _ = Schedule(c, "easy", now)
	assert.InDelta(t, 3.0, c.Ease, 0.001) // maksimum
	_, _, err := Schedule(SRS{}, "perfect", now)
	assert.Error(t, err)
	// ease 0 (boshlang'ich qiymat yo'q) -> default 2.5
	n, _, _ := Schedule(SRS{}, "good", now)
	assert.Equal(t, 3, n.Interval)
}

func TestPreviewIntervals(t *testing.T) {
	p := PreviewIntervals(SRS{Ease: 2.5}, time.Now())
	assert.Equal(t, map[string]int{"again": 0, "hard": 1, "good": 3, "easy": 7}, p)
}

// --- integratsiya ---

type fakeFlashcardAI struct {
	calls     int
	fail      bool
	notMedial bool
	gotInput  *m.FlashcardGenInput
}

func (f *fakeFlashcardAI) GenerateFlashcards(ctx context.Context, in *m.FlashcardGenInput) ([]m.FlashcardGen, *ai.Usage, error) {
	f.calls++
	f.gotInput = in
	if f.notMedial {
		return nil, &ai.Usage{PromptTokens: 100, OutputTokens: 100, TotalTokens: 200}, ai.ErrTopicNotMedical
	}
	if f.fail {
		return nil, &ai.Usage{PromptTokens: 100, OutputTokens: 100, TotalTokens: 200}, assert.AnError
	}
	var out []m.FlashcardGen
	for i := 0; i < in.Count; i++ {
		out = append(out, m.FlashcardGen{Front: "Savol " + string(rune('A'+i)) + " (" + in.Language + ")", Back: "Javob", UsedSources: []int{1}})
	}
	return out, &ai.Usage{PromptTokens: 800, OutputTokens: 700, TotalTokens: 1500}, nil
}

func TestFlashcards_SessionMistakes_CreateCards(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	e.setSetting(t, "coins_per_1k_tokens", "0")
	e.addSource(t, "ptest-kitob2", "text", "Klinik qo'llanma matni, yetarlicha uzun bo'lishi uchun yozilgan jumla.", nil, nil)

	sess, err := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "test", CategoryId: e.catId, TopicIds: []string{e.topic1}, Count: 3})
	require.NoError(t, err)
	// 3 savol: 1 ta to'g'ri, 2 ta xato
	for i := 0; i < 3; i++ {
		it, err := e.prac.Next(ctx, e.userId, sess.Id)
		require.NoError(t, err)
		key := "A"
		for _, o := range it.Options {
			if (i == 0) == (o.Text == "To'g'ri") { // i==0: to'g'ri variant; qolganlari: noto'g'ri
				key = o.Key
				break
			}
		}
		_, err = e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: it.Id, Answer: key})
		require.NoError(t, err)
	}

	got, err := e.prac.GetSession(ctx, e.userId, sess.Id)
	require.NoError(t, err)
	assert.Equal(t, m.PracticeStatusFinished, got.Status)
	assert.Equal(t, "done", got.FlashcardsStatus)
	assert.Equal(t, 4, got.FlashcardsCreated, "2 xato x 2 = 4 karta")
	assert.Equal(t, 1, e.cardAI.calls)
	require.Len(t, e.cardAI.gotInput.Mistakes, 2)
	assert.NotEmpty(t, e.cardAI.gotInput.Mistakes[0].CorrectAnswer)
	assert.NotEmpty(t, e.cardAI.gotInput.Mistakes[0].UserAnswer)

	// kartalar "Xatolarim" to'plamida, manba va mavzu bilan
	decks, err := e.flash.ListDecks(ctx, e.userId)
	require.NoError(t, err)
	require.Len(t, decks, 1)
	assert.Equal(t, "mistakes", decks[0].Kind)
	assert.Equal(t, 4, decks[0].CardCount)
	cards, total, err := e.flash.ListCards(ctx, e.userId, decks[0].Id, 20, 0)
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.NotEmpty(t, cards[0].Sources, "manba bog'langan")
	var topic string
	_ = e.pool.QueryRow(ctx, `SELECT COALESCE(topic_id::text,'') FROM flashcards WHERE user_id = $1 LIMIT 1`, e.userId).Scan(&topic)
	assert.Equal(t, e.topic1, topic)

	// ikkinchi sessiyadagi xato shu to'plamga qo'shiladi (takroriy old tomonlar o'tkazib yuboriladi)
	sess2, _ := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "test", CategoryId: e.catId, Count: 1})
	it, _ := e.prac.Next(ctx, e.userId, sess2.Id)
	wrong := "A"
	for _, o := range it.Options {
		if o.Text != "To'g'ri" {
			wrong = o.Key
		}
	}
	_, err = e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: it.Id, Answer: wrong})
	require.NoError(t, err)
	got2, _ := e.prac.GetSession(ctx, e.userId, sess2.Id)
	assert.Equal(t, "done", got2.FlashcardsStatus)
	assert.Equal(t, 0, got2.FlashcardsCreated, "fake AI bir xil old tomonlarni qaytardi - dublikatlar qo'shilmaydi")
	decks, _ = e.flash.ListDecks(ctx, e.userId)
	assert.Len(t, decks, 1)
	assert.Equal(t, 4, decks[0].CardCount)
}

func TestFlashcards_NoMistakes_NoCards(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	e.setSetting(t, "coins_per_1k_tokens", "0")
	sess, _ := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "test", CategoryId: e.catId, Count: 1})
	it, _ := e.prac.Next(ctx, e.userId, sess.Id)
	right := ""
	for _, o := range it.Options {
		if o.Text == "To'g'ri" {
			right = o.Key
		}
	}
	_, err := e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: it.Id, Answer: right})
	require.NoError(t, err)
	got, _ := e.prac.GetSession(ctx, e.userId, sess.Id)
	assert.Equal(t, "none", got.FlashcardsStatus)
	assert.Equal(t, 0, e.cardAI.calls)
}

func TestFlashcards_AIFails_StatusFailed(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	e.setSetting(t, "coins_per_1k_tokens", "0")
	e.cardAI.fail = true
	sess, _ := e.prac.Start(ctx, &m.PracticeStartReq{UserId: e.userId, Mode: "test", CategoryId: e.catId, Count: 1})
	it, _ := e.prac.Next(ctx, e.userId, sess.Id)
	wrong := ""
	for _, o := range it.Options {
		if o.Text != "To'g'ri" {
			wrong = o.Key
		}
	}
	_, err := e.prac.Answer(ctx, &m.PracticeAnswerReq{UserId: e.userId, ItemId: it.Id, Answer: wrong})
	require.NoError(t, err)
	got, _ := e.prac.GetSession(ctx, e.userId, sess.Id)
	assert.Equal(t, m.PracticeStatusFinished, got.Status, "sessiya baribir yakunlanadi")
	assert.Equal(t, "failed", got.FlashcardsStatus)
}

func TestFlashcards_ReviewFlow_Free_And_Streak(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	// xotira: kartalarni qo'lda qo'shamiz
	c1, err := e.flash.CreateCard(ctx, &m.FlashcardCreateReq{UserId: e.userId, Front: "Rovsing simptomi?", Back: "Chap yonbosh bosimida o'ngda og'riq"})
	require.NoError(t, err)
	_, err = e.flash.CreateCard(ctx, &m.FlashcardCreateReq{UserId: e.userId, Front: "rovsing SIMPTOMI?", Back: "dublikat"})
	assert.ErrorIs(t, err, errs.ErrConflict, "old tomoni bir xil karta qo'shilmaydi")
	_, err = e.flash.CreateCard(ctx, &m.FlashcardCreateReq{UserId: e.userId, Front: "", Back: "x"})
	assert.Error(t, err)

	st, err := e.flash.Stats(ctx, e.userId)
	require.NoError(t, err)
	assert.Equal(t, 1, st.Total)
	assert.Equal(t, 1, st.New)
	assert.Equal(t, 1, st.DueNow)

	due, err := e.flash.Due(ctx, e.userId, "", 20)
	require.NoError(t, err)
	require.Len(t, due.Cards, 1)
	assert.True(t, due.Cards[0].IsNew)
	assert.Equal(t, map[string]int{"again": 0, "hard": 1, "good": 3, "easy": 7}, due.Cards[0].NextIntervals)

	// boshqa foydalanuvchi baholay olmaydi
	_, err = e.flash.Review(ctx, &m.FlashcardReviewReq{UserId: "00000000-0000-0000-0000-000000000001", CardId: c1.Id, Rating: "good"})
	assert.ErrorIs(t, err, errs.ErrNotFound)
	_, err = e.flash.Review(ctx, &m.FlashcardReviewReq{UserId: e.userId, CardId: c1.Id, Rating: "super"})
	assert.Error(t, err)

	// good -> 3 kun, karta endi "due" emas; review BEPUL (tanga o'zgarmaydi), streak boshlanadi
	e.setSetting(t, "coins_per_1k_tokens", "1")
	var coinsBefore int
	_ = e.pool.QueryRow(ctx, `SELECT coins FROM users WHERE id = $1`, e.userId).Scan(&coinsBefore)
	rev, err := e.flash.Review(ctx, &m.FlashcardReviewReq{UserId: e.userId, CardId: c1.Id, Rating: "good"})
	require.NoError(t, err)
	assert.Equal(t, 3, rev.NextInterval)
	assert.Equal(t, 1, rev.StreakCount)
	assert.False(t, rev.Card.IsNew)
	var coinsAfter, freeUsed int
	_ = e.pool.QueryRow(ctx, `SELECT coins FROM users WHERE id = $1`, e.userId).Scan(&coinsAfter)
	_ = e.pool.QueryRow(ctx, `SELECT COALESCE(SUM(used),0) FROM user_daily_coins WHERE user_id = $1`, e.userId).Scan(&freeUsed)
	assert.Equal(t, coinsBefore, coinsAfter)
	assert.Equal(t, 0, freeUsed, "takrorlash tanga yechmaydi")

	due, _ = e.flash.Due(ctx, e.userId, "", 20)
	assert.Empty(t, due.Cards)
	st, _ = e.flash.Stats(ctx, e.userId)
	assert.Equal(t, 1, st.ReviewsToday)
	assert.Equal(t, 0, st.DueNow)
	assert.InDelta(t, 1.0, st.Retention30d, 0.001)

	// again -> 10 daqiqadan keyin (hozir due emas, lekin bugun due_today ichida)
	_, err = e.flash.Review(ctx, &m.FlashcardReviewReq{UserId: e.userId, CardId: c1.Id, Rating: "again"})
	require.NoError(t, err)
	st, _ = e.flash.Stats(ctx, e.userId)
	assert.Equal(t, 1, st.DueToday)
	assert.InDelta(t, 0.5, st.Retention30d, 0.001)

	// to'xtatilgan karta takrorlanmaydi
	susp := "suspended"
	require.NoError(t, e.flash.UpdateCard(ctx, &m.FlashcardUpdateReq{UserId: e.userId, Id: c1.Id, Status: &susp}))
	_, err = e.flash.Review(ctx, &m.FlashcardReviewReq{UserId: e.userId, CardId: c1.Id, Rating: "good"})
	assert.Error(t, err)
	require.NoError(t, e.flash.DeleteCard(ctx, e.userId, c1.Id))
	_, err = e.flash.Review(ctx, &m.FlashcardReviewReq{UserId: e.userId, CardId: c1.Id, Rating: "good"})
	assert.ErrorIs(t, err, errs.ErrNotFound)
}

func TestFlashcards_Deck_Billing(t *testing.T) {
	e := newPracticeEnv(t)
	ctx := context.Background()
	e.setSetting(t, "coins_per_1k_tokens", "1")
	e.setSetting(t, "daily_free_coins", "5")

	_, err := e.flash.CreateDeck(ctx, &m.FlashcardDeckCreateReq{UserId: e.userId})
	assert.Error(t, err, "mavzu yo'q")
	_, err = e.flash.CreateDeck(ctx, &m.FlashcardDeckCreateReq{UserId: e.userId, TopicText: "ab"})
	assert.Error(t, err)

	res, err := e.flash.CreateDeck(ctx, &m.FlashcardDeckCreateReq{UserId: e.userId, CategoryId: e.catId, TopicId: e.topic1, Count: 3, Language: "ru"})
	require.NoError(t, err)
	assert.Equal(t, 5, res.Created, "count minimum 5 ga ko'tariladi")
	assert.Equal(t, 1500, res.TokensUsed)
	assert.Equal(t, 2, res.CoinsSpent) // 1500 token = ceil(1.5) = 2 tanga (bepul tangalardan)
	assert.Equal(t, "topic", res.Deck.Kind)
	assert.Equal(t, 5, res.Deck.CardCount)
	assert.Equal(t, "OKS", res.Deck.Title)
	assert.Equal(t, "faqat STEMI", e.cardAI.gotInput.TopicHints[0])
	assert.Equal(t, "ru", e.cardAI.gotInput.Language)

	w, _ := e.biller().Wallet(ctx, e.userId)
	assert.Equal(t, 3, w.FreeCoinsLeft)

	// tibbiy bo'lmagan erkin mavzu: tanga baribir yechiladi (AI ishlatilgan), deck yaratilmaydi
	e.cardAI.notMedial = true
	_, err = e.flash.CreateDeck(ctx, &m.FlashcardDeckCreateReq{UserId: e.userId, TopicText: "Bitcoin narxi"})
	assert.ErrorIs(t, err, errs.ErrTopicNotMedical)
	decks, _ := e.flash.ListDecks(ctx, e.userId)
	assert.Len(t, decks, 1)
	e.cardAI.notMedial = false

	// hamyon bo'sh: buyurtma qilib bo'lmaydi
	_, _ = e.biller().Charge(ctx, e.userId, 100, "test", "t", "x")
	_, err = e.flash.CreateDeck(ctx, &m.FlashcardDeckCreateReq{UserId: e.userId, CategoryId: e.catId})
	assert.ErrorIs(t, err, errs.ErrInsufficientCoins)

	// o'chirilgan to'plam kartalarini ham olib ketadi
	require.NoError(t, e.flash.DeleteDeck(ctx, e.userId, res.Deck.Id))
	st, _ := e.flash.Stats(ctx, e.userId)
	assert.Equal(t, 0, st.Total)
}

func (e *practiceEnv) biller() *Biller { return NewBiller(e.pool, e.settings) }
