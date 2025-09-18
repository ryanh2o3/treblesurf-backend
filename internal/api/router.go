package api

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"treblesurf-backend/internal/auth"
	"treblesurf-backend/internal/controller"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRouter creates and configures a Gin router with all application routes
func SetupRouter(container *Container) *gin.Engine {
	r := gin.Default()

	isLocal := os.Getenv("GO_ENV")

	setupRoutes(r, container)

	if isLocal == "development" {
		// Add CORS middleware for development environment
		r.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"http://localhost:5173"},
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
		}))
		
		apiGroup := r.Group("/api")
		setupRoutes(apiGroup, container)
	} else {
		// Production CORS configuration for iOS PWA and other clients
		r.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"*"}, // Allow all origins in production
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-CSRF-Token", "User-Agent"},
			ExposeHeaders:    []string{"Content-Length", "X-CSRF-Token"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour, // Cache preflight for 12 hours
		}))
	}
	
	// Enhanced logging middleware
	return r
}

func setupRoutes(r gin.IRouter, container *Container) {
	log.Print(os.Getenv("GO_ENV"))
	
	// Apply iOS headers to all API routes
	r.Use(iOSHeadersMiddleware())
	
	r.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			newPath := strings.TrimPrefix(path, "/api")
			c.Request.URL.Path = newPath
		}
		c.Next()
	})

	// Public routes
	r.POST("/auth/google", auth.GoogleAuthHandler)
	r.GET("/auth/validate", auth.ValidateTokenHandler)
	r.POST("/auth/logout", auth.LogoutHandler)
	
	// Development-only endpoint for iOS simulator
	if os.Getenv("GO_ENV") == "development" {
		r.POST("/auth/dev-session", func(c *gin.Context) {
			var req struct {
				Email string `json:"email"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
				return
			}
			
			if err := auth.CreateDevSession(req.Email, c); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
				return
			}
			
			c.JSON(http.StatusOK, gin.H{"message": "Development session created"})
		})
	}

	// Location and forecast routes
	r.GET("/regions", controller.GetRegions)
	r.GET("/spots", controller.GetSpots)
	r.GET("/location", controller.GetCoordinates)
	r.GET("/listSpotsForecast", container.ForecastController.GetListSpotsForecast)
	r.GET("/regionForecast", container.ForecastController.GetRegionForecast)
	r.GET("/currentConditions", container.ForecastController.GetCurrentWeather)
	r.GET("/beforeAfterTide", container.ForecastController.GetBeforeAfterTides)
	r.GET("/tideExtremes", container.ForecastController.GetDayTides)
	r.GET("/getLiveBuoyData", controller.GetLiveBuoyData)
	r.GET("/getSingleBuoyData", controller.GetSingleBuoyData)
	r.GET("/getLast24BuoyData", controller.GetLast24HoursBuoyData)
	r.GET("/getMultipleBuoyData", controller.GetMultipleBuoyData)
	r.GET("/buoyLocationInfo", controller.BuoyLocationInfo)
	r.GET("/regionBuoys", controller.GetRegionBuoys)
	r.GET("/individualBuoyLocation", controller.IndividualBuoyLocationInfo)
	r.GET("/locationInfo", controller.GetLocationInfo)
	r.GET("/forecast", container.ForecastController.GetSpotForecast)

	isLocal := os.Getenv("GO_ENV") == "development"

	// Protected routes (auth required)
	authorized := r.Group("/")
	if isLocal {
		log.Print("using dev middleware")
		// In local development, use a mock auth middleware that automatically authenticates
		authorized.Use(DevAuthMiddleware())
	} else {
		// In production, use the real auth middleware
		log.Print("using production auth middleware")
		authorized.Use(auth.AuthMiddleware())
	}

	webModifyGroup := authorized.Group("/")
	if !isLocal {
		// Only apply CSRF in production, skip in local dev
		log.Print("using production CSRF middleware")
		webModifyGroup.Use(auth.CSRFMiddleware())
	}
	
	webModifyGroup.POST("/submitSurfReport", controller.SubmitCurrentSurfReport)
	webModifyGroup.POST("/submitSurfReportWithS3Image", controller.SubmitSurfReportWithS3Image)
	webModifyGroup.POST("/submitSurfReportWithIOSValidation", controller.SubmitSurfReportWithIOSValidation)
	webModifyGroup.GET("/generateImageUploadURL", controller.GenerateImageUploadURL)
	webModifyGroup.GET("/generateVideoUploadURL", controller.GenerateVideoUploadURL)
	webModifyGroup.DELETE("/deleteUploadedMedia", controller.DeleteUploadedMedia)
	webModifyGroup.DELETE("/deleteMyAccount", controller.DeleteMyAccount)
	webModifyGroup.PUT("/setTheme", controller.SetUserTheme)
	webModifyGroup.DELETE("/sessions/:sessionId", auth.TerminateSessionHandler)

	authorized.GET("/sessions", auth.GetUserSessionsHandler)
	authorized.GET("/getTheme", controller.GetUserTheme)
	authorized.GET("/getTodaySpotReports", controller.RetrieveTodaysSurfReports)
	authorized.GET("getReportImage", controller.GetReportImage)
	authorized.GET("/getReportVideo", controller.GetReportVideo)
	authorized.GET("/generateVideoViewURL", controller.GenerateVideoViewURL)
	authorized.GET("/ws-token", auth.GetWebSocketTokenHandler)
	authorized.GET("/streamUrl", controller.GetStreamPlaybackURL)
	authorized.GET("/latestSnapshot", controller.GetLatestSnapshotHandler)
	authorized.POST("/requestStream", controller.RequestStreamHandler)

	// API key routes
	apiKeyRoutes := r.Group("/")
	if isLocal {
		// In local dev, use mock auth
		apiKeyRoutes.Use(DevAuthMiddleware())
	} else {
		// In production, require API key
		apiKeyRoutes.Use(APIKeyAuthMiddleware("stream"))
	}
	
	apiKeyRoutes.POST("/streaming-credentials", controller.GetStreamingCredentials)
	apiKeyRoutes.GET("/check-streaming-requested", controller.CheckStreamRequestHandler)
	apiKeyRoutes.POST("/upload-snapshot", controller.UploadSnapshotHandler)

	// Admin routes
	adminRoutes := r.Group("/admin")
	if isLocal {
		// In local dev, use mock auth that gives admin privileges
		adminRoutes.Use(DevAdminAuthMiddleware())
	} else {
		// In production, use real auth with admin check
		log.Print("using production admin middleware")
		adminRoutes.Use(auth.AuthMiddleware(), AdminMiddleware())
	}
	
	adminRoutes.POST("/api-keys", controller.CreateAPIKeyHandler)
	adminRoutes.GET("/api-keys", controller.ListAPIKeysHandler)
	adminRoutes.DELETE("/api-keys/:keyID", controller.RevokeAPIKeyHandler)
}