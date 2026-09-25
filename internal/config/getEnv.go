package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cast"
)

type Config struct {
	AUTH_PORT string

	POSTGRES_HOST     string
	POSTGRES_PORT     int
	POSTGRES_USER     string
	POSTGRES_PASSWORD string
	POSTGRES_NAME     string

	REDIS_HOST string
	REDIS_PORT string

	NEW_BOT_TOKEN     string
	BOT_TOKEN         string
	GROUP_ID          string
	GROUP_FOR_INFO_ID string

	BASE_URL               string
	// WEBSITE_URL - Telegram bot menu tugmalari (Case yechish/Bo'limlar/Profil/Biz
	// haqimizda) bosilganda ochiladigan web ilova manzili (internal/bot).
	WEBSITE_URL            string
	CLICK_MERCHANT_ID      int
	CLICK_SERVICE_ID       int
	CLICK_SECRET_KEY       string
	PAYME_MERCHANT_ID      string
	PAYME_SECRET_KEY       string
	PAYMENT_PROVIDER_TOKEN string
	PAYMENT_CURRENCY       string
	PAYMENT_BOT_TOKEN      string

	FIREBASE_CREDENTIALS string

	MINIO_ENDPOINT    string
	MINIO_PORT        string
	MINIO_ACCESS_KEY  string
	MINIO_SECRET_KEY  string
	MINIO_BUCKET_NAME string
	MINIO_DOMAIN      string

	CONFIRM_PAGE_URL string

	EMAIL_NAME string
	EMAIL_LOGO string

	BREVO_SERVER string
	BREVO_PORT   string
	BREVO_LOGIN  string
	BREVO_KEY    string

	// AI (Gemini) - stub mode when GEMINI_API_KEY is empty
	GEMINI_API_KEY string
	GEMINI_MODEL   string

	// Google OAuth (mobile "1-bosqichli kirish")
	GOOGLE_CLIENT_ID string

	// Gamification / limits defaults (overridable from admin panel via app_settings table)
	DAILY_FREE_LIMIT int
}

func Load() Config {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found")
	}

	config := Config{}

	config.AUTH_PORT = cast.ToString(coalesce("AUTH_PORT", ":1001"))

	config.POSTGRES_HOST = cast.ToString(coalesce("POSTGRES_HOST", "postgres"))
	config.POSTGRES_PORT = cast.ToInt(coalesce("POSTGRES_PORT", 5432))
	config.POSTGRES_USER = cast.ToString(coalesce("POSTGRES_USER", "postgres"))
	config.POSTGRES_PASSWORD = cast.ToString(coalesce("POSTGRES_PASSWORD", "1111"))
	config.POSTGRES_NAME = cast.ToString(coalesce("POSTGRES_NAME", "test"))

	config.REDIS_HOST = cast.ToString(coalesce("REDIS_HOST", "redis"))
	config.REDIS_PORT = cast.ToString(coalesce("REDIS_PORT", ":1111"))

	config.NEW_BOT_TOKEN = cast.ToString(coalesce("NEW_BOT_TOKEN", "token"))
	config.BOT_TOKEN = cast.ToString(coalesce("BOT_TOKEN", "token"))
	config.GROUP_ID = cast.ToString(coalesce("GROUP_ID", "-1"))
	config.GROUP_FOR_INFO_ID = cast.ToString(coalesce("GROUP_FOR_INFO_ID", "-1"))

	config.BASE_URL = cast.ToString(coalesce("BASE_URL", "http://localhost:8004"))
	config.WEBSITE_URL = cast.ToString(coalesce("WEBSITE_URL", "https://tibcaseweb.vercel.app"))
	config.CLICK_MERCHANT_ID = cast.ToInt(coalesce("CLICK_MERCHANT_ID", 1))
	config.CLICK_SERVICE_ID = cast.ToInt(coalesce("CLICK_SERVICE_ID", 1))
	config.CLICK_SECRET_KEY = cast.ToString(coalesce("CLICK_SECRET_KEY", "test_click_secret_key"))
	config.PAYME_MERCHANT_ID = cast.ToString(coalesce("PAYME_MERCHANT_ID", ""))
	config.PAYME_SECRET_KEY = cast.ToString(coalesce("PAYME_SECRET_KEY", "test_payme_secret_key"))

	config.PAYMENT_PROVIDER_TOKEN = cast.ToString(coalesce("PAYMENT_PROVIDER_TOKEN", "TEST:631330408:TEST:631330408"))
	config.PAYMENT_CURRENCY = cast.ToString(coalesce("PAYMENT_CURRENCY", "UZS"))
	config.PAYMENT_BOT_TOKEN = cast.ToString(coalesce("PAYMENT_BOT_TOKEN", "token"))

	config.FIREBASE_CREDENTIALS = cast.ToString(coalesce("FIREBASE_CREDENTIALS", "firebase"))

	config.MINIO_ENDPOINT = cast.ToString(coalesce("MINIO_ENDPOINT", "minio"))
	config.MINIO_PORT = cast.ToString(coalesce("MINIO_PORT", ":1000"))
	config.MINIO_ACCESS_KEY = cast.ToString(coalesce("MINIO_ACCESS_KEY", "minio"))
	config.MINIO_SECRET_KEY = cast.ToString(coalesce("MINIO_SECRET_KEY", "minio"))
	config.MINIO_BUCKET_NAME = cast.ToString(coalesce("MINIO_BUCKET_NAME", "test-bucket"))
	config.MINIO_DOMAIN = cast.ToString(coalesce(("MINIO_DOMAIN"), "minio"))

	config.CONFIRM_PAGE_URL = cast.ToString(coalesce("CONFIRM_PAGE_URL", "test"))

	config.EMAIL_NAME = cast.ToString(coalesce("EMAIL_NAME", "test"))
	config.EMAIL_LOGO = cast.ToString(coalesce("EMAIL_LOGO", "test"))

	config.BREVO_SERVER = cast.ToString(coalesce("BREVO_SERVER", "smtp-relay.brevo.com"))
	config.BREVO_PORT = cast.ToString(coalesce("BREVO_PORT", "587"))
	config.BREVO_LOGIN = cast.ToString(coalesce("BREVO_LOGIN", "test"))
	config.BREVO_KEY = cast.ToString(coalesce("BREVO_KEY", "test"))

	config.GEMINI_API_KEY = cast.ToString(coalesce("GEMINI_API_KEY", ""))
	config.GEMINI_MODEL = cast.ToString(coalesce("GEMINI_MODEL", "gemini-2.0-flash"))

	config.GOOGLE_CLIENT_ID = cast.ToString(coalesce("GOOGLE_CLIENT_ID", ""))

	config.DAILY_FREE_LIMIT = cast.ToInt(coalesce("DAILY_FREE_LIMIT", 3))

	return config
}

func coalesce(key string, defaultValue interface{}) interface{} {
	val, exists := os.LookupEnv(key)

	if exists {
		return val
	}

	return defaultValue
}
