# Idiomatic Go Refactor Plan

Transform the treblesurf-backend into an exemplary, idiomatic Go codebase following modern best practices.

## Guiding Principles

1. **Explicit over implicit** - No global state, everything injected
2. **Depend on abstractions** - Services use interfaces, not concrete types
3. **Context flows everywhere** - From HTTP handler to database
4. **Fail fast** - Validate config at startup, not runtime
5. **Testable by design** - Every layer mockable via interfaces

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────────────┐
│                         Entry Points                                  │
│              cmd/api/main.go (Lambda)                                │
│              cmd/server/main.go (Container)                          │
└─────────────────────────────┬────────────────────────────────────────┘
                              │ creates
                              ▼
┌──────────────────────────────────────────────────────────────────────┐
│                         App (internal/app/)                          │
│    - Loads config                                                    │
│    - Initializes logging                                             │
│    - Wires all dependencies                                          │
│    - Returns http.Handler                                            │
│    - Provides Shutdown()                                             │
└─────────────────────────────┬────────────────────────────────────────┘
                              │ contains
                              ▼
┌──────────────────────────────────────────────────────────────────────┐
│                    Controller Layer (internal/controller/)            │
│    - HTTP request/response handling                                  │
│    - Input validation                                                │
│    - Calls services                                                  │
│    - Each controller is a struct with injected services              │
└─────────────────────────────┬────────────────────────────────────────┘
                              │ calls
                              ▼
┌──────────────────────────────────────────────────────────────────────┐
│                    Service Layer (internal/service/)                  │
│    - Business logic                                                  │
│    - Depends ONLY on Repository interfaces                           │
│    - Never imports aws-sdk directly                                  │
│    - Receives context.Context                                        │
└─────────────────────────────┬────────────────────────────────────────┘
                              │ uses interfaces
                              ▼
┌──────────────────────────────────────────────────────────────────────┐
│                 Repository Layer (internal/repository/)               │
│    - Data access abstraction                                         │
│    - One repository per domain entity                                │
│    - Interface defined, multiple implementations possible            │
└──────────┬─────────────────────────────────────┬─────────────────────┘
           │                                     │
           ▼                                     ▼
┌─────────────────────────┐          ┌─────────────────────────┐
│   DynamoDB Impl         │          │   Mock Impl             │
│   (internal/repository/ │          │   (internal/repository/ │
│    dynamodb/)           │          │    mock/)               │
└─────────────────────────┘          └─────────────────────────┘
```

---

## Package Structure (Target State)

```
treblesurf-backend/
├── cmd/
│   ├── api/
│   │   └── main.go              # Lambda entry point
│   ├── server/
│   │   └── main.go              # Container/HTTP server entry point
│   └── websocket/
│       └── main.go              # WebSocket Lambda
├── internal/
│   ├── app/
│   │   └── app.go               # Application wiring
│   ├── config/
│   │   └── config.go            # Centralized configuration
│   ├── logging/
│   │   └── logging.go           # Structured logging setup
│   ├── model/
│   │   ├── user.go
│   │   ├── report.go
│   │   ├── location.go
│   │   ├── forecast.go
│   │   ├── buoy.go
│   │   └── errors.go            # Domain errors
│   ├── repository/
│   │   ├── interfaces.go        # All repository interfaces
│   │   ├── dynamodb/
│   │   │   ├── user_repo.go
│   │   │   ├── report_repo.go
│   │   │   ├── location_repo.go
│   │   │   ├── forecast_repo.go
│   │   │   ├── buoy_repo.go
│   │   │   ├── session_repo.go
│   │   │   └── client.go        # DynamoDB client wrapper
│   │   ├── s3/
│   │   │   └── media_repo.go    # S3 media storage
│   │   └── mock/
│   │       ├── user_repo.go
│   │       ├── report_repo.go
│   │       └── ...              # Mocks for testing
│   ├── service/
│   │   ├── user_service.go
│   │   ├── report_service.go
│   │   ├── location_service.go
│   │   ├── forecast_service.go
│   │   ├── auth_service.go
│   │   └── websocket_service.go
│   ├── controller/
│   │   ├── user_controller.go
│   │   ├── report_controller.go
│   │   ├── location_controller.go
│   │   ├── forecast_controller.go
│   │   ├── auth_controller.go
│   │   ├── buoy_controller.go
│   │   └── middleware.go
│   └── api/
│       └── router.go            # Route registration
└── go.mod
```

---

## Phase 1: Foundation

### 1.1 Configuration Package

**File: `internal/config/config.go`**

```go
package config

