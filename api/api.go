package api

import (
	"log/slog"

	"time"

	"tibsphereai/api/handler"
	mid "tibsphereai/api/middleware"

	"github.com/casbin/casbin/v2"
	"github.com/go-redis/redis"

	_ "tibsphereai/api/docs"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title MedicAI API
// @version 1.0
// @description API for MedicAI clinical simulation platform
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func NewApi(h *handler.Handler, rdb *redis.Client) *gin.Engine {
	router := gin.Default()

	enforcer, err := casbin.NewEnforcer("./api/casbin/model.conf", "./api/casbin/policy.csv")
	if err != nil {
		slog.Error("Error while creating enforcer: ", "err", err)
	}

	if enforcer == nil {
		slog.Error("Enforcer is nil after initialization!")
	} else {
		slog.Info("Enforcer initialized successfully.")
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Accept-Language", "Authorization", "X-Captcha-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	}))

	router.GET("/api/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// --- Xavfsizlik: rate limiting va captcha (TZ_MVP) ---
	rlOff := h.Cfg != nil && h.Cfg.RATE_LIMIT_DISABLED
	rl := func(name string, limit int, window time.Duration, key mid.KeyFunc) gin.HandlerFunc {
		return mid.RateLimit(rdb, rlOff, name, limit, window, key)
	}
	var captcha gin.HandlerFunc = func(c *gin.Context) { c.Next() }
	if h.Cfg != nil {
		captcha = mid.Captcha(mid.NewCaptchaVerifier(h.Cfg.CAPTCHA_PROVIDER, h.Cfg.CAPTCHA_SECRET))
	}
	// Umumiy to'siq: bitta IP'dan daqiqasiga 600 so'rov (to'lov provayderlari webhook'lari oddiy IP'dan keladi).
	router.Use(rl("global", 600, time.Minute, mid.ByIP))

	// --- Public (unauthorized) ---
	router.POST("/web/auth/admin/login", rl("admin_login", 10, 10*time.Minute, mid.ByIP), h.Login)
	router.POST("/mobile/auth/user/check", rl("user_check", 30, time.Minute, mid.ByIP), h.CheckUser)
	router.POST("/mobile/auth/google", rl("google", 30, time.Minute, mid.ByIP), h.GoogleLogin)
	router.POST("/mobile/auth/telegram/start", rl("tg_start", 20, time.Hour, mid.ByIP), captcha, h.TelegramLoginStart)
	router.POST("/mobile/auth/telegram/check", rl("tg_check", 120, time.Minute, mid.ByIP), h.TelegramLoginCheck)
	router.POST("/click/prepare", h.ClickPrepare)
	router.POST("/click/complete", h.ClickComplete)
	router.POST("/payme", h.PaymeRPC)
	router.POST("/inpay/callback", h.InpayCallback)
	router.POST("/mobile/auth/user/otp/send", rl("otp_send_ip", 20, time.Hour, mid.ByIP), captcha, h.SendConfirmation)
	router.POST("/mobile/auth/user/otp/confirm", rl("otp_confirm_ip", 30, 10*time.Minute, mid.ByIP), h.VerifyConfirmation)
	router.POST("/auth/token/refresh", rl("refresh", 60, time.Minute, mid.ByIP), h.RefreshToken)
	router.GET("/mobile/app-route", h.GetAllAppRoutes)
	router.GET("/mobile/app-route/:id", h.GetByIdAppRoute)

	web := router.Group("/web")
	{
		auth := web.Group("/auth").Use(mid.NewAuth(enforcer, rdb))
		{
			auth.PUT("/password/change", h.ChangePasswordWeb)
			auth.PUT("/password/refresh", h.RefreshPassword)
		}

		user := web.Group("/user").Use(mid.NewAuth(enforcer, rdb))
		{
			user.GET("", h.GetAllUsers)
			user.GET("/:id", h.GetByIdUser)
			user.DELETE("/:id/delete", h.DeleteUser)
		}

		admin := web.Group("/admin").Use(mid.NewAuth(enforcer, rdb))
		{
			admin.POST("", h.CreateAdmin)
			admin.GET("", h.GetAllAdmins)
			admin.GET("/profile", h.GetProfileAdmin)
			admin.GET("/:id", h.GetByIdAdmin)
			admin.PUT("/:id/update", h.UpdateAdmin)
			admin.DELETE("/:id/delete", h.DeleteAdmin)
		}

		role := web.Group("/role").Use(mid.NewAuth(enforcer, rdb))
		{
			role.POST("", h.CreateRole)
			role.GET("", h.GetAllRoles)
			role.GET("/:id", h.GetByIdRole)
			role.PUT("/:id/update", h.UpdateRole)
			role.DELETE("/:id/delete", h.DeleteRole)
		}

		customPermission := web.Group("/custom-permission").Use(mid.NewAuth(enforcer, rdb))
		{
			customPermission.POST("", h.CreateCustomPermission)
			customPermission.GET("", h.GetRootCustomPermissions)
			customPermission.PUT("/:id/update", h.UpdateCustomPermission)
			customPermission.DELETE("/:id/delete", h.DeleteCustomPermission)
			customPermission.GET("/:id/children", h.GetChildCustomPermissions)
			customPermission.GET("/accesses", h.GetAccesses)
			customPermission.PUT("/accesses", h.UpdateAccesses)
		}

		banner := web.Group("/banner").Use(mid.NewAuth(enforcer, rdb))
		{
			banner.POST("", h.CreateBanner)
			banner.GET("", h.GetAllBanners)
			banner.GET("/:id", h.GetByIdBanner)
			banner.PUT("/:id/update", h.UpdateBanner)
			banner.DELETE("/:id/delete", h.DeleteBanner)
		}

		about := web.Group("/about").Use(mid.NewAuth(enforcer, rdb))
		{
			about.POST("", h.CreateAbout)
			about.GET("", h.GetAllAbouts)
			about.GET("/:id", h.GetByIdAbout)
			about.PUT("/:id/update", h.UpdateAbout)
			about.DELETE("/:id/delete", h.DeleteAbout)
		}

		app_route := web.Group("/app-route").Use(mid.NewAuth(enforcer, rdb))
		{
			app_route.POST("", h.CreateAppRoute)
			app_route.GET("", h.GetAllAppRoutes)
			app_route.GET("/:id", h.GetByIdAppRoute)
			app_route.PUT("/:id/update", h.UpdateAppRoute)
			app_route.DELETE("/:id/delete", h.DeleteAppRoute)
		}

		faq := web.Group("/faq").Use(mid.NewAuth(enforcer, rdb))
		{
			faq.POST("", h.CreateFaq)
			faq.GET("", h.GetAllFaqs)
			faq.GET("/:id", h.GetByIdFaq)
			faq.PUT("/:id/update", h.UpdateFaq)
			faq.DELETE("/:id/delete", h.DeleteFaq)
		}

		contact := web.Group("/contact").Use(mid.NewAuth(enforcer, rdb))
		{
			contact.POST("", h.CreateContact)
			contact.GET("", h.GetAllContacts)
			contact.GET("/:id", h.GetByIdContact)
			contact.PUT("/:id/update", h.UpdateContact)
			contact.DELETE("/:id/delete", h.DeleteContact)
		}

		tariff := web.Group("/tariff").Use(mid.NewAuth(enforcer, rdb))
		{
			tariff.POST("", h.CreateTariff)
			tariff.GET("", h.GetAllTariffs)
			tariff.GET("/:id", h.GetByIdTariff)
			tariff.PUT("/:id/update", h.UpdateTariff)
			tariff.DELETE("/:id/delete", h.DeleteTariff)
		}

		category := web.Group("/category").Use(mid.NewAuth(enforcer, rdb))
		{
			category.POST("", h.CreateCategory)
			category.GET("", h.GetAllCategories)
			category.GET("/:id", h.GetByIdCategory)
			category.PUT("/:id/update", h.UpdateCategory)
			category.DELETE("/:id/delete", h.DeleteCategory)
		}

		topic := web.Group("/topic").Use(mid.NewAuth(enforcer, rdb))
		{
			topic.POST("", h.CreateTopic)
			topic.GET("", h.GetAllTopics)
			topic.GET("/:id", h.GetByIdTopic)
			topic.PUT("/:id/update", h.UpdateTopic)
			topic.DELETE("/:id/delete", h.DeleteTopic)
		}

		caseGrp := web.Group("/case").Use(mid.NewAuth(enforcer, rdb))
		{
			caseGrp.POST("", h.CreateCase)
			caseGrp.POST("/ai-generate", h.GenerateCaseWithAI)
			caseGrp.POST("/ai-generate-batch", h.GenerateCaseBatch)
			caseGrp.POST("/import", h.ImportCases)
			caseGrp.GET("/import/template", h.CaseImportTemplate)
			caseGrp.GET("", h.GetAllCases)
			caseGrp.GET("/:id", h.GetByIdCase)
			caseGrp.PUT("/:id/update", h.UpdateCase)
			caseGrp.PUT("/:id/publish", h.PublishCase)
			caseGrp.DELETE("/:id/delete", h.DeleteCase)
		}

		caseSource := web.Group("/case-source").Use(mid.NewAuth(enforcer, rdb))
		{
			caseSource.POST("", h.CreateCaseSource)
			caseSource.GET("", h.GetAllCaseSources)
			caseSource.GET("/:id", h.GetByIdCaseSource)
			caseSource.PUT("/:id/update", h.UpdateCaseSource)
			caseSource.DELETE("/:id/delete", h.DeleteCaseSource)
		}

		source := web.Group("/source").Use(mid.NewAuth(enforcer, rdb))
		{
			source.POST("", h.CreateSource)
			source.GET("", h.GetAllSources)
			source.GET("/:id", h.GetByIdSource)
			source.PUT("/:id/update", h.UpdateSource)
			source.POST("/:id/reindex", h.ReindexSource)
			source.DELETE("/:id/delete", h.DeleteSource)
		}

		webFlashcard := web.Group("/flashcard").Use(mid.NewAuth(enforcer, rdb))
		{
			webFlashcard.GET("/stats", h.GetFlashcardAdminStats)
		}

		webFeedback := web.Group("/feedback").Use(mid.NewAuth(enforcer, rdb))
		{
			webFeedback.GET("", h.GetAllFeedback)
			webFeedback.PUT("/:id/status", h.SetFeedbackStatus)
		}

		webPractice := web.Group("/practice").Use(mid.NewAuth(enforcer, rdb))
		{
			webPractice.GET("/stats", h.GetPracticeStats)
		}

		partner := web.Group("/partner").Use(mid.NewAuth(enforcer, rdb))
		{
			partner.POST("", h.CreatePartner)
			partner.GET("", h.GetAllPartners)
			partner.GET("/:id", h.GetByIdPartner)
			partner.PUT("/:id/update", h.UpdatePartner)
			partner.DELETE("/:id/delete", h.DeletePartner)
		}

		promocode := web.Group("/promocode").Use(mid.NewAuth(enforcer, rdb))
		{
			promocode.POST("", h.CreatePromoCode)
			promocode.GET("", h.GetAllPromoCodes)
			promocode.GET("/:id", h.GetByIdPromoCode)
			promocode.DELETE("/:id/delete", h.DeletePromoCode)
		}

		level := web.Group("/level").Use(mid.NewAuth(enforcer, rdb))
		{
			level.POST("", h.CreateLevel)
			level.GET("", h.GetAllLevels)
			level.PUT("/:id/update", h.UpdateLevel)
			level.DELETE("/:id/delete", h.DeleteLevel)
		}

		setting := web.Group("/setting").Use(mid.NewAuth(enforcer, rdb))
		{
			setting.GET("", h.GetAllSettings)
			setting.PUT("", h.SetSetting)
			setting.GET("/difficulty-reward", h.GetDifficultyRewards)
			setting.PUT("/difficulty-reward", h.SetDifficultyRewards)
			setting.GET("/daily-free-coins", h.GetDailyFreeCoins)
			setting.PUT("/daily-free-coins", h.SetDailyFreeCoins)
			setting.GET("/coin-rate", h.GetCoinRate)
			setting.PUT("/coin-rate", h.SetCoinRate)
		}

		aiPrompt := web.Group("/ai-prompt").Use(mid.NewAuth(enforcer, rdb))
		{
			aiPrompt.GET("", h.GetAllAIPrompts)
			aiPrompt.PUT("", h.UpsertAIPrompt)
			aiPrompt.GET("/usage", h.GetAIUsageSummary)
			aiPrompt.POST("/test-patient", h.TestAIPatient)
			aiPrompt.POST("/test-debrief", h.TestAIDebrief)
		}

		notification := web.Group("/notification").Use(mid.NewAuth(enforcer, rdb))
		{
			notification.POST("", h.CreateNotification)
			notification.GET("", h.GetAllNotifications)
			notification.GET("/:id", h.GetByIdNotification)
			notification.PUT("/:id/update", h.UpdateNotification)
			notification.DELETE("/:id/delete", h.DeleteNotification)
			notification.PUT("/:id/read", h.ReadNotification)
			notification.GET("/user", h.GetAllUserNotifications)
		}

		order := web.Group("/order").Use(mid.NewAuth(enforcer, rdb))
		{
			order.GET("/:id", h.GetByIdOrder)
			order.GET("", h.GetAllOrders)
		}

		dashboard := web.Group("/dashboard").Use(mid.NewAuth(enforcer, rdb))
		{
			dashboard.GET("", h.GetDashboard)
		}

		coinTransaction := web.Group("/coin-transaction").Use(mid.NewAuth(enforcer, rdb))
		{
			coinTransaction.GET("", h.GetAllCoinTransactions)
			coinTransaction.GET("/summary", h.GetCoinMonitoringSummary)
		}

		web.POST("/file-upload", h.UploadFile).Use(mid.NewAuth(enforcer, rdb))
	}

	mobile := router.Group("/mobile")
	{
		user := mobile.Group("/user").Use(mid.NewAuth(enforcer, rdb))
		{
			user.GET("/get/profile", h.GetProfileUser)
			user.GET("/limit", h.GetUserLimit)
			user.PUT("/update/profile", h.UpdateProfileUser)
			user.DELETE("/delete/profile", h.DeleteProfileUser)
			user.POST("/activity", h.CreateActivity)
			user.GET("/activity", h.GetActivityStats)
			user.GET("/rating", h.GetRating)
			user.POST("/device", h.RegisterDevice)
			user.DELETE("/device", h.UnregisterDevice)
			user.POST("/phone/start", rl("phone_link", 10, time.Hour, mid.ByUser), h.PhoneLinkStart)
			user.POST("/phone/confirm", rl("phone_confirm", 120, time.Minute, mid.ByUser), h.PhoneLinkConfirm)
			user.DELETE("/phone", h.UnlinkPhone)
			user.POST("/email/send", rl("email_link", 10, time.Hour, mid.ByUser), h.EmailLinkSend)
			user.POST("/email/confirm", rl("email_confirm", 20, 10*time.Minute, mid.ByUser), h.EmailLinkConfirm)
			user.DELETE("/email", h.UnlinkEmail)
		}
		mobile.GET("/referral", mid.NewAuth(enforcer, rdb), h.GetReferralInfo)

		banner := mobile.Group("/banner").Use(mid.NewAuth(enforcer, rdb))
		{
			banner.GET("", h.GetAllMobileBanners)
			banner.GET("/:id", h.GetByIdMobileBanner)
		}

		about := mobile.Group("/about").Use(mid.NewAuth(enforcer, rdb))
		{
			about.GET("", h.GetAllMobileAbouts)
			about.GET("/:id", h.GetByIdMobileAbout)
		}

		faq := mobile.Group("/faq").Use(mid.NewAuth(enforcer, rdb))
		{
			faq.GET("", h.GetAllMobileFaqs)
			faq.GET("/:id", h.GetByIdMobileFaq)
		}

		contact := mobile.Group("/contact").Use(mid.NewAuth(enforcer, rdb))
		{
			contact.GET("", h.GetAllContacts)
			contact.GET("/:id", h.GetByIdContact)
		}

		mobileSetting := mobile.Group("/setting").Use(mid.NewAuth(enforcer, rdb))
		{
			mobileSetting.GET("/voice", h.GetVoiceSetting)
		}

		category := mobile.Group("/category").Use(mid.NewAuth(enforcer, rdb))
		{
			category.GET("", h.GetAllMobileCategories)
			category.GET("/:id", h.GetByIdMobileCategory)
		}

		topic := mobile.Group("/topic").Use(mid.NewAuth(enforcer, rdb))
		{
			topic.GET("", h.GetAllMobileTopics)
		}

		partner := mobile.Group("/partner").Use(mid.NewAuth(enforcer, rdb))
		{
			partner.GET("", h.GetAllMobilePartners)
		}

		// level - front tomonda "780/1000 XP" progress-bar hisoblash uchun
		// (joriy XP / keyingi levelning required_xp), o'qish uchun ochiq.
		mobile.GET("/level", mid.NewAuth(enforcer, rdb), h.GetAllLevels)

		caseGrp := mobile.Group("/case").Use(mid.NewAuth(enforcer, rdb))
		{
			caseGrp.GET("", h.GetAllMobileCases)
			caseGrp.GET("/random", h.GetRandomCase)
			caseGrp.GET("/:id", h.GetByIdMobileCase)
		}

		favorite := mobile.Group("/favorite").Use(mid.NewAuth(enforcer, rdb))
		{
			favorite.POST("", h.ToggleFavorite)
			favorite.GET("", h.GetFavorites)
		}

		studyPlan := mobile.Group("/study-plan").Use(mid.NewAuth(enforcer, rdb))
		{
			studyPlan.GET("", h.GetStudyPlan)
			studyPlan.PUT("", h.UpsertStudyPlan)
		}

		simulation := mobile.Group("/simulation").Use(mid.NewAuth(enforcer, rdb))
		{
			simulation.POST("/start", h.StartSimulation)
			simulation.GET("/ongoing", h.GetOngoingSimulations)
			simulation.GET("/completed", h.GetCompletedSimulations)
			simulation.GET("/:id", h.GetByIdSimulation)
			simulation.GET("/:id/ws", h.SimulationWS)
			simulation.POST("/:id/event", rl("sim_event", 60, time.Minute, mid.ByUser), h.AddSimulationEvent)
			simulation.PUT("/:id/finish", h.FinishSimulation)
		}

		practice := mobile.Group("/practice").Use(mid.NewAuth(enforcer, rdb))
		{
			practice.GET("/wallet", h.GetPracticeWallet)
			practice.POST("/sessions", rl("practice_start", 20, time.Minute, mid.ByUser), h.StartPractice)
			practice.GET("/sessions", h.ListPracticeSessions)
			practice.GET("/sessions/:id", h.GetPracticeSession)
			practice.POST("/sessions/:id/next", rl("practice_next", 20, time.Minute, mid.ByUser), h.NextPractice)
			practice.POST("/sessions/:id/finish", h.FinishPractice)
			practice.POST("/items/:id/answer", rl("practice_answer", 40, time.Minute, mid.ByUser), h.AnswerPractice)
			practice.POST("/items/:id/report", h.ReportPracticeItem)
		}

		mobile.GET("/roadmap", mid.NewAuth(enforcer, rdb), h.GetRoadmap)
		mobile.POST("/roadmap/refresh", mid.NewAuth(enforcer, rdb), rl("roadmap_refresh", 3, 10*time.Minute, mid.ByUser), h.RefreshRoadmap)
		mobile.POST("/feedback", mid.NewAuth(enforcer, rdb), rl("feedback", 10, time.Hour, mid.ByUser), h.CreateFeedback)

		flashcard := mobile.Group("/flashcard").Use(mid.NewAuth(enforcer, rdb))
		{
			flashcard.GET("/due", h.GetDueFlashcards)
			flashcard.GET("/stats", h.GetFlashcardStats)
			flashcard.GET("/decks", h.ListFlashcardDecks)
			flashcard.POST("/decks", rl("deck_create", 5, 10*time.Minute, mid.ByUser), h.CreateFlashcardDeck)
			flashcard.DELETE("/decks/:id", h.DeleteFlashcardDeck)
			flashcard.GET("/cards", h.ListFlashcards)
			flashcard.POST("/cards", h.CreateFlashcard)
			flashcard.PUT("/cards/:id", h.UpdateFlashcard)
			flashcard.DELETE("/cards/:id", h.DeleteFlashcard)
			flashcard.POST("/cards/:id/review", h.ReviewFlashcard)
		}

		debrief := mobile.Group("/debrief").Use(mid.NewAuth(enforcer, rdb))
		{
			debrief.GET("/:session_id", h.GetDebrief)
		}

		notification := mobile.Group("/notification").Use(mid.NewAuth(enforcer, rdb))
		{
			notification.PUT("/:id/read", h.ReadNotification)
			notification.GET("/user", h.GetAllUserNotifications)
		}

		tariff := mobile.Group("/tariff").Use(mid.NewAuth(enforcer, rdb))
		{
			tariff.GET("", h.GetAllTariffs)
			tariff.GET("/:id", h.GetByIdTariff)
		}

		subscription := mobile.Group("/subscription").Use(mid.NewAuth(enforcer, rdb))
		{
			subscription.POST("", h.CreateOrder)
		}

		promocode := mobile.Group("/promocode").Use(mid.NewAuth(enforcer, rdb))
		{
			promocode.GET("", h.GetMyPromoCodes)
			promocode.POST("/redeem", h.RedeemPromoCode)
		}
	}
	return router
}
