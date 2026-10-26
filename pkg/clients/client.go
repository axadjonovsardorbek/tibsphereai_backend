package clients

import (
	"tibsphereai/internal/ai"
	"tibsphereai/internal/config"
	"tibsphereai/internal/inpay"
	"tibsphereai/internal/push"
	"tibsphereai/service"
	"tibsphereai/storage/postgres"

	"github.com/go-redis/redis"
)

type Clients struct {
	Admin            *service.AdminService
	Role             *service.RoleService
	CustomPermission *service.CustomPermissionService
	About            *service.AboutService
	Banner           *service.BannerService
	Contact          *service.ContactService
	Faq              *service.FaqService
	Notification     *service.NotificationService
	AppRoute         *service.AppRouteService
	Order            *service.OrderService
	Tariff           *service.TariffService
	PromoCode        *service.PromoCodeService
	Partner          *service.PartnerService
	CaseSource       *service.CaseSourceService
	Source           *service.SourceService
	Practice         *service.PracticeService
	Dashboard        *service.DashboardService
	User             *service.UserService
	UserDevice       *service.UserDeviceService

	Category        *service.CategoryService
	Topic           *service.TopicService
	StudyPlan       *service.StudyPlanService
	Case            *service.CaseService
	Favorite        *service.FavoriteService
	Simulation      *service.SimulationService
	Debrief         *service.DebriefService
	Level           *service.LevelService
	AppSetting      *service.AppSettingService
	AIPrompt        *service.AIPromptService
	CoinTransaction *service.CoinTransactionService
}

func NewClients(cfg *config.Config, conn *postgres.Storage, rdb *redis.Client, pushClient *push.Client) (*Clients, error) {
	pushSvc := service.NewPushService(conn, pushClient)
	aiClient := ai.NewClient(cfg, conn.AIPromptS)
	appSettingSvc := service.NewAppSettingService(conn, cfg.DAILY_FREE_LIMIT)
	userSvc := service.NewUserService(conn, rdb, cfg, pushClient)

	return &Clients{
		Admin:            service.NewAdminService(conn),
		Role:             service.NewRoleService(conn),
		CustomPermission: service.NewCustomPermissionService(conn),
		About:            service.NewAboutService(conn, rdb),
		Banner:           service.NewBannerService(conn, rdb),
		Contact:          service.NewContactService(conn, rdb),
		Faq:              service.NewFaqService(conn, rdb),
		Notification:     service.NewNotificationService(conn, rdb, pushSvc),
		AppRoute:         service.NewAppRouteService(conn, rdb),
		Order:            service.NewOrderService(conn, rdb, pushSvc).WithInpay(inpay.NewClient(cfg.INPAY_BASE_URL, cfg.INPAY_MERCHANT_ID, cfg.INPAY_MERCHANT_TOKEN), cfg.BASE_URL, cfg.WEBSITE_URL),
		Tariff:           service.NewTariffService(conn, rdb),
		PromoCode:        service.NewPromoCodeService(conn, rdb, userSvc),
		Partner:          service.NewPartnerService(conn),
		CaseSource:       service.NewCaseSourceService(conn),
		Source:           service.NewSourceService(conn, aiClient),
		Practice:         service.NewPracticeService(conn, aiClient, appSettingSvc),
		Dashboard:        service.NewDashboardService(conn, rdb),
		User:             userSvc,
		UserDevice:       service.NewUserDeviceService(conn, pushClient),

		Category:        service.NewCategoryService(conn),
		Topic:           service.NewTopicService(conn),
		StudyPlan:       service.NewStudyPlanService(conn, pushSvc),
		Case:            service.NewCaseService(conn, aiClient),
		Favorite:        service.NewFavoriteService(conn),
		Simulation:      service.NewSimulationService(conn, aiClient, appSettingSvc),
		Debrief:         service.NewDebriefService(conn, aiClient),
		Level:           service.NewLevelService(conn),
		AppSetting:      appSettingSvc,
		AIPrompt:        service.NewAIPromptService(conn, aiClient),
		CoinTransaction: service.NewCoinTransactionService(conn),
	}, nil
}
