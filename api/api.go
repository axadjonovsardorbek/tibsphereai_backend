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
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           24 * time.Hour,
	}))

	router.GET("/api/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// --- Public (unauthorized) ---
	router.POST("/web/auth/admin/login", h.Login)
	router.POST("/mobile/auth/user/check", h.CheckUser)
	router.POST("/mobile/auth/google", h.GoogleLogin)
	router.POST("/click/prepare", h.ClickPrepare)
	router.POST("/click/complete", h.ClickComplete)
	router.POST("/payme", h.PaymeRPC)
	router.POST("/mobile/auth/user/otp/send", h.SendConfirmation)
	router.POST("/mobile/auth/user/otp/confirm", h.VerifyConfirmation)
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
			app_route.GET("", h.GetAllAppRoutes)
			app_route.GET("/:id", h.GetByIdAppRoute)
			app_route.PUT("/:id/update", h.UpdateAppRoute)
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
			caseGrp.GET("", h.GetAllCases)
			caseGrp.GET("/:id", h.GetByIdCase)
			caseGrp.PUT("/:id/update", h.UpdateCase)
			caseGrp.PUT("/:id/publish", h.PublishCase)
			caseGrp.DELETE("/:id/delete", h.DeleteCase)
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
		}

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

		category := mobile.Group("/category").Use(mid.NewAuth(enforcer, rdb))
		{
			category.GET("", h.GetAllMobileCategories)
			category.GET("/:id", h.GetByIdMobileCategory)
			category.GET("/:id/unlock", h.GetCategoryUnlockStatus)
			category.POST("/:id/unlock", h.PurchaseCategoryUnlock)
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
			simulation.POST("/:id/event", h.AddSimulationEvent)
			simulation.PUT("/:id/finish", h.FinishSimulation)
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
