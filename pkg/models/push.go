package models

import (
	"encoding/json"
	"fmt"
	"strings"
)

// pickLang - jsonStr ({"uz":"...","ru":"...","en":"..."}) ichidan berilgan tilni oladi,
// topilmasa "uz"ga tushadi. internal/helper.PickFromJSONStr bilan bir xil mantiq - shu yerda
// qaytarilgan (internal/helper pkg/models'ni import qiladi, aylanma bog'liqlik bo'lmasligi uchun).
func pickLang(jsonStr, lang string) (string, error) {
	jsonStr = strings.TrimSpace(jsonStr)
	if jsonStr == "" {
		return "", nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return "", fmt.Errorf("invalid json string: %w", err)
	}
	if v := strings.TrimSpace(m[lang]); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(m["uz"]); v != "" {
		return v, nil
	}
	return "", nil
}

type PushTargetType string

const (
	// TargetAllUsers - barcha foydalanuvchilarga FCM Topic orqali yuboriladi (uz/ru/en
	// alohida topic: all_uz/all_ru/all_en), har device o'z tiliga mos topicga obuna.
	TargetAllUsers PushTargetType = "all_users"
	// TargetUser - bitta foydalanuvchiga uning tokenlariga to'g'ridan-to'g'ri (topicsiz).
	TargetUser PushTargetType = "user"
)

// BroadcastTopic - "all" push uchun til bo'yicha FCM topic nomi.
func BroadcastTopic(lang string) string {
	switch lang {
	case "ru", "en":
		return "all_" + lang
	default:
		return "all_uz"
	}
}

type PushTarget struct {
	Type   PushTargetType
	UserID string // TargetUser uchun
}

type PushEventType string

const (
	EventStudyReminder     PushEventType = "study_reminder"  // TZ 2-bo'lim: kunlik eslatma
	EventNewCase           PushEventType = "new_case"        // admin yangi keys publish qilganda
	EventPaymentSuccess    PushEventType = "payment_success" // TZ 5-bo'lim: obuna to'lovi
	EventPaymentFailed     PushEventType = "payment_failed"
	EventAdminNotification PushEventType = "admin_notification"
)

type PushEvent struct {
	Type PushEventType

	CaseID   string
	CaseName string // uz nom (title->>'uz'), fallback sifatida ishlatiladi

	OrderID    string
	Amount     string
	FailReason string

	// AdminTitleJSON/AdminBodyJSON - master_notifications.title/message kabi
	// {"uz":"...","ru":"...","en":"..."} JSON string (helper.PickFromJSONStr bilan o'qiladi).
	AdminTitleJSON string
	AdminBodyJSON  string
}

type PushMessage struct {
	Title string
	Body  string
	Data  map[string]string
}

// BuildLocalizedPushMessage - berilgan til (uz/ru/en) uchun push matnini tayyorlaydi.
// TargetAllUsers uchun har topic (all_uz/all_ru/all_en) o'z tilida chaqiradi,
// TargetUser uchun o'sha foydalanuvchining users.language'i bo'yicha chaqiriladi.
func BuildLocalizedPushMessage(e PushEvent, lang string) PushMessage {
	switch lang {
	case "ru":
		return buildRuPushMessage(e)
	case "en":
		return buildEnPushMessage(e)
	default:
		return buildUzPushMessage(e)
	}
}

