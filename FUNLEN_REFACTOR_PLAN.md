# Function Length (funlen) Refactoring Plan

This document outlines a comprehensive plan to refactor all 22 functions that exceed the funlen limits, following modern Go best practices.

## funlen Limits
- **Default**: 40 statements for regular functions
- **Stricter**: 60 statements for functions with higher complexity

## Refactoring Principles

1. **Single Responsibility Principle**: Each function should do one thing well
2. **Extract Helper Functions**: Identify logical blocks and extract them
3. **Preserve Functionality**: All refactoring must maintain existing behavior
4. **Improve Testability**: Smaller functions are easier to test
5. **Follow Go Conventions**: 
   - Use descriptive function names
   - Keep functions focused and cohesive
   - Group related helper functions near their usage
   - Use package-private helpers (lowercase) when appropriate

---

## Functions to Refactor (22 total)

### Category 1: Route Setup & Configuration (2 functions)

#### 1.1 `internal/api/router.go:53` - `setupRoutes` (80 statements)

**Current Issues**:
- Handles multiple concerns: middleware setup, route grouping, route registration
- Mixes public routes, authenticated routes, and development routes

**Refactoring Strategy**:
```go
// Extract route setup into logical groups
func setupRoutes(r gin.IRouter, container *Container) {
    setupMiddleware(r)
    setupPublicRoutes(r)
    setupAuthRoutes(r, container)
    setupLocationRoutes(r, container)
    setupForecastRoutes(r, container)
    setupReportRoutes(r, container)
    setupAdminRoutes(r, container)
    setupDevRoutes(r)
}

// Helper functions:
func setupMiddleware(r gin.IRouter)
func setupPublicRoutes(r gin.IRouter)
func setupAuthRoutes(r gin.IRouter, container *Container)
func setupLocationRoutes(r gin.IRouter, container *Container)
func setupForecastRoutes(r gin.IRouter, container *Container)
func setupReportRoutes(r gin.IRouter, container *Container)
func setupAdminRoutes(r gin.IRouter, container *Container)
func setupDevRoutes(r gin.IRouter)
```

**Benefits**:
- Clear separation of concerns
- Easier to maintain and test
- Better organization
- No duplication, each route group is unique

---

#### 1.2 `internal/api/container.go:47` - `NewContainer` (50 statements)

**Current Issues**:
- Initializes multiple services in a single function
- Mixes local vs production logic

**Refactoring Strategy**:
```go
func NewContainer() (*Container, error) {
    cfg, err := loadConfig()
    if err != nil {
        return nil, err
    }
    
    isLocal := cfg.IsLocal()
    
    storage, err := initializeStorage(cfg, isLocal)
    if err != nil {
        return nil, err
    }
    
    services, err := initializeServices(storage, cfg, isLocal)
    if err != nil {
        return nil, err
    }
    
    controllers := initializeControllers(services)
    
    setupGlobalDependencies(storage, services, isLocal)
    
    return buildContainer(storage, services, controllers), nil
}

// Helper functions:
func loadConfig() (*config, error)
func initializeStorage(cfg *config, isLocal bool) (*storage, error)
func initializeServices(storage *storage, cfg *config, isLocal bool) (*services, error)
func initializeControllers(services *services) *controllers
func setupGlobalDependencies(storage *storage, services *services, isLocal bool)
func buildContainer(storage *storage, services *services, controllers *controllers) *Container
```

**Benefits**:
- Clear initialization phases
- Easier to test each phase
- Better error handling per phase
- No duplication, logical separation

---

### Category 2: Authentication Handlers (2 functions)

#### 2.1 `internal/auth/service.go:503` - `GoogleAuthHandler` (70 statements)

**Current Issues**:
- Handles validation, token creation, session creation, user creation, response formatting
- Mixes multiple concerns in one handler

**Refactoring Strategy**:
```go
func GoogleAuthHandler(c *gin.Context) {
    req, err := parseTokenRequest(c)
    if err != nil {
        respondError(c, http.StatusBadRequest, err)
        return
    }
    
    token, err := validateGoogleToken(req.IDToken)
    if err != nil {
        respondError(c, http.StatusUnauthorized, err)
        return
    }
    
    user, err := getOrCreateUser(token)
    if err != nil {
        respondError(c, http.StatusInternalServerError, err)
        return
    }
    
    session, err := createUserSession(user, c)
    if err != nil {
        respondError(c, http.StatusInternalServerError, err)
        return
    }
    
    respondWithSession(c, user, session)
}

// Helper functions:
func parseTokenRequest(c *gin.Context) (*TokenRequest, error)
func validateGoogleToken(idToken string) (*idtoken.Payload, error)
func getOrCreateUser(token *idtoken.Payload) (*User, error)
func createUserSession(user *User, c *gin.Context) (*Session, error)
func respondWithSession(c *gin.Context, user *User, session *Session)
func respondError(c *gin.Context, status int, err error)
```

