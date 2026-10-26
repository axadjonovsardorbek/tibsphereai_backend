package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"tibsphereai/internal/ai"
	"tibsphereai/pkg/errs"
	m "tibsphereai/pkg/models"
	st "tibsphereai/storage/postgres"
	"time"
)

const (
	perfLimit         = 1000
	perTopicWindow    = 30
	recencyHalfLife   = 14.0 // kun: eski natijalar ta'siri yarimga tushadi
	maxPathSteps      = 6
	weakBelow         = 50
	strongFrom        = 75
	learningBelowAns  = 3
	staleDays         = 21
	roadmapCacheHours = 24
)

// RoadmapService - bilim xaritasi (qaysi mavzu kuchli/zaif) va shaxsiy o'qish yo'li (TZ_MVP).
// Xarita va qadamlar har so'rovda hisoblanadi (bepul, deterministik); AI faqat qadam izohlari va
// xulosani yozadi (tokenga tanga yechiladi, natija 24 soat keshlanadi).
type RoadmapService struct {
	storage  st.Storage
	ai       *ai.Client
	settings *AppSettingService
	biller   *Biller
	now      func() time.Time
}

func NewRoadmapService(storage *st.Storage, aiClient *ai.Client, settings *AppSettingService) *RoadmapService {
	return &RoadmapService{storage: *storage, ai: aiClient, settings: settings, biller: NewBiller(storage.Db, settings), now: time.Now}
}

func levelFor(answered int, mastery int) string {
	switch {
	case answered == 0:
		return m.MasteryUnexplored
	case answered < learningBelowAns:
		return m.MasteryLearning
	case mastery < weakBelow:
		return m.MasteryWeak
	case mastery < strongFrom:
		return m.MasteryMedium
	default:
		return m.MasteryStrong
	}
}

// BuildMap - katalogdagi har bir faol mavzu uchun o'zlashtirish darajasi. mastery - so'nggi (eng ko'pi bilan 30)
// javob ballarining yangilik bo'yicha vaznlangan o'rtachasi (yarim umr 14 kun). perf: eng yangisi birinchi.
func BuildMap(catalog []m.CatalogTopic, perf []m.PerfRow, cards map[string]m.TopicCards, now time.Time) []m.TopicMastery {
	byTopic := map[string][]m.PerfRow{}
	for _, p := range perf {
		if len(byTopic[p.TopicId]) < perTopicWindow {
			byTopic[p.TopicId] = append(byTopic[p.TopicId], p)
		}
	}
	out := make([]m.TopicMastery, 0, len(catalog))
	for _, ct := range catalog {
		tm := m.TopicMastery{
			CategoryId: ct.CategoryId, CategoryName: ct.CategoryName, TopicId: ct.TopicId, TopicName: ct.TopicName,
		}
		if c, ok := cards[ct.TopicId]; ok {
			tm.Cards, tm.CardsDue = c.Total, c.Due
		}
		rows := byTopic[ct.TopicId]
		tm.Answered = len(rows)
		if len(rows) > 0 {
			var wSum, sSum float64
			correct := 0
			for _, p := range rows {
				age := math.Max(0, now.Sub(p.AnsweredAt).Hours()/24)
				w := math.Pow(0.5, age/recencyHalfLife)
				wSum += w
				sSum += w * float64(p.Score)
				if p.IsCorrect {
					correct++
				}
			}
			if wSum > 0 {
				tm.Mastery = int(math.Round(sSum / wSum))
			}
			tm.Accuracy = float64(correct) / float64(len(rows))
			last := rows[0].AnsweredAt.Format(time.RFC3339)
			tm.LastPracticed = &last
		}
		tm.Level = levelFor(tm.Answered, tm.Mastery)
		out = append(out, tm)
	}
	return out
}

func daysSince(ts *string, now time.Time) int {
	if ts == nil {
		return 9999
	}
	t, err := time.Parse(time.RFC3339, *ts)
	if err != nil {
		return 9999
	}
	return int(now.Sub(t).Hours() / 24)
}

// priority - mavzuni o'qish yo'liga qo'shish ustuvorligi (katta = muhimroq).
func priority(t m.TopicMastery, now time.Time, practicedCats map[string]bool) float64 {
	stale := float64(min(daysSince(t.LastPracticed, now), 60)) / 60 * 15
	switch t.Level {
	case m.MasteryWeak:
		return 100 - float64(t.Mastery) + stale
	case m.MasteryLearning:
		return 60 + stale
	case m.MasteryUnexplored:
		p := 40.0
		if practicedCats[t.CategoryId] {
			p += 10 // foydalanuvchi allaqachon shu bo'limda ishlagan
		}
		return p
	case m.MasteryMedium:
		return 30 + stale
	default: // strong: faqat uzoq vaqt mashq qilinmagan bo'lsa
		if daysSince(t.LastPracticed, now) >= staleDays {
			return 20 + stale
		}
		return 0
	}
}

