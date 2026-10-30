// Package tglink - Telegram bot orqali OTP'siz tasdiqlash (deep-link): ilova bir martalik token yaratadi,
// foydalanuvchi bot havolasini ochadi (t.me/<bot>?start=<purpose>_<token>), bot Telegram o'zi tasdiqlagan
// shaxsni (Telegram ID + o'z kontakti) Redis'ga yozadi, ilova token bo'yicha natijani oladi.
// Bot va API ikkalasi ham shu paketdan foydalanadi (umumiy Redis).
package tglink

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis"
)

const (
	PurposeLogin = "login" // kirish (yoki ro'yxatdan o'tish)
	PurposePhone = "phone" // joriy hisobga telefon raqamni ulash/almashtirish

	StatusPending   = "pending"
	StatusConfirmed = "confirmed"

	// TTL - token amal qilish muddati.
	TTL = 5 * time.Minute
)

var (
	ErrNotFound   = errors.New("link token not found or expired")
	ErrNotPending = errors.New("link token is not pending")
	ErrOwner      = errors.New("link token belongs to another user")
)

// Info - token holati va (tasdiqlangach) Telegram tasdiqlagan ma'lumotlar.
type Info struct {
	Status string `json:"status"`
	Owner  string `json:"owner,omitempty"` // phone maqsadida: tokenni yaratgan foydalanuvchi ID'si
	Phone  string `json:"phone,omitempty"`
	Name   string `json:"name,omitempty"`
	TgID   int64  `json:"tg_id,omitempty"`
	Web    bool   `json:"web,omitempty"` // token oddiy brauzerdan boshlangan (Mini App emas)
}

func key(purpose, token string) string { return "tgl:" + purpose + ":" + token }

// NewToken - 24 baytli kriptografik tasodifiy token (hex).
func NewToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand ishlamasa xavfsiz davom etib bo'lmaydi
	}
	return hex.EncodeToString(b)
}

// Create - kutilayotgan (pending) token yaratadi.
func Create(rdb *redis.Client, purpose, token, owner string, web bool) error {
	b, _ := json.Marshal(Info{Status: StatusPending, Owner: owner, Web: web})
	return rdb.Set(key(purpose, token), b, TTL).Err()
}

// Get - token holatini o'qiydi (ErrNotFound - yo'q yoki muddati o'tgan).
func Get(rdb *redis.Client, purpose, token string) (Info, error) {
	raw, err := rdb.Get(key(purpose, token)).Result()
	if err == redis.Nil {
		return Info{}, ErrNotFound
	}
	if err != nil {
		return Info{}, err
	}
	var in Info
	if err := json.Unmarshal([]byte(raw), &in); err != nil {
		return Info{}, err
	}
	return in, nil
}

// Confirm - bot tomonidan chaqiriladi: faqat mavjud pending token tasdiqlanadi (o'zboshimcha token
// yaratib bo'lmaydi). phone maqsadida egasi saqlanadi.
func Confirm(rdb *redis.Client, purpose, token string, who Info) error {
	cur, err := Get(rdb, purpose, token)
	if err != nil {
		return err
	}
	if cur.Status != StatusPending {
		return ErrNotPending
	}
	who.Status, who.Owner, who.Web = StatusConfirmed, cur.Owner, cur.Web
	b, _ := json.Marshal(who)
	return rdb.Set(key(purpose, token), b, TTL).Err()
}

// Consume - tasdiqlangan tokenni bir marta o'qib o'chiradi (qayta ishlatib bo'lmaydi).
// Pending bo'lsa o'chirmaydi va ok=false, status pending qaytadi.
func Consume(rdb *redis.Client, purpose, token string) (info Info, ok bool, err error) {
	info, err = Get(rdb, purpose, token)
	if err != nil {
		return info, false, err
	}
	if info.Status != StatusConfirmed {
		return info, false, nil
	}
	n, err := rdb.Del(key(purpose, token)).Result()
	if err != nil {
		return info, false, err
	}
	return info, n == 1, nil // n == 0: parallel so'rov allaqachon olgan
}

// ParseStartPayload - "/start <purpose>_<token>" payload'idan maqsad va tokenni ajratadi.
func ParseStartPayload(payload string) (purpose, token string, ok bool) {
	for _, p := range []string{PurposeLogin, PurposePhone} {
		if strings.HasPrefix(payload, p+"_") {
			t := strings.TrimPrefix(payload, p+"_")
			if len(t) >= 16 && len(t) <= 64 && isHex(t) {
				return p, t, true
			}
		}
	}
	return "", "", false
}

func isHex(s string) bool {
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

// --- Telegram ID -> telefon xotirasi (kontaktni har safar so'ramaslik uchun) ---

type Known struct {
	Phone string `json:"phone"`
	Name  string `json:"name"`
}

func knownKey(tgID int64) string { return fmt.Sprintf("tgid:%d", tgID) }

// RememberUser - foydalanuvchi o'z kontaktini bir marta yuborgach, Telegram ID bo'yicha eslab qolinadi.
func RememberUser(rdb *redis.Client, tgID int64, k Known) {
	b, _ := json.Marshal(k)
	rdb.Set(knownKey(tgID), b, 0)
}

func KnownUser(rdb *redis.Client, tgID int64) (Known, bool) {
	raw, err := rdb.Get(knownKey(tgID)).Result()
	if err != nil {
		return Known{}, false
	}
	var k Known
	if json.Unmarshal([]byte(raw), &k) != nil || k.Phone == "" {
		return Known{}, false
	}
	return k, true
}

// --- Kontakt kutish holati: bot "raqamni yuboring" so'ragan chat uchun qaysi token kutilayotgani ---

func waitKey(chatID int64) string { return fmt.Sprintf("tgwait:%d", chatID) }

func SetWaiting(rdb *redis.Client, chatID int64, purpose, token string) {
	rdb.Set(waitKey(chatID), purpose+"_"+token, TTL)
}

func PopWaiting(rdb *redis.Client, chatID int64) (purpose, token string, ok bool) {
	raw, err := rdb.Get(waitKey(chatID)).Result()
	if err != nil {
		return "", "", false
	}
	rdb.Del(waitKey(chatID))
	return ParseStartPayload(raw)
}
