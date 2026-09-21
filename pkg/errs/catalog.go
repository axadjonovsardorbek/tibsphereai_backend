package errs

import "net/http"

// Umumiy (har qanday modulda qayta ishlatiladigan) xatolar.
var (
	ErrValidation = New(http.StatusBadRequest, "validation_error", Text{
		Uz: "Noto'g'ri so'rov ma'lumotlari",
		Ru: "Некорректные данные запроса",
		En: "Invalid request data",
	})
	ErrUnauthorized = New(http.StatusUnauthorized, "unauthorized", Text{
		Uz: "Avtorizatsiyadan o'tilmagan",
		Ru: "Не авторизован",
		En: "Unauthorized",
	})
	ErrForbidden = New(http.StatusForbidden, "forbidden", Text{
		Uz: "Bu amalga ruxsat yo'q",
		Ru: "Доступ запрещён",
		En: "Forbidden",
	})
	ErrNotFound = New(http.StatusNotFound, "not_found", Text{
		Uz: "Ma'lumot topilmadi",
		Ru: "Данные не найдены",
		En: "Not found",
	})
	ErrConflict = New(http.StatusConflict, "conflict", Text{
		Uz: "Bunday ma'lumot allaqachon mavjud",
		Ru: "Такие данные уже существуют",
		En: "Already exists",
	})
	ErrInternal = New(http.StatusInternalServerError, "internal_error", Text{
		Uz: "Server xatoligi, birozdan so'ng qayta urinib ko'ring",
		Ru: "Ошибка сервера, попробуйте позже",
		En: "Internal server error, try again later",
	})
)

// ============ Auth (BACKEND_STRUCTURE.md 1-bo'lim) ============

var (
	ErrInvalidCredentials = New(http.StatusBadRequest, "invalid_credentials", Text{
		Uz: "Login yoki parol noto'g'ri",
		Ru: "Неверный логин или пароль",
		En: "Invalid login or password",
	})
	ErrUserExists = New(http.StatusConflict, "user_already_exists", Text{
		Uz: "Bu telefon/email bilan foydalanuvchi allaqachon ro'yxatdan o'tgan",
		Ru: "Пользователь с этим телефоном/email уже зарегистрирован",
		En: "User with this phone/email already exists",
	})
	ErrUserNotFound = New(http.StatusNotFound, "user_not_found", Text{
		Uz: "Foydalanuvchi topilmadi",
		Ru: "Пользователь не найден",
		En: "User not found",
	})
	ErrInvalidOTP = New(http.StatusBadRequest, "invalid_otp", Text{
		Uz: "Tasdiqlash kodi noto'g'ri",
		Ru: "Неверный код подтверждения",
		En: "Invalid confirmation code",
	})
	ErrOTPExpired = New(http.StatusBadRequest, "otp_expired", Text{
		Uz: "Tasdiqlash kodining muddati tugagan, qaytadan so'rang",
		Ru: "Срок действия кода истёк, запросите новый",
		En: "Confirmation code expired, request a new one",
	})
	ErrGoogleTokenInvalid = New(http.StatusUnauthorized, "google_token_invalid", Text{
		Uz: "Google token yaroqsiz",
		Ru: "Недействительный токен Google",
		En: "Invalid Google token",
	})
)

// ============ Klinik keyslar & Sevimlilar (2, 3-bo'lim) ============

var (
	ErrCaseNotFound = New(http.StatusNotFound, "case_not_found", Text{
		Uz: "Klinik keys topilmadi",
		Ru: "Клинический кейс не найден",
		En: "Clinical case not found",
	})
	ErrCaseNotPublished = New(http.StatusForbidden, "case_not_published", Text{
		Uz: "Bu keys hali e'lon qilinmagan",
		Ru: "Этот кейс ещё не опубликован",
		En: "This case is not published yet",
	})
	ErrAlreadyFavorited = New(http.StatusConflict, "already_favorited", Text{
		Uz: "Keys sevimlilarga allaqachon qo'shilgan",
		Ru: "Кейс уже добавлен в избранное",
		En: "Case already in favorites",
	})
)

// ============ Simulyatsiya dvigateli (4-bo'lim) ============

var (
	ErrDailyLimitReached = New(http.StatusForbidden, "daily_limit_reached", Text{
		Uz: "Bugungi bepul urinishlar limiti tugadi, obuna bo'ling",
		Ru: "Дневной лимит бесплатных попыток исчерпан, оформите подписку",
		En: "Daily free attempts limit reached, subscribe to continue",
	})
	ErrNoActiveSubscription = New(http.StatusForbidden, "no_active_subscription", Text{
		Uz: "Bu amal uchun faol obuna kerak",
		Ru: "Для этого действия нужна активная подписка",
		En: "An active subscription is required for this action",
	})
	ErrSessionNotFound = New(http.StatusNotFound, "session_not_found", Text{
		Uz: "Simulyatsiya sessiyasi topilmadi",
		Ru: "Сессия симуляции не найдена",
		En: "Simulation session not found",
	})
	ErrSessionNotActive = New(http.StatusConflict, "session_not_active", Text{
		Uz: "Simulyatsiya sessiyasi allaqachon yakunlangan",
		Ru: "Сессия симуляции уже завершена",
		En: "Simulation session already finished",
	})
)

// ============ Obuna, Promokod, To'lov (5-bo'lim) ============

var (
	ErrTariffNotFound = New(http.StatusNotFound, "tariff_not_found", Text{
		Uz: "Tarif topilmadi",
		Ru: "Тариф не найден",
		En: "Tariff not found",
	})
	ErrPromoCodeInvalid = New(http.StatusBadRequest, "promo_code_invalid", Text{
		Uz: "Promokod noto'g'ri yoki mavjud emas",
		Ru: "Промокод неверный или не существует",
		En: "Invalid or nonexistent promo code",
	})
	ErrPromoCodeUsed = New(http.StatusConflict, "promo_code_used", Text{
		Uz: "Promokod allaqachon ishlatilgan",
		Ru: "Промокод уже использован",
		En: "Promo code already used",
	})
	ErrPromoCodeExpired = New(http.StatusBadRequest, "promo_code_expired", Text{
		Uz: "Promokodning muddati tugagan",
		Ru: "Срок действия промокода истёк",
		En: "Promo code expired",
	})
	ErrOrderNotFound = New(http.StatusNotFound, "order_not_found", Text{
		Uz: "Buyurtma topilmadi",
		Ru: "Заказ не найден",
		En: "Order not found",
	})
	ErrPaymentFailed = New(http.StatusBadRequest, "payment_failed", Text{
		Uz: "To'lovni amalga oshirib bo'lmadi",
		Ru: "Не удалось выполнить оплату",
		En: "Payment could not be completed",
	})
	ErrInsufficientCoins = New(http.StatusBadRequest, "insufficient_coins", Text{
		Uz: "Tanga balansi yetarli emas",
		Ru: "Недостаточно монет на балансе",
		En: "Insufficient coin balance",
	})
)
