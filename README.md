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

```bash
cp .env.example .env      # qiymatlarni to'ldiring
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
