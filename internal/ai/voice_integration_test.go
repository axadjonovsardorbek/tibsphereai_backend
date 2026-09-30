package ai

import (
	"context"
	"os"
	"strings"
	"testing"

	"tibsphereai/internal/config"

	"github.com/joho/godotenv"
)

// realGeminiClientForTest - .env'dagi haqiqiy GEMINI_API_KEY bilan geminiClient
// quradi; kalit topilmasa (CI, boshqa dasturchi mashinasi) test skip qilinadi -
// bu haqiqiy tarmoq/billing chaqiruvi, doim ishlaydigan unit test emas.
func realGeminiClientForTest(t *testing.T) *geminiClient {
	t.Helper()
	_ = godotenv.Load("../../.env")
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("skipping: GEMINI_API_KEY yo'q (.env)")
	}
	ttsModel := os.Getenv("GEMINI_TTS_MODEL")
	if ttsModel == "" {
		ttsModel = "gemini-3.8-flash-tts"
	}
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-2.0-flash"
	}
	return newGeminiClient(&config.Config{
		GEMINI_API_KEY:   apiKey,
		GEMINI_MODEL:     model,
		GEMINI_TTS_MODEL: ttsModel,
	}, nil)
}

// TestGeminiClient_Synthesize_ProducesUzbekAudio - haqiqiy Gemini TTS chaqiruvi:
// o'zbekcha matn audioga aylantiriladi, keyin SHU audio yana Geminiga (oddiy
// chat modeliga) qaytarilib, "qaysi til/nima deyilgan" deb so'raladi - shu
// orqali chiqqan ovoz haqiqatan o'zbekcha va to'g'ri ekanligi mustaqil tekshiriladi
// (nafaqat "xato qaytmadi", balki "kontent to'g'ri" darajasida).
func TestGeminiClient_Synthesize_ProducesUzbekAudio(t *testing.T) {
	g := realGeminiClientForTest(t)
	ctx := context.Background()

	const phrase = "Salom, bugun o'zingizni qanday his qilyapsiz?"
	audioB64, usage, err := g.Synthesize(ctx, phrase)
	if err != nil {
		t.Fatalf("Synthesize error: %v", err)
	}
	if len(audioB64) < 1000 {
		t.Fatalf("audio juda qisqa/bo'sh ko'rinadi: %d belgi", len(audioB64))
	}
	if usage == nil {
		t.Fatal("usage nil bo'lmasligi kerak")
	}
	t.Logf("Synthesize OK: audio %d base64 belgi, usage=%+v", len(audioB64), *usage)

	// Mustaqil tekshiruv: yaratilgan audio'ni yana Geminiga (oddiy chat modeli)
	// yuborib, tilini va matnini so'raymiz.
	verifyReq := geminiRequest{
		Contents: []geminiContent{{
			Role: "user",
			Parts: []geminiPart{
				{Text: "Bu audio qaysi tilda? Nima deyilgan? Faqat til nomi va aynan aytilgan gapni yoz."},
				{InlineData: &geminiInlineData{MimeType: "audio/wav", Data: audioB64}},
			},
		}},
	}
	gr, _, err := g.doRequest(ctx, g.model, verifyReq)
	if err != nil {
		t.Fatalf("tekshiruv chaqiruvi xato: %v", err)
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		t.Fatal("tekshiruv javobida candidate yo'q")
	}
	var verifyText strings.Builder
	for _, p := range gr.Candidates[0].Content.Parts {
		verifyText.WriteString(p.Text)
	}
	got := verifyText.String()
	t.Logf("Gemini tasdiqlovi: %s", got)

	lower := strings.ToLower(got)
	if !strings.Contains(lower, "o'zbek") && !strings.Contains(lower, "ozbek") && !strings.Contains(lower, "uzbek") {
		t.Fatalf("kutilgan: audio tili o'zbekcha deb tasdiqlanishi kerak edi, olindi: %s", got)
	}
}

// TestGeminiClient_Transcribe_SmokeTest - Transcribe metodi haqiqiy Gemini bilan
// so'rov yuboradi va xatosiz matn qaytarishini tekshiradi. Sintetik audio input
// tayyorlash qiyin bo'lgani uchun (mikrofon yozuvi kerak), avval Synthesize bilan
// audio yaratib, keyin o'sha audio'ni Transcribe'ga uzatamiz - to'liq round-trip.
func TestGeminiClient_Transcribe_SmokeTest(t *testing.T) {
	g := realGeminiClientForTest(t)
	ctx := context.Background()

	const original = "Bemorga aspirin uch yuz milligramm berildi"
	audioB64, _, err := g.Synthesize(ctx, original)
	if err != nil {
		t.Fatalf("Synthesize error: %v", err)
	}

	text, usage, err := g.Transcribe(ctx, audioB64, "audio/wav")
	if err != nil {
		t.Fatalf("Transcribe error: %v", err)
	}
	if strings.TrimSpace(text) == "" {
		t.Fatal("Transcribe bo'sh matn qaytardi")
	}
	if usage == nil {
		t.Fatal("usage nil bo'lmasligi kerak")
	}
	t.Logf("Transcribe OK: %q (asl: %q)", text, original)
}
