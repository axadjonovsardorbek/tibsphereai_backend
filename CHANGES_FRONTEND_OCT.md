# Backend o'zgarishlari (1–15 oktyabr 2026) — frontend uchun

Bu hujjat [`CHANGES_FRONTEND_USER.md`](CHANGES_FRONTEND_USER.md) va
[`CHANGES_FRONTEND_ADMIN.md`](CHANGES_FRONTEND_ADMIN.md) (27–30 sentabr) dan **keyingi**
o'zgarishlarni o'z ichiga oladi. Ishlaydigan referens implementatsiya:
[`user-panel.html`](user-panel.html) (user-app) va [`admin-panel.html`](admin-panel.html) (admin).
To'liq endpoint tavsifi: [`API_MOBILE.md`](API_MOBILE.md), [`API_WEB.md`](API_WEB.md), Swagger (`/api/swagger/index.html`).

Belgilar: **[USER]** — user-app (web/mobile), **[ADMIN]** — admin panel.

## Qisqa jadval

| Sana | Nima | Kim |
|---|---|---|
| 10-15 | Case'ga urinishlar limiti `max_attempts` (default 10) | USER + ADMIN |
| 10-15 | Chat endi "javob yozish" sifatida ishlaydi: to'g'ri javobda sessiya avtomatik tugaydi | USER |
| 10-10 | Case fayldan ommaviy yuklash (xlsx/csv) + shablon yuklab olish | ADMIN |
| 10-10 | Limit tugaganda case'ni tanga evaziga boshlash (`case_coin_cost`, `use_coins`) | USER + ADMIN |
| 10-10 | Promokod: har bir foydalanuvchi uchun limit, `0 = cheksiz` | ADMIN + USER |
| 10-08/10 | To'lov faqat inPAY (Click olib tashlandi) | USER |
| 10-08 | Hech qanday klinik harakatsiz tugatilsa `final_score = 0` | USER |
| 10-03 | Universal promokodlar (`scope`), hamkor ixtiyoriy | ADMIN |
| 10-09 | Coin tranzaksiyalar ro'yxati javobi o'zgardi (pagination) | ADMIN |
| 10-05 | Push: shaxsiy topic tilga bog'langan, obunani frontend qiladi | USER |

---

## 1. Case urinishlar limiti — `max_attempts` **[USER + ADMIN]**

Har bir case'da `max_attempts` (integer, default `10`): foydalanuvchi **bitta sessiyada** shu case'ga nechta
xabar yubora olishi. Limitga yetsa sessiya avtomatik yakunlanadi.

**Nima hisoblanadi:** faqat `type: "question"` turidagi xabarlar (matn ham, ovozli ham). Ko'rik/tahlil/dori
(`exam`, `lab`, `imaging`, `medication`, `procedure`) hisoblanmaydi.

### 1.1 Admin
- `POST /web/case`, `PUT /web/case`: yangi maydon `max_attempts`. Bo'sh yoki `0` — default 10 (create) / eski qiymat
  o'zgarmaydi (update).