import (
    "fmt"
    "os"
    "strings"
)

type Config struct {
    AWS       AWSConfig
    Auth      AuthConfig
    WebSocket WebSocketConfig
    Server    ServerConfig
    Env       Environment
}

type Environment string

const (
    EnvDevelopment Environment = "development"
    EnvProduction  Environment = "production"
)

type AWSConfig struct {
    Region     string
    BucketName string
}

type AuthConfig struct {
    JWTSecret       string
    GoogleClientIDs []string
}

type WebSocketConfig struct {
    Endpoint string
    Stage    string
}

type ServerConfig struct {
    Port string
}

func Load() (*Config, error) {
    env := Environment(getEnvOrDefault("GO_ENV", "production"))
    
    cfg := &Config{
        Env: env,
        AWS: AWSConfig{
            Region:     getEnvOrDefault("AWS_REGION", "eu-west-1"),
            BucketName: getEnvOrDefault("S3_BUCKET_NAME", "treblesurf-images"),
        },
        WebSocket: WebSocketConfig{
            Endpoint: os.Getenv("WEBSOCKET_API_ENDPOINT"),
            Stage:    getEnvOrDefault("WEBSOCKET_API_STAGE", "production"),
        },
        Server: ServerConfig{
            Port: getEnvOrDefault("PORT", "8080"),
        },
    }
    
    // JWT Secret - required in production
    cfg.Auth.JWTSecret = os.Getenv("JWT_SECRET")
    if cfg.Auth.JWTSecret == "" && env == EnvProduction {
        return nil, fmt.Errorf("JWT_SECRET is required in production")
    }
    if cfg.Auth.JWTSecret == "" {
        cfg.Auth.JWTSecret = "dev-secret-do-not-use-in-prod"
    }
    
    // Google Client IDs
    if ids := os.Getenv("GOOGLE_CLIENT_IDS"); ids != "" {
        cfg.Auth.GoogleClientIDs = strings.Split(ids, ",")
    }
    
    return cfg, nil
}

func MustLoad() *Config {
    cfg, err := Load()
    if err != nil {
        panic(fmt.Sprintf("failed to load config: %v", err))
    }
    return cfg
}

func (c *Config) IsDevelopment() bool {
    return c.Env == EnvDevelopment
}

func getEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

### 1.2 Structured Logging

**File: `internal/logging/logging.go`**

```go
package logging

import (
    "context"
    "log/slog"
    "os"
    
    "treblesurf-backend/internal/config"
)

type ctxKey struct{}

// Init initializes the global logger based on environment.
func Init(cfg *config.Config) {
    var handler slog.Handler
    
    opts := &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }
    
    if cfg.IsDevelopment() {
        opts.Level = slog.LevelDebug
        handler = slog.NewTextHandler(os.Stdout, opts)
    } else {
        handler = slog.NewJSONHandler(os.Stdout, opts)
    }
    
    logger := slog.New(handler)
    slog.SetDefault(logger)
}

// FromContext retrieves a logger from context, or returns default.
func FromContext(ctx context.Context) *slog.Logger {
    if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
        return l
    }
    return slog.Default()
}

// WithLogger adds a logger to context.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
    return context.WithValue(ctx, ctxKey{}, l)
}

// WithRequestID returns a logger with request ID attached.
func WithRequestID(l *slog.Logger, requestID string) *slog.Logger {
    return l.With(slog.String("request_id", requestID))
}
```

---

## Phase 2: Repository Layer (The Key to Testability)

### 2.1 Repository Interfaces

**File: `internal/repository/interfaces.go`**