**Benefits**:
- Clear request flow
- Each step is testable
- Better error handling
- Reusable helper functions

---

#### 2.2 `internal/auth/service.go:696` - `ValidateTokenHandler` (81 statements)

**Current Issues**:
- Handles token validation, user retrieval, session validation, response formatting
- Multiple conditional branches

**Refactoring Strategy**:
```go
func ValidateTokenHandler(c *gin.Context) {
    token, err := extractToken(c)
    if err != nil {
        respondUnauthorized(c)
        return
    }
    
    user, err := getUserFromToken(token)
    if err != nil {
        respondUnauthorized(c)
        return
    }
    
    session := getCurrentSession(c)
    response := buildValidationResponse(user, session)
    
    respondSuccess(c, response)
}

// Helper functions:
func extractToken(c *gin.Context) (string, error)
func getUserFromToken(token string) (*User, error)
func getCurrentSession(c *gin.Context) *Session
func buildValidationResponse(user *User, session *Session) map[string]interface{}
func respondUnauthorized(c *gin.Context)
func respondSuccess(c *gin.Context, data interface{})
```

**Benefits**:
- Simplified flow
- Clear separation of concerns
- Reusable authentication helpers

---

### Category 3: Controller Handlers (6 functions)

#### 3.1-3.6 Controller Functions Pattern

**General Strategy for Controllers**:
- Extract validation logic
- Extract response formatting
- Extract business logic calls
- Extract error handling

**Functions**:
1. `internal/controller/report_controller.go:67` - `SubmitCurrentSurfReport` (53 statements)
2. `internal/controller/report_controller.go:198` - `SubmitSurfReportWithS3Image` (50 statements)
3. `internal/controller/report_controller.go:468` - `GetSurfReportsWithSimilarBuoyData` (76 statements)
4. `internal/controller/report_controller.go:747` - `GenerateVideoViewURL` (71 statements)
5. `internal/controller/report_controller.go:825` - `SubmitSurfReportWithIOSValidation` (67 statements)
6. `internal/controller/report_controller.go:1004` - `DeleteUploadedMedia` (96 statements)
7. `internal/controller/snapshot_controller.go:22` - `UploadSnapshotHandler` (50 statements)

**Common Refactoring Pattern**:
```go
// Before: Long handler function
func HandlerFunction(c *gin.Context) {
    // 50+ statements mixing validation, business logic, error handling
}

// After: Extracted handlers
func HandlerFunction(c *gin.Context) {
    req, err := parseRequest(c)
    if err != nil {
        respondError(c, http.StatusBadRequest, err)
        return
    }
    
    result, err := service.Method(req)
    if err != nil {
        handleServiceError(c, err)
        return
    }
    
    respondSuccess(c, result)
}

// Helper functions (shared across controllers):
func parseRequest(c *gin.Context) (*Request, error)
func handleServiceError(c *gin.Context, err error)
func respondError(c *gin.Context, status int, err error)
func respondSuccess(c *gin.Context, data interface{})
```

**Specific Extractions Needed**:
- **SubmitCurrentSurfReport**: Extract parameter validation, report building, service call, response formatting
- **SubmitSurfReportWithS3Image**: Extract image validation, S3 handling, report creation
- **GetSurfReportsWithSimilarBuoyData**: Extract parameter parsing, query building, response formatting
- **GenerateVideoViewURL**: Extract validation, URL generation, response formatting
- **SubmitSurfReportWithIOSValidation**: Extract iOS validation, report processing
- **DeleteUploadedMedia**: Extract media type detection, deletion logic, response formatting
- **UploadSnapshotHandler**: Extract file handling, upload logic, response formatting

**Benefits**:
- Consistent error handling pattern
- Reusable validation helpers
- Easier to test
- Better maintainability

---

### Category 4: Service Layer Functions (5 functions)

#### 4.1-4.5 Service Functions Pattern

**Functions**:
1. `internal/service/report_service.go:47` - `SubmitSurfReport` (62 statements)
2. `internal/service/report_service.go:221` - `SubmitSurfReportWithS3Image` (46 statements)
3. `internal/service/report_service.go:469` - `GetSpotSurfReports` (66 statements)
4. `internal/service/report_service.go:629` - `SubmitSurfReportWithIOSValidation` (50 statements)
5. `internal/service/report_service.go:1591` - `getCurrentWindConditions` (79 statements)

