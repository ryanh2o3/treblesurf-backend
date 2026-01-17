# Treblesurf Backend - Architecture Overview

A comprehensive analysis of the current application state, architecture patterns, and Go best practices compliance.

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Application Overview](#application-overview)
3. [Technology Stack](#technology-stack)
4. [Project Structure](#project-structure)
5. [Architecture Patterns](#architecture-patterns)
6. [Data Flow](#data-flow)
7. [Domain Model](#domain-model)
8. [What's Good](#whats-good)
9. [What Needs Improvement](#what-needs-improvement)
10. [Go Best Practices Scorecard](#go-best-practices-scorecard)
11. [Security Analysis](#security-analysis)
12. [Scalability Considerations](#scalability-considerations)

---

## Executive Summary

**Treblesurf Backend** is a Go-based REST API serving a surf conditions tracking application. It runs on AWS Lambda and provides endpoints for:

- User authentication (Google OAuth)
- Surf report submission with image/video validation
- Location and forecast data retrieval
- Buoy data analysis and historical matching
- Real-time WebSocket notifications

### Quick Assessment

| Area                  | Rating   | Notes                                           |
| --------------------- | -------- | ----------------------------------------------- |
| Project Structure     | ⭐⭐⭐⭐ | Good use of `cmd/` and `internal/`              |
| Dependency Management | ⭐⭐⭐   | Partial DI, global state issues                 |
| Error Handling        | ⭐⭐⭐   | Custom errors exist, inconsistent usage         |
| Testing               | ⭐⭐     | Global state makes testing difficult            |
| Documentation         | ⭐⭐⭐   | Package docs exist, could be more comprehensive |
| Security              | ⭐⭐⭐⭐ | Good auth patterns, CSRF protection             |

---

## Application Overview

### Purpose

Treblesurf is a crowdsourced surf conditions platform that allows surfers to:

1. Submit real-time surf reports with photos/videos
2. View current conditions at surf spots
3. Get forecast data and predictions
4. See historical reports with similar conditions
5. Receive real-time updates via WebSocket

### Entry Points

| Entry Point | Location                | Purpose                              |
| ----------- | ----------------------- | ------------------------------------ |
| HTTP API    | `cmd/api/main.go`       | Lambda handler for REST API          |
| WebSocket   | `cmd/websocket/main.go` | Lambda handler for real-time updates |
| Local Dev   | `local/cmd/server.go`   | HTTP server for development          |

### Deployment Model

```
┌─────────────────────────────────────────────────────────────────┐
│                        AWS Lambda                                │
├─────────────────────────────────────────────────────────────────┤
│  API Gateway ──► HTTP Lambda ──► Gin Router ──► Controllers     │
│                                                                  │
│  API Gateway ──► WS Lambda ──► WebSocket Handler                │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                      AWS Services                                │
├─────────────────┬─────────────────┬─────────────────────────────┤
│    DynamoDB     │       S3        │      Rekognition            │
│  (all tables)   │  (media files)  │  (image validation)         │
└─────────────────┴─────────────────┴─────────────────────────────┘
```

---

## Technology Stack

### Core

| Component      | Technology              | Version     |
| -------------- | ----------------------- | ----------- |
| Language       | Go                      | 1.24.0      |
| HTTP Framework | Gin                     | 1.10.1      |
| Lambda Adapter | aws-lambda-go-api-proxy | 0.16.2      |
| AWS SDK        | aws-sdk-go              | 1.55.6 (v1) |

### AWS Services Used

| Service     | Purpose                                                      |
| ----------- | ------------------------------------------------------------ |
| DynamoDB    | Primary database (Users, Reports, Forecasts, Sessions, etc.) |
| S3          | Media storage (images, videos)                               |
| Rekognition | Image validation (surf-related content detection)            |
| API Gateway | HTTP/WebSocket routing                                       |
| Lambda      | Compute                                                      |

### Key Dependencies

```go
// Core
github.com/gin-gonic/gin              // HTTP framework
github.com/aws/aws-lambda-go          // Lambda runtime
github.com/awslabs/aws-lambda-go-api-proxy  // Gin-Lambda adapter

// Auth
github.com/golang-jwt/jwt             // JWT handling (v3 - outdated)
github.com/golang-jwt/jwt/v5          // JWT handling (v5 - also present)
github.com/adam-hanna/sessions        // Session management
google.golang.org/api/idtoken         // Google OAuth validation

// AWS
github.com/aws/aws-sdk-go             // AWS SDK v1
```

⚠️ **Note:** Both `jwt` v3 and v5 are in dependencies - should standardize on v5.

---

## Project Structure

### Current Layout

```
treblesurf-backend/
├── cmd/
│   ├── api/
│   │   └── main.go              ✅ Lambda HTTP entry point
│   └── websocket/
│       └── main.go              ✅ Lambda WebSocket entry point
│
├── internal/
│   ├── api/
│   │   ├── container.go         ✅ DI container
│   │   ├── middleware.go        ✅ HTTP middleware
│   │   └── router.go            ✅ Route registration
│   │
│   ├── auth/
│   │   ├── middleware.go        ✅ Auth middleware
│   │   ├── service.go           ⚠️ Has global state
│   │   └── service_helpers.go
│   │
│   ├── constants/
│   │   └── constants.go         ✅ App constants
│   │
│   ├── controller/
│   │   ├── *_controller.go      ✅ Controller structs
│   │   └── *_helpers.go
│   │
│   ├── model/
│   │   ├── user.go              ✅ Domain types
│   │   ├── report.go            ✅ Domain types
│   │   ├── errors.go            ✅ Custom errors
│   │   └── *.go
│   │
│   ├── service/
│   │   ├── report_service.go    ⚠️ 1600+ lines, needs splitting
│   │   └── *.go                 ⚠️ Mix of interfaces and concrete types
│   │
│   ├── storage/
│   │   ├── dynamodb.go          ✅ Has interface
│   │   └── s3.go                ✅ Has interface
│   │
│   ├── validation/
│   │   └── surfValidation.go    ✅ Input validation
│   │
│   └── websocket/
│       ├── handler.go           ✅ WebSocket handling
│       └── handler_helpers.go
│
├── local/
│   ├── cmd/server.go            ✅ Local dev server
│   ├── config/config.go         ⚠️ Local-only config
│   ├── storage/                 ✅ Local DynamoDB/S3 mocks
│   └── scripts/
│
├── scripts/
│   └── deploy.sh
│
├── Dockerfile                   ❌ Python Flask! Not Go!
├── docker-compose.yml
├── go.mod
└── Makefile
```

### Structure Issues

| Issue                      | Location                                  | Severity |
| -------------------------- | ----------------------------------------- | -------- |
| Dockerfile is Python Flask | `Dockerfile`                              | High     |
| Global state               | Multiple locations                        | High     |
| Large files                | `report_service.go` (1600+ lines)         | Medium   |

---

## Architecture Patterns

### Current Pattern: Modified MVC

```
┌──────────────────────────────────────────────────────────────────┐
│                         Router Layer                              │
│                    (internal/api/router.go)                       │
│         Routes HTTP requests to controller functions              │
└─────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                       Controller Layer                            │
│                   (internal/controller/*.go)                      │
│    Controller structs with injected services                      │
└─────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                        Service Layer                              │
│                    (internal/service/*.go)                        │
│    Business logic - some use interfaces, some use concrete        │
│    DynamoDB client directly                                       │
└─────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                        Storage Layer                              │
│                    (internal/storage/*.go)                        │
│         Interfaces defined, but inconsistently used               │
└──────────────────────────────────────────────────────────────────┘
```

### Dependency Flow (Current)

```go
// How dependencies currently flow:

// 1. main.go creates container
container, _ := httphandler.NewContainer()

// 2. Container creates services
services := &containerServices{
    forecastService: service.NewForecastService(storage.dynamoDBClient),
    // ...
}

// 3. But then sets them as GLOBALS
controller.SetUserService(services.userService)
controller.SetReportService(services.reportService)

// 4. Controllers access via globals
func SubmitCurrentSurfReport(c *gin.Context) {
    ReportService.SubmitSurfReport(...)  // Global!
}
```

### Problems with Current Approach

1. **Testing is difficult** - Can't inject mocks
2. **Hidden dependencies** - Not clear what a controller needs
3. **Race conditions possible** - During initialization
4. **Harder to reason about** - State scattered across packages

---

## Data Flow

### Surf Report Submission

```
┌────────────┐     ┌────────────┐     ┌───────────────┐     ┌──────────┐
│   Client   │────►│ Controller │────►│ ReportService │────►│ DynamoDB │
│  (iOS App) │     │            │     │               │     │          │
└────────────┘     └────────────┘     └───────┬───────┘     └──────────┘
                                              │
                                              ▼
                                      ┌───────────────┐     ┌──────────┐
                                      │ Rekognition   │     │    S3    │
                                      │ (validate)    │     │ (store)  │
                                      └───────────────┘     └──────────┘
```

### Authentication Flow

```
┌────────────┐     ┌────────────┐     ┌───────────────┐
│  Google    │     │    Auth    │     │   DynamoDB    │
│  Sign-In   │────►│  Service   │────►│   (Users)     │
└────────────┘     └─────┬──────┘     └───────────────┘
                         │
                         ▼
                  ┌───────────────┐     ┌───────────────┐
                  │    Session    │────►│   DynamoDB    │
                  │   Service     │     │  (Sessions)   │
                  └───────────────┘     └───────────────┘
```

---

## Domain Model

### DynamoDB Tables

| Table                | Partition Key       | Sort Key           | Purpose               |
| -------------------- | ------------------- | ------------------ | --------------------- |
| Users                | email               | -                  | User accounts         |
| Sessions             | session_id          | -                  | User sessions         |
| SurfReports          | country_region_spot | date_reported      | Surf reports          |
| LocationData         | country_region_spot | -                  | Spot information      |
| SpotForecastData     | spot_id             | forecast_timestamp | Weather forecasts     |
| BuoyData             | region_buoy         | dataDateTime       | Buoy measurements     |
| BuoyLocations        | region_buoy         | -                  | Buoy positions        |
| WebSocketConnections | connection_id       | -                  | Active WS connections |
| APIKeys              | key_id              | -                  | API keys              |

### Core Domain Types

```go
// User - central entity
type User struct {
    UUID       string
    Email      string  // Primary key
    Name       string
    Theme      string
    Role       string  // "admin" or empty
    CreatedAt  string
    LastLogin  string
}

// SurfReport - user-submitted conditions
type SurfReport struct {
    CountryRegionSpot string  // PK: "Ireland_Donegal_Bundoran"
    DateReported      string  // SK: "2024-01-15T10:30:00Z_uuid"
    SurfSize          string
    WindAmount        string
    WindDirection     string
    ImageKey          string  // S3 key
    VideoKey          string  // S3 key
    UserUUID          string
}

// LocationInfo - surf spot metadata
type LocationInfo struct {
    CountryRegionSpot   string
    Latitude            float64
    Longitude           float64
    BeachDirection      int
    IdealSwellDirection string
}
```

---

## What's Good

### ✅ Project Layout

- Proper use of `cmd/` for entry points
- `internal/` package prevents external imports
- Clear separation: `controller/`, `service/`, `model/`, `storage/`
- Separate `local/` for development tooling

### ✅ Interface Design (Partial)

Storage interfaces exist and are well-designed:

```go
// internal/storage/dynamodb.go
type DynamoDBStorage interface {
    Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error)
    Query(input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
    GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error)
    PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error)
    UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error)
    DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error)
}
```

### ✅ Custom Error Types

Good start on domain errors:

```go
// internal/model/errors.go
var (
    ErrImageNotSurfRelated   = errors.New("image does not appear to be surf-related")
    ErrImageAnalysisFailed   = errors.New("image analysis failed")
    ErrImageUploadFailed     = errors.New("image upload failed")
)

type ImageValidationError struct {
    Err     error
    Message string
}

func (e *ImageValidationError) Unwrap() error { return e.Err }
```

### ✅ Authentication & Security

- Google OAuth integration
- Session-based auth with DynamoDB storage
- CSRF protection for state-changing operations
- API key authentication for external services
- Admin role checking

### ✅ Input Validation

Dedicated validation package:

```go
// internal/validation/surfValidation.go
func IsValidSurfSize(size string) bool
func IsValidWindAmount(wind string) bool
func IsValidWindDirection(dir string) bool
```

### ✅ Container Pattern (Started)

The `Container` struct in `container.go` shows intent for proper DI:

```go
type Container struct {
    DynamoDBStorage        storagepkg.DynamoDBStorage
    S3Storage              storagepkg.S3Storage
    ForecastService        *service.ForecastService
    ForecastController     *controller.ForecastController
    // ...
}
```

### ✅ Package Documentation

Most packages have doc comments:

```go
// Package httphandler provides HTTP API routing configuration,
// route registration, and middleware setup for the Treble Surf backend API.
package httphandler
```

### ✅ Error Wrapping

Good use of `%w` in many places:

```go
return nil, fmt.Errorf("failed to query reports: %w", err)
```

### ✅ Development/Production Separation

Clear environment handling:

```go
isLocal := os.Getenv("GO_ENV") == constants.EnvDevelopment
if isLocal {
    authorized.Use(DevAuthMiddleware())
} else {
    authorized.Use(auth.Middleware())
}
```

---

## What Needs Improvement

### ❌ Global State Everywhere

**Problem:** Multiple global variable declarations scattered across packages.

```go
// internal/auth/service.go
var jwtSecret []byte                          // GLOBAL
var db *dynamodb.DynamoDB                     // GLOBAL
var sessionService *sessions.Service         // GLOBAL
```

**Impact:**

- Testing requires global state manipulation
- Race conditions during initialization
- Hidden dependencies
- Harder to trace code flow

### ❌ Inconsistent Interface Usage

**Problem:** Some services use interfaces, others use concrete types.

```go
// Good: Uses interface
type ReportService struct {
    dbStorage storage.DynamoDBStorage  // Interface ✅
    s3Storage storage.S3Storage        // Interface ✅
}

// Bad: Uses concrete type
type ForecastService struct {
    db *dynamodb.DynamoDB              // Concrete ❌
}

type UserService struct {
    db *dynamodb.DynamoDB              // Concrete ❌
}
```

### ❌ Controllers as Package Functions

**Problem:** Controllers are package-level functions, not struct methods.

```go
// Current: Package function with implicit dependencies
func SubmitCurrentSurfReport(c *gin.Context) {
    if err := ReportService.SubmitSurfReport(...); err != nil {  // Global!
        // ...
    }
}

// Better: Struct method with explicit dependencies
func (rc *ReportController) SubmitReport(c *gin.Context) {
    if err := rc.reports.Submit(...); err != nil {  // Injected
        // ...
    }
}
```

### ❌ Missing Context Propagation

**Problem:** `context.Context` not passed through the call stack.

```go
// Current: No context
func (s *ReportService) GetTodaysSurfReports(
    countryName, regionName, spotName string,
) ([]map[string]interface{}, error)

// Better: With context
func (s *ReportService) GetTodaysSurfReports(
    ctx context.Context,
    countryName, regionName, spotName string,
) ([]map[string]interface{}, error)
```

### ❌ No Structured Logging

**Problem:** Using `log.Printf` throughout.

```go
// Current
log.Printf("🗑️ [CLEANUP] Deleting media: %s", mediaKey)
log.Printf("Broadcasting message to %d subscribers: %v", len(subscribers), message)

// Better: Structured logging
slog.Info("deleting media", slog.String("key", mediaKey))
slog.Info("broadcasting", slog.Int("subscribers", len(subscribers)))
```

### ❌ Dockerfile is Wrong Language

**Problem:** `Dockerfile` is for Python Flask, not Go.

```dockerfile
# Current (WRONG)
FROM python:3.8-slim-buster
RUN pip install flask
CMD ["gunicorn", "surfable_flask:APP"...]
```

### ❌ Duplicate Type Definitions

**Problem:** `User` struct defined twice.

```go
// internal/model/user.go
type User struct {
    UUID       string `json:"uuid"`
    Email      string `json:"email"`
    // ...
}

// internal/auth/service.go (DUPLICATE)
type User struct {
    UUID       string `json:"uuid"`
    Email      string `json:"email"`
    // ...
}
```

### ❌ Large Files

**Problem:** `report_service.go` is 1600+ lines.

Should be split into:

- `report_service.go` - Core CRUD
- `report_service_media.go` - Image/video handling
- `report_service_similarity.go` - Buoy matching logic

### ❌ Scattered Configuration

**Problem:** `os.Getenv()` calls throughout codebase.

```go
// Found in multiple files:
region := os.Getenv("AWS_REGION")
jwtSecret := os.Getenv("JWT_SECRET")
endpoint := os.Getenv("WEBSOCKET_API_ENDPOINT")
bucketName := os.Getenv("S3_BUCKET_NAME")
```

### ❌ Mixed JWT Versions

**Problem:** Both v3 and v5 in dependencies.

```go
// go.mod shows both:
github.com/golang-jwt/jwt v3.2.2+incompatible
github.com/golang-jwt/jwt/v5 v5.2.2
```

---

## Go Best Practices Scorecard

| Practice                     | Status | Details                                 |
| ---------------------------- | ------ | --------------------------------------- |
| `cmd/` for entry points      | ✅     | Properly organized                      |
| `internal/` for private code | ✅     | Correct usage                           |
| No `init()` for globals      | ❌     | Uses `init()` in Lambda main            |
| Depend on interfaces         | ⚠️     | Partial - storage yes, some services no |
| Constructor injection        | ⚠️     | Started but bypassed with globals       |
| Context propagation          | ❌     | Not implemented                         |
| Structured logging           | ❌     | Using `log.Printf`                      |
| Error wrapping with `%w`     | ⚠️     | Inconsistent                            |
| No package-level state       | ❌     | Multiple global vars                    |
| Small interfaces             | ✅     | Storage interface is good               |
| Table-driven tests           | ❓     | No tests visible                        |
| Graceful shutdown            | ❌     | Lambda doesn't need, but server does    |
| Config validation at startup | ⚠️     | Partial                                 |

---

## Security Analysis

### ✅ Strengths

| Feature                   | Implementation                     |
| ------------------------- | ---------------------------------- |
| Authentication            | Google OAuth + session cookies     |
| CSRF Protection           | Token-based for state-changing ops |
| API Key Auth              | Scope-based validation             |
| Admin Authorization       | Role checking middleware           |
| Path Traversal Prevention | Media key validation               |

```go
// Good: Path traversal prevention
func isValidMediaKey(mediaKey string) bool {
    if strings.Contains(mediaKey, "..") || strings.Contains(mediaKey, "//") {
        return false
    }
    // ...
}
```

### ⚠️ Concerns

| Issue                     | Risk   | Location                  |
| ------------------------- | ------ | ------------------------- |
| Hardcoded admin list      | Medium | `middleware.go`           |
| Default JWT secret in dev | Low    | `local/cmd/server.go`     |
| CORS `*` in production    | Low    | `router.go` (for iOS PWA) |

```go
// Hardcoded admin - should be in database or config
func isAdminUser(email string) bool {
    adminUsers := map[string]bool{
        "ryancpatton0@gmail.com": true,  // Hardcoded!
    }
    return adminUsers[email]
}
```

---

## Scalability Considerations

### Current Strengths

- **Serverless:** Lambda scales automatically
- **DynamoDB:** Scales horizontally
- **S3:** Unlimited storage
- **Stateless:** No in-memory state between requests

### Potential Bottlenecks

| Area                 | Concern                                 | Mitigation          |
| -------------------- | --------------------------------------- | ------------------- |
| Cold starts          | Lambda initialization takes time        | Keep functions warm |
| DynamoDB scans       | Some queries use Scan                   | Add GSIs            |
| Large report queries | GetSurfReportsWithSimilarBuoyData scans | Pagination, caching |
| Image validation     | Rekognition API call per image          | Batch processing    |

### If Moving to Containers

Would need to add:

1. Connection pooling (though DynamoDB SDK handles this)
2. Graceful shutdown
3. Health check endpoint
4. Horizontal scaling configuration
5. Load balancer

---

## Summary

### The Good

- Solid project structure following Go conventions
- Storage layer interface design
- Authentication/authorization implementation
- Input validation
- Environment-aware configuration

### The Bad

- Global state everywhere
- Inconsistent dependency injection
- Missing context propagation
- No structured logging
- Wrong Dockerfile

### The Ugly

- 1600+ line service file
- Duplicate type definitions
- Mixed JWT library versions

### Priority Actions

1. **High:** Remove global state, complete DI
2. **High:** Fix Dockerfile (or remove if Lambda-only)
3. **Medium:** Add context propagation
4. **Medium:** Switch to structured logging
5. **Low:** Split large files
6. **Low:** Clean up duplicates

---

_Generated: Analysis of treblesurf-backend current state_
