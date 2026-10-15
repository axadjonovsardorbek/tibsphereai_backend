# Backend o'zgarishlari (27–30 sentabr 2026) — user-app frontend uchun

Bu hujjat oxirgi 3 kunda backendga kirgan commitlardan **mobil/user-web
ilova** (o'quvchi/shifokor tomoni, `/mobile/...` endpointlar) frontendiga
tegishlilarini o'z ichiga oladi. Admin panelga tegishli o'zgarishlar uchun
[`CHANGES_FRONTEND_ADMIN.md`](CHANGES_FRONTEND_ADMIN.md) ga qarang.

Ishlaydigan referens implementatsiya (barcha o'zgarishlar allaqachon
qo'llangan): [`user-panel.html`](user-panel.html).

## Qisqa jadval

| Sana | Commit | Nima |
|---|---|---|
| 09-27 | `689ae7e` | Telegram bot menu tugmalari endi Mini App sifatida ochiladi |
| 09-28 | `c4e8b4e` | Google Sign-In haqiqiy tugma, study-plan eslatma cron, push per-user topic |
| 09-29 | `b291c5b` | health_percent 0'da auto-finish, debrief jarima, session_not_active xatosi |
| 09-29 | `c749ccc` | TTS: uz-UZ ovoz topilmasa jim qolish tuzatildi (eskirgan, pastga qarang) |
| 09-29 | `096f7e9` | Ovozli kiritish (STT, brauzer), AI xatolari uchun tushunarli xabar |
| 09-29 | `43e5a1b` | Ovozli kiritish Safari'da "service-not-allowed" xatosi tuzatildi (eskirgan) |
| 09-30 | `63f432c` | **Ovoz to'liq Gemini orqali (STT+TTS)** — brauzer speech API butunlay olib tashlandi |
| 09-30 | `a0bcc25`, `ea38636` | Har bir ovozli xabarda "qayta eshitish" (replay) tugmasi |

Eng muhim va FE tomonda majburiy ish talab qiladigan qism — pastdagi **1** va
**2**-bo'limlar (ovozli chat va auto-finish).

---

## 1. Ovozli chat — endi to'liq Gemini orqali (STT + TTS)

**MUHIM: Brauzer Speech API (`SpeechRecognition`/`webkitSpeechRecognition`,
`speechSynthesis`) endi ishlatilmaydi.** Avvalgi 2 kunda (`096f7e9`, `43e5a1b`,
`c749ccc`) qo'shilgan brauzer-asosli ovoz kodi `63f432c` commitida butunlay
almashtirildi — Safari/Firefox'da ishlamaslik, uz-UZ ovoz topilmasligi kabi
muammolarning barchasi shu bilan bartaraf bo'ldi, chunki endi hech narsa
brauzer imkoniyatiga bog'liq emas.

### 1.1 Kiritish (ovozli savol/harakat yuborish)

Frontend mikrofon orqali audio yozadi (`MediaRecorder`), va uni matn o'rniga
`POST /mobile/simulation/{id}/event` so'roviga quyidagi qo'shimcha
`payload` kalitlari bilan yuboradi:

```json
{
  "session_id": "...",
  "type": "question",
  "payload": {
    "audio_base64": "<MediaRecorder blob'ining base64'i>",
    "audio_mime": "audio/webm"
  }
}
```

- `audio_base64` — majburiy, raw base64 (data URL prefiksisiz, ya'ni
  `data:audio/webm;base64,` qismisiz).
- `audio_mime` — ixtiyoriy, berilmasa backend `audio/webm` deb hisoblaydi.
- Backend audio'ni Gemini orqali matnga aylantiradi va **xuddi shu matnni
  qo'lda yozilgandek** pastki oqimga (PatientAI/ActionEvaluator) yuboradi —
  `type` maydoni (`question` yoki boshqa) qanday ishlagan bo'lsa, shundayligicha
  qoladi.
- Javobda qo'shimcha maydonlar:
  - `response.transcript` (string) — Gemini nima eshitganini matn ko'rinishida
    qaytaradi, chatda "siz aytdingiz: ..." qilib ko'rsatish uchun.
  - `response.transcribe_error` (string, o'zbekcha, foydalanuvchiga
    ko'rsatsa bo'ladi) — audio tushunarsiz bo'lsa/AI xato bersa; bu holda
    PatientAI/Evaluator umuman chaqirilmaydi, faqat shu xabar qaytadi.

### 1.2 Chiqish (bemor/feedback javobini eshittirish)

Endi **har bir matnli javob** (`response.patient_reply` yoki
`response.feedback`) bilan birga backend Gemini TTS orqali audio ham
generatsiya qiladi va javobga qo'shadi:

```json
{
  "health_delta": 0,
  "health_percent": 92,
  "is_correct": true,
  "response": {
    "patient_reply": "Ha, ko'krak qafasida og'riq bor...",
    "reply_audio_base64": "<WAV audio, base64>",
    "reply_audio_mime": "audio/wav"
  }
}
```

- `reply_audio_base64` bo'lsa — frontend shuni ijro etadi (masalan
  `Audio` obyektiga `data:audio/wav;base64,...` sifatida beriladi).
- Bo'lmasa (AI xato bergan/GEMINI_API_KEY yo'q) — bu holatda frontend
  brauzer `speechSynthesis`'iga **fallback** qilishi mumkin (majburiy emas,
  lekin ovoz butunlay jim qolmasligi uchun tavsiya etiladi — `user-panel.html`
  ichida `speakText` funksiyasi shu naqshni ko'rsatadi).
