package storage

import (
	"context"
	m "tibsphereai/pkg/models"
)

type AdminI interface {
	Login(req *m.LoginReq) (*m.TokenRes, error)
	GetById(req *m.ById) (*m.AdminRes, error)
	Create(req *m.AdminCreateReq) (*m.ById, error)
	GetAll(req *m.AdminGetListReq) (*m.PaginationResponse, *m.AdminList, error)
	Update(req *m.AdminUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
}

// RoleI - admin panel rollari (superadmin + qo'shimcha). "superadmin" tizim roli -
// migratsiyada seed qilingan, Delete/Update'da himoyalanadi (is_system=true).
type RoleI interface {
	Create(req *m.RoleCreateReq) (*m.ById, error)
	GetById(req *m.ById) (*m.RoleRes, error)
	GetAll(req *m.RoleGetListReq) (*m.PaginationResponse, *m.RoleList, error)
	Update(req *m.RoleUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
}

// CustomPermissionI - admin panel modullari (U-Code custom-permission patterni,
// API_DOCUMENTATION_EN.md). Boshqaruv qatlami - haqiqiy ruxsat tekshiruvi hamon
// Casbin (policy.csv) orqali, bu yerdagi ma'lumot hozircha faqat admin panelda
// ko'rsatish/sozlash uchun.
type CustomPermissionI interface {
	Create(req *m.CustomPermissionCreateReq) (*m.ById, error)
	Delete(req *m.ById) (*m.Void, error)
	Update(req *m.CustomPermissionUpdateReq) (*m.Void, error)
	// GetRoots - parent_id=NULL bo'lgan (root) modullar ro'yxati.
	GetRoots() (*m.CustomPermissionList, error)
	// GetChildren - berilgan parent_id ostidagi modullar.
	GetChildren(parentId string) (*m.CustomPermissionList, error)
	// HasChildren - bitta modulning farzand-modullari bor-yo'qligi (accesses ro'yxatida
	// "ichiga kirish mumkin"ligini ko'rsatish uchun).
	HasChildren(id string) (bool, error)
	// GetAccesses - root yoki (parentId berilsa) child modullar ro'yxatini, shu rol
	// uchun hozirgi ruxsatlar (role_accesses LEFT JOIN, belgilanmagan bo'lsa hammasi false)
	// bilan birga qaytaradi.
	GetAccesses(roleId, parentId string) (*m.RoleAccessList, error)
	// UpsertAccess - bitta rol+modul uchun ruxsatlarni yaratadi yoki yangilaydi.
	UpsertAccess(req *m.RoleAccessUpsertReq) (*m.Void, error)
}

type BannerI interface {
	Create(req *m.BannerCreateReq) (*m.ById, error)
	GetById(req *m.ById) (*m.BannerRes, error)
	GetAll(req *m.BannerGetListReq) (*m.PaginationResponse, *m.BannerList, error)
	Update(req *m.BannerUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
	GetByIdMobile(req *m.ByIdMobile) (*m.BannerMobileRes, error)
	GetAllMobile(req *m.BannerGetListMobileReq) (*m.PaginationResponse, *m.BannerMobileList, error)
}

type ContactI interface {
	Create(req *m.ContactCreateReq) (*m.ById, error)
	GetById(req *m.ById) (*m.ContactRes, error)
	GetAll(req *m.ContactGetListReq) (*m.PaginationResponse, *m.ContactList, error)
	Update(req *m.ContactUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
}

// UserI - mobil foydalanuvchi (parolsiz: faqat OTP email/telegram + Google, TZ 3-bo'lim).
type UserI interface {
	EmailLogin(req *m.ConfirmationReq) (*m.TokenRes, error)
	TelegramLogin(req *m.TelegramLoginReq) (*m.TokenRes, error)
	GoogleLogin(req *m.GoogleAuthReq) (*m.TokenRes, error)
	Check(req *m.UserCheckReq) (*m.UserCheckRes, error)
	GetById(req *m.ById) (*m.UserRes, error)
	GetAll(req *m.UserGetListReq) (*m.PaginationResponse, *m.UserList, error)
	Update(req *m.UserUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
	// ChangePassword/RefreshPassword - faqat role="admin" uchun ishlaydi (adminlar hamon
	// parol bilan kiradi). Mobil user'da parol yo'q.
	ChangePassword(req *m.ChangePasswordReq) (*m.Void, error)
	RefreshPassword(req *m.RefreshPasswordReq) (*m.RefreshPassword, error)
	CreateActivity(req *m.UserActivityCreateReq) (*m.Void, error)
	GetActivities(req *m.UserActivityGetListReq) (*m.ActivityStatsResponse, error)
	GetRating(req *m.RatingRequest) (*m.RatingResponse, error)
	GetTop(req *m.RatingRequest) ([]m.RatingUserItem, error)
	GetMe(req *m.RatingRequest) (*m.RatingUserItem, error)
	// Gamifikatsiya: XP/Coin qo'shish va level'ni levels jadvaliga qarab qayta hisoblash
	AddPoints(req *m.AddPointsReq) (*m.Void, error)
}

// UserDeviceI - ko'p qurilma push token (users.fcm_token o'rnini bosadi). Register/Unregister
// paytida til bo'yicha FCM topic (all_uz/all_ru/all_en) obuna/unobuna qilinadi (service qatlamida).
type UserDeviceI interface {
	Register(req *m.UserDeviceRegisterReq) (*m.Void, error)
	// Unregister - o'chirilgan qurilmaning user_id'sini qaytaradi (topikdan unsubscribe qilish
	// uchun til kerak bo'ladi), topilmasa bo'sh string.
	Unregister(req *m.UserDeviceUnregisterReq) (string, error)
	GetTokensAll(ctx context.Context) ([]string, error)
	GetTokensByUser(ctx context.Context, userId string) ([]string, error)
}

type NotificationI interface {
	Create(req *m.NotificationCreateReq) (*m.ById, error)
	GetById(req *m.ById) (*m.MasterNotificationRes, error)
	GetAll(req *m.NotificationGetListReq) (*m.PaginationResponse, *m.MasterNotificationList, error)
	Update(req *m.MasterNotificationUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
	UserNotifications(req *m.NotificationGetListReq) (*m.PaginationResponse, *m.UserNotificationList, error)
	MarkAsRead(req *m.NotificationMarkReadReq) (*m.Void, error)
}

type FaqI interface {
	Create(req *m.FaqCreateReq) (*m.ById, error)
	GetById(req *m.ById) (*m.FaqRes, error)
	GetAll(req *m.FaqGetListReq) (*m.PaginationResponse, *m.FaqList, error)
	Update(req *m.FaqUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
	GetByIdMobile(req *m.ByIdMobile) (*m.FaqMobileRes, error)
	GetAllMobile(req *m.FaqGetListMobileReq) (*m.PaginationResponse, *m.FaqMobileList, error)
}

type AboutI interface {
	Create(req *m.AboutCreateReq) (*m.ById, error)
	GetById(req *m.ById) (*m.AboutRes, error)
	GetAll(req *m.AboutGetListReq) (*m.PaginationResponse, *m.AboutList, error)
	Update(req *m.AboutUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
	GetByIdMobile(req *m.ByIdMobile) (*m.AboutMobileRes, error)
	GetAllMobile(req *m.AboutGetListMobileReq) (*m.PaginationResponse, *m.AboutMobileList, error)
}

type AppRouteI interface {
	Create(req *m.AppRouteCreateReq) (*m.ById, error)
	GetById(req *m.ById) (*m.AppRouteRes, error)
	GetAll(req *m.AppRouteGetListReq) (*m.PaginationResponse, *m.AppRouteList, error)
	Update(req *m.AppRouteUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
}

type TariffI interface {
	Create(req *m.TariffCreateReq) (*m.ById, error)
	GetById(req *m.ById) (*m.TariffRes, error)
	GetAll(req *m.TariffGetListReq) (*m.PaginationResponse, *m.TariffList, error)
	Update(req *m.TariffUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
}

type OrderI interface {
	Create(req *m.OrderCreateReq) (*m.OrderCreateRes, error)
	GetById(req *m.ById) (*m.OrderStatisticRes, error)
	GetAll(req *m.OrderGetListReq) (*m.PaginationResponse, *m.OrderList, error)

	// Click callbacks uchun
	ApplyPrepare(req *m.ApplyPrepareParams) (*m.PrepareResult, error)
	// ApplyComplete - to'lovni PAID qiladi va faol subscription'ni yaratadi/uzaytiradi.
	// Muvaffaqiyatli bo'lsa PaidUserId/PaidOrderId to'ldiriladi - service qatlami shu orqali
	// push/in-app bildirishnoma yuboradi.
	ApplyComplete(req *m.ApplyCompleteParams) (*m.CompleteResult, error)

	// Payme JSON-RPC callbacklari uchun (docs: furasentr-payment/internal/api/handlers/payme.go).
	// orderId = Payme ac.order_id = bizning orders.id.
	PaymeCheckPerform(orderId string, amountTiyin int64) error
	PaymeCreateTransaction(orderId, paymeId string, amountTiyin, timeMs int64) (*m.PaymeCreateResult, error)
	// PaymePerformTransaction - state 1->2, PAID qiladi va subscription yaratadi. JustPaid=true
	// bo'lsa service qatlami promokod/push logikasini ishga tushiradi (ApplyComplete bilan bir xil).
	PaymePerformTransaction(paymeId string) (*m.PaymeTransactionResult, error)
	PaymeCancelTransaction(paymeId string, reason int) (*m.PaymeTransactionResult, error)
	PaymeCheckTransaction(paymeId string) (*m.PaymeTransactionResult, error)
	PaymeGetStatement(fromMs, toMs int64) ([]m.PaymeStatementItem, error)
}

// PromoCodeI - Hamkor promokodlari (TZ 5-6-bo'lim). Hamkor/admin OLDINDAN tanga
// qiymati bilan yaratadi; foydalanuvchi Redeem orqali kodni ishlatib tanga oladi.
type PromoCodeI interface {
	Create(req *m.PromoCodeCreateReq) (*m.ById, error)
	GetById(req *m.ById) (*m.PromoCodeRes, error)
	GetAll(req *m.PromoCodeGetListReq) (*m.PaginationResponse, *m.PromoCodeList, error)
	GetMy(req *m.PromoCodeGetListReq) (*m.PaginationResponse, *m.PromoCodeList, error)
	// Redeem - kodni "used" qiladi va tanga miqdorini qaytaradi (service shu orqali
	// UserS.AddPoints chaqiradi - tanga hisoblash logikasi bir joyda markazlashgan).
	Redeem(req *m.PromoCodeRedeemReq) (int, error)
	Delete(req *m.ById) (*m.Void, error)
}

type PartnerI interface {
	Create(req *m.PartnerCreateReq) (*m.ById, error)
	GetById(req *m.ById) (*m.PartnerRes, error)
	GetAll(req *m.PartnerGetListReq) (*m.PaginationResponse, *m.PartnerList, error)
	Update(req *m.PartnerUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
	// GetAllMobile - Home ekranidagi "Hamkor kurslari" banneri (TZ 2.1-bo'lim).
	GetAllMobile() (*m.PartnerMobileList, error)
}

type DashBoardI interface {
	GetDashboard(req *m.GetDashboardReq) (*m.DashboardRes, error)
}

// --- Home / Kategoriyalar ---

type CategoryI interface {
	Create(req *m.CategoryCreateReq) (*m.ById, error)
	GetById(req *m.ById) (*m.CategoryRes, error)
	GetAll(req *m.CategoryGetListReq) (*m.PaginationResponse, *m.CategoryList, error)
	Update(req *m.CategoryUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
	GetByIdMobile(req *m.ByIdMobile) (*m.CategoryMobileRes, error)
	GetAllMobile(req *m.CategoryGetListMobileReq) (*m.PaginationResponse, *m.CategoryMobileList, error)
}

// TopicI - Bo'lim (categories) ichidagi Mavzular (TZ 6-bo'lim: Bo'lim -> Mavzu -> Keys).
type TopicI interface {
	Create(req *m.TopicCreateReq) (*m.ById, error)
	GetById(req *m.ById) (*m.TopicRes, error)
	GetAll(req *m.TopicGetListReq) (*m.PaginationResponse, *m.TopicList, error)
	Update(req *m.TopicUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
	GetAllMobile(req *m.TopicGetListMobileReq) (*m.PaginationResponse, *m.TopicMobileList, error)
}

// CategoryUnlockI - Premium obunasiz, tanga evaziga bitta bo'limdan N ta case
// ishlash huquqini sotib olish (BACKEND_STRUCTURE.md: "50 ta case = 50 tanga" pattern).
type CategoryUnlockI interface {
	// Purchase - atomik: userning tanga balansi tekshiriladi (yetarli bo'lmasa
	// errs.ErrInsufficientCoins), yetarli bo'lsa ayiriladi va cases_remaining oshiriladi.
	Purchase(req *m.CategoryUnlockPurchaseReq, coinCost int) (*m.CategoryUnlockRes, error)
	GetRemaining(userId, categoryId string) (int, error)
	// ConsumeOne - simulyatsiya boshlanganda 1 ta case-huquqini sarflaydi
	// (cases_remaining - 1). Yetarli bo'lmasa false qaytaradi (xato emas).
	ConsumeOne(userId, categoryId string) (bool, error)
}

type StudyPlanI interface {
	Upsert(req *m.StudyPlanUpsertReq) (*m.Void, error)
	GetByUserId(userId string) (*m.StudyPlanRes, error)
	GetAllEnabled() ([]m.StudyPlanRes, error)
}

// --- Klinik keyslar & Sevimlilar ---

type CaseI interface {
	Create(req *m.CaseCreateReq) (*m.ById, error)
	GetById(req *m.ById) (*m.CaseRes, error)
	GetAll(req *m.CaseGetListReq) (*m.PaginationResponse, *m.CaseList, error)
	Update(req *m.CaseUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
	Publish(req *m.ById) (*m.Void, error)
	GetByIdMobile(req *m.ByIdMobile) (*m.CaseMobileRes, error)
	GetAllMobile(req *m.CaseGetListMobileReq) (*m.PaginationResponse, *m.CaseMobileList, error)
	GetRandom(req *m.CaseRandomReq) (*m.CaseMobileRes, error)
	// GetCategoryId - case'ning qaysi bo'limga tegishliligi (cases.topic_id -> topics.category_id
	// orqali), SimulationService.Start'da tanga-huquqini tekshirish uchun ishlatiladi.
	GetCategoryId(caseId string) (string, error)
}

type FavoriteI interface {
	Toggle(req *m.FavoriteToggleReq) (*m.FavoriteToggleRes, error)
	GetAll(req *m.FavoriteGetListReq) (*m.PaginationResponse, *m.CaseMobileList, error)
}

// --- Simulyatsiya dvigateli ---

type SimulationI interface {
	Start(req *m.SimulationStartReq) (*m.SimulationStartRes, error)
	// AddEvent - isCorrect/healthDelta service qatlamida hisoblanadi (question
	// turi - PatientAI chat, boshqalari - ActionEvaluatorAI klinik baho,
	// [[internal/ai/ai.go]]); storage faqat berilgan qiymatlarni yozadi.
	AddEvent(req *m.SimulationEventReq, isCorrect bool, healthDelta int) (*m.SimulationEventRes, error)
	Finish(req *m.SimulationFinishReq) (*m.SimulationFinishRes, error)
	GetById(req *m.ById) (*m.SimulationSessionRes, error)
	GetOngoing(req *m.ById) (*m.PaginationResponse, *m.SimulationSessionList, error)
	GetCompleted(req *m.ById) (*m.PaginationResponse, *m.SimulationSessionList, error)
	// GetEvents - debrief generatsiyasi uchun sessiya qadamlari logi.
	GetEvents(sessionId string) ([]m.SimulationEventReq, error)
}

type DebriefI interface {
	GetBySessionId(sessionId string) (*m.DebriefReportRes, error)
	Create(req *m.DebriefReportCreateReq) (*m.Void, error)
}

// --- Gamifikatsiya ---

type LevelI interface {
	Create(req *m.LevelCreateReq) (*m.ById, error)
	GetAll() ([]m.LevelRes, error)
	Update(req *m.LevelUpdateReq) (*m.Void, error)
	Delete(req *m.ById) (*m.Void, error)
	GetLevelForXP(xp int) (*m.LevelRes, error)
}

type AppSettingI interface {
	Get(key string) (*m.AppSettingRes, error)
	Set(req *m.AppSettingSetReq) (*m.Void, error)
	GetAll() ([]m.AppSettingRes, error)
	// Kunlik bepul limit tekshiruvi (BACKEND_STRUCTURE.md 6-bo'lim)
	GetTodayFreeAttemptsUsed(userId string) (int, error)
	IncrTodayFreeAttempts(userId string) error
}

// --- Admin AI ---

type AIPromptI interface {
	GetByKey(key string) (*m.AIPromptRes, error)
	GetAll() ([]m.AIPromptRes, error)
	Upsert(req *m.AIPromptUpsertReq) (*m.Void, error)
}

type AIUsageLogI interface {
	Log(req *m.AIUsageLogCreateReq) error
	GetSummary() (*m.AIUsageSummaryRes, error)
}
