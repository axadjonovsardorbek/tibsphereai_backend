package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"tibsphereai/internal/config"
	"tibsphereai/internal/tglink"

	"github.com/go-redis/redis"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// Asosiy menu tugmalari - bosilganda web ilovaga (config.WEBSITE_URL) o'tkazib
// yuboradi (inline URL tugma orqali - ReplyKeyboardMarkup'da URL bo'lmaydi).
const (
	menuBtnCases      = "🩺 Case yechish"
	menuBtnCategories = "📚 Bo'limlar"
	menuBtnProfile    = "👤 Profil"
	menuBtnAbout      = "ℹ️ Biz haqimizda"
)

func mainMenuKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.ReplyKeyboardMarkup{
		Keyboard: [][]tgbotapi.KeyboardButton{
			{{Text: menuBtnCases}, {Text: menuBtnCategories}},
			{{Text: menuBtnProfile}, {Text: menuBtnAbout}},
		},
		ResizeKeyboard: true,
	}
}

// --- Telegram Mini App (web_app) tugmasi ---
//
// go-telegram-bot-api (v4.6.4, shu jumladan eng so'nggi v5.5.1 ham) InlineKeyboardButton'da
// "web_app" maydonini umuman qo'llab-quvvatlamaydi - shu sabab oddiy NewInlineKeyboardButtonURL
// ishlatilsa, Telegram tugmani tashqi brauzerda ochadi (ilova ichida emas, "orqaga"
// bosilganda ilovadan butunlay chiqib ketadi). tgbotapi.BaseChat.ReplyMarkup interface{}
// bo'lgani va json.Marshal generik ishlagani uchun, kutubxonani yangilamasdan
// o'zimizning JSON strukturamiz bilan Telegram Bot API'ning web_app formatini
// to'g'ridan-to'g'ri yuboramiz (bot Telegram ichida - Mini App sifatida ochiladi,
// pastki panelga minimizatsiya bo'ladi).
type webAppInfo struct {
	URL string `json:"url"`
}

type webAppInlineButton struct {
	Text   string     `json:"text"`
	WebApp webAppInfo `json:"web_app"`
}

type webAppInlineKeyboard struct {
	InlineKeyboard [][]webAppInlineButton `json:"inline_keyboard"`
}

// sendWebLink - berilgan matn bilan, websiteURL'ni Telegram Mini App sifatida
// (bot ichida) ochadigan bitta web_app tugmali xabar yuboradi.
func sendWebLink(bot *tgbotapi.BotAPI, chatID int64, websiteURL, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = webAppInlineKeyboard{
		InlineKeyboard: [][]webAppInlineButton{
			{{Text: "📱 Ilovani ochish", WebApp: webAppInfo{URL: websiteURL}}},
		},
	}
	bot.Send(msg)
}

// setMenuButton - chat oynasidagi doimiy "Menu" tugmasini (matn kiritish maydoni
// yonida) to'g'ridan-to'g'ri web_app'ga sozlaydi (BotFather -> Menu Button orqali
// qo'lda ham qilsa bo'ladi, lekin shu funksiya kodda, deploy'da avtomatik bajaradi).
// tgbotapi kutubxonasida setChatMenuButton yo'q, shu sabab xom HTTP so'rov.
func setMenuButton(token, websiteURL string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/setChatMenuButton", token)
	body, _ := json.Marshal(map[string]any{
		"menu_button": map[string]any{
			"type": "web_app",
			"text": "📱 Ilovani ochish",
			"web_app": map[string]any{
				"url": websiteURL,
			},
		},
	})
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("setChatMenuButton failed: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("setChatMenuButton unexpected status: %d", resp.StatusCode)
	}
}