**General Strategy**:
- Extract validation logic
- Extract database operations
- Extract business logic computations
- Extract response building

**Specific Refactoring Examples**:

**SubmitSurfReport**:
```go
func (s *ReportService) SubmitSurfReport(report *model.SurfReport) error {
    if err := validateReport(report); err != nil {
        return err
    }
    
    if err := enrichReport(report); err != nil {
        return err
    }
    
    if err := saveReport(report); err != nil {
        return err
    }
    
    notifySubscribers(report)
    return nil
}

// Helper functions:
func (s *ReportService) validateReport(report *model.SurfReport) error
func (s *ReportService) enrichReport(report *model.SurfReport) error
func (s *ReportService) saveReport(report *model.SurfReport) error
func (s *ReportService) notifySubscribers(report *model.SurfReport)
```

**GetSpotSurfReports**:
```go
func (s *ReportService) GetSpotSurfReports(...) ([]map[string]interface{}, error) {
    query := buildReportQuery(...)
    items, err := s.executeQuery(query)
    if err != nil {
        return nil, err
    }
    
    reports := transformItems(items)
    return enrichReports(reports), nil
}

// Helper functions:
func buildReportQuery(...) *dynamodb.QueryInput
func (s *ReportService) executeQuery(query *dynamodb.QueryInput) ([]map[string]*dynamodb.AttributeValue, error)
func transformItems(items []map[string]*dynamodb.AttributeValue) []map[string]interface{}
func (s *ReportService) enrichReports(reports []map[string]interface{}) []map[string]interface{}
```

**getCurrentWindConditions**:
```go
func (s *ReportService) getCurrentWindConditions(...) (map[string]interface{}, error) {
    forecast, err := s.getCurrentForecast(...)
    if err != nil {
        return nil, err
    }
    
    conditions := extractWindConditions(forecast)
    return formatWindConditions(conditions), nil
}

// Helper functions:
func (s *ReportService) getCurrentForecast(...) (map[string]interface{}, error)
func extractWindConditions(forecast map[string]interface{}) *WindConditions
func formatWindConditions(conditions *WindConditions) map[string]interface{}
```

**Benefits**:
- Clear business logic flow
- Testable components
- Reusable logic
- Better error handling

---

### Category 5: WebSocket Handler (1 function)

#### 5.1 `internal/websocket/handler.go:47` - `handleConnect` (61 statements)

**Refactoring Strategy**:
```go
func (h *WebSocketHandler) handleConnect(request events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
    token, err := extractToken(request)
    if err != nil {
        return unauthorizedResponse(), nil
    }
    
    claims, err := validateToken(token)
    if err != nil {
        return unauthorizedResponse(), nil
    }
    
    connectionID := request.RequestContext.ConnectionID
    if err := h.registerConnection(connectionID, claims.Email); err != nil {
        return errorResponse(err), nil
    }
    
    return successResponse(), nil
}

// Helper functions:
func extractToken(request events.APIGatewayWebsocketProxyRequest) (string, error)
func validateToken(token string) (*jwt.MapClaims, error)
func (h *WebSocketHandler) registerConnection(connectionID, userEmail string) error
func unauthorizedResponse() events.APIGatewayProxyResponse
func errorResponse(err error) events.APIGatewayProxyResponse
func successResponse() events.APIGatewayProxyResponse
```

**Benefits**:
- Clear connection flow
- Testable authentication
- Reusable token handling

---

### Category 6: Swell Prediction Service (1 function)

#### 6.1 `internal/service/swell_prediction_service.go:292` - `GetClosestAIPredictionForSpot` (54 statements)

**Refactoring Strategy**:
```go
func (s *SwellPredictionService) GetClosestAIPredictionForSpot(...) (map[string]interface{}, error) {
    query := buildPredictionQuery(...)
    
    result, err := s.executeQuery(query)
    if err != nil {
        return nil, err
    }
    
    if len(result.Items) == 0 {
        return s.fallbackQuery(...)
    }
    
    prediction := findClosestPrediction(result.Items, ...)
    return transformPrediction(prediction), nil
}

// Helper functions:
func buildPredictionQuery(...) *dynamodb.QueryInput
func (s *SwellPredictionService) executeQuery(query *dynamodb.QueryInput) (*dynamodb.QueryOutput, error)
func (s *SwellPredictionService) fallbackQuery(...) (map[string]interface{}, error)
func findClosestPrediction(items []map[string]*dynamodb.AttributeValue, ...) map[string]*dynamodb.AttributeValue
func transformPrediction(item map[string]*dynamodb.AttributeValue) map[string]interface{}
```

