package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go-oauth-rbac-service/internal/config"
	"go-oauth-rbac-service/internal/database"
	"go-oauth-rbac-service/internal/http/handler"
	"go-oauth-rbac-service/internal/http/router"
	"go-oauth-rbac-service/internal/observability"
	"go-oauth-rbac-service/internal/repository"
	"go-oauth-rbac-service/internal/security"
	"go-oauth-rbac-service/internal/service"
)

type App struct {
	Config *config.Config
	Logger *slog.Logger
	Server *http.Server
}

func New() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	logger := observability.NewLogger()
	if err := observability.InitMetrics(); err != nil {
		return nil, err
	}
	if err := observability.InitTracing(); err != nil {
		return nil, err
	}

	db, err := database.Open(cfg)
	if err != nil {
		return nil, err
	}
	if err := database.Migrate(db); err != nil {
		return nil, err
	}
	if err := database.Seed(db, cfg.BootstrapAdminEmail); err != nil {
		return nil, err
	}

	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	oauthRepo := repository.NewOAuthRepository(db)

	jwtMgr := security.NewJWTManager(cfg.JWTIssuer, cfg.JWTAudience, cfg.JWTAccessSecret, cfg.JWTRefreshSecret)
	cookieMgr := security.NewCookieManager(cfg.CookieDomain, cfg.CookieSecure, cfg.CookieSameSite)

	rbacSvc := service.NewRBACService()
	userSvc := service.NewUserService(userRepo, rbacSvc)
	tokenSvc := service.NewTokenService(jwtMgr, sessionRepo, cfg.RefreshTokenPepper, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	oauthSvc := service.NewOAuthService(service.NewGoogleOAuthProvider(cfg), userRepo, oauthRepo, roleRepo)
	authSvc := service.NewAuthService(cfg, oauthSvc, tokenSvc, userSvc, rbacSvc)

	authHandler := handler.NewAuthHandler(authSvc, cookieMgr, cfg.StateSigningSecret, cfg.JWTRefreshTTL)
	userHandler := handler.NewUserHandler(userSvc)
	adminHandler := handler.NewAdminHandler(userSvc, roleRepo, permRepo)

	h := router.NewRouter(router.Dependencies{
		AuthHandler:      authHandler,
		UserHandler:      userHandler,
		AdminHandler:     adminHandler,
		JWTManager:       jwtMgr,
		RBACService:      rbacSvc,
		CORSOrigins:      cfg.CORSAllowedOrigins,
		AuthRateLimitRPM: cfg.AuthRateLimitPerMin,
		APIRateLimitRPM:  cfg.APIRateLimitPerMin,
	})

	server := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           h,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{Config: cfg, Logger: logger, Server: server}, nil
}

func RunMigrationOnly() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.Open(cfg)
	if err != nil {
		return err
	}
	if err := database.Migrate(db); err != nil {
		return err
	}
	if err := database.Seed(db, cfg.BootstrapAdminEmail); err != nil {
		return err
	}
	fmt.Println("migration complete")
	return nil
}