```go
package repository

import (
    "context"
    "time"
    
    "treblesurf-backend/internal/model"
)

// UserRepository handles user data persistence.
type UserRepository interface {
    GetByEmail(ctx context.Context, email string) (*model.User, error)
    GetByUUID(ctx context.Context, uuid string) (*model.User, error)
    Create(ctx context.Context, user *model.User) error
    Update(ctx context.Context, user *model.User) error
    Delete(ctx context.Context, email string) error
    UpdateTheme(ctx context.Context, email, theme string) error
    UpdateLastLogin(ctx context.Context, email string) error
}

// ReportRepository handles surf report persistence.
type ReportRepository interface {
    Create(ctx context.Context, report *model.SurfReport) error
    GetBySpot(ctx context.Context, country, region, spot string, limit int) ([]*model.SurfReport, error)
    GetBySpotAndTimeRange(ctx context.Context, country, region, spot string, start, end time.Time) ([]*model.SurfReport, error)
}

// LocationRepository handles location data persistence.
type LocationRepository interface {
    GetRegions(ctx context.Context, country string) ([]string, error)
    GetSpots(ctx context.Context, country, region string) ([]*model.LocationInfo, error)
    GetLocationInfo(ctx context.Context, country, region, spot string) (*model.LocationInfo, error)
    GetCoordinates(ctx context.Context, country, region, spot string) (lat, lon float64, err error)
}

// ForecastRepository handles forecast data persistence.
type ForecastRepository interface {
    GetSpotForecast(ctx context.Context, country, region, spot string) ([]*model.Forecast, error)
    GetCurrentConditions(ctx context.Context, country, region, spot string) (*model.Forecast, error)
    GetForecastAtTime(ctx context.Context, country, region, spot string, t time.Time) (*model.Forecast, error)
}

// BuoyRepository handles buoy data persistence.
type BuoyRepository interface {
    GetLiveData(ctx context.Context, buoyName string) (*model.BuoyData, error)
    GetDataAtTime(ctx context.Context, buoyName string, t time.Time) (*model.BuoyData, error)
    GetDataRange(ctx context.Context, buoyName string, start, end time.Time) ([]*model.BuoyData, error)
    GetLocations(ctx context.Context) (map[string]*model.BuoyLocation, error)
}

// SessionRepository handles session persistence.
type SessionRepository interface {
    Save(ctx context.Context, session *model.Session) error
    Get(ctx context.Context, sessionID string) (*model.Session, error)
    Delete(ctx context.Context, sessionID string) error
    GetByUserID(ctx context.Context, userID string) ([]*model.Session, error)
}

// MediaRepository handles media file storage (S3).
type MediaRepository interface {
    Upload(ctx context.Context, key string, data []byte, contentType string) error
    Download(ctx context.Context, key string) ([]byte, error)
    Delete(ctx context.Context, key string) error
    GenerateUploadURL(ctx context.Context, key string, expires time.Duration) (string, error)
    GenerateViewURL(ctx context.Context, key string, expires time.Duration) (string, error)
}

// APIKeyRepository handles API key persistence.
type APIKeyRepository interface {
    Create(ctx context.Context, key *model.APIKey) error
    GetByKey(ctx context.Context, key string) (*model.APIKey, error)
    List(ctx context.Context) ([]*model.APIKey, error)
    Revoke(ctx context.Context, keyID string) error
}

// WebSocketRepository handles WebSocket connection persistence.
type WebSocketRepository interface {
    SaveConnection(ctx context.Context, conn *model.ConnectionInfo) error
    GetConnection(ctx context.Context, connectionID string) (*model.ConnectionInfo, error)
    DeleteConnection(ctx context.Context, connectionID string) error
    UpdateSpot(ctx context.Context, connectionID, spot string) error
    GetConnectionsByUserIDs(ctx context.Context, userIDs []string) ([]*model.ConnectionInfo, error)
}
```

### 2.2 DynamoDB Implementation Example

**File: `internal/repository/dynamodb/user_repo.go`**