- `GET /web/case`, `GET /web/case/{id}` javobida `max_attempts`.
- Fayldan yuklashda ixtiyoriy `max_attempts` ustuni (bo'sh = 10). Shablon yangilangan.
- Case formasiga raqam maydoni, ro'yxat jadvaliga ustun qo'shing (qarang `admin-panel.html`).

### 1.2 User-app
- Case ro'yxati (`GET /mobile/case`), detal (`/mobile/case/{id}`), random, sevimlilar: `max_attempts`.
- `POST /mobile/simulation/start` javobida `max_attempts`.
- `GET /mobile/simulation/{id}`, `/ongoing`, `/completed`: `max_attempts` + `attempts_used` — sessiya qayta
  ochilganda hisoblagichni tiklash uchun.
- `POST /mobile/simulation/{id}/event` javobida har doim:
  ```json
  { "max_attempts": 10, "attempts_used": 3, "attempts_left": 7, "session_ended": false }
  ```
- `attempts_left = 0` bo'lsa backend sessiyani **o'zi** yakunlaydi: `session_ended: true` va `finish_result`
  qaytadi (tugash sababi `attempts_exhausted`, sessiya holati `failed`, tanga berilmaydi, XP to'g'ri javob
  topilmagani uchun 0). Shundan keyin yana xabar yuborilsa `409 session_not_active`.

**FE nima qiladi:**
1. Chat yonida "Qolgan urinishlar: N / M" ko'rsatadi (`attempts_left` / `max_attempts`).
2. `session_ended: true` kelsa kiritishni yopadi va `finish_result` bilan natija ekranini ko'rsatadi
   (`health_zero` uchun yozilgan oqim bilan bir xil — qayta `PUT /finish` chaqirilmaydi).

---

## 2. Chat — "javob yozish" rejimi, to'g'ri javobda avtomatik tugash **[USER]**

Chatga yozilgan har bir xabar endi AI tomonidan **javob urinishi** sifatida baholanishi mumkin (bemor gapirmaydigan
case'lar uchun prompt shunday sozlanadi: `ai_prompts.patient_persona`).

Event javobi (`POST /mobile/simulation/{id}/event`, `type: "question"`) ikki xil bo'lishi mumkin:

| Holat | `response` ichida | Ma'nosi |
|---|---|---|
| Oddiy savol | `patient_reply` | Bemor javobi (eski xatti-harakat) |
| Javob urinishi | `feedback`, `is_correct` (tashqarida) | Platforma izohi: to'g'ri/xato + ishora |

- **To'g'ri javob**: `is_correct: true`, `session_ended: true`, `finish_result` (AI debrief, XP, coin). Tugash sababi
  `solved`. Frontend natija ekraniga o'tadi.
- **Xato javob**: `is_correct: false`, `feedback` da to'g'ri yo'nalishga ishora (javobning o'zi ochilmaydi),
  `health_delta` odatda `0`. Sessiya davom etadi (vaqt yoki urinishlar tugaguncha).
- `feedback` ni chatda platforma xabari sifatida ko'rsating (✅ / ⚠️), bemor xabari sifatida emas.
  `reply_audio_base64` bo'lsa o'ynatiladi (avvalgidek).
- Hech qanday to'g'ri javob topilmasa va vaqt tugasa `PUT /finish` `reason: "timeout"` — `final_score = 0`.

`reason` qiymatlari (`PUT /finish` yuboradigan va backend o'zi qo'yadiganlar): `manual`, `timeout`,
`health_zero`, `solved`, `attempts_exhausted`.

---

## 3. Hech qanday klinik harakatsiz tugatish — `final_score = 0` **[USER]**

Foydalanuvchi hech qanday klinik harakat qilmay (faqat anamnez chat) `finish` bossa, `final_score`, `xp_earned`,
`coins_earned` — hammasi `0`. (Avval health 100% bo'lgani uchun to'liq ball berilardi.) Chatdagi **to'g'ri javob**
(2-bo'lim) klinik harakat hisoblanadi. Xato javoblar hisoblanmaydi.

FE: natija ekranida 0 ball kutilgan holat; alohida ishlov shart emas.

---

## 4. Limit tugaganda case'ni tanga evaziga boshlash **[USER + ADMIN]**

- **Admin:** `PUT /web/setting` `{"key":"case_coin_cost","value":"5"}` — narx (tanga), `0` = o'chiq.
- **User:** `GET /mobile/user/limit` javobida `case_coin_cost`.
- Bepul limit tugagan va obunasiz foydalanuvchida `POST /mobile/simulation/start` → `403 daily_limit_reached`.
  `case_coin_cost > 0` bo'lsa foydalanuvchidan tasdiq so'rang (`"Case N tanga evaziga boshlansinmi?"`), roziligida
  shu so'rovni `"use_coins": true` bilan qayta yuboring. Narx balansdan yechiladi; yetarli bo'lmasa
  `insufficient_coins`. Tranzaksiya sababi: `case_unlock`.
- `use_coins` yuborilmasa tanga hech qachon yechilmaydi.

Referens: `startSim()` — `user-panel.html`.

---

## 5. Promokodlar **[ADMIN + USER]**

### 5.1 Universal promokodlar (`scope`)
`POST /web/promocode` body:
- `scope`: `user` (aynan bitta odam, `target_user_id` majburiy) | `users` (aniq bir nechta odam,
  `target_user_ids: string[]`) | `limited` (ko'pi bilan `max_uses` ta odam) | `all` (hamma uchun).
- `partner_id` — ixtiyoriy (bo'sh = hamkorsiz kod). `created_by` token'dan olinadi, body'da yuborilmaydi.
- Noto'g'ri `scope` → `400 promo_code_scope_invalid`.
- Admin formada "kim uchun" tanlovi; `users` uchun ID yozish o'rniga qidiruvli ro'yxatdan tanlash
  (`admin-panel.html`).
- Javobda: `scope`, `target_user_id`, `target_user_ids`, `max_uses` (`null` = cheksiz), `used_count`.

### 5.2 Ishlatish limitlari
- `max_uses` — kod **jami** necha marta ishlatilishi (`scope=limited`).
- `max_uses_per_user` — **bitta foydalanuvchi** bir kodni necha marta ishlata oladi.
- Ikkalasida ham: ko'rsatilmasa — **1**, **`0` — cheksiz**, manfiy — xato.
- `scope=user` da jami = `max_uses_per_user`; `scope=users` da jami = odamlar soni × `max_uses_per_user`
  (0 bo'lsa cheksiz).

### 5.3 User-app xatolari (`POST /mobile/promocode/redeem`)
`promo_code_invalid` (400), `promo_code_expired` (400), `promo_code_used` (409, jami limit tugagan),
`promo_code_already_redeemed` (409, shaxsiy limit tugagan), `promo_code_not_for_you` (403, kod boshqa odamga).
Xato javobi formati: `{error: {code, slug, message}}`; `message` `Accept-Language` (uz|ru|en) bo'yicha keladi, `slug` bo'yicha shartlash mumkin.

---

## 6. To'lov — faqat inPAY **[USER]**

`POST /mobile/subscription` body: `{"tariff_id": "...", "coins_used": 0, "provider": "inpay"}`
(`provider` ixtiyoriy; Click olib tashlangan, berilmasa ham inPAY).

Javob: `{"order_id": "...", "url": "<pay_url>", "just_paid": false}`.
- Frontend `url` ni ochadi (yangi tab / webview).
- To'lov tugagach backend webhook orqali order'ni `PAID` qiladi; frontend `GET /web/order/{id}` yoki obuna holatini
  poll qilib natijani bilib oladi (backendda fon reconciler ham bor — webhook kechiksa ham order yopiladi).
- `just_paid: true` (to'liq tangaga yopilgan) bo'lsa `url` bo'sh.
- Minimal to'lov **1000 so'm**.
- UI'dan Click tugmasi/tanlovini olib tashlang.

---

## 7. Case'larni fayldan ommaviy yuklash **[ADMIN]**

- `POST /web/case/import` (multipart): `file` (`.xlsx`/`.csv`, ≤5MB, ≤500 qator), `publish=true|false`.
- `GET /web/case/import/template` — `.xlsx` shablonni yuklab olish (Authorization header kerak, shuning uchun
  `fetch` + blob; `admin-panel.html` dagi "Namuna Excel" tugmasiga qarang).
- Ustunlar: `topic`, `title_uz/ru/en`, `subtitle_*`, `chief_complaint_*`, `difficulty`, `patient_age`,
  `patient_gender`, `expected_duration_minutes`, **`max_attempts`**, `hr`, `bp`, `spo2`, `rr`, `temp`, `gcs`,
  `visual_state`, `expected_answer`, `cover_image_url`, `order_num`, `scenario` (JSON).
- Javob: `{total, created, published, failed, ids[], errors[{row, message}]}`; xatoli qatorlar o'tkazib yuboriladi.

---

## 8. Boshqa o'zgarishlar

- **[ADMIN] Coin tranzaksiyalar** (`GET /web/coin-transaction`): javob endi `PaginationResponse` —
  `data: CoinTransactionRes[]` + `total, page, limit, total_pages, has_previous, has_next`
  (avval `{count, transactions}`).
- **[USER] Push bildirishnoma:** shaxsiy topic endi tilga bog'liq: `user_<id>_<uz|ru|en>`. Backend qurilmani topicga
  **obuna qilmaydi ham, obunadan chiqarmaydi** — faqat `all_<lang>` va `user_<id>_<lang>` ga yuboradi.
  Obunani frontend o'zi (Firebase SDK) qilishi kerak; til o'zgarganda eski topicdan chiqib, yangisiga obuna bo'ling.
- **[USER] Telegram bot:** login kodi ostida "ilovaga qaytish" (`web_app`) tugmasi (FE ta'siri yo'q).
- **[USER] Referral:** `GET /mobile/referral` (kod, taklif qilganlar soni, olingan tangalar) va login paytida
  referral kod yuborish `user-panel.html` da qo'shilgan.
- **[ADMIN] AI case generatsiya:** natija har doim `CaseCreateReq` formatida qaytadi (`expected_answer` string yoki
  `{uz,ru,en}` obyekt bo'lishi mumkin; sarlavha tekshiriladi). `max_attempts` bo'sh bo'lsa 10 qo'yiladi.

---

## 9. Muhit (deploy) — frontend env

Dev backend: `https://dev.tibstation.uz`. Web uchun build vaqtida (`VITE_*`) va admin uchun
(`NEXT_PUBLIC_*`) env kerak; ular kodga build paytida joylanadi, o'zgarsa **qayta build** shart.

| Web (`tibcase-web`) | Admin (`tibcase-admin`) |
|---|---|
| `VITE_API_BASE`, `VITE_BACKEND_API_URL` | `NEXT_PUBLIC_API_BASE` |
| `BACKEND_HOST` (nginx proxy, runtime) | `BACKEND_API_URL` (runtime, `/api/...` proxy) |
| `VITE_GOOGLE_CLIENT_ID`, `VITE_FIREBASE_*` | `SWAGGER` |

GitHub Actions deploy: `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_SSH_KEY` (to'liq **private** kalit,
`-----BEGIN OPENSSH PRIVATE KEY-----` dan `-----END ...-----` gacha; `SHA256:...` fingerprint emas).

---

## Tekshirish ro'yxati

**USER**
- [ ] Chatda "Qolgan urinishlar: N / M" (`attempts_left`/`max_attempts`), resume'da `GET /mobile/simulation/{id}` dan
- [ ] `session_ended: true` (to'g'ri javob / urinishlar tugadi / HP 0) → kiritishni yopib `finish_result` ko'rsatish
- [ ] Chatda `response.feedback` ni platforma xabari (✅/⚠️) sifatida ko'rsatish
- [ ] `use_coins` bilan qayta start (limit tugaganda, `case_coin_cost > 0`)
- [ ] Promokod xatolari (`promo_code_already_redeemed`, `promo_code_not_for_you`, ...) matnlari
- [ ] To'lov: faqat inPAY, `url` ni ochish, order holatini poll qilish, Click'ni olib tashlash
- [ ] Push: `user_<id>_<lang>` topicga o'zi obuna bo'lish, til o'zgarganda qayta obuna

**ADMIN**
- [ ] Case formasi/jadvali: `max_attempts` maydoni (default 10)
- [ ] Fayldan yuklash: `max_attempts` ustuni, shablon yuklab olish tugmasi
- [ ] Promokod formasi: `scope`, `target_user_ids` (qidiruvli tanlov), `max_uses`, `max_uses_per_user` (0 = cheksiz)
- [ ] `case_coin_cost` sozlamasi
- [ ] Coin tranzaksiyalar: yangi pagination javobi (`data`, `total_pages`, ...)
