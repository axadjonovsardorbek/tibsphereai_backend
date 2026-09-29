# Google Sign-In — frontend integratsiya qo'llanmasi

Bu hujjat TibSphereAI backend bilan Google orqali kirishni frontendda (web) ulash
uchun. Backend tomon tayyor va o'zgartirilmaydi — faqat frontend qismini shu
bo'yicha yozasiz. Ishlaydigan referens implementatsiya: [`user-panel.html`](user-panel.html)
(`googlePane`, `initGoogleSignIn`, `onGoogleCredential` funksiyalari) — savol
tug'ilsa o'sha faylga qarang, u aynan shu oqim bo'yicha ishlaydi.

## 1. Nima kerak (Google Cloud tomonda)

1. https://console.cloud.google.com/apis/credentials ga kiring (loyihaning
   Google Cloud project'i ostida).
2. **OAuth 2.0 Client ID** yarating (agar hali yo'q bo'lsa) — turi **Web application**.
3. **Authorized JavaScript origins** ga frontend qaysi domen(lar)dan ochilsa,
   o'shani qo'shing. Masalan:
   - `https://tibcaseweb.vercel.app`
   - `http://localhost:5173` (local dev uchun)
   - `https://dev-medic.axadjonovsardorbek.uz` (agar shu domendan ham test qilsangiz)

   > Muhim: origin ro'yxatiga kiritilmagan domendan ochsangiz, Google
   > **"no registered origin" / "Ошибка 401: invalid_client"** deb bloklaydi.
   > Path/portsiz, faqat `scheme://host[:port]` yoziladi, oxirida `/` bo'lmaydi.

4. **Authorized redirect URIs** — GIS popup/One Tap oqimi uchun shart emas
   (redirect emas, JS orqali ishlaydi), bo'sh qoldirsa ham bo'ladi.
5. Client ID'ni nusxalab oling — ko'rinishi: `NNNNNNNNNNNN-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx.apps.googleusercontent.com`

**Bu Client ID backenddagi `GOOGLE_CLIENT_ID` environment o'zgaruvchisi bilan
bir xil bo'lishi SHART.** Backend `idtoken.Validate(ctx, id_token, GOOGLE_CLIENT_ID)`
orqali tokenning audience (`aud`) claim'ini shu qiymat bilan solishtiradi —
mos kelmasa `401 google_token_invalid` qaytadi. Qaysi Client ID backendda
turganini backend jamoasidan (yoki `.env`dan) so'rab tasdiqlab oling.

## 2. Frontendda oqim (umumiy)

```
[Google Sign-In tugmasi] -> foydalanuvchi akkount tanlaydi
        |
        v
Google Identity Services (GIS) brauzerda id_token (JWT) qaytaradi
        |
        v
POST {API_BASE}/mobile/auth/google   body: { "id_token": "<JWT>" }
        |
        v
Backend: Google bilan tokenni tekshiradi -> user topadi/yaratadi -> access/refresh token qaytaradi
        |
        v
Frontend: access_token/refresh_token'ni saqlaydi, keyingi so'rovlarga Authorization: Bearer qo'shadi
```

Muhim: **frontend hech qachon Google Client Secret ishlatmaydi** — bu faqat
server-to-server OAuth uchun kerak, GIS (One Tap / tugma) oqimida secret
umuman kerak emas, faqat public Client ID.

## 3. GIS skriptini ulash