```go
package dynamodb

import (
    "context"
    "fmt"
    
    "treblesurf-backend/internal/model"
    "treblesurf-backend/internal/repository"
    
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/service/dynamodb"
    "github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

// Ensure implementation satisfies interface at compile time
var _ repository.UserRepository = (*UserRepo)(nil)

type UserRepo struct {
    client    *dynamodb.DynamoDB
    tableName string
}

func NewUserRepo(client *dynamodb.DynamoDB, tableName string) *UserRepo {
    return &UserRepo{
        client:    client,
        tableName: tableName,
    }
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
    input := &dynamodb.GetItemInput{
        TableName: aws.String(r.tableName),
        Key: map[string]*dynamodb.AttributeValue{
            "email": {S: aws.String(email)},
        },
    }
    
    result, err := r.client.GetItemWithContext(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("getting user by email: %w", err)
    }
    
    if result.Item == nil {
        return nil, model.ErrUserNotFound
    }
    
    var user model.User
    if err := dynamodbattribute.UnmarshalMap(result.Item, &user); err != nil {
        return nil, fmt.Errorf("unmarshaling user: %w", err)
    }
    
    return &user, nil
}

func (r *UserRepo) Create(ctx context.Context, user *model.User) error {
    item, err := dynamodbattribute.MarshalMap(user)
    if err != nil {
        return fmt.Errorf("marshaling user: %w", err)
    }
    
    input := &dynamodb.PutItemInput{
        TableName: aws.String(r.tableName),
        Item:      item,
    }
    
    if _, err := r.client.PutItemWithContext(ctx, input); err != nil {
        return fmt.Errorf("creating user: %w", err)
    }
    
    return nil
}

// ... implement remaining methods
```

### 2.3 Mock Implementation for Testing

**File: `internal/repository/mock/user_repo.go`**

```go
package mock

import (
    "context"
    
    "treblesurf-backend/internal/model"
    "treblesurf-backend/internal/repository"
)

var _ repository.UserRepository = (*UserRepo)(nil)

type UserRepo struct {
    GetByEmailFn     func(ctx context.Context, email string) (*model.User, error)
    GetByUUIDFn      func(ctx context.Context, uuid string) (*model.User, error)
    CreateFn         func(ctx context.Context, user *model.User) error
    UpdateFn         func(ctx context.Context, user *model.User) error
    DeleteFn         func(ctx context.Context, email string) error
    UpdateThemeFn    func(ctx context.Context, email, theme string) error
    UpdateLastLoginFn func(ctx context.Context, email string) error
}

func (m *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
    if m.GetByEmailFn != nil {
        return m.GetByEmailFn(ctx, email)
    }
    return nil, model.ErrUserNotFound
}

func (m *UserRepo) Create(ctx context.Context, user *model.User) error {
    if m.CreateFn != nil {
        return m.CreateFn(ctx, user)
    }
    return nil
}

// ... implement remaining methods
```

---

## Phase 3: Service Layer

Services contain business logic and depend ONLY on repository interfaces.

**File: `internal/service/user_service.go`**

```go
package service

import (
    "context"
    "fmt"
    "time"
    
    "github.com/google/uuid"
    
    "treblesurf-backend/internal/model"
    "treblesurf-backend/internal/repository"
)

type UserService struct {
    users repository.UserRepository
}

func NewUserService(users repository.UserRepository) *UserService {
    return &UserService{users: users}
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (*model.User, error) {
    user, err := s.users.GetByEmail(ctx, email)
    if err != nil {
        return nil, fmt.Errorf("getting user: %w", err)
    }
    return user, nil
}

func (s *UserService) CreateUser(ctx context.Context, email, name, picture string) (*model.User, error) {
    // Check if user exists
    existing, err := s.users.GetByEmail(ctx, email)
    if err != nil && err != model.ErrUserNotFound {
        return nil, fmt.Errorf("checking existing user: %w", err)
    }
    if existing != nil {
        return nil, model.ErrUserAlreadyExists
    }
    
    user := &model.User{
        UUID:      uuid.New().String(),
        Email:     email,
        Name:      name,
        Picture:   picture,
        CreatedAt: time.Now().UTC().Format(time.RFC3339),
        LastLogin: time.Now().UTC().Format(time.RFC3339),
        Theme:     model.DefaultTheme,
    }
    
    if err := s.users.Create(ctx, user); err != nil {
        return nil, fmt.Errorf("creating user: %w", err)
    }
    
    return user, nil
}

func (s *UserService) UpdateTheme(ctx context.Context, email, theme string) error {
    if !model.IsValidTheme(theme) {
        return model.ErrInvalidTheme
    }
    
    if err := s.users.UpdateTheme(ctx, email, theme); err != nil {
        return fmt.Errorf("updating theme: %w", err)
    }
    
    return nil
}
```

---

## Phase 4: Controller Layer