func BotAuth(config *config.Config) {

	log.Println("Bot started")

	rAddr := fmt.Sprintf("%s%s", config.REDIS_HOST, config.REDIS_PORT)

	rdb := redis.NewClient(&redis.Options{
		Addr: rAddr,
	})

	bot, err := tgbotapi.NewBotAPI(config.NEW_BOT_TOKEN)
	if err != nil {
		fmt.Println(err)
		log.Panic(err)
	}

	setMenuButton(config.NEW_BOT_TOKEN, config.WEBSITE_URL)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID

		// Bosqich 1: /start
		if update.Message.Text != "" {
			text := strings.TrimSpace(update.Message.Text)
			log.Printf("incoming text: %q", text)

			if strings.HasPrefix(text, "/start") {
				fields := strings.Fields(text)

				payload := ""
				if len(fields) > 1 {
					payload = fields[1]
				}

				// OTP'siz kirish / telefon ulash: /start login_<token> yoki phone_<token>
				if purpose, token, ok := tglink.ParseStartPayload(payload); ok {
					handleLinkStart(bot, rdb, config, update.Message, purpose, token)
					continue
				}

				if payload == "login" {
					msg := tgbotapi.NewMessage(chatID, "📱 Iltimos, raqamingizni yuboring:")
					msg.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
						Keyboard: [][]tgbotapi.KeyboardButton{
							{
								{
									Text:           "📲 Raqamni yuborish",
									RequestContact: true,
								},
							},
						},
						ResizeKeyboard:  true,
						OneTimeKeyboard: true,
					}
					bot.Send(msg)
					continue
				}

				// Oddiy /start (login payload'siz) - asosiy menu ko'rsatiladi
				// (Case yechish/Bo'limlar/Profil/Biz haqimizda - har biri web
				// ilovaga o'tkazib yuboradi).
				msg := tgbotapi.NewMessage(chatID, "👋 Xush kelibsiz! Kerakli bo'limni tanlang:")
				msg.ReplyMarkup = mainMenuKeyboard()
				bot.Send(msg)
				continue
			}

			if text == "/menu" {
				msg := tgbotapi.NewMessage(chatID, "Kerakli bo'limni tanlang:")
				msg.ReplyMarkup = mainMenuKeyboard()
				bot.Send(msg)
				continue
			}

			switch text {
			case menuBtnCases:
				sendWebLink(bot, chatID, config.WEBSITE_URL, "🩺 Case yechish uchun web ilovani oching:")
				continue
			case menuBtnCategories:
				sendWebLink(bot, chatID, config.WEBSITE_URL, "📚 Bo'limlarni ko'rish uchun web ilovani oching:")
				continue
			case menuBtnProfile:
				sendWebLink(bot, chatID, config.WEBSITE_URL, "👤 Profilingizni ko'rish uchun web ilovani oching:")
				continue
			case menuBtnAbout:
				sendWebLink(bot, chatID, config.WEBSITE_URL, "ℹ️ Biz haqimizda ma'lumot uchun web ilovani oching:")
				continue
			}
		}

		// Bosqich 2: Raqam yuborildi
		if update.Message.Contact != nil {
			phone := update.Message.Contact.PhoneNumber
			userID := update.Message.Contact.UserID
			name := update.Message.Contact.FirstName + update.Message.Contact.LastName

			// Xavfsizlik: faqat o'z kontaktini yuborgan foydalanuvchi qabul qilinadi (boshqa odamning
			// raqamini "forward" qilib, uning nomidan kirib bo'lmasin).
			if update.Message.From == nil || userID != update.Message.From.ID {
				bot.Send(tgbotapi.NewMessage(chatID, "⚠️ Iltimos, «📲 Raqamni yuborish» tugmasi orqali o'zingizning raqamingizni yuboring."))
				continue
			}

			// OTP'siz oqim: shu chat uchun kutilayotgan token bo'lsa, kontakt egasi tasdiqlanadi.
			if purpose, token, ok := tglink.PopWaiting(rdb, chatID); ok {
				tglink.RememberUser(rdb, int64(userID), tglink.Known{Phone: NormalizePhone(phone), Name: name})
				confirmLink(bot, rdb, config, chatID, purpose, token, tglink.Info{Phone: NormalizePhone(phone), Name: name, TgID: int64(userID)})
				continue
			}

			log.Printf("✅ Raqam qabul qilindi: %s (Telegram ID: %d)", phone, userID)

			phone_number := NormalizePhone(phone)
			// Bosqich 3: Tasdiqlash kodi yaratish
			otp, err := rdb.Get(phone_number).Result()

			if err == redis.Nil {
				msg := tgbotapi.NewMessage(chatID,
					"❌ Bu telefon raqam uchun login so‘rovi topilmadi.\n\nAvval sayt yoki ilova orqali login qilishni boshlang.")
				bot.Send(msg)
				continue
			} else if err != nil {
				log.Printf("redis get error: %v", err)
				msg := tgbotapi.NewMessage(chatID,
					"⚠️ Xatolik yuz berdi. Iltimos, keyinroq urinib ko‘ring.")
				bot.Send(msg)
				continue
			}
			fmt.Println("-------------------------------")
			fmt.Println("Generated OTP:", otp)
			fmt.Println("-------------------------------")

			// Bosqich 4: Kodingni bot orqali foydalanuvchiga yuborish
			text := fmt.Sprintf(`👋 Salom %s

			Siz *TibStation* akkauntingizga kirish uchun so‘rov yubordingiz.
			
			🔐 *Tasdiqlash kodingiz:*
			
			`+"`%s`"+`
			
			⏳ Kod *5 daqiqa* davomida amal qiladi.
			
			⚠️ Hech qachon bu kodni boshqalar bilan ulashmang.
			Agar bu so‘rovni siz yubormagan bo‘lsangiz, xabarni e’tiborsiz qoldiring.

			📋 Kod ustiga bosib, uni nusxalang va pastdagi *«Ilovaga qaytish»* tugmasini bosib, kodni kiriting.`,
				name, otp,
			)

			msg := tgbotapi.NewMessage(chatID, text)
			msg.ParseMode = "Markdown"
			// Mini App botga o'tganda pastga minimizatsiya bo'ladi va uni dasturiy
			// ravishda qaytarib bo'lmaydi - shu sabab xabar tagiga ilovani qayta ochadigan
			// tugma qo'yamiz (foydalanuvchi kodni nusxalab, shu tugma orqali qaytadi).
			msg.ReplyMarkup = webAppInlineKeyboard{
				InlineKeyboard: [][]webAppInlineButton{
					{{Text: "↩️ Ilovaga qaytish", WebApp: webAppInfo{URL: config.WEBSITE_URL}}},
				},
			}
			bot.Send(msg)

			rdb.Set(fmt.Sprintf("tg:%s", phone_number), name, time.Minute*5)
		}
	}
}

