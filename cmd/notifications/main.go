// Package main provides the hourly good-surf notifications Lambda handler.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"treblesurf-backend/internal/app"
	"treblesurf-backend/internal/config"
	"treblesurf-backend/internal/service"

	"github.com/aws/aws-lambda-go/lambda"
)

var errNotificationsUnavailable = errors.New("notification service unavailable")

var notificationService *service.NotificationService

func initialize() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	application, err := app.New(cfg)
	if err != nil {
		return err
	}
	notificationService = application.NotificationService()
	if notificationService == nil {
		return errNotificationsUnavailable
	}
	return nil
}

func Handler(ctx context.Context) error {
	if notificationService == nil {
		return errNotificationsUnavailable
	}
	return notificationService.RunGoodSurfAlerts(ctx)
}

func main() {
	if err := initialize(); err != nil {
		slog.Error("failed to initialize notifications handler", slog.Any("error", err))
		os.Exit(1)
	}
	lambda.Start(Handler)
}