// PlanPath - o'qish yo'li: 1) bugungi fleshkarta takrorlash, 2) ustuvorlik bo'yicha mavzular.
func PlanPath(mp []m.TopicMastery, dueCards int, lang string, now time.Time) []m.RoadmapStep {
	var steps []m.RoadmapStep
	add := func(s m.RoadmapStep) {
		s.Order = len(steps) + 1
		steps = append(steps, s)
	}
	if dueCards > 0 {
		add(m.RoadmapStep{Action: m.RoadmapActionFlashcardReview, Count: min(dueCards, 20), Reason: reasonText(lang, "due", map[string]any{"n": dueCards})})
	}

	practiced := map[string]bool{}
	for _, t := range mp {
		if t.Answered > 0 {
			practiced[t.CategoryId] = true
		}
	}
	type cand struct {
		t m.TopicMastery
		p float64
	}
	var cands []cand
	for _, t := range mp {
		if p := priority(t, now, practiced); p > 0 {
			cands = append(cands, cand{t, p})
		}
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].p > cands[j].p })

	for _, c := range cands {
		if len(steps) >= maxPathSteps {
			break
		}
		t := c.t
		name := nameIn(t.TopicName, lang)
		base := m.RoadmapStep{CategoryId: t.CategoryId, TopicId: t.TopicId, TopicName: name}
		switch t.Level {
		case m.MasteryWeak:
			s := base
			s.Action, s.Count = m.RoadmapActionTest, 10
			s.Difficulty = "medium"
			if t.Mastery < 35 {
				s.Difficulty = "easy"
			}
			s.Reason = reasonText(lang, "weak", map[string]any{"topic": name, "mastery": t.Mastery})
			add(s)
			if t.Cards == 0 && len(steps) < maxPathSteps {
				d := base
				d.Action, d.Count = m.RoadmapActionDeck, 10
				d.Reason = reasonText(lang, "deck", map[string]any{"topic": name})
				add(d)
			}
		case m.MasteryLearning:
			s := base
			s.Action, s.Count, s.Difficulty = m.RoadmapActionTest, 5, "medium"
			s.Reason = reasonText(lang, "learning", map[string]any{"topic": name, "n": t.Answered})
			add(s)
		case m.MasteryUnexplored:
			s := base
			s.Action, s.Count, s.Difficulty = m.RoadmapActionTest, 5, "easy"
			s.Reason = reasonText(lang, "unexplored", map[string]any{"topic": name})
			add(s)
		case m.MasteryMedium:
			s := base
			s.Action, s.Difficulty = m.RoadmapActionCase, "medium"
			s.Reason = reasonText(lang, "medium", map[string]any{"topic": name, "mastery": t.Mastery})
			add(s)
		default: // strong, lekin uzoq vaqt mashq qilinmagan
			s := base
			s.Action, s.Count, s.Difficulty = m.RoadmapActionTest, 10, "hard"
			s.Reason = reasonText(lang, "stale", map[string]any{"topic": name, "days": daysSince(t.LastPracticed, now)})
			add(s)
		}
	}
	if len(steps) == 0 { // hamma narsa a'lo yoki katalog bo'sh: umumiy tavsiya
		add(m.RoadmapStep{Action: m.RoadmapActionTest, Count: 10, Difficulty: "hard", Reason: reasonText(lang, "keep", nil)})
	}
	return steps
}

func nameIn(name map[string]string, lang string) string {
	if v := name[lang]; v != "" {
		return v
	}
	return pickTopicName(name)
}

