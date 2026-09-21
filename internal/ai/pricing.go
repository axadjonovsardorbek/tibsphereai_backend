package ai

import "strings"

// pricePerMTok - 1 million token narxi (USD), input/output alohida.
// Manba: ai.google.dev/pricing (taxminiy, narxlar o'zgarganda shu yerni
// yangilang - kod boshqa joyda narxga tegmaydi).
type pricePerMTok struct {
	Input  float64
	Output float64
}

var geminiPricing = map[string]pricePerMTok{
	"gemini-2.0-flash":      {Input: 0.10, Output: 0.40},
	"gemini-2.0-flash-lite": {Input: 0.075, Output: 0.30},
	"gemini-1.5-flash":      {Input: 0.075, Output: 0.30},
	"gemini-1.5-pro":        {Input: 1.25, Output: 5.00},
	"gemini-2.5-flash":      {Input: 0.15, Output: 0.60},
	"gemini-2.5-pro":        {Input: 1.25, Output: 5.00},
	// gemini-3.6-flash - TAXMINIY narx (ai.google.dev/pricing'da tasdiqlang,
	// bu model chiqarilish sanasi bilim chegarasidan keyingi bo'lishi mumkin).
	"gemini-3.6-flash": {Input: 0.10, Output: 0.40},
}

// defaultPricing - narxlar jadvalida topilmagan model uchun (gemini-2.0-flash
// bilan bir xil, eng ko'p ishlatiladigan arzon model).
var defaultPricing = pricePerMTok{Input: 0.10, Output: 0.40}

func pricingFor(model string) pricePerMTok {
	if p, ok := geminiPricing[model]; ok {
		return p
	}
	// versiyasiz nom bo'yicha moslashtirish (masalan "models/gemini-2.0-flash-001")
	for key, p := range geminiPricing {
		if strings.Contains(model, key) {
			return p
		}
	}
	return defaultPricing
}

func calcCost(model string, promptTokens, outputTokens int) float64 {
	p := pricingFor(model)
	return float64(promptTokens)/1_000_000*p.Input + float64(outputTokens)/1_000_000*p.Output
}