Controllers handle HTTP concerns and delegate to services.

**File: `internal/controller/user_controller.go`**

```go
package controller

import (
    "errors"
    "log/slog"
    "net/http"
    
    "github.com/gin-gonic/gin"
    
    "treblesurf-backend/internal/logging"
    "treblesurf-backend/internal/model"
    "treblesurf-backend/internal/service"
)

type UserController struct {
    users *service.UserService
}

func NewUserController(users *service.UserService) *UserController {
    return &UserController{users: users}
}

func (c *UserController) GetTheme(gc *gin.Context) {
    ctx := gc.Request.Context()
    log := logging.FromContext(ctx)
    
    email, ok := gc.Get("email")
    if !ok {
        gc.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
        return
    }
    
    user, err := c.users.GetByEmail(ctx, email.(string))
    if err != nil {
        log.Error("failed to get user", slog.String("error", err.Error()))
        handleError(gc, err)
        return
    }
    
    gc.JSON(http.StatusOK, gin.H{"theme": user.Theme})
}

func (c *UserController) SetTheme(gc *gin.Context) {
    ctx := gc.Request.Context()
    log := logging.FromContext(ctx)
    
    email, ok := gc.Get("email")
    if !ok {
        gc.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
        return
    }
    
    var req struct {
        Theme string `json:"theme" binding:"required"`
    }
    if err := gc.ShouldBindJSON(&req); err != nil {
        gc.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
        return
    }
    
    if err := c.users.UpdateTheme(ctx, email.(string), req.Theme); err != nil {
        log.Error("failed to update theme", slog.String("error", err.Error()))
        handleError(gc, err)
        return
    }
    
    gc.JSON(http.StatusOK, gin.H{"message": "theme updated"})
}

// handleError maps domain errors to HTTP responses
func handleError(gc *gin.Context, err error) {
    switch {
    case errors.Is(err, model.ErrUserNotFound):
        gc.JSON(http.StatusNotFound, ErrorResponse{
            Error:   "user_not_found",
            Message: "User not found",
        })
    case errors.Is(err, model.ErrInvalidTheme):
        gc.JSON(http.StatusBadRequest, ErrorResponse{
            Error:   "invalid_theme",
            Message: "Invalid theme value",
        })
    default:
        gc.JSON(http.StatusInternalServerError, ErrorResponse{
            Error: "internal_error",
        })
    }
}

type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message,omitempty"`
}
```

---

## Phase 5: Application Wiring

**File: `internal/app/app.go`**

```go
package app

import (
    "context"
    "net/http"
    
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/dynamodb"
    "github.com/aws/aws-sdk-go/service/rekognition"
    "github.com/aws/aws-sdk-go/service/s3"
    "github.com/gin-gonic/gin"
    
    "treblesurf-backend/internal/api"
    "treblesurf-backend/internal/config"
    "treblesurf-backend/internal/controller"
    "treblesurf-backend/internal/logging"
    repodynamo "treblesurf-backend/internal/repository/dynamodb"
    repos3 "treblesurf-backend/internal/repository/s3"
    "treblesurf-backend/internal/service"
)

type App struct {
    config  *config.Config
    router  *gin.Engine
    
    // AWS clients (for cleanup if needed)
    dynamoDB    *dynamodb.DynamoDB
    s3Client    *s3.S3
    rekognition *rekognition.Rekognition
}