var reasonTemplates = map[string]map[string]string{
	"due": {
		"uz": "Bugun takrorlashga %[1]v ta fleshkarta tayyor: takrorlash (bepul) esda saqlashni mustahkamlaydi.",
		"ru": "Сегодня к повторению готово карточек: %[1]v. Повторение (бесплатно) закрепляет знания.",
		"en": "%[1]v flashcards are due today: reviewing (free) locks in what you know.",
	},
	"weak": {
		"uz": "«%[1]v» mavzusi hozircha zaif (%[2]v/100): avval osonroq savollar bilan poydevorni mustahkamlang.",
		"ru": "Тема «%[1]v» пока слабая (%[2]v/100): сначала закрепите основу на более простых вопросах.",
		"en": "“%[1]v” is still weak (%[2]v/100): build the foundation with easier questions first.",
	},
	"deck": {
		"uz": "«%[1]v» bo'yicha fleshkartalar hali yo'q: AI to'plam yarating va har kuni takrorlang.",
		"ru": "По теме «%[1]v» ещё нет карточек: создайте набор с ИИ и повторяйте каждый день.",
		"en": "No flashcards for “%[1]v” yet: generate an AI deck and review it daily.",
	},
	"learning": {
		"uz": "«%[1]v» bo'yicha ma'lumot kam (%[2]v ta javob): baholash aniqroq bo'lishi uchun yana mashq qiling.",
		"ru": "По теме «%[1]v» мало данных (ответов: %[2]v): потренируйтесь ещё, чтобы оценка стала точнее.",
		"en": "Little data on “%[1]v” (%[2]v answers): practice more for a clearer picture.",
	},
	"unexplored": {
		"uz": "«%[1]v» mavzusini hali boshlamagansiz: qisqa oson test bilan tanishib chiqing.",
		"ru": "Тему «%[1]v» вы ещё не начинали: познакомьтесь с ней коротким лёгким тестом.",
		"en": "You haven't started “%[1]v” yet: try a short easy test.",
	},
	"medium": {
		"uz": "«%[1]v» o'rtacha (%[2]v/100): klinik keys bilan bilimni amaliyotda mustahkamlang.",
		"ru": "Тема «%[1]v» на среднем уровне (%[2]v/100): закрепите знания клиническим кейсом.",
		"en": "“%[1]v” is at a medium level (%[2]v/100): reinforce it with a clinical case.",
	},
	"stale": {
		"uz": "«%[1]v» ni yaxshi bilasiz, lekin %[2]v kundan beri mashq qilinmagan: qiyin test bilan darajani saqlang.",
		"ru": "Тему «%[1]v» вы знаете хорошо, но не практиковали %[2]v дн.: поддержите уровень сложным тестом.",
		"en": "You know “%[1]v” well but haven't practiced for %[2]v days: keep it sharp with a hard test.",
	},
	"keep": {
		"uz": "Natijalaringiz a'lo: bilimni saqlash uchun qiyin darajada mashq qilishda davom eting.",
		"ru": "Результаты отличные: продолжайте практиковаться на высоком уровне сложности.",
		"en": "Great results: keep practicing at a hard level to stay sharp.",
	},
}

// reasonText - shablon (uz/ru/en). args: n, topic, mastery, days kalitlari bo'yicha tartibli.
func reasonText(lang, key string, args map[string]any) string {
	t := reasonTemplates[key]
	tpl := t[lang]
	if tpl == "" {
		tpl = t["uz"]
	}
	switch key {
	case "due":
		return fmt.Sprintf(tpl, args["n"])
	case "weak", "medium":
		return fmt.Sprintf(tpl, args["topic"], args["mastery"])
	case "learning":
		return fmt.Sprintf(tpl, args["topic"], args["n"])
	case "stale":
		return fmt.Sprintf(tpl, args["topic"], args["days"])
	case "deck", "unexplored":
		return fmt.Sprintf(tpl, args["topic"])
	default:
		return tpl
	}
}

// summaryLines - AI uchun xulosa satrlari (raqamlar backenddan; AI to'qimaydi).
func summaryLines(mp []m.TopicMastery, lang string) []string {
	var lines []string
	counts := map[string]int{}
	for _, t := range mp {
		counts[t.Level]++
	}
	lines = append(lines, fmt.Sprintf("jami mavzular: %d; kuchli: %d, o'rtacha: %d, zaif: %d, ma'lumot kam: %d, boshlanmagan: %d",
		len(mp), counts[m.MasteryStrong], counts[m.MasteryMedium], counts[m.MasteryWeak], counts[m.MasteryLearning], counts[m.MasteryUnexplored]))
	sorted := append([]m.TopicMastery{}, mp...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Mastery > sorted[j].Mastery })
	for _, t := range sorted {
		if t.Answered > 0 && len(lines) < 12 {
			lines = append(lines, fmt.Sprintf("%s: o'zlashtirish %d/100, %d ta javob, daraja=%s", nameIn(t.TopicName, lang), t.Mastery, t.Answered, t.Level))
		}
	}
	return lines
}

func stepLines(path []m.RoadmapStep) []string {
	out := make([]string, 0, len(path))
	for _, s := range path {
		l := fmt.Sprintf("harakat=%s", s.Action)
		if s.TopicName != "" {
			l += fmt.Sprintf(", mavzu=%s", s.TopicName)
		}
		if s.Difficulty != "" {
			l += fmt.Sprintf(", qiyinlik=%s", s.Difficulty)
		}
		if s.Count > 0 {
			l += fmt.Sprintf(", soni=%d", s.Count)
		}
		out = append(out, l+" | asosiy sabab: "+s.Reason)
	}
	return out
}

type roadmapPayload struct {
	Summary string   `json:"summary"`
	Reasons []string `json:"reasons"`
}

