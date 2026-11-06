# Admin statistikasi — 1-bosqich (frontend uchun)

TZ_STATISTICS 1-bosqichi: 1 (moliya), 2 (AI xarajati), 3 (premium), 5 (faollik), 9 (foydalanuvchilar). Aniq javob tiplari swagger'da (`stats` tegi).

## Umumiy

- Hammasi admin uchun, `Authorization: Bearer <admin token>`.
- Davr: `?from=YYYY-MM-DD&to=YYYY-MM-DD`. Ikkala sana ham kiradi, kun chegarasi Toshkent vaqti bo'yicha.
- Davr berilmasa oxirgi 30 kun olinadi. Eng ko'pi 366 kun.
- Sana noto'g'ri bo'lsa 400 qaytadi.
- Natija 60 soniya keshlanadi.

## Endpointlar

| Endpoint | Nima qaytaradi |
|---|---|
| `GET /web/stats/finance` | `FinanceStatsRes`: tushum, buyurtmalar, AI xarajati (USD/so'm), foyda, ARPU/ARPPU, `daily[]`, `by_tariff[]` |
| `GET /web/stats/ai-cost` | `AICostStatsRes`: jami, `by_feature[]`, `per_action[]` (case_session, test_session, simulation, flashcard_deck, roadmap), `by_tier[]` (free, coin, premium, system), `cost_per_coin_usd` |
| `GET /web/stats/premium-cost` | `PremiumCostStatsRes`: premiumlar soni, o'rtacha AI xarajati, zarar keltirayotganlar, `top_spenders[20]` |
| `GET /web/stats/engagement` | `EngagementStatsRes`: DAU/WAU/MAU (bugungi kunga nisbatan), `daily[]`, `funnel` |
| `GET /web/user/{id}/stats` | `UserStatsRes`: oxirgi 30 kun grafigi, top 5 bo'lim, oxirgi 10 tanga yozuvi va 10 buyurtma |

Atamalar:

- **Tushum**: pul bilan to'langan (`PAID`, `amount > 0`) buyurtmalar. Butunlay tanga bilan yopilgan buyurtma tushumga kirmaydi.
- **Faol foydalanuvchi**: davrda sessiya boshlagan, practice savoliga javob bergan yoki fleshkarta takrorlagan. Faqat login qilish faollik hisoblanmaydi.
- **ARPU**: tushum ÷ davrdagi faol foydalanuvchilar. **ARPPU**: tushum ÷ davrda to'lov qilganlar.
- **Zarar keltirayotgan premium**: umr bo'yi AI xarajati (so'mda) umr bo'yi to'lagan pulidan ko'p bo'lgan premium foydalanuvchi.
- **Voronka**: davrda ro'yxatdan o'tganlar bo'yicha. "Ertasi kuni qaytdi" — ro'yxatdan o'tgan kunining ertasiga faol bo'lgan.
- **`cost_per_coin_usd`**: bepul va tanga bilan ishlaydigan foydalanuvchilar AI xarajati ÷ davrda yechilgan tangalar soni. Bu 1 tanganing haqiqiy tannarxi.

## `GET /web/user` — yangi maydonlar

Har bir foydalanuvchida endi `stats` obyekti bor:

```json
"stats": {
  "last_active_at": "2026-11-05 14:02:11+05", "sessions_count": 12,
  "ai_cost_usd": 0.42, "ai_tokens": 120000, "coins_spent": 35,
  "paid_uzs": 50000, "orders_count": 1,
  "has_subscription": true, "subscription_until": "2026-12-04 21:16:43+05",
  "signup_method": "telegram", "platform": "android", "app_version": "1.4.0",
  "feedback_count": 2
}
```

Yangi query parametrlar:

- `email`
- `created_from`, `created_to` (`YYYY-MM-DD`)
- `inactive_days=N`: N kundan beri faol emas, hech qachon faol bo'lmaganlar ham kiradi
- `has_subscription=true|false`
- `paid=true|false`
- `sort_by=created_at|last_active|ai_cost|paid|sessions`

## Mobil ilova va mini app: har bir so'rovga 2 ta header qo'shilsin

```
X-Platform: ios | android | web
X-App-Version: 1.4.0
```

Bu headerlardan foydalanuvchining `platform` va `app_version` maydonlari to'ldiriladi. Headerlar yuborilmasa, bu maydonlar bo'sh qoladi.

## Sozlama

`usd_to_uzs_rate` (default 12800) mavjud `app-setting` orqali o'zgartiriladi.
