package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"tibsphereai/internal/config"

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

// sendWebLink - berilgan matn bilan, websiteURL'ni ochadigan bitta inline
// URL tugmali xabar yuboradi.
func sendWebLink(bot *tgbotapi.BotAPI, chatID int64, websiteURL, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("🌐 Ochish", websiteURL),
		),
	)
	bot.Send(msg)
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

			Siz *MedicAI.uz* akkauntingizga kirish uchun so‘rov yubordingiz.
			
			🔐 *Tasdiqlash kodingiz:*
			
			`+"`%s`"+`
			
			⏳ Kod *5 daqiqa* davomida amal qiladi.
			
			⚠️ Hech qachon bu kodni boshqalar bilan ulashmang.
			Agar bu so‘rovni siz yubormagan bo‘lsangiz, xabarni e’tiborsiz qoldiring.

			📋 Kod ustiga bosib, uni nusxalashingiz mumkin.`,
				name, otp,
			)

			msg := tgbotapi.NewMessage(chatID, text)
			msg.ParseMode = "Markdown"
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