`<head>` ichiga (yoki komponent mount bo'lishidan oldin):

```html
<script src="https://accounts.google.com/gsi/client" async defer></script>
```

React/Vue/Next kabi SPA'da bu skriptni `index.html`ga qo'yish yoki
`next/script` (`strategy="afterInteractive"`) bilan yuklash mumkin.

## 4. Tugmani render qilish

```html
<div id="googleBtn"></div>
```

```js
function initGoogleSignIn() {
  if (!window.google?.accounts?.id) {
    // skript hali yuklanmagan bo'lishi mumkin — biroz kutib qayta urinish
    setTimeout(initGoogleSignIn, 250);
    return;
  }

  google.accounts.id.initialize({
    client_id: 'YOUR_GOOGLE_CLIENT_ID.apps.googleusercontent.com',
    callback: onGoogleCredential, // pastda
  });

  google.accounts.id.renderButton(document.getElementById('googleBtn'), {
    theme: 'filled_black', // yoki 'outline'
    size: 'large',
    width: 280,
    text: 'signin_with',
  });

  // Ixtiyoriy: One Tap oynasini ham ko'rsatish (sahifa ochilganda avtomatik taklif)
  // google.accounts.id.prompt();
}

initGoogleSignIn();
```

## 5. id_token'ni backendga yuborish

```js
async function onGoogleCredential(response) {
  // response.credential - Google bergan id_token (JWT)
  const res = await fetch(`${API_BASE}/mobile/auth/google`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      id_token: response.credential,
      // referral_code: 'ABC123', // ixtiyoriy - referal orqali ro'yxatdan o'tsa
    }),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    // err.message (yoki .error) - errs.Bad formatidagi javob, pastdagi jadvalga qarang
    console.error('Google login failed:', err);
    return;
  }

  const data = await res.json();
  // data: { access_token, refresh_token, id, role }
  saveSession(data);
}
```

### So'rov/javob shakli

**Request** — `POST /mobile/auth/google`
```json
{
  "id_token": "eyJhbGciOi...",
  "referral_code": ""
}
```
`referral_code` ixtiyoriy, bo'sh string yuborsa ham bo'ladi/umuman yubormasa ham bo'ladi.

**Response — 200 OK**
```json
{
  "access_token": "...",
  "refresh_token": "...",
  "id": "user-uuid",
  "role": "user"
}
```

**Xatolar**
| HTTP | Sabab | Frontendda ko'rsatiladigan matn (backend qaytaradi) |
|---|---|---|
| 401 | `id_token` yaroqsiz/muddati tugagan/audience mos emas | "Google token yaroqsiz" |
| 400 | so'rov body noto'g'ri (masalan `id_token` yo'q) | validatsiya xabari |

## 6. Session saqlash va keyingi so'rovlar

```js
function saveSession(tokenRes) {
  localStorage.setItem('access_token', tokenRes.access_token);
  localStorage.setItem('refresh_token', tokenRes.refresh_token);
  localStorage.setItem('user_id', tokenRes.id);
}

// Himoyalangan so'rovlarda:
fetch(`${API_BASE}/mobile/...`, {
  headers: { Authorization: `Bearer ${localStorage.getItem('access_token')}` },
});
```

`access_token` muddati tugaganda (`401` kelganda) yangilash:

```js
async function refreshToken() {
  const res = await fetch(`${API_BASE}/auth/token/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: localStorage.getItem('refresh_token') }),
  });
  if (!res.ok) { /* logout qiling */ return null; }
  const data = await res.json();
  saveSession(data);
  return data.access_token;
}
```

> Diqqat: refresh endpoint `/mobile/...` ostida emas, to'g'ridan-to'g'ri
> `/auth/token/refresh` (root darajada).

## 7. Web push bilan bog'liqlik (agar kerak bo'lsa)

Google orqali kirgandan keyin, agar ilova web push bildirishnomalarini
ishlatsa, FCM tokenni olib `/mobile/user-device/register`ga `platform:"web"`
bilan yuborish kerak — bu alohida mavzu, google auth bilan bevosita bog'liq
emas, faqat login muvaffaqiyatli bo'lgandan keyingi qadam sifatida qo'shiladi.

## 8. Troubleshooting

- **"Доступ заблокирован: ошибка авторизации" / `no registered origin` / `invalid_client`**
  → Client ID'ning Google Cloud Console'dagi Authorized JavaScript origins
  ro'yxatida joriy domen yo'q. §1-bandga qarang.
- **Backend `401 google_token_invalid` qaytaradi, lekin Google popup xatosiz o'tdi**
  → Frontendda ishlatilgan Client ID bilan backend `GOOGLE_CLIENT_ID` env
  qiymati **bir xil emas**. Ikkalasini solishtirib tekshiring.
- **Tugma umuman chiqmaydi (bo'sh joy)**
  → `accounts.google.com/gsi/client` skripti yuklanmagan (tarmoq/AdBlock) yoki
  `initGoogleSignIn()` skript yuklanishidan oldin chaqirilgan — `setTimeout`
  bilan qayta urinish yoki `window.onload`da chaqiring.
- **`file://` orqali local test qilyapman, ishlamayapti**
  → GIS `file://` originni qo'llamaydi. Local HTTP server orqali oching
  (masalan `npx serve` yoki `python3 -m http.server`) va shu originni ham
  Authorized origins'ga qo'shing.
