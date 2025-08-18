package controller

import (
	"treblesurf-backend/internal/service"
)

// ServiceRegistry holds all services for easy access across controllers
var (
	UserService      *service.UserService
	ReportService    *service.ReportService
	LocationService  *service.LocationService
	APIKeyService    *service.APIKeyService
	WebSocketService *service.WebSocketService
)

// SetUserService sets the user service in the registry
func SetUserService(us *service.UserService) {
	UserService = us
}

// SetReportService sets the report service in the registry
func SetReportService(rs *service.ReportService) {
	ReportService = rs
}

// SetLocationService sets the location service in the registry
func SetLocationService(ls *service.LocationService) {
	LocationService = ls
}

// SetAPIKeyService sets the API key service in the registry
func SetAPIKeyService(aks *service.APIKeyService) {
	APIKeyService = aks
}

// SetWebSocketService sets the WebSocket service in the registry
func SetWebSocketService(ws *service.WebSocketService) {
	WebSocketService = ws
}
