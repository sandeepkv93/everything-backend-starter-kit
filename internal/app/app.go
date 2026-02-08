package app

import (
	"log/slog"
	"net/http"

	"go-oauth-rbac-service/internal/config"
)

type App struct {
	Config *config.Config
	Logger *slog.Logger
	Server *http.Server
}

func New(cfg *config.Config, logger *slog.Logger, server *http.Server) *App {
	return &App{Config: cfg, Logger: logger, Server: server}
}
