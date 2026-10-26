# MedicAI Backend (tibsphereai)

Klinik simulyatsiya ilovasi backend'i. Arxitektura `/Users/sardorbek/Documents/imed_backend` asosida qurilgan
(Gin + layered: handler → service → storage/postgres, Casbin RBAC, Redis, MinIO, Firebase push), funksional
talablar `BACKEND_STRUCTURE.md` da.

## Stack

- Go 1.26, Gin
- PostgreSQL — `jackc/pgx/v5` + `pgxpool` (lib/pq emas)
- Redis — sessiya/OTP/kesh
- Casbin — RBAC (`api/casbin/model.conf`, `policy.csv`)
- MinIO — fayl/rasm yuklash
- Firebase Cloud Messaging — push bildirishnomalar
- Click va Payme — to'lov (obuna/tarif uchun, `orders` jadvali umumiy)
- Gemini AI — `internal/ai` orqali (GEMINI_API_KEY bo'sh bo'lsa mock rejim)
- gorilla/websocket — simulyatsiya real-vaqt monitori

## Ishga tushirish

`.env*.example` fayllari xavfsizlik uchun git'da saqlanmaydi. `.env` faylini quyidagi o'zgaruvchilar bilan o'zingiz yarating
(qiymatlarni jamoadan oling; hech qachon repoga qo'shmang):

```
AUTH_PORT POSTGRES_HOST POSTGRES_PORT POSTGRES_USER POSTGRES_PASSWORD POSTGRES_NAME REDIS_HOST REDIS_PORT
BOT_TOKEN NEW_BOT_TOKEN GROUP_ID GROUP_FOR_INFO_ID BASE_URL WEBSITE_URL
CLICK_MERCHANT_ID CLICK_SERVICE_ID CLICK_SECRET_KEY PAYME_MERCHANT_ID PAYME_SECRET_KEY
INPAY_BASE_URL INPAY_MERCHANT_ID INPAY_MERCHANT_TOKEN FIREBASE_CREDENTIALS
MINIO_ENDPOINT MINIO_PORT MINIO_ACCESS_KEY MINIO_SECRET_KEY MINIO_BUCKET_NAME MINIO_DOMAIN
CONFIRM_PAGE_URL EMAIL_NAME EMAIL_LOGO BREVO_SERVER BREVO_PORT BREVO_LOGIN BREVO_KEY
GEMINI_API_KEY GEMINI_MODEL GEMINI_TTS_MODEL GEMINI_EMBEDDING_MODEL GEMINI_IMAGE_MODEL GOOGLE_CLIENT_ID DAILY_FREE_LIMIT JWT_SECRET
TG_BOT_USERNAME CAPTCHA_PROVIDER CAPTCHA_SECRET RATE_LIMIT_DISABLED   # oxirgi to'rttasi ixtiyoriy (default bor)
```

```bash
task compose-up           # postgres + redis
task migrate-up           # migratsiyalar
task run                  # yoki: go run ./cmd/main.go
```

Swagger: `http://localhost:1001/api/swagger/index.html` (`task swag-init` orqali generatsiya qilinadi).

## Papka strukturasi

- `api/` — router (`api.go`), handlerlar, middleware, casbin, JWT token
- `service/` — biznes-logika
- `storage/` — `storage.go` interfeyslari, `storage/postgres/` implementatsiyasi
- `pkg/models/` — DTO'lar
- `pkg/clients/` — barcha service'larni yig'uvchi `Clients`
- `internal/ai/` — Gemini AI qatlami (mock + gemini stub)
- `internal/ws/` — simulyatsiya WebSocket hub
- `internal/push`, `internal/bot`, `internal/config`, `internal/helper`
- `migrations/` — goose SQL migratsiyalar

## Hozircha TODO (keyingi bosqichlar)

- `internal/ai/gemini.go` — haqiqiy Gemini API chaqiruvi (hozircha `GEMINI_API_KEY` bo'sh bo'lsa mock javob)
- `internal/ws/hub.go` — vitallarni tibbiy jihatdan to'g'ri hisoblash (hozircha health_percent'ni qaytaradi)
- Simulyatsiya scoring algoritmi (`service/simulation.go` — hozircha soddalashtirilgan)
- Push-reminder scheduler (study_plans asosida cron — hozircha joy tayyor, ishga tushmaydi)
- `api/docs` — `task swag-init` orqali to'liq swagger generatsiyasi

<!-- ci test: 2026-09-21T18:58:02Z -->
