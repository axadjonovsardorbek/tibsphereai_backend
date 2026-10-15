package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"tibsphereai/internal/config"
	"tibsphereai/pkg/models"
	"tibsphereai/storage"
)

// geminiClient - Gemini "generateContent" REST API'ga real so'rov yuboruvchi
// klient (google.golang.org/genai o'rniga oddiy net/http - qo'shimcha
// bog'liqlik shart emas). GEMINI_API_KEY berilganda ishlatiladi.
type geminiClient struct {
	apiKey     string
	model      string
	ttsModel   string
	promptRepo storage.AIPromptI
	httpClient *http.Client
}

func newGeminiClient(cfg *config.Config, promptRepo storage.AIPromptI) *geminiClient {
	return &geminiClient{
		apiKey:     cfg.GEMINI_API_KEY,
		model:      cfg.GEMINI_MODEL,
		ttsModel:   cfg.GEMINI_TTS_MODEL,
		promptRepo: promptRepo,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// --- Default promptlar (admin panelda ai_prompts jadvali orqali kodni
// o'zgartirmasdan qayta yozilishi mumkin, BACKEND_STRUCTURE.md 7-bo'lim) ---

// defaultPatientPersonaPrompt - shifokorning chatdagi xabari ikki xil bo'ladi:
// (1) bemordan anamnez so'ragan savol - shunda bemor tilidan tabiiy javob kerak;
// (2) klinik qaror/harakat bayonoti ("epinefrin beraman", "EKG qilamiz") - bu ENDI
// bemorga emas, platformaga aytilgan deb hisoblanadi va ExpectedAnswer/expected_steps'ga
// solishtirib to'g'ri/xato deb baholanadi (bemor "aniq bo'lmagan javob" bermaydi).
const defaultPatientPersonaPrompt = `Sen klinik simulyatsiya platformasining ekspert-baholovchisan. Bemor GAPIRMAYDI. Shifokor (foydalanuvchi) chatga case savoliga javob (tashxis yoki davolash qarori) yozdi. Uni "Kutilgan to'g'ri javob" bilan solishtir.

Qoidalar:
- Har doim is_action=true qaytar. patient_reply bo'sh qoldir.
- Javob mazmunan to'g'ri bo'lsa (sinonim, boshqa so'zlar bilan aytilgan yoki asosiy qismi to'g'ri bo'lsa ham): is_correct=true, health_delta=0, feedback - qisqa tabrik (1 gap).
- Javob xato yoki to'liq bo'lmasa: is_correct=false, health_delta=0, feedback - 1-2 gapli ishora: to'g'ri yo'nalishni ayt (qaysi belgi/tekshiruvga e'tibor berish kerak), lekin to'g'ri javobning o'zini AYTMA.
- Savol bilan bog'liq bo'lmagan yoki ma'nosiz xabar: is_correct=false, health_delta=0, feedback - javob yozishni so'ra.

Bemor holati (case scenario, JSON):
%s

Kutilgan to'g'ri javob (admin kiritgan, asosiy mezon):
%s

Javobni FAQAT quyidagi JSON formatda qaytar, boshqa hech qanday matn yoki markdown qo'shma:
{"is_action": true, "patient_reply": "", "is_correct": true, "health_delta": 0, "feedback": "izoh"}`

const defaultDebriefPrompt = `Sen tibbiyot bo'yicha ekspert-baholovchisan. Quyidagi klinik simulyatsiya case'i va shifokor (foydalanuvchi) bosgan qadamlar logi asosida, AHA/ERC kabi xalqaro klinik qo'llanmalarga tayangan holda debriefing tahlili tuz.

Case scenario:
%s

Shifokor bosgan qadamlar (JSON ro'yxat, "type" va "payload" bilan):
%s

Javobni FAQAT quyidagi JSON formatda qaytar, boshqa hech qanday matn, izoh yoki markdown belgisi qo'shma:
{"correct_steps": ["..."], "incorrect_steps": ["..."], "guideline_notes": "...", "weak_topics": ["..."]}`

const defaultCaseGenerationPrompt = `Sen tajribali shoshilinch tibbiy yordam shifokori va ta'lim dizaynerisan. "%s" mavzusi bo'yicha, "%s" qiyinlik darajasida (easy|medium|hard) to'liq klinik simulyatsiya keysi (case) generatsiya qil.

Javobni FAQAT quyidagi JSON formatda qaytar (o'zbek tilida, boshqa hech qanday matn/markdown qo'shma):
{
  "title": {"uz": "...", "ru": "...", "en": "..."},
  "subtitle": {"uz": "...", "ru": "...", "en": "..."},
  "expected_answer": "kutilgan to'g'ri davolash algoritmi matn ko'rinishida",
  "difficulty": "%s",
  "chief_complaint": {"uz": "...", "ru": "...", "en": "..."},
  "patient_age": 45,
  "patient_gender": "male|female",
  "expected_duration_minutes": 10,
  "initial_vitals": {"hr": 0, "bp": "120/80", "spo2": 98, "rr": 16, "temp": 36.6, "gcs": 15},
  "visual_state": "conscious|pain|unconscious|seizure|coma",
  "scenario": {"patient_persona": "bemor personaji tavsifi", "expected_steps": ["..."], "labs": {}, "imaging": {}, "medications": []}
}`

const caseGenFormatSuffix = `

MAJBURIY CHIQISH FORMATI (yuqoridagi barcha format ko'rsatmalaridan USTUN; boshqa kalit nomlari, o'rovchi obyekt va qo'shimcha maydonlar ishlatma; faqat shu JSON, markdown va izohsiz):
{
  "title": {"uz": "...", "ru": "...", "en": "..."},
  "subtitle": {"uz": "...", "ru": "...", "en": "..."},
  "expected_answer": "bitta matn (o'zbekcha): tashxis va birinchi qadamlar",
  "difficulty": "easy|medium|hard",
  "chief_complaint": {"uz": "...", "ru": "...", "en": "..."},
  "patient_age": 45,
  "patient_gender": "male|female",
  "expected_duration_minutes": 10,
  "initial_vitals": {"hr": 0, "bp": "120/80", "spo2": 98, "rr": 16, "temp": 36.6, "gcs": 15},
  "visual_state": "stable|unwell|critical",
  "scenario": {"history": "...", "comorbidities": [], "allergies": [], "medications": [], "expected_steps": ["..."]}
}`

func (g *geminiClient) Reply(ctx context.Context, caseScenario map[string]any, expectedAnswer, question string) (*PatientReplyRes, *Usage, error) {
	tmpl, params := g.loadPrompt(models.AIPromptPatientPersona, defaultPatientPersonaPrompt)
	scenarioJSON, _ := json.MarshalIndent(caseScenario, "", "  ")

	system := tmpl
	if n := strings.Count(system, "%s"); n > 0 {
		args := []any{string(scenarioJSON), expectedAnswer}
		if n < len(args) {
			args = args[:n]
		}
		system = fmt.Sprintf(tmpl, args...)
	}
	prompt := system + "\n\nShifokorning chatdagi xabari: " + question

	text, usage, err := g.generateContent(ctx, prompt, params, true)
	if err != nil {
		return nil, usage, err
	}

	var raw struct {
		IsAction     bool   `json:"is_action"`
		PatientReply string `json:"patient_reply"`
		IsCorrect    bool   `json:"is_correct"`
		HealthDelta  int    `json:"health_delta"`
		Feedback     string `json:"feedback"`
	}
	if err := unmarshalJSONLoose(text, &raw); err != nil {
		return nil, usage, fmt.Errorf("gemini chat response parse error: %w (raw: %s)", err, text)
	}
	return &PatientReplyRes{
		IsAction:     raw.IsAction,
		PatientReply: strings.TrimSpace(raw.PatientReply),
		IsCorrect:    raw.IsCorrect,
		HealthDelta:  raw.HealthDelta,
		Feedback:     strings.TrimSpace(raw.Feedback),
	}, usage, nil
}

func (g *geminiClient) Generate(ctx context.Context, caseScenario map[string]any, events []models.SimulationEventReq) (*models.DebriefReportCreateReq, *Usage, error) {
	tmpl, params := g.loadPrompt(models.AIPromptDebrief, defaultDebriefPrompt)
	scenarioJSON, _ := json.MarshalIndent(caseScenario, "", "  ")
	eventsJSON, _ := json.MarshalIndent(events, "", "  ")

	prompt := tmpl
	if strings.Contains(prompt, "%s") {
		prompt = fmt.Sprintf(tmpl, string(scenarioJSON), string(eventsJSON))
	}

	text, usage, err := g.generateContent(ctx, prompt, params, true)
	if err != nil {
		return nil, usage, err
	}

	var report models.DebriefReportCreateReq
	if err := unmarshalJSONLoose(text, &report); err != nil {
		return nil, usage, fmt.Errorf("gemini debrief response parse error: %w (raw: %s)", err, text)
	}
	return &report, usage, nil
}

// geminiCaseGen - CaseGeneratorAI uchun alohida tip (geminiClient'da
// allaqachon boshqa signature bilan Generate metodi bor - DebriefAI).
type geminiCaseGen struct {
	g *geminiClient
}

func (c geminiCaseGen) Generate(ctx context.Context, genReq *models.CaseGenReq) (*models.CaseCreateReq, *Usage, error) {
	g := c.g
	tmpl, params := g.loadPrompt(models.AIPromptCaseGeneration, defaultCaseGenerationPrompt)
	topic := genReq.Topic
	difficulty := genReq.Difficulty
	if difficulty == "" {
		difficulty = "medium"
	}

	prompt := tmpl
	if strings.Contains(prompt, "%s") {
		// default shablon 3 ta %s kutadi (topic, difficulty, difficulty); admin
		// o'zi yozgan shablonda bironta %s bo'lmasligi ham mumkin - shunda
		// topic/difficulty pastda alohida qo'shib qo'yiladi.
		n := strings.Count(prompt, "%s")
		args := make([]any, n)
		for i := range args {
			switch i {
			case 0:
				args[i] = topic
			default:
				args[i] = difficulty
			}
		}
		prompt = fmt.Sprintf(prompt, args...)
	} else {
		prompt = fmt.Sprintf("%s\n\nMavzu: %s\nQiyinlik darajasi: %s", prompt, topic, difficulty)
	}

	// Admin qo'lda shikoyat/javob kiritgan bo'lsa - AI ularni MAJBURIY asos qilib olishi
	// kerak (o'zidan o'ylab topmaydi, faqat shu ikkisi atrofida case quradi).
	if genReq.ChiefComplaint != "" {
		prompt = fmt.Sprintf("%s\n\nBemorning asosiy shikoyati (albatta shu asosda case tuz, chief_complaint maydonini shu ma'noda barcha tillarga tarjima qil): %s", prompt, genReq.ChiefComplaint)
	}
	if genReq.ExpectedAnswer != "" {
		prompt = fmt.Sprintf("%s\n\nKutilgan to'g'ri javob/tashxis (albatta shu bo'lishi kerak): %s", prompt, genReq.ExpectedAnswer)
	}

	// Admin shabloni boshqa JSON tuzilmasini so'rashi mumkin (masalan patient_info,
	// case_id) - backend esa aynan CaseCreateReq maydonlarini kutadi, shu sabab
	// chiqish formatini har doim oxirida majburan qayta belgilaymiz.
	prompt += caseGenFormatSuffix

	text, usage, err := g.generateContent(ctx, prompt, params, true)
	if err != nil {
		return nil, usage, err
	}

	// expected_answer ba'zan string o'rniga {"uz","ru","en"} obyekti bo'lib keladi
	// (admin prompti shuni so'rasa) - shuni string'ga keltiramiz.
	var req struct {
		models.CaseCreateReq
		ExpectedAnswer json.RawMessage `json:"expected_answer"`
	}
	if err := unmarshalJSONLoose(text, &req); err != nil {
		return nil, usage, fmt.Errorf("gemini case-generation response parse error: %w (raw: %s)", err, text)
	}
	// Ba'zan model natijani {"case": {...}} kabi o'rab qaytaradi - ochamiz.
	if len(req.Title) == 0 {
		var wrap map[string]json.RawMessage
		if unmarshalJSONLoose(text, &wrap) == nil && len(wrap) == 1 {
			for _, inner := range wrap {
				_ = unmarshalJSONLoose(string(inner), &req)
			}
		}
	}
	if len(req.Title) == 0 {
		return nil, usage, fmt.Errorf("gemini case-generation response has no title (raw: %s)", truncate(text, 500))
	}
	out := req.CaseCreateReq
	out.ExpectedAnswer = flattenExpectedAnswer(req.ExpectedAnswer)
	return finishCaseGen(&out, difficulty, genReq.ExpectedAnswer), usage, nil
}

// flattenExpectedAnswer string yoki {"uz","ru","en"} obyektdan bitta matn chiqaradi
// (uz ustun, bo'lmasa en/ru, bo'lmasa istalgan birinchi qiymat).
func flattenExpectedAnswer(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var m map[string]string
	if json.Unmarshal(raw, &m) == nil {
		for _, k := range []string{"uz", "en", "ru"} {
			if m[k] != "" {
				return m[k]
			}
		}
		for _, v := range m {
			if v != "" {
				return v
			}
		}
	}
	return ""
}

func finishCaseGen(req *models.CaseCreateReq, difficulty, adminExpected string) *models.CaseCreateReq {
	if req.Difficulty == "" {
		req.Difficulty = difficulty
	}
	if req.MaxAttempts <= 0 {
		req.MaxAttempts = models.DefaultCaseMaxAttempts
	}
	// ExpectedAnswer bitta til (string) - admin kiritgan bo'lsa AI natijasidan
	// qat'iy nazar aynan shuni saqlaymiz (aniqlik uchun, AI qayta yozib qo'ymasin).
	if adminExpected != "" {
		req.ExpectedAnswer = adminExpected
	}
	return req
}

const defaultActionEvalPrompt = `Sen shoshilinch tibbiy yordam bo'yicha ekspert-baholovchisan. Quyidagi klinik simulyatsiya case'i (bemor holati, kutilgan qadamlar) asosida, shifokor (foydalanuvchi) hozir bajargan BITTA harakatni AHA/ERC xalqaro qo'llanmalariga tayanib baho.

Case scenario (JSON):
%s

Kutilgan to'g'ri javob/davolash algoritmi (admin kiritgan, baholashda asosiy mezon):
%s

Shifokor bajargan harakat turi: %s
Harakat tafsilotlari (JSON): %s

Baholash mezoni:
- to'g'ri, o'z vaqtida va klinik jihatdan asosli harakat: health_delta ijobiy (+5 dan +20 gacha)
- ortiqcha zarar keltirmaydigan, lekin foydasiz/kerak bo'lmagan harakat: health_delta 0
- noto'g'ri, xavfli yoki qarshi ko'rsatilgan (contraindicated) harakat: health_delta manfiy (-10 dan -40 gacha)

Javobni FAQAT quyidagi JSON formatda qaytar, boshqa hech qanday matn yoki markdown qo'shma:
{"is_correct": true, "health_delta": 10, "feedback": "Qisqa (1-2 gap) klinik izoh - nima uchun to'g'ri/noto'g'ri"}`

// action_eval hozircha ai_prompts jadvalida yo'q (CHECK constraint faqat
// patient_persona/debrief/case_generation'ni ruxsat beradi), shu sabab
// hozircha faqat kod ichidagi default (defaultActionEvalPrompt) ishlatiladi.
func (g *geminiClient) Evaluate(ctx context.Context, caseScenario map[string]any, expectedAnswer, eventType string, payload map[string]any) (*ActionEvalRes, *Usage, error) {
	scenarioJSON, _ := json.MarshalIndent(caseScenario, "", "  ")
	payloadJSON, _ := json.MarshalIndent(payload, "", "  ")
	prompt := fmt.Sprintf(defaultActionEvalPrompt, string(scenarioJSON), expectedAnswer, eventType, string(payloadJSON))

	text, usage, err := g.generateContent(ctx, prompt, nil, true)
	if err != nil {
		return nil, usage, err
	}

	var raw struct {
		IsCorrect   bool   `json:"is_correct"`
		HealthDelta int    `json:"health_delta"`
		Feedback    string `json:"feedback"`
	}
	if err := unmarshalJSONLoose(text, &raw); err != nil {
		return nil, usage, fmt.Errorf("gemini action-eval response parse error: %w (raw: %s)", err, text)
	}
	return &ActionEvalRes{IsCorrect: raw.IsCorrect, HealthDelta: raw.HealthDelta, Feedback: raw.Feedback}, usage, nil
}

// --- Gemini REST chaqiruvi ---

type geminiGenerationConfig struct {
	Temperature      *float64 `json:"temperature,omitempty"`
	TopP             *float64 `json:"topP,omitempty"`
	TopK             *float64 `json:"topK,omitempty"`
	MaxOutputTokens  *int     `json:"maxOutputTokens,omitempty"`
	ResponseMimeType string   `json:"responseMimeType,omitempty"`
	// ResponseModalities/SpeechConfig - faqat *-tts modellar bilan (Synthesize)
	// ishlatiladi: ["AUDIO"] javobni matn o'rniga audio (inlineData) qilib qaytaradi.
	ResponseModalities []string            `json:"responseModalities,omitempty"`
	SpeechConfig       *geminiSpeechConfig `json:"speechConfig,omitempty"`
}

type geminiSpeechConfig struct {
	VoiceConfig *geminiVoiceConfig `json:"voiceConfig,omitempty"`
}

type geminiVoiceConfig struct {
	PrebuiltVoiceConfig *geminiPrebuiltVoiceConfig `json:"prebuiltVoiceConfig,omitempty"`
}

type geminiPrebuiltVoiceConfig struct {
	VoiceName string `json:"voiceName"`
}

type geminiRequest struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

// geminiPart - matn (Text) YOKI audio/rasm kabi ikkilik ma'lumot (InlineData)
// bo'lishi mumkin: kiritishda (Transcribe) audio yuboriladi, chiqishda
// (Synthesize) audio qaytadi - ikkalasida ham shu maydon ishlatiladi.
type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // baza64
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiPart `json:"parts"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		ThoughtsTokenCount   int `json:"thoughtsTokenCount"` // "fikrlash" (reasoning) tokenlari - narxda output sifatida hisoblanadi
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// generateContent - Gemini "generateContent" endpointiga bitta so'rov,
// javob matni + token/xarajat (Usage) bilan qaytadi.
func (g *geminiClient) generateContent(ctx context.Context, prompt string, modelParams map[string]any, jsonMode bool) (string, *Usage, error) {
	genCfg := &geminiGenerationConfig{}
	if jsonMode {
		genCfg.ResponseMimeType = "application/json"
	}
	if t, ok := modelParams["temperature"].(float64); ok {
		genCfg.Temperature = &t
	}
	if t, ok := modelParams["top_p"].(float64); ok {
		genCfg.TopP = &t
	}
	if t, ok := modelParams["max_output_tokens"].(float64); ok {
		v := int(t)
		genCfg.MaxOutputTokens = &v
	}

	reqBody := geminiRequest{
		Contents:         []geminiContent{{Role: "user", Parts: []geminiPart{{Text: prompt}}}},
		GenerationConfig: genCfg,
	}
	gr, usage, err := g.doRequest(ctx, g.model, reqBody)
	if err != nil {
		return "", usage, err
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return "", usage, fmt.Errorf("gemini returned no candidates")
	}

	var text strings.Builder
	for _, p := range gr.Candidates[0].Content.Parts {
		text.WriteString(p.Text)
	}
	return text.String(), usage, nil
}

// Transcribe - VoiceAI: audioB64 (masalan audio/webm, brauzer MediaRecorder'idan)
// ni Gemini'ning o'zi (bosh chat modeli - alohida STT modeli shart emas,
// ko'p tilni audio bo'yicha ham tushunadi) o'zbek tilidagi matnga aylantiradi.
func (g *geminiClient) Transcribe(ctx context.Context, audioB64, mimeType string) (string, *Usage, error) {
	if mimeType == "" {
		mimeType = "audio/webm"
	}
	reqBody := geminiRequest{
		Contents: []geminiContent{{
			Role: "user",
			Parts: []geminiPart{
				{Text: "Ushbu audioda shifokor tomonidan aytilgan gapni eshit va so'zma-so'z, hech narsa qo'shmasdan yoki izohlamasdan, o'zbek tilida yoz. Faqat transkripsiya matnini qaytar, boshqa hech qanday matn yoki izoh qo'shma."},
				{InlineData: &geminiInlineData{MimeType: mimeType, Data: audioB64}},
			},
		}},
	}
	gr, usage, err := g.doRequest(ctx, g.model, reqBody)
	if err != nil {
		return "", usage, err
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return "", usage, fmt.Errorf("gemini returned no candidates")
	}
	var text strings.Builder
	for _, p := range gr.Candidates[0].Content.Parts {
		text.WriteString(p.Text)
	}
	return strings.TrimSpace(text.String()), usage, nil
}

// synthesizeVoiceName - Gemini prebuilt ovozlaridan biri (Kore - neytral,
// tabiiy ohang). Til inputning o'zidan (text) avtomatik aniqlanadi - alohida
// "til" parametri kerak emas, matn o'zbekcha bo'lsa ovoz ham o'zbekcha chiqadi
// (bu haqiqiy chaqiruv bilan tekshirib tasdiqlangan).
const synthesizeVoiceName = "Kore"

// Synthesize - VoiceAI: matnni (bemor javobi/klinik feedback, o'zbekcha)
// gemini-*-tts model orqali tabiiy ovozli audioga (WAV, baza64) aylantiradi.
func (g *geminiClient) Synthesize(ctx context.Context, text string) (string, *Usage, error) {
	if strings.TrimSpace(text) == "" {
		return "", nil, fmt.Errorf("synthesize: bo'sh matn")
	}
	reqBody := geminiRequest{
		Contents: []geminiContent{{Role: "user", Parts: []geminiPart{{Text: text}}}},
		GenerationConfig: &geminiGenerationConfig{
			ResponseModalities: []string{"AUDIO"},
			SpeechConfig: &geminiSpeechConfig{
				VoiceConfig: &geminiVoiceConfig{
					PrebuiltVoiceConfig: &geminiPrebuiltVoiceConfig{VoiceName: synthesizeVoiceName},
				},
			},
		},
	}
	gr, usage, err := g.doRequest(ctx, g.ttsModel, reqBody)
	if err != nil {
		return "", usage, err
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return "", usage, fmt.Errorf("gemini tts: audio qaytmadi")
	}
	for _, p := range gr.Candidates[0].Content.Parts {
		if p.InlineData != nil && p.InlineData.Data != "" {
			return p.InlineData.Data, usage, nil
		}
	}
	return "", usage, fmt.Errorf("gemini tts: javobda audio topilmadi")
}

// doRequest - Gemini "generateContent" endpointiga xom so'rov (model tanlanadi -
// oddiy chat/transkripsiya uchun g.model, ovozli chiqish uchun g.ttsModel), xom
// javob + token/xarajat (Usage) bilan qaytadi. generateContent/Transcribe/Synthesize
// shu ustiga qurilgan.
func (g *geminiClient) doRequest(ctx context.Context, model string, reqBody geminiRequest) (*geminiResponse, *Usage, error) {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("gemini request marshal error: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, nil, fmt.Errorf("gemini request build error: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", g.apiKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("gemini response read error: %w", err)
	}

	var gr geminiResponse
	if err := json.Unmarshal(body, &gr); err != nil {
		return nil, nil, fmt.Errorf("gemini response decode error: %w (status=%d, body=%s)", err, resp.StatusCode, truncate(string(body), 500))
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("gemini API error (status %d)", resp.StatusCode)
		if gr.Error != nil && gr.Error.Message != "" {
			msg = fmt.Sprintf("gemini API error (status %d): %s", resp.StatusCode, gr.Error.Message)
		}
		return nil, nil, errors.New(msg)
	}

	// OutputTokens narx uchun candidates + thoughts (reasoning) tokenlarini
	// birga oladi - Google ikkalasini ham output narxida hisoblaydi, aks
	// holda "thinking" modellarda xarajat kamroq ko'rsatiladi.
	usage := &Usage{
		PromptTokens: gr.UsageMetadata.PromptTokenCount,
		OutputTokens: gr.UsageMetadata.CandidatesTokenCount + gr.UsageMetadata.ThoughtsTokenCount,
		TotalTokens:  gr.UsageMetadata.TotalTokenCount,
	}
	usage.CostUSD = calcCost(model, usage.PromptTokens, usage.OutputTokens)

	return &gr, usage, nil
}

// loadPrompt - admin panelda tahrirlangan template'ni (ai_prompts jadvali)
// o'qiydi, topilmasa yoki DB xato bo'lsa kodda qattiq yozilgan default'ga
// qaytadi (BACKEND_STRUCTURE.md 7-bo'lim: promptlarni kodsiz sozlash).
func (g *geminiClient) loadPrompt(key, fallback string) (string, map[string]any) {
	if g.promptRepo == nil {
		return fallback, nil
	}
	row, err := g.promptRepo.GetByKey(key)
	if err != nil || row == nil || strings.TrimSpace(row.Template) == "" {
		return fallback, nil
	}
	return row.Template, row.ModelParams
}

// unmarshalJSONLoose - Gemini ba'zan JSON'ni ```json ... ``` fensasi ichida
// qaytarishi mumkin (responseMimeType=application/json bo'lsa kamdan-kam,
// lekin ehtiyot uchun tozalab olamiz).
func unmarshalJSONLoose(text string, v any) error {
	t := strings.TrimSpace(text)
	t = strings.TrimPrefix(t, "```json")
	t = strings.TrimPrefix(t, "```")
	t = strings.TrimSuffix(t, "```")
	t = strings.TrimSpace(t)
	return json.Unmarshal([]byte(t), v)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
