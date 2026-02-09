package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

type VerificationNotification struct {
	UserID          uint
	Email           string
	Token           string
	ExpiresAt       time.Time
	VerificationURL string
}

type EmailVerificationNotifier interface {
	SendEmailVerification(ctx context.Context, notification VerificationNotification) error
}

type DevEmailVerificationNotifier struct {
	logger *slog.Logger
}

func NewDevEmailVerificationNotifier(logger *slog.Logger) *DevEmailVerificationNotifier {
	return &DevEmailVerificationNotifier{logger: logger}
}

func (n *DevEmailVerificationNotifier) SendEmailVerification(ctx context.Context, notification VerificationNotification) error {
	link := notification.VerificationURL
	if strings.TrimSpace(link) == "" {
		link = fmt.Sprintf("token=%s", notification.Token)
	}
	n.logger.InfoContext(ctx, "email verification token issued",
		"user_id", notification.UserID,
		"email", notification.Email,
		"expires_at", notification.ExpiresAt,
		"verification", link,
	)
	return nil
}
