package handler

import (
	"tibsphereai/internal/config"
	"tibsphereai/pkg/clients"
	"tibsphereai/pkg/minio"
)

type Handler struct {
	Clients *clients.Clients
	MinIO   *minio.MinIO
	Cfg     *config.Config
}

func NewHandler(client clients.Clients, minio *minio.MinIO, cfg *config.Config) *Handler {
	return &Handler{
		Clients: &client,
		MinIO:   minio,
		Cfg:     cfg,
	}
}
