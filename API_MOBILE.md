# 📱 Mobil ilova API — Frontend uchun qo'llanma

> **Diqqat:** bu fayl `api/docs/swagger.json` asosida avtomatik generatsiya qilingan
> (`scripts/gen_api_doc.py` skripti orqali — qo'lda tahrirlamang, backend o'zgarsa qayta
> generatsiya qilinadi: `swag init -g api/api.go -o api/docs && python3 scripts/gen_api_doc.py`).
>
> - **Base URL:** deploy qilingan domen (masalan `https://api.tibsphereai.uz`), path'lar shu yerda ko'rsatilganidek qo'shiladi (masalan `/mobile/category`).
> - **Auth:** login/OTP oqimi orqali (`POST /mobile/auth/google` yoki `/mobile/auth/user/otp/*`) olingan `access_token`ni `Authorization: Bearer <access_token>` header bilan yuboring. Token muddati tugasa (`401 "access token expired"`), `POST /auth/token/refresh` ga `{"refresh_token": "..."}` yuborib yangi juft token oling (bu endpoint `/mobile` ostida emas, root'da).
> - **Til:** ko'p javoblarda matnlar tilga bog'liq bo'lsa, `Accept-Language: uz|ru|en` header yuboriladi (default `uz`).
> - **Pagination:** ro'yxat endpointlari odatda `limit`, `page` query parametr oladi.
> - **Xatolik formati:** xatolik bo'lsa `4xx/5xx` status + `{"error": "..."}` shaklidagi JSON qaytadi.
> - Pastdagi "Auth" qatori shu endpoint uchun haqiqatda routerda auth middleware borligini bildiradi (`api/api.go`dagi guruhlashga qarab tekshirilgan, ba'zi joylarda swagger annotatsiyasidan ko'ra ishonchliroq).

---


## Autentifikatsiya (Login/Register/Token)

### `POST /mobile/auth/google`
**Google orqali kirish (1 bosqichli)**
BACKEND_STRUCTURE.md 1-bo'lim: Google ID token orqali bir bosqichli kirish
- Auth: kerak emas
**Body** (GoogleLoginReq) (majburiy):
  - `id_token`: string
  - `referral_code`: string
**Javob (200):**
- `access_token`: string
- `id`: string
- `refresh_token`: string
- `role`: string

### `POST /mobile/auth/user/check`
**Check user**
Check user
- Auth: kerak emas
**Body** (UserCheckReq) (majburiy):
  - `identifier`: string
**Javob (200):**
- `has_account`: boolean

### `POST /mobile/auth/user/otp/confirm`
**Verify confirmation code**
Verify the provided confirmation code and return a token if valid
- Auth: kerak emas
**Body** (ConfirmationReq) (majburiy):
  - `confirmation_code`: string
  - `identifier`: string
  - `referral_code`: string
  - `type`: string
**Javob (200):**
`string`

### `POST /mobile/auth/user/otp/send`
**Send confirmation code to email**
Send a confirmation code to the provided email address
- Auth: kerak emas
**Body** (SendConfirmationCodeReq) (majburiy):
  - `identifier`: string
  - `type`: string
**Javob (200):**
`string`

## Foydalanuvchi profili

### `GET /mobile/user/activity`
**Get user activity stats**
Returns user activity stats by type: day|week|month|year|range. For day: optional ?date=YYYY-MM-DD. For range: required ?from=YYYY-MM-DD&to=YYYY-MM-DD
- Auth: kerak (Bearer token)
**Parametrlar:**
- `type` (query, string) (majburiy): Stats type: day|week|month|year|range
- `date` (query, string): For day only: YYYY-MM-DD (default: today)
- `from` (query, string): For range only: YYYY-MM-DD
- `to` (query, string): For range only: YYYY-MM-DD
**Javob (200):**
- `items`: ActivityStatItem[]
- `total`: integer
- `type`: string
- `user_id`: string

### `POST /mobile/user/activity`
**Activity user**
Activity user
- Auth: kerak (Bearer token)
**Body** (UserActivityCreateBody) (majburiy):
  - `activity`: integer
**Javob (200):**
`string`

### `DELETE /mobile/user/delete/profile`
**Delete a profile**
Delete a profile
- Auth: kerak (Bearer token)
**Javob (200):**
`string`

### `POST /mobile/user/device`
**Push token ro'yxatdan o'tkazish (ko'p qurilma support)**
- Auth: kerak (Bearer token)
**Body** (object) (majburiy):
  - `fcm_token`: string
  - `platform`: string
**Javob (200):**
`string`

### `DELETE /mobile/user/device`
**Push token'ni o'chirish (logout)**
- Auth: kerak (Bearer token)
**Body** (object) (majburiy):
  - `fcm_token`: string
**Javob (200):**
`string`

### `GET /mobile/user/get/profile`
**Get user profile**
Get user profile
- Auth: kerak (Bearer token)
**Javob (200):**
- `coins`: integer
- `created_at`: string
- `email`: string
- `id`: string
- `image_url`: string
- `language`: string
- `level`: integer
- `name`: string
- `phone_number`: string
- `specialization`: string
- `streak_count`: integer
- `updated_at`: string
- `xp`: integer

### `GET /mobile/user/limit`
**Kunlik bepul urinish limiti holati (home ekranidagi "35/50 Limit" belgisi)**
- Auth: kerak (Bearer token)
**Javob (200):**
- `has_subscription`: boolean
- `remaining`: integer
- `total`: integer
- `used`: integer

### `GET /mobile/user/rating`
**Reyting (leaderboard) olish**
Reytingni qaytaradi: day|week|month|year|total. Default: total. 0 activity bo'lgan userlar avtomatik oxirida turadi. Response ichida TOP list va "Me" (mening o'rnim) qaytadi.
- Auth: kerak (Bearer token)
**Parametrlar:**
- `type` (query, string): Filter type: day|week|month|year|total (default: total)
- `limit` (query, integer): Top list limit (default: 10, max: 100)
**Javob (200):**
- `items`: RatingUserItem[]
- `limit`: integer
- `me`: RatingUserItem
- `type`: RatingType

### `PUT /mobile/user/update/profile`
**Update a user profile**
Updates user profile fields via multipart/form-data. Image is optional.
- Auth: kerak (Bearer token)
**Parametrlar:**
- `name` (formData, string): User name
- `phone_number` (formData, string): Phone number
- `email` (formData, string): Email
- `language` (formData, string): Language (e.g. uz, ru, en)
- `image` (formData, file): Profile image (jpg/png/webp)
**Javob (200):**
`string`

## Foydalanuvchi (qo'shimcha)

### `GET /mobile/referral`
**Referral ma'lumotlari (do'stlarni taklif qilish)**
O'z referral_code'i, taklif qilganlar soni va ular evaziga olingan tangalar
- Auth: kerak (Bearer token)
**Javob (200):**
- `invited`: ReferralInvitedItem[]
- `invited_count`: integer
- `referral_code`: string
- `referred_reward`: integer
- `referrer_reward`: integer
- `total_coins_earned`: integer

