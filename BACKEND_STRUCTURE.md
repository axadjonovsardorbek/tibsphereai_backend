# 📋 MedicAI — Backend Funksiyalar Ro'yxati (Functional Specification)

**Backend dasturchi uchun aniq modullar va funksiyalar ro'yxatidir**.

---

## 📱 Asosiy Mobil Ilova Bo'limlari (Bottom Navigation)

Ilova pastki menyusida quyidagi **4 ta asosiy bo'lim** mavjud:
1. **Home (Bosh sahifa):** Foydalanuvchi mini-statusi (XP, Level, Streak), Kategoriyalar (Kardiologiya, Terapiya va h.k.), Tavsiya etilgan va tasodifiy keyslar.

2. **Simulation (Simulyatsiya):** Barcha klinik keyslar katalogi (In Progress, Oson/O'rta/Murakkab), qidiruv va simulyatsiyani boshlash.
3. **Favorites (Sevimlilar):** Foydalanuvchi o'zi saqlab qo'ygan (yurakcha bosilgan) keyslar ro'yxati.
4. **Profile (Profil & Sozlamalar):** Foydalanuvchi ma'lumotlari, Level/XP, Tangalar, Keyslar tarixi (Completed/Ongoing), Obuna, **Mening Promokodlarim** va tozalangan Sozlamalar.


---

## 1. Foydalanuvchilar va Autentifikatsiya (Auth & Profile)

- [ ] **Ro'yxatdan o'tish va Kirish:**
  - Telefon raqam yoki Email + parol orqali ro'yxatdan o'tish.
  - SMS / Telegram OTP orqali tasdiqlash.
  - Google OAuth orqali 1 bosqichli kirish.
  - JWT Access va Refresh tokenlar mexanizmi.
  - Tizimdan chiqish (Logout) va hisobni butunlay o'chirish (Delete Account).
- [ ] **Parol boshqaruvi:**
  - **Parolni almashtirish:** Eski parolni kiritib, yangi parol o'rnatish.
  - **Parolni unutdingizmi? (Forgot Password):** Telefon raqam orqali SMS yoki Email orqali tasdiqlash kodi yuboriladi, kod to'g'ri kiritilgach yangi parol o'rnatish imkoniyati beriladi.
- [ ] **Foydalanuvchi Profili:**
  - F.I.Sh / Username o'zgartirish (`Change Username`).
  - Tibbiy ixtisoslik va daraja (Talaba, Rezident, Shifokor).
  - Avatar / rasm yuklash.
- [ ] **Profil Statistikasi va Daraja (Level & XP):**
  - Foydalanuvchi darajasi (`Level 1`, `Level 2` va `XP bar`: masalan, 350/1000 XP).
  - Streak hisoblagichi (kunlik ketma-ket kirishlar soni).
  - Tangalar (Coins) balansi.
  - Yechilgan to'g'ri keyslar soni (`Correct`).
- [ ] **Keyslar Tarixi (Profile Tabs):**
  - **Completed:** Muvaffaqiyatli yakunlangan keyslar ro'yxati va olingan ballar.
  - **Ongoing (In Progress):** Hozirda chala qolgan / davom etayotgan keyslar.
- [ ] **Sozlamalar (Settings):**
  - Tilni tanlash (`Language Selection`: O'zbekcha, Ruscha, Inglizcha).
  - Do'stlar bilan ulashish (`Share with Friends`).
  - Ilovani baholash (`Rate Us`).
  - Foydalanish shartlari va Maxfiylik siyosati (`Terms of Use`, `Privacy Policy`).

---

## 2. Bosh Sahifa va Kategoriyalar (Home API)

- [ ] **Home ma'lumotlari (Dashboard Feed):**
  - Yuqori promo-banner (Chegirmali obuna taklifi va taymer).
  - Foydalanuvchi mini-profil kartochkasi (Ism, unvon, Level, XP, Streak, Obuna statusi).
  - Agar foydalanuvchi obuna sotib olmagan bo'lsa, qolgan bepul urinishlar soni ko'rsatiladi (masalan: "3 ta bepul keys qoldi").
- [ ] **Kategoriyalar (Categories):**
  - Tibbiy yo'nalishlar ro'yxati (Kardiologiya, Terapiya, Shoshilinch yordam, Nevrologiya, Pediatriya, Travmatologiya va h.k.).
  - Kategoriya bo'yicha tegishli keyslarni filtrlash.
- [ ] **O'quv Rejasi va Bildirishnomalar (Study Plan & Push Notifications):**
  - Foydalanuvchi ilovaga kirish uchun kunlik vaqtni o'zi belgilaydi (masalan: har kuni soat 20:00).
  - Necha minut avval eslatish kerakligini ham o'zi tanlaydi (masalan: 15 daqiqa oldin).
  - Belgilangan vaqtda tizim foydalanuvchiga **push bildirishnoma** yuboradi (masalan: "Bugun ham bir keys yechib, bilimingizni sinab ko'ring!").
  - Backend: rejalashtirilgan vaqt va eslatish daqiqasi DBda saqlanadi; FCM / APNs orqali bildirishnoma yuboriladi.

---

## 3. Klinik Keyslar va Sevimlilar (Cases & Favorites)

- [ ] **Keyslar katalogi:**
  - Barcha keyslar ro'yxati (nomi, muqova rasmi, qiyinchilik darajasi: Easy / Medium / Hard).
  - Filterlar: Kategoriya, Qiyinchilik, Holati (Yangi, Jarayonda, Yakunlangan).
  - Qidiruv tizimi.
- [ ] **Tasodifiy keys (Random Case):**
  - Foydalanuvchi darajasiga mos tasodifiy keys tavsiya qilish ("Try your luck!" — bitta "Random" tugmasi bilan ishlaydi).
- [ ] **Keys tafsilotlari (Case Detail):**
  - Bemorning dastlabki shikoyati, yoshi, jinsi, kutilayotgan davomiylik va talablar.
- [ ] **Sevimlilar (Favorites) tizimi:**
  - Keysni sevimlilarga qo'shish / olib tashlash (`Toggle Favorite`).
  - Sevimlilar ro'yxatini olish (`GET /api/v1/favorites`).
- [ ] **Kirish huquqini tekshirish:**
  - Kunlik bepul limit borligini yoki Premium obuna faolligini tekshirish.

---

## 4. Simulyatsiya Dvigateli (Simulation Engine — Core)

- [ ] **Sessiyani boshlash (Start Simulation):**
  - Bepul limitdan 1 ta urinish ayirish (Premium foydalanuvchilarda cheksiz).
  - Bemorning dastlabki holatini yuklash:
    - Vital parametrlar: Puls (HR), Qon bosimi (BP), Kislorod (SpO2), Nafas (RR), Harorat (Temp), Glazgo (GCS).
    - Vizual holat (hushida, og'riqda, hushsiz, tirishish, koma).
    - Sog'liq shkalasi (`Health Bar`: 100%).
    - Shoshilinch taymer (masalan: 10 daqiqa).
- [ ] **Bemor bilan interaktiv muloqot (Anamnez):**
  - Shifokor savol beradi -> Gemini AI bemor personajida tabiiy va klinik to'g'ri javob qaytaradi.
- [ ] **Fizikal ko'rik amallari:**
  - Auskultatsiya (yurak, o'pka tovushlari), palpatsiya, perkussiya, qorachiqlar reaksiyasi.
- [ ] **Diagnostika va tahlillar:**
  - Laboratoriya tahlillari (Umumiy qon, biokimyo, qon gazi, troponin va h.k.).
  - Instrumental tekshiruvlar (EKG, Rentgen, UTT, KT).
- [ ] **Muolaja va Dori-darmonlar:**
  - Dori nomi + doza + yuborish yo'li (IV, IM, Oral).
  - Shoshilinch muolajalar (defibrillatsiya, kislorod niqobi, intubatsiya, infuziya).
- [ ] **Holat dinamikasi va Sog'liq shkalasi:**
  - To'g'ri harakat -> Sog'liq oshadi, vitallar normallashadi.
  - Noto'g'ri/kechiktirilgan harakat -> Sog'liq tushadi, bemor holati og'irlashadi.
  - Vaqt o'tishi bilan passivlik jazosi (Inaction penalty).
- [ ] **Real-vaqt monitori (WebSocket / SSE):**
  - Vital parametrlar oqimi (har soniyada yangilanuvchi ma'lumotlar).
  - Monitor tovush signallari (Puls urishi, Sirena/Alarm, Yurak to'xtashi - Flatline).
- [ ] **Sessiyani yakunlash:**
  - Foydalanuvchi "Tugatish" tugmasini bosganda, sog'liq 0 ga tushganda yoki vaqt tugaganda sessiya yopiladi va barcha qadamlar logi saqlanadi.

---

## 5. Debriefing, Obuna va Hamkor Kurslar Promokodi

- [ ] **AI orqali Debriefing hisoboti:**
  - Qilingan to'g'ri va noto'g'ri qadamlar ketma-ketligi jadvali.
  - Yakuniy ball (0-100%) va qo'shilgan XP/Tangalar.
  - Xalqaro klinik qo'llanmalar (AHA, ERC va b.) bo'yicha tahliliy izoh.
  - Bilim bo'shliqlari / Zaif mavzularni aniqlash (masalan: "EKG - Miokard infarkti").
- [ ] **Hamkor kurslar taklifi va Obuna savdosi (Funnel):**
  - Keys yakunlanganda xatolar tahlilidan kelib chiqib taklif chiqadi:
    > *"Bizning 1 oylik obunamizni sotib oling va 'Hamkor' kursiga 15% lik chegirmani qo'lga kiriting!"*
  - Foydalanuvchi "Obunani faollashtirish va Chegirmani olish" tugmasini bosadi.
- [ ] **To'lov tizimi integratsiyasi (Payment):**
  - Payme va Click orqali 1 oylik Premium obunaga to'lov qilish.
  - To'lov tasdiqlangach (Webhook):
    1. Foydalanuvchiga 1 oylik Premium obuna beriladi.
    2. Tizim avtomatik ravishda **1 martalik unikal promokod** yaratadi.
    3. Foydalanuvchiga xabar ko'rsatiladi:
       > *"Bu bir martalik promokod, uni hech kimga bermang. 'Hamkor' adminiga shu promokodni berasiz yoki hamkor platformasida kiritasiz va 15% chegirmani qo'lga kiritasiz. Promokodlaringizni profil qismidagi Promokodlar bo'limida kuzatishingiz mumkin."*
- [ ] **Mening Promokodlarim (Profile -> Promocodes):**
  - Foydalanuvchi profilida mavjud barcha promokodlar ro'yxati.
  - Maydonlar: Promokod matni, Hamkor nomi, Chegirma foizi, Berilgan sana, Amal qilish muddati.
  - Statuslar: `Faol` (Active), `Ishlatildi` (Used), `Muddati o'tdi` (Expired).
  - Promokoddan nusxa olish (Copy) imkoniyati.

---

## 6. Limitlar va Tangachalar (Gamification & Limits)

- [ ] **Kunlik bepul keyslar limiti:**
  - Bepul foydalanuvchilarga kunlik urinishlar limiti (masalan: kuniga 3 ta).
  - Har kecha soat 00:00 da limit avtomatik yangilanadi.
  - **Asosiy qoida:** Keys muvaffaqiyatsiz yechilsa, tanga kamaytirilmaydi — faqat bepul urinishlar soni kamayadi.
  - Premium obunachilar uchun keys yechish limiti cheksiz.
- [ ] **Tangalar (Coins) hisoblash:**
  - Keys muvaffaqiyatli topshirilganda yakuniy ballga mutanosib tangalar taqdim etiladi.
  - Suiiste'mollikni (farming) oldini olish: qayta yechilgan keys uchun qayta to'liq tanga berilmaydi.
- [ ] **Tangalar balansi va tarixi (Audit Log):**
  - Tangalar kirim-chiqim operatsiyalari auditi.

---

## 7. Admin Paneli (Admin Console)

- [ ] **Dashboard va Analitika:**
  - Foydalanuvchilar soni (DAU/MAU), faol obunalar.
  - Yechilgan keyslar statistikasi va eng ko'p xato qilinayotgan mavzular.
  - Gemini AI xarajatlari va sarflangan tokenlar.
- [ ] **Tizim sozlamalari (Limits Config):**
  - **Bepul foydalanish sonini admin panel orqali o'zgartira olish** (masalan, kunlik bepul limitni 3 tadan 5 taga yoki istalgan songa dinamik o'zgartirish).
- [ ] **Foydalanuvchilar boshqaruvi:**
  - Qidiruv, filtrlash, bloklash/faollashtirish.
  - Foydalanuvchi balansi va obunasini boshqarish.
  - Har bir foydalanuvchining promokodlari va simulyatsiya tarixini ko'rish.
- [ ] **Klinik Keyslar boshqaruvi (CMS):**
  - Keyslarni qo'lda qo'shish, tahrirlash, o'chirish.
  - Bemor shikoyatlari, dastlabki vitallar, laboratoriya natijalari va to'g'ri davolash algoritmini kiritish.
  - **AI yordamida avtomatik keys yaratish:** Admin mavzuni va darajani kiritadi -> Gemini AI to'liq ssenariyni bir zumda tayyorlaydi -> Admin ko'rib chiqib, bitta tugma bilan tasdiqlaydi (Publish).
- [ ] **AI Promptlari va Model parametrlari:**
  - Bemor personaji va debriefing promptlarini kodni o'zgartirmasdan admin paneldan sozlash.
- [ ] **Hamkorlar va Promokodlar boshqaruvi:**
  - Hamkor o'quv markazlari ro'yxati va ularga tegishli chegirma foizlarini kiritish.
  - Generatsiya qilingan promokodlar ro'yxati va ularning holati (ishlatilgan/ishlatilmagan).
  - Admin tomonidan promokodni "Ishlatildi" statusiga o'tkazish imkoniyati.
  
  -moliyaviy hisobot:
  qo'shimcha click payme orqali to'lovlar, obunalar statistikasi va hokazo
