package app

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"tibsphereai/api"
	"tibsphereai/api/handler"
	"tibsphereai/internal/bot"
	"tibsphereai/internal/config"
	"tibsphereai/internal/push"
	"tibsphereai/pkg/clients"
	"tibsphereai/pkg/minio"
	"tibsphereai/storage/postgres"

	"github.com/go-redis/redis"
)

func Run(cf *config.Config) {
	ctx := context.Background()

	rAddr := fmt.Sprintf("%s%s", cf.REDIS_HOST, cf.REDIS_PORT)

	rdb := redis.NewClient(&redis.Options{
		Addr: rAddr,
	})

	_, err := rdb.Ping().Result()
	if err != nil {
		slog.Error("Failed to connect redis:", "err", err)
		return
	}
	slog.Info("Redis connected successfully")

	conn, err := postgres.NewPostgresStorage(*cf, rdb)
	if err != nil {
		slog.Error("Failed to connect postgres:", "err", err)
		return
	}
	defer conn.Db.Close()
	slog.Info("Postgres storage connected successfully")

	pushClient, err := push.NewClientFromServiceAccountFile(
		ctx,
		cf.FIREBASE_CREDENTIALS,
	)
	if err != nil {
		slog.Warn("firebase init failed, push notifications disabled:", "err", err)
	}

	minioClient, err := minio.MinIOConnect(cf)
	if err != nil {
		slog.Error("Failed to connect to MinIO", "err", err)
		return
	}

	services, err := clients.NewClients(cf, conn, rdb, pushClient)
	if err != nil {
		log.Fatalf("error while connecting clients. err: %s", err.Error())
	}

	services.Practice.SetImageStore(minioClient)
	handlers := handler.NewHandler(*services, minioClient, cf)

	go bot.BotAuth(cf)
	go services.StudyPlan.RunReminderScheduler(ctx)
	go services.Order.RunInpayReconciler(ctx)

	router := api.NewApi(handlers, rdb)
	log.Println("Server is running on port ", cf.AUTH_PORT)
	if err := router.Run(cf.AUTH_PORT); err != nil {
		slog.Error("Error:", "err", err)
	}
}