- Til/ovoz tanlash frontend tomonda kerak emas — matn qaysi tilda bo'lsa
  (hozircha o'zbekcha), audio ham shu tilda chiqadi.

### 1.3 Replay tugmasi (`a0bcc25`, `ea38636`)

Har bir ovozli xabar (bemor javobi HAM, shifokorning o'z ovozli xabari HAM)
chatda saqlanib qolishi va qayta bosib eshitib bo'lishi kerak — bir martalik
avtoijro emas. Amalda: chat xabar obyektiga `audioBase64`/`audioMime`'ni
saqlab qo'ying, xabar yonida kichik "🔊 qayta eshitish" tugmasi chiqaring,
bosilganda saqlangan audio'ni qayta ijro eting. Referens: `user-panel.html`
ichidagi `playBase64Audio` va xabar render qismidagi replay tugmasi.

### 1.4 AI xatolari — endi tushunarli xabar (`096f7e9`)

`response.patient_reply_error` / `response.feedback_error` /
`response.transcribe_error` endi Gemini'ning xom inglizcha xatosini emas,
umumiy o'zbekcha xabarni qaytaradi: *"AI xizmati hozir band yoki vaqtincha
ishlamayapti. Birozdan so'ng qayta urinib ko'ring."* — frontend bu matnni
to'g'ridan-to'g'ri foydalanuvchiga ko'rsatishi mumkin (ilgari xom matnni
ko'rsatish/parslashga urinish shart emas edi va endi ham emas).

---

## 2. Simulyatsiya auto-finish (`health_percent` 0'ga tushganda)

**Muammo edi:** health_percent 0'ga tushgach ham sessiya "active" bo'lib
qolardi, frontend "Yakunlash" bosishini kutish kerak edi va shu oralda yana
buyruq yuborsa bo'lardi ("o'lgan" bemorga cheksiz muolaja).

**Endi:** `POST /mobile/simulation/{id}/event` javobida `health_percent` shu
harakatdan keyin 0'ga tushsa, backend **avtomatik** `finish(reason="health_zero")`
chaqiradi va javobga qo'shadi:

```json
{
  "health_delta": -30,
  "health_percent": 0,
  "is_correct": false,
  "response": { "...": "..." },
  "session_ended": true,
  "finish_result": {
    "session_id": "...",
    "final_score": 0,
    "xp_earned": 12,
    "coins_earned": 0,
    "debrief_ready": true
  }
}
```

**Frontend qilishi kerak bo'lgan tuzatish:**
- Har bir `/event` javobida `session_ended` bayrog'ini tekshiring.
- `true` bo'lsa: keyingi savol/harakat yuborishni **to'xtating** (input/mikrofon
  tugmalarini disable qiling), `finish_result` ichidagi XP/coin/final_score'ni
  ko'rsating va debrief ekraniga o'ting — **endi qo'lda `PUT /finish`
  chaqirmang**, chunki sessiya allaqachon backend tomonidan yakunlangan.
- Eski kod hali ham qo'lda `PUT /mobile/simulation/{id}/finish` chaqirsa (masalan
  foydalanuvchi "Yakunlash" tugmasini bossa, sessiya allaqachon auto-finish
  bo'lgan bo'lsa) — endi bu so'rov **`409 Conflict`** (`session_not_active`)
  qaytaradi, `200` emas. Frontend bu holatni ham "sessiya allaqachon
  yakunlangan, natija allaqachon ko'rsatilgan" deb ushlab, xato sifatida
  ko'rsatmasligi kerak.

### Yangi xato kodlari

| HTTP | `error` (kod) | Qachon | FE nima qilishi kerak |
|---|---|---|---|
| 404 | `session_not_found` | noto'g'ri/mavjud bo'lmagan `session_id` | umumiy "sessiya topilmadi" xatosi |
| 409 | `session_not_active` | allaqachon yakunlangan sessiyaga `/event` yoki `/finish` yuborilsa | xato sifatida ko'rsatmang — sessiya natijasini (agar hali ko'rsatilmagan bo'lsa `GET /mobile/simulation/{id}` yoki `GET /mobile/debrief/{id}` orqali) ko'rsating |

### Ball hisoblash o'zgardi (FE'da kod o'zgartirish shart emas, faqat bilib qo'ying)

`final_score` endi sof `health_percent` emas — sessiya tugagach AI debrief
hisobotidagi har bir `incorrect_steps` elementi uchun 10% jarima ayiriladi
(masalan health_percent=80%, 2 ta klinik xato topilsa → final_score=60%).
XP/coin shu `final_score`'ga mutanosib hisoblanadi. `GET /mobile/debrief/{id}`
javobidagi `final_score`/`xp_earned`/`coins_earned` — bularning barchasi shu
yangi formula bilan kelgan holda mos keladi, alohida hisoblash frontendda
kerak emas. Bazaviy XP/coin qiymatlari (level va qiyinlik bo'yicha) admin
panelda sozlanadi — batafsil [`CHANGES_FRONTEND_ADMIN.md`](CHANGES_FRONTEND_ADMIN.md).

---

## 3. Google Sign-In — haqiqiy tugma (`c4e8b4e`)

Google login oqimi endi id_token'ni qo'lda paste qilish emas, Google
Identity Services (GIS) tugmasi orqali. Bu allaqachon alohida hujjatlangan —
to'liq integratsiya qo'llanmasi: [`GOOGLE_AUTH_FRONTEND.md`](GOOGLE_AUTH_FRONTEND.md).
Backend endpoint (`POST /mobile/auth/google`) o'zgarmagan, faqat frontend
tomon (GIS tugmasi ulash) kerak.

---

## 4. Push bildirishnomalar — per-user topic, study-plan eslatma (`c4e8b4e`)

- **Endpoint o'zgarishi yo'q.** `POST /mobile/user/device` (device
  register) va `PUT /mobile/study-plan` allaqachon mavjud edi.
- **Topic obunasini frontend o'zi qiladi** (FCM SDK `subscribeToTopic`), backend
  obuna qilmaydi. Login/FCM token olingandan keyin quyidagilarga obuna bo'ling
  (`<lang>` = `uz` | `ru` | `en`, foydalanuvchi tili):
  - `all_<lang>` — hammaga broadcast;
  - `user_<userId>_<lang>` — shaxsiy push (to'lov, study-plan eslatma va h.k.).
  Til o'zgarganda eski til topic'laridan `unsubscribeFromTopic` qilib, yangisiga
  obuna bo'ling; logout'da ikkalasidan ham chiqing. `POST /mobile/user/device`
  faqat tokenni saqlaydi.
- Study-plan eslatma cron endi haqiqatan ishlaydi (avval TODO edi): foydalanuvchi
  `PUT /mobile/study-plan`'da belgilagan `remind_time`/`remind_minutes_before`
  vaqtida push (`EventStudyReminder`) yetib boradi. FE'da alohida ish kerak
  emas, faqat shuni bilib, bu eslatma push'ini UI'da to'g'ri ko'rsating (agar
  hali handle qilinmagan bo'lsa).

---

## 5. Boshqa kichik tuzatishlar (FE ta'siri yo'q/minimal)

- `689ae7e` — Telegram bot menu tugmalari Mini App (`web_app`) sifatida
  ochiladi (avval tashqi brauzer ochardi). Agar web-app Telegram Mini App
  sifatida ham ishlatilsa, sahifa `window.Telegram.WebApp` muhitida ochilishini
  hisobga oling (masalan `WebApp.ready()`/`expand()` chaqirish tavsiya
  etiladi) — backend tomondan qo'shimcha o'zgarish yo'q.
- `c749ccc`, `43e5a1b`, `096f7e9` (dastlabki brauzer-asosli STT/TTS
  tuzatishlari) — **`63f432c` bilan butunlay eskirgan**, chunki brauzer
  speech API endi ishlatilmaydi. Frontendda hali eski `SpeechRecognition`/
  `speechSynthesis` kodi bo'lsa, 1-bo'limdagi yangi oqim bilan almashtiring.

---

## 6. Case urinishlar limiti (`max_attempts`)

- `GET /mobile/case`, `/mobile/case/{id}`, random va sevimlilar: har case'da `max_attempts` (default 10).
- `POST /mobile/simulation/start` javobida `max_attempts`; `GET /mobile/simulation/{id}`, `ongoing` va `completed` da
  `max_attempts` + `attempts_used` (davom ettirilganda hisoblagichni tiklash uchun).
- `POST /mobile/simulation/{id}/event` javobida `max_attempts`, `attempts_used`, `attempts_left`. Faqat `question`
  turidagi xabarlar (matn va ovozli) hisoblanadi, ko'rik/tahlil/dori hisoblanmaydi.
- `attempts_left = 0` bo'lsa backend sessiyani o'zi yakunlaydi: `session_ended: true` + `finish_result`
  (`failed`, tanga berilmaydi). To'g'ri javob topilsa limitdan oldin ham `session_ended: true` bo'ladi.
- FE: chat yonida "Qolgan urinishlar: N / M" ko'rsatish, `session_ended` kelsa kiritishni yopish (qarang: `user-panel.html`).

## Tekshirish ro'yxati

- [ ] Mikrofon: `MediaRecorder` orqali audio yozib, `audio_base64`+`audio_mime`
      bilan `/event`'ga yuborish (eski `SpeechRecognition` kodini olib tashlash)
- [ ] Har bir javobda `reply_audio_base64` bo'lsa ijro etish, bo'lmasa
      ixtiyoriy browser-TTS fallback
- [ ] Har bir ovozli xabarda replay tugmasi (audio saqlab qolish)
- [ ] `session_ended`/`finish_result`'ni `/event` javobida tekshirish, auto-finish
      bo'lganda inputni to'xtatish va qo'lda `/finish` chaqirmaslik
- [ ] `409 session_not_active` javobini xato emas, "sessiya allaqachon
      yakunlangan" holati sifatida ushlash
- [ ] Login/FCM token olingandan keyin `POST /mobile/user/device` chaqirilishini
      tasdiqlash (push va study-plan eslatmasi ishlashi uchun shart)
- [ ] Google Sign-In tugmasi — `GOOGLE_AUTH_FRONTEND.md` bo'yicha