// Get - bilim xaritasi + o'qish yo'li. Kesh (24 soat, til mos) bo'lsa AI izohlari ham qo'shiladi, aks holda
// shablon izohlari (bepul). Qadamlar kesh yaratilgandan keyin o'zgargan bo'lsa ham xarita/qadamlar yangi,
// AI matni faqat qadamlar soni mos kelganda ishlatiladi.
func (s *RoadmapService) Get(ctx context.Context, userId, lang string) (*m.RoadmapRes, error) {
	res, err := s.compute(ctx, userId, lang)
	if err != nil {
		return nil, err
	}
	payload, cLang, at, err := s.storage.RoadmapS.GetCache(userId)
	if err == nil && payload != nil && cLang == lang && s.now().Sub(at) < roadmapCacheHours*time.Hour {
		var p roadmapPayload
		if json.Unmarshal(payload, &p) == nil && len(p.Reasons) == len(res.Path) && p.Summary != "" {
			for i := range res.Path {
				res.Path[i].Reason = p.Reasons[i]
			}
			res.Summary, res.IsAI = p.Summary, true
			t := at.Format(time.RFC3339)
			res.GeneratedAt = &t
		}
	}
	return res, nil
}

func (s *RoadmapService) compute(ctx context.Context, userId, lang string) (*m.RoadmapRes, error) {
	if lang != "ru" && lang != "en" {
		lang = "uz"
	}
	catalog, err := s.storage.RoadmapS.Catalog()
	if err != nil {
		return nil, err
	}
	perf, err := s.storage.RoadmapS.Performance(userId, perfLimit)
	if err != nil {
		return nil, err
	}
	cards, err := s.storage.RoadmapS.CardsByTopic(userId)
	if err != nil {
		return nil, err
	}
	due, err := s.storage.RoadmapS.DueCards(userId)
	if err != nil {
		return nil, err
	}
	now := s.now()
	mp := BuildMap(catalog, perf, cards, now)
	res := &m.RoadmapRes{Map: mp, Path: PlanPath(mp, due, lang, now)}
	for _, t := range mp {
		switch t.Level {
		case m.MasteryStrong:
			res.Strong++
		case m.MasteryWeak:
			res.Weak++
		case m.MasteryUnexplored:
			res.Unexplored++
		}
	}
	return res, nil
}

// Refresh - AI izohlari va xulosani (qayta) yaratadi; token sarfiga tanga yechiladi (obunachilarga bepul).
func (s *RoadmapService) Refresh(ctx context.Context, userId, lang string) (*m.RoadmapRes, error) {
	if lang != "ru" && lang != "en" {
		lang = "uz"
	}
	if !s.biller.HasSubscription(ctx, userId) {
		if err := s.biller.RequireFunds(ctx, userId); err != nil {
			return nil, err
		}
	}
	res, err := s.compute(ctx, userId, lang)
	if err != nil {
		return nil, err
	}
	if !hasAnswers(res.Map) {
		// Hech qanday natija yo'q: AI'ga berish uchun ma'lumot yo'q (token sarflanmaydi).
		return nil, errors.New("not enough data: solve some cases or tests first")
	}

	var (
		summary string
		reasons []string
		usage   *ai.Usage
		aiErr   error
	)
	for attempt := 0; attempt < 2; attempt++ {
		var u *ai.Usage
		summary, reasons, u, aiErr = s.ai.Roadmap.Advise(ctx, lang, summaryLines(res.Map, lang), stepLines(res.Path))
		usage = addUsage(usage, u)
		if aiErr == nil {
			break
		}
	}
	coins := 0
	if usage != nil {
		logAIUsage(&s.storage, "roadmap", usage, userId, "")
		if c := CoinsForTokens(usage.PromptTokens+usage.OutputTokens, s.settings.CoinsPer1kTokens(ctx)); c > 0 {
			charged, cErr := s.biller.Charge(ctx, userId, c, "roadmap", "roadmap", "")
			if cErr != nil {
				fmt.Println("failed to charge roadmap:", cErr)
			}
			coins = charged
		}
	}
	if aiErr != nil {
		return nil, errs.ErrPracticeGenFailed.Wrap(aiErr)
	}
	b, _ := json.Marshal(roadmapPayload{Summary: summary, Reasons: reasons})
	if err := s.storage.RoadmapS.SaveCache(userId, b, lang); err != nil {
		return nil, err
	}
	for i := range res.Path {
		res.Path[i].Reason = reasons[i]
	}
	res.Summary, res.IsAI, res.CoinsSpent = summary, true, coins
	t := s.now().Format(time.RFC3339)
	res.GeneratedAt = &t
	return res, nil
}

func hasAnswers(mp []m.TopicMastery) bool {
	for _, t := range mp {
		if t.Answered > 0 {
			return true
		}
	}
	return false
}