## Bo'limlar (Category)

### `GET /mobile/category`
**Get all categories (mobile)**
- Auth: kerak (Bearer token)
**Parametrlar:**
- `Accept-Language` (header, string, default=`uz`): Language (uz|ru|en)
**Javob (200):**
- `categories`: CategoryMobileRes[]
- `count`: integer

### `GET /mobile/category/{id}`
**Get category by ID (mobile)**
- Auth: kerak (Bearer token)
**Parametrlar:**
- `id` (path, string) (majburiy): Category ID
- `Accept-Language` (header, string, default=`uz`): Language (uz|ru|en)
**Javob (200):**
- `audience`: string
- `cases_count`: integer
- `icon_url`: string
- `id`: string
- `name`: string
- `order_num`: integer

## Mavzular (Topic)

### `GET /mobile/topic`
**Mavzular ro'yxati (mobile)**
- Auth: kerak (Bearer token)
**Parametrlar:**
- `category_id` (query, string): Category ID
- `limit` (query, integer): Limit
- `page` (query, integer): Page
- `Accept-Language` (header, string, default=`uz`): Language (uz|ru|en)
**Javob (200):**
- `count`: integer
- `topics`: TopicMobileRes[]

## Klinik Case'lar

### `GET /mobile/case`
**Get cases catalog (mobile)**
- Auth: kerak (Bearer token)
**Parametrlar:**
- `topic_id` (query, string): Topic ID
- `category_id` (query, string): Category ID
- `difficulty` (query, string): easy|medium|hard
- `status` (query, string): new|active|completed|failed|abandoned (foydalanuvchi holati)
- `search` (query, string): Search
- `limit` (query, integer): Limit
- `page` (query, integer): Page
- `Accept-Language` (header, string, default=`uz`): Language (uz|ru|en)
**Javob (200):**
- `cases`: CaseMobileRes[]
- `count`: integer