func buildUzPushMessage(e PushEvent) PushMessage {
	switch e.Type {
	case EventStudyReminder:
		return PushMessage{
			Title: "Vaqt keldi! ⏰",
			Body:  "Bugun ham bir keys yechib, bilimingizni sinab ko'ring!",
			Data:  map[string]string{"type": "study_reminder"},
		}
	case EventNewCase:
		return PushMessage{
			Title: "Yangi klinik keys! 🩺",
			Body:  fmt.Sprintf("Yangi keys qo'shildi: %s", e.CaseName),
			Data:  map[string]string{"type": "new_case", "case_id": e.CaseID},
		}
	case EventPaymentSuccess:
		return PushMessage{
			Title: "To'lov qabul qilindi ✅",
			Body:  "Premium obunangiz faollashtirildi.",
			Data:  map[string]string{"type": "payment_success", "order_id": e.OrderID, "amount": e.Amount},
		}
	case EventPaymentFailed:
		body := "To'lov qabul qilinmadi."
		if e.FailReason != "" {
			body = body + " Sabab: " + e.FailReason
		}
		return PushMessage{
			Title: "To'lov bekor qilindi ❌",
			Body:  body,
			Data:  map[string]string{"type": "payment_failed", "order_id": e.OrderID, "amount": e.Amount},
		}
	case EventAdminNotification:
		title, _ := pickLang(e.AdminTitleJSON, "uz")
		body, _ := pickLang(e.AdminBodyJSON, "uz")
		return PushMessage{Title: title, Body: body, Data: map[string]string{"type": "admin_notification"}}
	default:
		return PushMessage{Title: "Bildirishnoma", Body: "Yangi xabar bor", Data: map[string]string{"type": "unknown"}}
	}
}

func buildRuPushMessage(e PushEvent) PushMessage {
	switch e.Type {
	case EventStudyReminder:
		return PushMessage{
			Title: "Время пришло! ⏰",
			Body:  "Решите ещё один кейс сегодня и проверьте свои знания!",
			Data:  map[string]string{"type": "study_reminder"},
		}
	case EventNewCase:
		return PushMessage{
			Title: "Новый клинический кейс! 🩺",
			Body:  fmt.Sprintf("Добавлен новый кейс: %s", e.CaseName),
			Data:  map[string]string{"type": "new_case", "case_id": e.CaseID},
		}
	case EventPaymentSuccess:
		return PushMessage{
			Title: "Оплата принята ✅",
			Body:  "Ваша Premium подписка активирована.",
			Data:  map[string]string{"type": "payment_success", "order_id": e.OrderID, "amount": e.Amount},
		}
	case EventPaymentFailed:
		body := "Оплата не прошла."
		if e.FailReason != "" {
			body = body + " Причина: " + e.FailReason
		}
		return PushMessage{
			Title: "Платёж отменён ❌",
			Body:  body,
			Data:  map[string]string{"type": "payment_failed", "order_id": e.OrderID, "amount": e.Amount},
		}
	case EventAdminNotification:
		title, _ := pickLang(e.AdminTitleJSON, "ru")
		body, _ := pickLang(e.AdminBodyJSON, "ru")
		return PushMessage{Title: title, Body: body, Data: map[string]string{"type": "admin_notification"}}
	default:
		return PushMessage{Title: "Уведомление", Body: "Новое сообщение", Data: map[string]string{"type": "unknown"}}
	}
}

func buildEnPushMessage(e PushEvent) PushMessage {
	switch e.Type {
	case EventStudyReminder:
		return PushMessage{
			Title: "Time to practice! ⏰",
			Body:  "Solve a case today and test your knowledge!",
			Data:  map[string]string{"type": "study_reminder"},
		}
	case EventNewCase:
		return PushMessage{
			Title: "New clinical case! 🩺",
			Body:  fmt.Sprintf("A new case was added: %s", e.CaseName),
			Data:  map[string]string{"type": "new_case", "case_id": e.CaseID},
		}
	case EventPaymentSuccess:
		return PushMessage{
			Title: "Payment received ✅",
			Body:  "Your Premium subscription is now active.",
			Data:  map[string]string{"type": "payment_success", "order_id": e.OrderID, "amount": e.Amount},
		}
	case EventPaymentFailed:
		body := "Payment was not completed."
		if e.FailReason != "" {
			body = body + " Reason: " + e.FailReason
		}
		return PushMessage{
			Title: "Payment cancelled ❌",
			Body:  body,
			Data:  map[string]string{"type": "payment_failed", "order_id": e.OrderID, "amount": e.Amount},
		}
	case EventAdminNotification:
		title, _ := pickLang(e.AdminTitleJSON, "en")
		body, _ := pickLang(e.AdminBodyJSON, "en")
		return PushMessage{Title: title, Body: body, Data: map[string]string{"type": "admin_notification"}}
	default:
		return PushMessage{Title: "Notification", Body: "You have a new message", Data: map[string]string{"type": "unknown"}}
	}
}
