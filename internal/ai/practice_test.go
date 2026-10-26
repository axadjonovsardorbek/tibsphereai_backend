package ai

import (
	"strings"
	"testing"

	"tibsphereai/pkg/models"

	"github.com/stretchr/testify/assert"
)

func TestValidateTest(t *testing.T) {
	ok := func() *models.PracticeTestGen {
		return &models.PracticeTestGen{Question: "Q?", Options: []string{"a", "b", "c", "d"}, CorrectIndex: 1, Explanation: "e"}
	}
	assert.NoError(t, ValidateTest(ok()))

	for name, mut := range map[string]func(*models.PracticeTestGen){
		"3 variant":       func(t *models.PracticeTestGen) { t.Options = t.Options[:3] },
		"5 variant":       func(t *models.PracticeTestGen) { t.Options = append(t.Options, "e") },
		"bo'sh variant":   func(t *models.PracticeTestGen) { t.Options[2] = "  " },
		"bir xil variant": func(t *models.PracticeTestGen) { t.Options[3] = "A" },
		"indeks 4":        func(t *models.PracticeTestGen) { t.CorrectIndex = 4 },
		"indeks -1":       func(t *models.PracticeTestGen) { t.CorrectIndex = -1 },
		"bo'sh savol":     func(t *models.PracticeTestGen) { t.Question = "" },
		"bo'sh izoh":      func(t *models.PracticeTestGen) { t.Explanation = " " },
	} {
		tg := ok()
		mut(tg)
		assert.Error(t, ValidateTest(tg), name)
	}
}

func TestValidateCase(t *testing.T) {
	c := &models.PracticeCaseGen{ChiefComplaint: " shikoyat ", ExpectedAnswer: "javob", Explanation: "izoh"}
	assert.NoError(t, ValidateCase(c))
	assert.Equal(t, "shikoyat", c.ChiefComplaint)
	assert.NotNil(t, c.Patient)
	assert.Error(t, ValidateCase(&models.PracticeCaseGen{ChiefComplaint: "x", Explanation: "y"}))
	assert.Error(t, ValidateCase(&models.PracticeCaseGen{ExpectedAnswer: "x", Explanation: "y"}))
}

// Verdict har doim score'ga mos keladi (AI "correct" deb 20 ball bersa ham to'g'ri hisoblanmaydi).
func TestNormalizeEval(t *testing.T) {
	cases := []struct {
		in      int
		verdict string
		want    int
	}{{100, "correct", 100}, {85, "wrong", 85}, {84, "correct", 84}, {35, "wrong", 35}, {34, "partial", 34}, {-5, "correct", 0}, {250, "wrong", 100}}
	for _, c := range cases {
		ev := &models.PracticeEval{Score: c.in, Verdict: c.verdict, Feedback: "  f "}
		NormalizeEval(ev)
		assert.Equal(t, c.want, ev.Score)
		switch {
		case ev.Score >= 85:
			assert.Equal(t, "correct", ev.Verdict)
		case ev.Score >= 35:
			assert.Equal(t, "partial", ev.Verdict)
		default:
			assert.Equal(t, "wrong", ev.Verdict)
		}
		assert.Equal(t, "f", ev.Feedback)
	}
}

func TestBuildCommon_IncludesContextAndPrevious(t *testing.T) {
	in := &models.PracticeGenInput{
		Language: "ru", Difficulty: "hard", CategoryName: "Kardio", TopicNames: []string{"OKS"}, TopicHints: []string{"faqat STEMI"},
		Previous: []string{"oldingi case"}, Chunks: []models.SourceChunk{{SourceName: "Kitob", Content: "MATN"}},
	}
	p := buildCommon(in)
	for _, want := range []string{"Kardio", "OKS", "faqat STEMI", "rus", "oldingi case", "[1] (Kitob)", "MATN", "atipik"} {
		assert.Contains(t, p, want)
	}
	// manba yo'q - umumiy bilim eslatmasi
	assert.Contains(t, buildCommon(&models.PracticeGenInput{Language: "uz"}), "Manba berilmagan")
	assert.True(t, strings.Contains(buildCommon(&models.PracticeGenInput{Language: "xx"}), "o'zbek"))
}

func TestValidateFlashcards(t *testing.T) {
	long := make([]rune, 400)
	for i := range long {
		long[i] = 'a'
	}
	in := []models.FlashcardGen{
		{Front: " Savol 1 ", Back: " Javob 1 "},
		{Front: "savol 1", Back: "takroriy old tomon"},
		{Front: "", Back: "bo'sh front"},
		{Front: "Back yo'q", Back: "  "},
		{Front: string(long), Back: "juda uzun front"},
		{Front: "Savol 2", Back: "Javob 2"},
		{Front: "Savol 3", Back: "Javob 3"},
	}
	out := ValidateFlashcards(in, 2)
	assert.Len(t, out, 2)
	assert.Equal(t, "Savol 1", out[0].Front)
	assert.Equal(t, "Javob 1", out[0].Back)
	assert.Equal(t, "Savol 2", out[1].Front)
	assert.Len(t, ValidateFlashcards(in, 0), 3) // 0 = cheklovsiz: faqat yaroqlilari
}
