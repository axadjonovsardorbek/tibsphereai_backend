package bot

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWebAppInlineKeyboard_JSONShape - Telegram Bot API "web_app" tugmasi aynan
// {"text":"...","web_app":{"url":"..."}} shaklida bo'lishi kerak (oddiy "url"
// maydoni emas - aks holda Telegram tugmani tashqi brauzerda ochadi, Mini App
// sifatida emas). go-telegram-bot-api kutubxonasi bu maydonni qo'llamagani
// uchun qo'lda yozilgan struct ishlatiladi - shu formatni saqlab qolish muhim.
func TestWebAppInlineKeyboard_JSONShape(t *testing.T) {
	kb := webAppInlineKeyboard{
		InlineKeyboard: [][]webAppInlineButton{
			{{Text: "📱 Ilovani ochish", WebApp: webAppInfo{URL: "https://tibcaseweb.vercel.app"}}},
		},
	}

	data, err := json.Marshal(kb)
	assert.NoError(t, err)
	assert.JSONEq(t, `{
		"inline_keyboard": [[
			{"text": "📱 Ilovani ochish", "web_app": {"url": "https://tibcaseweb.vercel.app"}}
		]]
	}`, string(data))
}