func New(cfg *config.Config) (*App, error) {
    // Initialize logging
    logging.Init(cfg)
    
    // Create AWS session
    sess := session.Must(session.NewSession(&aws.Config{
        Region: aws.String(cfg.AWS.Region),
    }))
    
    // Create AWS clients
    dynamoClient := dynamodb.New(sess)
    s3Client := s3.New(sess)
    rekognitionClient := rekognition.New(sess)
    
    // Create repositories
    userRepo := repodynamo.NewUserRepo(dynamoClient, "Users")
    reportRepo := repodynamo.NewReportRepo(dynamoClient, "SurfReports")
    locationRepo := repodynamo.NewLocationRepo(dynamoClient, "LocationData")
    forecastRepo := repodynamo.NewForecastRepo(dynamoClient, "SpotForecastData")
    buoyRepo := repodynamo.NewBuoyRepo(dynamoClient, "BuoyData", "BuoyLocations")
    sessionRepo := repodynamo.NewSessionRepo(dynamoClient, "Sessions")
    mediaRepo := repos3.NewMediaRepo(s3Client, cfg.AWS.BucketName)
    apiKeyRepo := repodynamo.NewAPIKeyRepo(dynamoClient, "APIKeys")
    
    // Create services
    userService := service.NewUserService(userRepo)
    reportService := service.NewReportService(reportRepo, mediaRepo, userRepo, rekognitionClient)
    locationService := service.NewLocationService(locationRepo, mediaRepo)
    forecastService := service.NewForecastService(forecastRepo)
    buoyService := service.NewBuoyService(buoyRepo)
    authService := service.NewAuthService(userRepo, sessionRepo, cfg.Auth)
    apiKeyService := service.NewAPIKeyService(apiKeyRepo)
    
    // Create controllers
    userController := controller.NewUserController(userService)
    reportController := controller.NewReportController(reportService, userService)
    locationController := controller.NewLocationController(locationService)
    forecastController := controller.NewForecastController(forecastService)
    buoyController := controller.NewBuoyController(buoyService)
    authController := controller.NewAuthController(authService, userService)
    apiKeyController := controller.NewAPIKeyController(apiKeyService)
    
    // Create router
    router := api.NewRouter(cfg, api.Controllers{
        User:     userController,
        Report:   reportController,
        Location: locationController,
        Forecast: forecastController,
        Buoy:     buoyController,
        Auth:     authController,
        APIKey:   apiKeyController,
    })
    
    return &App{
        config:      cfg,
        router:      router,
        dynamoDB:    dynamoClient,
        s3Client:    s3Client,
        rekognition: rekognitionClient,
    }, nil
}

func (a *App) Handler() http.Handler {
    return a.router
}

func (a *App) GinEngine() *gin.Engine {
    return a.router
}

func (a *App) Shutdown(ctx context.Context) error {
    // Cleanup resources if needed
    // (AWS SDK clients don't need explicit cleanup)
    return nil
}
```

---

## Phase 6: Entry Points

### Lambda Entry Point

**File: `cmd/api/main.go`**

```go
package main

import (
    "context"
    "strings"
    
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
    
    "treblesurf-backend/internal/app"
    "treblesurf-backend/internal/config"
)

var ginLambda *ginadapter.GinLambda

func init() {
    cfg := config.MustLoad()
    application, err := app.New(cfg)
    if err != nil {
        panic(err)
    }
    ginLambda = ginadapter.New(application.GinEngine())
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    req.Path = strings.TrimPrefix(req.Path, "/api")
    return ginLambda.ProxyWithContext(ctx, req)
}

func main() {
    lambda.Start(Handler)
}
```

### Container Entry Point

**File: `cmd/server/main.go`**

```go
package main

import (
    "context"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "treblesurf-backend/internal/app"
    "treblesurf-backend/internal/config"
)

func main() {
    cfg := config.MustLoad()
    
    application, err := app.New(cfg)
    if err != nil {
        slog.Error("failed to create application", slog.String("error", err.Error()))
        os.Exit(1)
    }
    
    server := &http.Server{
        Addr:         ":" + cfg.Server.Port,
        Handler:      application.Handler(),
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }
    
    // Start server in goroutine
    go func() {
        slog.Info("starting server", slog.String("addr", server.Addr))
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            slog.Error("server error", slog.String("error", err.Error()))
            os.Exit(1)
        }
    }()
    
    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    slog.Info("shutting down server")
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        slog.Error("server shutdown error", slog.String("error", err.Error()))
    }
    
    if err := application.Shutdown(ctx); err != nil {
        slog.Error("application shutdown error", slog.String("error", err.Error()))
    }
    
    slog.Info("server stopped")
}
```

---

## Phase 7: Domain Errors

**File: `internal/model/errors.go`**

```go
package model

import "errors"

// User errors
var (
    ErrUserNotFound      = errors.New("user not found")
    ErrUserAlreadyExists = errors.New("user already exists")
    ErrInvalidTheme      = errors.New("invalid theme")
)

