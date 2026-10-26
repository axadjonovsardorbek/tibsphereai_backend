// Package ai - Gemini AI integratsiya qatlami (BACKEND_STRUCTURE.md 4,5,7-bo'lim):
// bemor personaji bilan muloqot, debriefing hisoboti va admin uchun avtomatik
// keys generatsiyasi. GEMINI_API_KEY bo'sh bo'lganda mock implementatsiya
// ishlatiladi - shu tufayli service/handler qatlamlari haqiqiy kalit kelguncha
// ham to'liq ishlaydi (deterministik, lekin real bo'lmagan javoblar bilan).
package ai

import (
	"context"
	"tibsphereai/internal/config"
	"tibsphereai/pkg/models"
	"tibsphereai/storage"
)

// Usage - bitta Gemini chaqiruvi uchun sarflangan token/xarajat (dashboard va
// ai_usage_logs uchun, BACKEND_STRUCTURE.md 7-bo'lim: "Gemini AI xarajatlari
// va sarflangan tokenlar").
type Usage struct {
	PromptTokens int
	OutputTokens int
	TotalTokens  int
	CostUSD      float64
}

// PatientReplyRes - chat xabariga javob: shifokor xabari ikki xil bo'lishi mumkin -
// (1) bemordan anamnez so'ragan savol ("qayerda og'riq bor?") - shunda IsAction=false,
// PatientReply bemor tilidan tabiiy javob; (2) klinik qaror/harakat bayonoti
// ("epinefrin beraman", "EKG qilamiz") - shunda IsAction=true va bu ENDI bemorga emas,
// balki platformaga aytilgan deb hisoblanadi: admin kiritgan ExpectedAnswer/expected_steps'ga
// solishtirib to'g'ri/xato deb baholanadi (xuddi ActionEvaluatorAI kabi).
// Ko'ring: "bemor bilan gaplashtirish xato, platforma bilan gaplashishi kerak" (audit talabi).
type PatientReplyRes struct {
	IsAction     bool
	PatientReply string // faqat IsAction=false bo'lsa mazmunli
	IsCorrect    bool   // faqat IsAction=true bo'lsa mazmunli
	HealthDelta  int    // faqat IsAction=true bo'lsa mazmunli (-100..+100)
	Feedback     string // faqat IsAction=true bo'lsa mazmunli - platforma tilidan qisqa klinik izoh
}

// PatientAI - Chat (BACKEND_STRUCTURE.md 4-bo'lim): shifokorning chatdagi xabari
// anamnez savolimi yoki klinik qaror bayonotimi aniqlaydi va shunga qarab bemor
// tilidan javob YOKI platforma tilidan to'g'ri/xato bahosini qaytaradi.
// expectedAnswer - admin kiritgan kutilgan to'g'ri javob (Case.ExpectedAnswer),
// baholash shu asosda amalga oshiriladi.
type PatientAI interface {
	Reply(ctx context.Context, caseScenario map[string]any, expectedAnswer, question string) (*PatientReplyRes, *Usage, error)
}

// DebriefAI - Yakuniy hisobot (BACKEND_STRUCTURE.md 5-bo'lim): qadamlar logi
// asosida tahliliy debriefing tayyorlaydi.
type DebriefAI interface {
	Generate(ctx context.Context, caseScenario map[string]any, events []models.SimulationEventReq) (*models.DebriefReportCreateReq, *Usage, error)
}

// CaseGeneratorAI - Admin uchun avtomatik keys yaratish (BACKEND_STRUCTURE.md 7-bo'lim).
type CaseGeneratorAI interface {
	Generate(ctx context.Context, req *models.CaseGenReq) (*models.CaseCreateReq, *Usage, error)
}

// ActionEvalRes - shifokorning bitta klinik harakati (ko'rik/tahlil/instrumental/
// dori/protsedura - "question" bundan mustasno, u alohida PatientAI orqali
// suhbat sifatida ishlaydi) qanchalik to'g'ri bajarilganini bildiradi.
type ActionEvalRes struct {
	IsCorrect   bool
	HealthDelta int // -100..+100 (manfiy - noto'g'ri/kechiktirilgan harakat, ijobiy - to'g'ri)
	Feedback    string
}

// ActionEvaluatorAI - "question" bo'lmagan qadamlarni (BACKEND_STRUCTURE.md
// 4-bo'lim: fizikal ko'rik, diagnostika, muolaja) case scenario/expected_steps
// asosida real vaqtda baholaydi - shu orqali Sog'liq shkalasi va Ball haqiqiy
// klinik to'g'rilikka mos ravishda o'zgaradi (avval har doim health_delta=0 edi).
type ActionEvaluatorAI interface {
	Evaluate(ctx context.Context, caseScenario map[string]any, expectedAnswer, eventType string, payload map[string]any) (*ActionEvalRes, *Usage, error)
}

// VoiceAI - "voice orqali suhbatlashish": tibsimulyatsiyada shifokorning
// aytgan gapini (audio) matnga aylantiradi (Transcribe) va platforma/bemor
// javobini tabiiy ovozga (Synthesize, o'zbekcha) aylantiradi - ikkalasi ham
// Gemini native audio orqali (gemini-*-tts model, contents+inlineData audio),
// brauzerning Web Speech API'siga (STT/TTS) umuman bog'liq emas.
type VoiceAI interface {
	// Transcribe - audioB64 (raw baza64, masalan audio/webm) ni matnga aylantiradi.
	Transcribe(ctx context.Context, audioB64, mimeType string) (text string, usage *Usage, err error)
	// Synthesize - matnni (o'zbekcha) tabiiy ovozli audioga aylantiradi;
	// audioB64 - WAV formatidagi audio, baza64 kodlangan.
	Synthesize(ctx context.Context, text string) (audioB64 string, usage *Usage, err error)
}

// Client - AI xizmatlarini bitta joyda ushlab turadi.
type Client struct {
	Practice   PracticeAI
	Flashcards FlashcardAI
	Roadmap    RoadmapAI
	Image      ImageAI
	Embedder   EmbedderAI
	Patient    PatientAI
	Debrief    DebriefAI
	CaseGen    CaseGeneratorAI
	Action     ActionEvaluatorAI
	Voice      VoiceAI
	Enabled    bool // haqiqiy Gemini API ulanganmi (GEMINI_API_KEY bor)
}

// NewClient - GEMINI_API_KEY bo'sh bo'lsa mock, aks holda haqiqiy Gemini
// klientini qaytaradi. promptRepo - admin panelda tahrirlangan promptlarni
// (ai_prompts jadvali) o'qish uchun, kodda qattiq yozilgan default'larni
// GEMINI_API_KEY bo'lsa ham o'zgartirmasdan ishlatish mumkin bo'lishi uchun.
func NewClient(cfg *config.Config, promptRepo storage.AIPromptI) *Client {
	if cfg.GEMINI_API_KEY == "" {
		m := &mockClient{}
		return &Client{Patient: m, Debrief: m, Action: m, CaseGen: mockCaseGen{}, Voice: m, Practice: mockPractice{}, Flashcards: mockPractice{}, Roadmap: mockPractice{}, Image: mockPractice{}, Embedder: mockPractice{}, Enabled: false}
	}

	g := newGeminiClient(cfg, promptRepo)
	return &Client{Patient: g, Debrief: g, Action: g, CaseGen: geminiCaseGen{g}, Voice: g, Practice: geminiPractice{g}, Flashcards: geminiPractice{g}, Roadmap: geminiPractice{g}, Image: geminiPractice{g}, Embedder: geminiPractice{g}, Enabled: true}
}
