package ai

import (
	"context"
	"fmt"
	"tibsphereai/pkg/models"
)

// mock* - GEMINI_API_KEY berilmagunicha ishlatiladigan deterministik stub'lar.
// Skelet bosqichida router/service/DB oqimini sinash uchun yetarli.

type mockClient struct{}

func (mockClient) Reply(ctx context.Context, caseScenario map[string]any, question string) (string, *Usage, error) {
	return fmt.Sprintf("(AI stub) Savolingizni tushundim: %q. Gemini ulanganda bemor shu yerda haqiqiy javob beradi.", question), &Usage{}, nil
}

func (mockClient) Generate(ctx context.Context, caseScenario map[string]any, events []models.SimulationEventReq) (*models.DebriefReportCreateReq, *Usage, error) {
	correct := make([]string, 0, len(events))
	for i := range events {
		correct = append(correct, fmt.Sprintf("Qadam %d bajarildi", i+1))
	}
	return &models.DebriefReportCreateReq{
		CorrectSteps:   correct,
		IncorrectSteps: []string{},
		GuidelineNotes: "(AI stub) Gemini ulanganda AHA/ERC qo'llanmalariga asoslangan tahlil shu yerda bo'ladi.",
		WeakTopics:     []string{},
	}, &Usage{}, nil
}

func (mockClient) Evaluate(ctx context.Context, caseScenario map[string]any, eventType string, payload map[string]any) (*ActionEvalRes, *Usage, error) {
	return &ActionEvalRes{
		IsCorrect:   true,
		HealthDelta: 0,
		Feedback:    fmt.Sprintf("(AI stub) %q harakati qabul qilindi. Gemini ulanganda bu yerda haqiqiy klinik baholash bo'ladi.", eventType),
	}, &Usage{}, nil
}

// mockCaseGen - CaseGeneratorAI uchun alohida tip, chunki mockClient'da
// allaqachon boshqa signature bilan Generate metodi bor (DebriefAI).
type mockCaseGen struct{}

func (mockCaseGen) Generate(ctx context.Context, topic, difficulty string) (*models.CaseCreateReq, *Usage, error) {
	return nil, nil, fmt.Errorf("AI case generation is not configured: set GEMINI_API_KEY")
}