// Auth errors
var (
    ErrInvalidToken       = errors.New("invalid token")
    ErrTokenExpired       = errors.New("token expired")
    ErrSessionNotFound    = errors.New("session not found")
    ErrSessionExpired     = errors.New("session expired")
    ErrUnauthorized       = errors.New("unauthorized")
    ErrInvalidCredentials = errors.New("invalid credentials")
)

// Report errors
var (
    ErrReportNotFound      = errors.New("report not found")
    ErrInvalidReportData   = errors.New("invalid report data")
    ErrImageNotSurfRelated = errors.New("image does not appear to be surf-related")
    ErrImageUploadFailed   = errors.New("image upload failed")
    ErrVideoUploadFailed   = errors.New("video upload failed")
)

// Location errors
var (
    ErrLocationNotFound = errors.New("location not found")
    ErrInvalidLocation  = errors.New("invalid location parameters")
)

// Media errors
var (
    ErrMediaNotFound    = errors.New("media not found")
    ErrInvalidMediaKey  = errors.New("invalid media key")
    ErrMediaAccessDenied = errors.New("media access denied")
)

// API Key errors
var (
    ErrAPIKeyNotFound = errors.New("api key not found")
    ErrAPIKeyRevoked  = errors.New("api key revoked")
    ErrAPIKeyInvalid  = errors.New("api key invalid")
)
```

---

## Testing Example

**File: `internal/service/user_service_test.go`**

```go
package service_test

import (
    "context"
    "testing"
    
    "treblesurf-backend/internal/model"
    "treblesurf-backend/internal/repository/mock"
    "treblesurf-backend/internal/service"
)

func TestUserService_GetByEmail(t *testing.T) {
    tests := []struct {
        name      string
        email     string
        mockSetup func(*mock.UserRepo)
        want      *model.User
        wantErr   error
    }{
        {
            name:  "user found",
            email: "test@example.com",
            mockSetup: func(m *mock.UserRepo) {
                m.GetByEmailFn = func(ctx context.Context, email string) (*model.User, error) {
                    return &model.User{
                        Email: email,
                        Name:  "Test User",
                    }, nil
                }
            },
            want:    &model.User{Email: "test@example.com", Name: "Test User"},
            wantErr: nil,
        },
        {
            name:  "user not found",
            email: "notfound@example.com",
            mockSetup: func(m *mock.UserRepo) {
                m.GetByEmailFn = func(ctx context.Context, email string) (*model.User, error) {
                    return nil, model.ErrUserNotFound
                }
            },
            want:    nil,
            wantErr: model.ErrUserNotFound,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := &mock.UserRepo{}
            if tt.mockSetup != nil {
                tt.mockSetup(mockRepo)
            }
            
            svc := service.NewUserService(mockRepo)
            got, err := svc.GetByEmail(context.Background(), tt.email)
            
            if err != tt.wantErr {
                t.Errorf("GetByEmail() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if tt.want != nil && got.Email != tt.want.Email {
                t.Errorf("GetByEmail() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

---

## Summary of Changes

| Current | After Refactor |
|---------|----------------|
| Global `DB`, `S3Client` variables | Injected via constructors |
| Package-level service singletons | Service instances in App |
| Controllers as package functions | Controller structs with methods |
| Direct DynamoDB calls in services | Repository interface calls |
| No context propagation | Context flows through all layers |
| stdlib `log.Printf` | Structured `log/slog` |
| Scattered `os.Getenv` | Centralized config package |
| Lambda-only entry point | Lambda + HTTP server options |

---

## Implementation Order

1. **Foundation** (no dependencies)
   - `internal/config/config.go`
   - `internal/logging/logging.go`
   - `internal/model/errors.go`

2. **Repository Layer** (depends on: model)
   - `internal/repository/interfaces.go`
   - `internal/repository/dynamodb/*.go`
   - `internal/repository/s3/*.go`
   - `internal/repository/mock/*.go`

3. **Service Layer** (depends on: repository, model)
   - `internal/service/*.go`

4. **Controller Layer** (depends on: service, model)
   - `internal/controller/*.go`

5. **App & Routing** (depends on: everything)
   - `internal/api/router.go`
   - `internal/app/app.go`

6. **Entry Points**
   - `cmd/api/main.go`
   - `cmd/server/main.go`

7. **Cleanup**
   - Remove old files
   - Update tests
