package postgres

import (
	"context"
	"fmt"

	"tibsphereai/internal/config"
	"tibsphereai/storage"

	"github.com/go-redis/redis"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	Db *pgxpool.Pool

	AdminS            storage.AdminI
	RoleS             storage.RoleI
	CustomPermissionS storage.CustomPermissionI
	AboutS            storage.AboutI
	BannerS           storage.BannerI
	ContactS          storage.ContactI
	FaqS              storage.FaqI
	NotificationS     storage.NotificationI
	AppRouteS         storage.AppRouteI
	OrderS            storage.OrderI
	TariffS           storage.TariffI
	PromoCodeS        storage.PromoCodeI
	PartnerS          storage.PartnerI
	DashboardS        storage.DashBoardI
	UserS             storage.UserI
	UserDeviceS       storage.UserDeviceI

	CategoryS       storage.CategoryI
	CategoryUnlockS storage.CategoryUnlockI
	TopicS          storage.TopicI
	StudyPlanS      storage.StudyPlanI
	CaseS           storage.CaseI
	FavoriteS       storage.FavoriteI
	SimulationS     storage.SimulationI
	DebriefS        storage.DebriefI
	LevelS          storage.LevelI
	AppSettingS     storage.AppSettingI
	AIPromptS       storage.AIPromptI
	AIUsageLogS     storage.AIUsageLogI
}

func NewPostgresStorage(cf config.Config, rdb *redis.Client) (*Storage, error) {
	ctx := context.Background()

	connString := fmt.Sprintf("host=%s user=%s dbname=%s password=%s port=%d sslmode=disable",
		cf.POSTGRES_HOST, cf.POSTGRES_USER, cf.POSTGRES_NAME, cf.POSTGRES_PASSWORD, cf.POSTGRES_PORT)

	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	return &Storage{
		Db: pool,

		AdminS:            NewAdminRepo(pool, rdb),
		RoleS:             NewRoleRepo(pool),
		CustomPermissionS: NewCustomPermissionRepo(pool),
		AboutS:            NewAboutRepo(pool),
		BannerS:           NewBannerRepo(pool),
		ContactS:          NewContactRepo(pool),
		FaqS:              NewFaqRepo(pool),
		NotificationS:     NewNotificationRepo(pool),
		AppRouteS:         NewAppRouteRepo(pool),
		OrderS:            NewOrderRepo(pool, cf.CLICK_MERCHANT_ID, cf.CLICK_SERVICE_ID),
		TariffS:           NewTariffRepo(pool),
		PromoCodeS:        NewPromoCodeRepo(pool),
		PartnerS:          NewPartnerRepo(pool),
		DashboardS:        NewDashboardRepo(pool),
		UserS:             NewUserRepo(pool, rdb),
		UserDeviceS:       NewUserDeviceRepo(pool),

		CategoryS:       NewCategoryRepo(pool),
		CategoryUnlockS: NewCategoryUnlockRepo(pool),
		TopicS:          NewTopicRepo(pool),
		StudyPlanS:      NewStudyPlanRepo(pool),
		CaseS:           NewCaseRepo(pool),
		FavoriteS:       NewFavoriteRepo(pool),
		SimulationS:     NewSimulationRepo(pool),
		DebriefS:        NewDebriefRepo(pool),
		LevelS:          NewLevelRepo(pool),
		AppSettingS:     NewAppSettingRepo(pool),
		AIPromptS:       NewAIPromptRepo(pool),
		AIUsageLogS:     NewAIUsageLogRepo(pool),
	}, nil
}