**Benefits**:
- Clear query logic
- Separated fallback logic
- Testable components

---

### Category 7: Local Development Scripts (3 functions)

**Functions**:
1. `local/scripts/seed_data.go:49` - `seedRealForecastData` (70 statements)
2. `local/scripts/seed_data.go:700` - `seedBuoyLocationData` (63 statements)
3. `local/scripts/seed_data.go:766` - `seedBuoyMeasurements` (42 statements)

**Strategy**: These are scripts, not production code. Refactor to:
- Extract data generation logic
- Extract batch insertion logic
- Extract error handling

**Example Pattern**:
```go
func seedRealForecastData() error {
    data := generateForecastData()
    
    batches := createBatches(data, batchSize)
    for _, batch := range batches {
        if err := insertBatch(batch); err != nil {
            return err
        }
    }
    
    return nil
}

// Helper functions:
func generateForecastData() []ForecastItem
func createBatches(data []ForecastItem, size int) [][]ForecastItem
func insertBatch(batch []ForecastItem) error
```

**Note**: Scripts are lower priority, but refactoring improves maintainability.

---

### Category 8: Local Storage (1 function)

#### 8.1 `local/storage/dynamodb.go:88` - `createLocalTables` (200 statements)

**Current Issues**: 
- Extremely long function (200 statements!)
- Creates multiple tables with similar patterns

**Refactoring Strategy**:
```go
func createLocalTables() error {
    tables := []tableDefinition{
        newUsersTable(),
        newSurfReportsTable(),
        newForecastTable(),
        // ... other tables
    }
    
    for _, table := range tables {
        if err := createTable(table); err != nil {
            return fmt.Errorf("failed to create %s: %w", table.name, err)
        }
    }
    
    return nil
}

// Helper types and functions:
type tableDefinition struct {
    name    string
    input   *dynamodb.CreateTableInput
}

func newUsersTable() tableDefinition
func newSurfReportsTable() tableDefinition
func newForecastTable() tableDefinition
// ... other table definitions

func createTable(def tableDefinition) error
```

**Benefits**:
- Massive reduction in code duplication
- Easier to add new tables
- Clear table definitions
- Testable table creation

---

## Execution Strategy

### Phase 1: High-Impact, Low-Risk (Start Here)
1. `setupRoutes` - Clear separation, low risk
2. `NewContainer` - Well-defined initialization phases
3. `createLocalTables` - Huge improvement, isolated code

### Phase 2: Controller Handlers
4. All controller functions - Apply consistent pattern
5. Extract shared controller helpers to common package/helpers

### Phase 3: Service Layer
6. Service functions - Extract business logic components
7. Create reusable service helpers

### Phase 4: Authentication & WebSocket
8. Authentication handlers - Critical but well-defined
9. WebSocket handler - Isolated, low risk

### Phase 5: Scripts (Optional/Lower Priority)
10. Seed data scripts - Improve but not critical

---

## Testing Strategy

For each refactored function:

1. **Unit Tests**: Test extracted helper functions independently
2. **Integration Tests**: Test main function still works correctly
3. **Behavior Preservation**: Ensure all edge cases handled
4. **Error Handling**: Verify error paths work correctly

---

## Anti-Patterns to Avoid

1. **Over-Abstraction**: Don't create abstractions just for the sake of it
2. **Premature Optimization**: Keep refactoring focused on readability
3. **Breaking APIs**: Maintain all function signatures
4. **Creating Duplication**: Extract common patterns, not duplicate code
5. **Circular Dependencies**: Keep helper functions in same package or clearly separate

---

## Success Criteria

- ✅ All functions under funlen limits
- ✅ No new linter errors introduced
- ✅ All tests pass
- ✅ Code compiles successfully
- ✅ Functionality preserved
- ✅ Improved testability
- ✅ Better code organization
- ✅ No duplication introduced

---

## Estimated Effort

- **Phase 1**: 2-3 hours (high-impact, foundational)
- **Phase 2**: 4-5 hours (controllers, moderate complexity)
- **Phase 3**: 3-4 hours (services, business logic)
- **Phase 4**: 2-3 hours (auth, websocket)
- **Phase 5**: 2-3 hours (scripts, optional)

**Total**: ~13-18 hours of focused refactoring

---

## Notes

- All refactoring maintains backward compatibility
- Helper functions should be package-private (lowercase) when only used internally
- Consider creating helper packages (e.g., `internal/controller/helpers`) for shared controller logic
- Document extracted functions with clear purpose
- Follow Go naming conventions and idioms