### `GET /mobile/case/random`
**"Try your luck" - tasodifiy keys (BACKEND_STRUCTURE.md 3-bo'lim)**
- Auth: kerak (Bearer token)
**Parametrlar:**
- `Accept-Language` (header, string, default=`uz`): Language (uz|ru|en)
**Javob (200):**
- `category_id`: string
- `category_name`: string
- `chief_complaint`: string
- `cover_image_url`: string
- `difficulty`: string
- `expected_duration_minutes`: integer
- `id`: string
- `is_favorite`: boolean
- `patient_age`: integer
- `patient_gender`: string
- `status`: string
- `subtitle`: string
- `title`: string
- `topic_id`: string
- `topic_name`: string

### `GET /mobile/case/{id}`
**Get case detail (mobile)**
- Auth: kerak (Bearer token)
**Parametrlar:**
- `id` (path, string) (majburiy): Case ID
- `Accept-Language` (header, string, default=`uz`): Language (uz|ru|en)
**Javob (200):**
- `category_id`: string
- `category_name`: string
- `chief_complaint`: string
- `cover_image_url`: string
- `difficulty`: string
- `expected_duration_minutes`: integer
- `id`: string
- `is_favorite`: boolean
- `patient_age`: integer
- `patient_gender`: string
- `status`: string
- `subtitle`: string
- `title`: string
- `topic_id`: string
- `topic_name`: string

## Sevimlilar (Favorites)

### `GET /mobile/favorite`
**Sevimlilar ro'yxati (GET /api/v1/favorites - BACKEND_STRUCTURE.md 3-bo'lim)**
- Auth: kerak (Bearer token)
**Parametrlar:**
- `limit` (query, integer): Limit
- `page` (query, integer): Page
- `Accept-Language` (header, string, default=`uz`): Language (uz|ru|en)
**Javob (200):**
- `cases`: CaseMobileRes[]
- `count`: integer

### `POST /mobile/favorite`
**Sevimlilarga qo'shish/olib tashlash (BACKEND_STRUCTURE.md 3-bo'lim)**
- Auth: kerak (Bearer token)
**Body** (object) (majburiy):
  - `case_id`: string
**Javob (200):**
- `is_favorite`: boolean

## Simulyatsiya dvigateli