func NormalizePhone(phone string) string {
	if !strings.HasPrefix(phone, "+") {
		return "+" + phone
	}
	return phone
}

// handleLinkStart - /start <purpose>_<token>: Telegram ID oldin tasdiqlangan bo'lsa darhol tasdiqlaydi,
// aks holda foydalanuvchidan o'z raqamini (bir marta) so'raydi.
func handleLinkStart(bot *tgbotapi.BotAPI, rdb *redis.Client, cfg *config.Config, msg *tgbotapi.Message, purpose, token string) {
	chatID := msg.Chat.ID
	if _, err := tglink.Get(rdb, purpose, token); err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "⌛ Havola eskirgan yoki noto'g'ri. Ilovaga qaytib, qaytadan urinib ko'ring."))
		return
	}
	if msg.From != nil {
		if k, ok := tglink.KnownUser(rdb, int64(msg.From.ID)); ok {
			confirmLink(bot, rdb, cfg, chatID, purpose, token, tglink.Info{Phone: k.Phone, Name: k.Name, TgID: int64(msg.From.ID)})
			return
		}
	}
	tglink.SetWaiting(rdb, chatID, purpose, token)
	m := tgbotapi.NewMessage(chatID, "📱 Tasdiqlash uchun raqamingizni yuboring (bir marta):")
	m.ReplyMarkup = tgbotapi.ReplyKeyboardMarkup{
		Keyboard:        [][]tgbotapi.KeyboardButton{{{Text: "📲 Raqamni yuborish", RequestContact: true}}},
		ResizeKeyboard:  true,
		OneTimeKeyboard: true,
	}
	bot.Send(m)
}

// confirmLink - tokenni tasdiqlaydi va foydalanuvchini ilovaga qaytaradi.
func confirmLink(bot *tgbotapi.BotAPI, rdb *redis.Client, cfg *config.Config, chatID int64, purpose, token string, who tglink.Info) {
	if err := tglink.Confirm(rdb, purpose, token, who); err != nil {
		log.Printf("link confirm failed (%s): %v", purpose, err)
		bot.Send(tgbotapi.NewMessage(chatID, "⌛ Havola eskirgan. Ilovaga qaytib, qaytadan urinib ko'ring."))
		return
	}
	text := "✅ Tasdiqlandi! Ilovaga qayting - kirish avtomatik davom etadi."
	if purpose == tglink.PurposePhone {
		text = "✅ Raqamingiz tasdiqlandi! Ilovaga qayting - u hisobingizga ulanadi."
	}
	m := tgbotapi.NewMessage(chatID, text)
	m.ReplyMarkup = webAppInlineKeyboard{
		InlineKeyboard: [][]webAppInlineButton{{{Text: "↩️ Ilovaga qaytish", WebApp: webAppInfo{URL: cfg.WEBSITE_URL}}}},
	}
	bot.Send(m)
}
