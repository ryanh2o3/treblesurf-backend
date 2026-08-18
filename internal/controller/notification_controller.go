package controller

import (
	"errors"
	"net/http"
	"strings"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// NotificationController handles device token and spot-alert preference routes.
type NotificationController struct {
	notifications *service.NotificationService
	users         *service.UserService
}

func NewNotificationController(
	notifications *service.NotificationService,
	users *service.UserService,
) *NotificationController {
	return &NotificationController{
		notifications: notifications,
		users:         users,
	}
}

func (nc *NotificationController) PutDeviceToken(c *gin.Context) {
	user, ok := nc.requireUser(c)
	if !ok {
		return
	}
	var req model.DeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}
	if err := nc.notifications.RegisterDeviceToken(c.Request.Context(), user.UUID, req.Token, req.Environment); err != nil {
		if errors.Is(err, repository.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Device token registered"})
}

func (nc *NotificationController) DeleteDeviceToken(c *gin.Context) {
	user, ok := nc.requireUser(c)
	if !ok {
		return
	}
	var req model.DeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}
	if err := nc.notifications.UnregisterDeviceToken(c.Request.Context(), user.UUID, req.Token); err != nil {
		if errors.Is(err, repository.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove device token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Device token removed"})
}

func (nc *NotificationController) GetPreferences(c *gin.Context) {
	user, ok := nc.requireUser(c)
	if !ok {
		return
	}
	prefs, err := nc.notifications.GetPreferences(c.Request.Context(), user.UUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load notification preferences"})
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func (nc *NotificationController) PutSpotAlert(c *gin.Context) {
	user, ok := nc.requireUser(c)
	if !ok {
		return
	}
	country, region, spot, ok := spotQuery(c)
	if !ok {
		return
	}
	var req model.SpotAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "reportsEnabled and goodSurfEnabled are required"})
		return
	}
	if err := nc.notifications.UpsertSpotAlert(
		c.Request.Context(),
		user.UUID,
		country,
		region,
		spot,
		req.ReportsEnabled,
		req.GoodSurfEnabled,
	); err != nil {
		if errors.Is(err, repository.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update spot alerts"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Spot alerts updated"})
}

func (nc *NotificationController) DeleteSpotAlert(c *gin.Context) {
	user, ok := nc.requireUser(c)
	if !ok {
		return
	}
	country, region, spot, ok := spotQuery(c)
	if !ok {
		return
	}
	if err := nc.notifications.DeleteSpotAlert(c.Request.Context(), user.UUID, country, region, spot); err != nil {
		if errors.Is(err, repository.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove spot alerts"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Spot alerts removed"})
}

func (nc *NotificationController) requireUser(c *gin.Context) (*model.User, bool) {
	if nc == nil || nc.notifications == nil || nc.users == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Notifications unavailable"})
		return nil, false
	}
	emailStr, err := getEmailFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return nil, false
	}
	user, ok := loadUserOr404(c, nc.users, emailStr)
	if !ok {
		return nil, false
	}
	if strings.TrimSpace(user.UUID) == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User is missing a UUID"})
		return nil, false
	}
	return user, true
}

func spotQuery(c *gin.Context) (country, region, spot string, ok bool) {
	country = strings.TrimSpace(c.Query("country"))
	region = strings.TrimSpace(c.Query("region"))
	spot = strings.TrimSpace(c.Query("spot"))
	if country == "" || region == "" || spot == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "country, region, and spot are required"})
		return "", "", "", false
	}
	return country, region, spot, true
}