### `GET /mobile/simulation/completed`
**Profile -> Completed (BACKEND_STRUCTURE.md 1-bo'lim)**
- Auth: kerak (Bearer token)
**Javob (200):**
- `count`: integer
- `sessions`: SimulationSessionRes[]

### `GET /mobile/simulation/ongoing`
**Profile -> Ongoing (BACKEND_STRUCTURE.md 1-bo'lim)**
- Auth: kerak (Bearer token)
**Javob (200):**
- `count`: integer
- `sessions`: SimulationSessionRes[]

### `POST /mobile/simulation/start`
**Sessiyani boshlash (BACKEND_STRUCTURE.md 4-bo'lim)**
- Auth: kerak (Bearer token)
**Body** (object) (majburiy):
  - `case_id`: string
**Javob (200):**
- `case_id`: string
- `health_percent`: integer
- `initial_vitals`: CaseVitals
- `session_id`: string
- `time_limit_seconds`: integer
- `visual_state`: string
- `ws_url`: string

### `GET /mobile/simulation/{id}`
- Auth: kerak (Bearer token)
**Parametrlar:**
- `id` (path, string) (majburiy): Session ID
**Javob (200):**
- `case_id`: string
- `case_title`: string
- `coins_earned`: integer
- `ended_at`: string
- `final_score`: integer
- `health_percent`: integer
- `id`: string
- `started_at`: string
- `status`: string
- `user_id`: string
- `xp_earned`: integer

### `POST /mobile/simulation/{id}/event`
**Anamnez/ko'rik/tahlil/dori qadamini yuborish (BACKEND_STRUCTURE.md 4-bo'lim)**
- Auth: kerak (Bearer token)
**Parametrlar:**
- `id` (path, string) (majburiy): Session ID
**Body** (SimulationEventReq) (majburiy):
  - `payload`: object (key->object)
  - `session_id`: string
  - `type`: string
**Javob (200):**
- `health_delta`: integer
- `health_percent`: integer
- `is_correct`: boolean
- `response`: object (key->object)

### `PUT /mobile/simulation/{id}/finish`
**Sessiyani yakunlash (BACKEND_STRUCTURE.md 4,5-bo'lim)**
- Auth: kerak (Bearer token)
**Parametrlar:**
- `id` (path, string) (majburiy): Session ID
**Body** (object) (majburiy):
  - `reason`: string
**Javob (200):**
- `coins_earned`: integer
- `debrief_ready`: boolean
- `final_score`: integer
- `session_id`: string
- `xp_earned`: integer

### `GET /mobile/simulation/{id}/ws`
**Real-vaqt vitallar monitori (BACKEND_STRUCTURE.md 4-bo'lim, WebSocket)**
- Auth: kerak emas
**Parametrlar:**
- `id` (path, string) (majburiy): Session ID

## Debriefing hisoboti

### `GET /mobile/debrief/{session_id}`
**AI orqali Debriefing hisoboti (BACKEND_STRUCTURE.md 5-bo'lim)**
- Auth: kerak (Bearer token)
**Parametrlar:**
- `session_id` (path, string) (majburiy): Session ID
**Javob (200):**
- `coins_earned`: integer
- `correct_steps`: string[]
- `created_at`: string
- `final_score`: integer
- `guideline_notes`: string
- `incorrect_steps`: string[]
- `session_id`: string
- `weak_topics`: string[]
- `xp_earned`: integer

## Sozlamalar (limit, ovoz)

### `GET /mobile/setting/voice`
**Ovozli funksiya (shikoyat/chat matnini o'qib berish) global yoqilganmi**
Audio backendda generatsiya qilinmaydi - frontend shu bayroqqa qarab
qurilma TTS'idan (device text-to-speech) foydalanadi yoki yashiradi.
- Auth: kerak (Bearer token)
**Javob (200):**
- `enabled`: boolean

## Promokodlar

### `GET /mobile/promocode`
**Mening Promokodlarim (ishlatganlarim, BACKEND_STRUCTURE.md 5-bo'lim)**
- Auth: kerak (Bearer token)
**Parametrlar:**
- `limit` (query, integer): Limit
- `page` (query, integer): Page
**Javob (200):**
- `count`: integer
- `promocodes`: PromoCodeRes[]

### `POST /mobile/promocode/redeem`
**Promokodni kiritib tanga olish (TZ 2.3-bo'lim)**
- Auth: kerak (Bearer token)
**Body** (object) (majburiy):
  - `code`: string
**Javob (200):**
- `coins_added`: integer
**Xatolar:** `promo_code_invalid` (400), `promo_code_expired` (400), `promo_code_used` (409, limit tugagan), `promo_code_already_redeemed` (409, siz allaqachon ishlatgansiz), `promo_code_not_for_you` (403, boshqa odam uchun kod)

## Tariflar (Obuna/Coin paket)

### `GET /mobile/tariff`
**Get all tariff**
Get all tariff with optional filtering
- Auth: kerak (Bearer token)
**Parametrlar:**
- `duration` (query, integer): Duration
**Javob (200):**
- `count`: integer
- `tariffs`: TariffRes[]

### `GET /mobile/tariff/{id}`
**Get tariff by ID**
Get a tariff by their ID
- Auth: kerak (Bearer token)
**Parametrlar:**
- `id` (path, string): Tariff ID
**Javob (200):**
- `coins`: integer
- `created_at`: string
- `description`: string
- `duration`: integer
- `id`: string
- `kind`: string
- `name`: string
- `price`: number
- `updated_at`: string

## Obuna

### `POST /mobile/subscription`
**Premium obunaga buyurtma (BACKEND_STRUCTURE.md 5-bo'lim, Click orqali to'lov)**
- Auth: kerak (Bearer token)
**Body** (OrderCreateBody) (majburiy):
  - `coins_used`: integer
  - `tariff_id`: string
**Javob (200):**
`string`

## O'quv rejasi (eslatma)

### `GET /mobile/study-plan`
- Auth: kerak (Bearer token)
**Javob (200):**
- `is_enabled`: boolean
- `remind_minutes_before`: integer
- `remind_time`: string
- `user_id`: string

### `PUT /mobile/study-plan`
**O'quv rejasi va eslatma vaqtini belgilash (BACKEND_STRUCTURE.md 2-bo'lim)**
- Auth: kerak (Bearer token)
**Body** (StudyPlanUpsertReq) (majburiy):
  - `is_enabled`: boolean
  - `remind_minutes_before`: integer
  - `remind_time`: string
  - `user_id`: string
**Javob (200):**
`string`

## Bildirishnomalar

### `GET /mobile/notification/user`
**Get all notification**
Get all notification with optional filtering
- Auth: kerak (Bearer token)
**Parametrlar:**
- `is_read` (query, boolean): Is Read
- `Accept-Language` (header, string, default=`uz`): Language (uz|ru|en)
**Javob (200):**
- `count`: integer
- `notifications`: UserNotificationRes[]

### `PUT /mobile/notification/{id}/read`
**Read notification**
Read notification
- Auth: kerak (Bearer token)
**Parametrlar:**
- `id` (path, string): Notification ID
**Javob (200):**
`string`

## Bannerlar

### `GET /mobile/banner`
**Get all banner**
Get all banner with optional filtering
- Auth: kerak (Bearer token)
**Parametrlar:**
- `Accept-Language` (header, string, default=`uz`): Language (uz|ru|en)
**Javob (200):**
- `banners`: BannerMobileRes[]
- `count`: integer

### `GET /mobile/banner/{id}`
**Get banner by ID**
Get a banner by their ID
- Auth: kerak (Bearer token)
**Parametrlar:**
- `id` (path, string): Banner ID
- `Accept-Language` (header, string, default=`uz`): Language (uz|ru|en)
**Javob (200):**
- `created_at`: string
- `description`: string
- `id`: string
- `image_url`: string
- `link_url`: string
- `order_num`: integer
- `title`: string
- `updated_at`: string

## Biz haqimizda

### `GET /mobile/about`
**Get all about**
Get all about with optional filtering
- Auth: kerak (Bearer token)
**Parametrlar:**
- `title` (query, string): Title
- `Accept-Language` (header, string, default=`uz`): Language (uz|ru|en)
**Javob (200):**
- `abouts`: AboutMobileRes[]
- `count`: integer

### `GET /mobile/about/{id}`
**Get about by ID**
Get a about by their ID
- Auth: kerak (Bearer token)
**Parametrlar:**
- `id` (path, string): About ID
- `Accept-Language` (header, string, default=`uz`): Language (uz|ru|en)
**Javob (200):**
- `created_at`: string
- `description`: string
- `id`: string
- `link_url`: string
- `order_num`: integer
- `title`: string
- `updated_at`: string

## FAQ

### `GET /mobile/faq`
**Get all faq**
Get all faq with optional filtering
- Auth: kerak (Bearer token)
**Parametrlar:**
- `question` (query, string): Question
- `Accept-Language` (header, string, default=`uz`): Language (uz|ru|en)
**Javob (200):**
- `count`: integer
- `faqs`: FaqMobileRes[]

### `GET /mobile/faq/{id}`
**Get faq by ID**
Get a faq by their ID
- Auth: kerak (Bearer token)
**Parametrlar:**
- `id` (path, string): Faq ID
- `Accept-Language` (header, string, default=`uz`): Language (uz|ru|en)
**Javob (200):**
- `answer`: string
- `created_at`: string
- `id`: string
- `order_num`: integer
- `question`: string
- `updated_at`: string

## Kontaktlar

### `GET /mobile/contact`
**Get all contact**
Get all contact with optional filtering
- Auth: kerak (Bearer token)
**Parametrlar:**
- `name` (query, string): Name
- `phone_number` (query, string): PhoneNumber
- `limit` (query, integer): Limit
- `page` (query, integer): Page
**Javob (200):**
- `contacts`: ContactRes[]
- `count`: integer

### `GET /mobile/contact/{id}`
**Get contact by ID**
Get a contact by their ID
- Auth: kerak (Bearer token)
**Parametrlar:**
- `id` (path, string): Contact ID
**Javob (200):**
- `created_at`: string
- `id`: string
- `link_url`: string
- `name`: string
- `phone_number`: string
- `updated_at`: string

## App Route (ilova ichki linklar)

### `GET /mobile/app-route`
**Get all appRoute**
Get all appRoute with optional filtering
- Auth: kerak emas
**Javob (200):**
- `app_routes`: AppRouteRes[]
- `count`: integer

### `GET /mobile/app-route/{id}`
**Get appRoute by ID**
Get a appRoute by their ID
- Auth: kerak emas
**Parametrlar:**
- `id` (path, string): AppRoute ID
**Javob (200):**
- `app_links`: object
- `app_version`: object
- `buy_course`: boolean
- `call_center`: string
- `created_at`: string
- `id`: string
- `payment_min_version`: string
- `support_url`: string
- `updated_at`: string

## Hamkorlar

### `GET /mobile/partner`
**Home ekranidagi "Hamkor kurslari" banneri (TZ 2.1-bo'lim)**
- Auth: kerak (Bearer token)
**Javob (200):**
- `count`: integer
- `partners`: PartnerMobileRes[]
