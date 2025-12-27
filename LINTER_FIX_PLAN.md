# Linter Errors Fix Plan

This document outlines the systematic plan to fix all 222 linter errors identified by golangci-lint.

## Summary by Error Type

| Category                             | Count | Priority | Difficulty |
| ------------------------------------ | ----- | -------- | ---------- |
| revive (missing comments)            | 120   | Medium   | Low        |
| lll (line length)                    | 40    | Low      | Very Low   |
| funlen (long functions)              | 19    | High     | Medium     |
| errcheck (unchecked errors)          | 12    | High     | Low        |
| gosec (security)                     | 10    | Critical | Medium     |
| dupl (duplicate code)                | 10    | Medium   | Medium     |
| gocritic (code quality)              | 3     | Medium   | Low        |
| gocyclo (complexity)                 | 2     | High     | High       |
| gochecknoinits (init funcs)          | 2     | Medium   | Medium     |
| unused (unused code)                 | 2     | Low      | Very Low   |
| goconst (magic strings)              | 1     | Low      | Very Low   |
| ineffassign (ineffective assignment) | 1     | Low      | Very Low   |

## Fix Strategy by Category

### 1. revive - Missing Comments (120 errors)

**Root Cause**: Go exported identifiers must have documentation comments.

**Fix Approach**:

- Add package comments to all packages (11 packages)
- Add comments to all exported types, functions, and methods (109 identifiers)
- Follow Go documentation conventions: start with identifier name, use complete sentences

**Files Affected**:

- All packages missing package comments
- All exported types, functions, and methods missing comments

**Priority**: Medium (code quality/documentation)

---

### 2. lll - Line Length (40 errors)

**Root Cause**: Lines exceed 120 character limit (configured in .golangci.yml:53).

**Fix Approach**:

- Break long lines into multiple lines
- Extract long string literals to constants
- Use line breaks in function calls and struct initializations

**Examples**:

- Long HTTP headers → constants or multi-line
- Long function signatures → parameter grouping
- Long string messages → constants or line breaks

**Priority**: Low (readability)

---

### 3. funlen - Long Functions (19 errors)

**Root Cause**: Functions exceed statement limits (40 or 60 statements depending on complexity).

**Fix Approach**:

- Extract logical blocks into helper functions
- Separate validation logic from business logic
- Extract error handling into helper functions
- Split large functions into smaller, focused functions

**Functions to Refactor**:

- `setupRoutes` (80 statements → split into multiple route setup functions)
- `GoogleAuthHandler` (64 statements → extract validation and token handling)
- Multiple controller functions (extract error handling helpers)
- Service functions (extract business logic helpers)
- Local script functions (extract data setup helpers)

**Priority**: High (maintainability)

---

### 4. errcheck - Unchecked Errors (12 errors)

**Root Cause**: Error returns are not checked, potentially leading to silent failures.

**Fix Approach**:

- Check all error returns explicitly
- For defer Close() calls, use explicit error checking or log warnings
- For critical errors, propagate or handle appropriately
- For non-critical errors (like Close() in defer), at minimum log them

**Locations**:

- `defer result.Body.Close()` → Check and log if error occurs
- `defer src.Close()` → Check and log if error occurs
- `UpdateConnectionLastActive()` → Check and handle error
- `SendToConnection()` → Check and handle error
- `os.Setenv()` → Check and handle error (non-fatal, log warning)
- `file.Close()` → Check and log if error occurs
- `os.Remove()` → Check and log if error occurs

**Priority**: High (correctness)

---

### 5. gosec - Security Issues (10 errors)

**Root Cause**: Security vulnerabilities identified by static analysis.

**Fix Approach**:

#### 5a. Hardcoded Credentials (4 errors)

- Move Google OAuth client IDs to environment variables or config
- Move JWT secrets to environment variables (already mostly done, but need to remove hardcoded fallbacks in production code)
- Use secure defaults only in local/dev environments with clear warnings

#### 5b. File Permissions (3 errors)

- Change directory permissions from 0755 to 0750 (local storage)
- Change file permissions from 0644 to 0600 (local storage)
- These are local-only, so security impact is limited but should follow best practices

#### 5c. File Inclusion (3 errors)

- Validate file paths to prevent directory traversal
- Use `filepath.Clean()` and validate paths are within expected directories
- For local development scripts, add path validation

**Priority**: Critical (security)

---

### 6. dupl - Duplicate Code (10 errors)

**Root Cause**: Code duplication reduces maintainability.

**Fix Approach**:

- Extract duplicate error handling into helper functions
- Extract duplicate URL generation logic into generic helper
- Extract duplicate DynamoDB item creation into helper function
- For seed data duplication, consider if consolidation is possible

**Specific Duplications**:

1. Error handling in `report_controller.go` (lines 140-184 vs 309-353) → Extract `handleImageError()`
2. URL generation functions (`GenerateImageUploadURL` vs `GenerateVideoUploadURL`) → Extract common logic
3. DynamoDB item creation → Extract `createReportItem()` helper
4. Seed data structures → Consider if they can be shared

**Priority**: Medium (maintainability)

---

### 7. gocritic - Code Quality (3 errors)

**Root Cause**: If-else chains should be switch statements for better readability.

**Fix Approach**:

- Convert if-else chains to switch statements
- Improves code readability and maintainability

**Locations**:

- `internal/auth/middleware.go:30` - Convert to switch
- `internal/service/report_service.go:631` - Convert to switch
- `internal/service/report_service.go:1011` - Convert to switch

**Priority**: Medium (code quality)

---

### 8. gocyclo - High Complexity (2 errors)

**Root Cause**: Functions have cyclomatic complexity > 30 (threshold is 15, but these are much higher).

**Fix Approach**:

- Break down complex functions into smaller functions
- Extract conditional logic into helper functions
- Use early returns to reduce nesting
- Consider using strategy pattern or state machine for complex logic

**Functions**:

- `GetSurfReportsWithSimilarBuoyData` (complexity 45) → Extract matching logic, filtering logic
- `GetSurfReportsWithMatchingConditions` (complexity 56) → Extract condition matching, filtering logic

**Priority**: High (maintainability, testability)

---

### 9. gochecknoinits - Init Functions (2 errors)

**Root Cause**: `init()` functions make testing harder and have unpredictable execution order.

**Fix Approach**:

- Refactor to explicit initialization functions
- For Lambda handlers, create `Initialize()` functions called from `main()`
- Makes initialization explicit and testable

**Locations**:

- `cmd/api/main.go` - Move init logic to `Initialize()` function
- `cmd/websocket/main.go` - Move init logic to `Initialize()` function

**Priority**: Medium (testability)

---

### 10. unused - Unused Functions (2 errors)

**Root Cause**: Functions are defined but never called.

**Fix Approach**:

- Remove unused functions if truly not needed
- Or mark with build tags if reserved for future use
- Check if they should be exported for external use

**Functions**:

- `regenerateCSRFToken` in `internal/auth/middleware.go` - Remove or implement if needed
- `minInt` in `internal/service/report_service.go` - Remove (Go 1.21+ has built-in `min()`)

**Priority**: Low (cleanup)

---

### 11. goconst - Magic Strings (1 error)

**Root Cause**: String literal "development" appears 6 times, should be a constant.

**Fix Approach**:

- Create a constant for the environment string
- Use the constant throughout the codebase

**Location**:

- `internal/api/router.go:24` - Create `const EnvDevelopment = "development"`

**Priority**: Low (code quality)

---

### 12. ineffassign - Ineffective Assignment (1 error)

**Root Cause**: Variable is assigned but never used, or assignment has no effect.

**Fix Approach**:

- Remove the ineffective assignment
- Or fix the logic if the assignment was intended to be used

**Location**:

- `internal/service/report_service.go:1925` - `targetBuoyTime` assignment is ineffective, use `reportTime` directly or fix logic

**Priority**: Low (correctness)

---

## Execution Order

1. **Phase 1: Quick Wins (Low Hanging Fruit)**

   - unused (2) - Remove unused functions
   - ineffassign (1) - Fix ineffective assignment
   - goconst (1) - Extract magic string to constant
   - unused-parameter (1) - Fix parameter naming

2. **Phase 2: Critical Security Fixes**

   - gosec (10) - Fix security issues (credentials, file permissions, path validation)

3. **Phase 3: Correctness Fixes**

   - errcheck (12) - Handle all error returns
   - gochecknoinits (2) - Refactor init functions

4. **Phase 4: Code Quality Improvements**

   - gocritic (3) - Convert if-else to switch
   - lll (40) - Fix line length issues
   - dupl (10) - Extract duplicate code

5. **Phase 5: Documentation**

   - revive (120) - Add all missing comments

6. **Phase 6: Complex Refactoring**
   - gocyclo (2) - Reduce complexity (may require significant refactoring)
   - funlen (19) - Break down long functions (may require significant refactoring)

## Testing Strategy

After each phase:

1. Run linter to verify fixes
2. Run tests to ensure no regressions
3. Build project to ensure it compiles
4. Check for any new issues introduced

## Estimated Impact

- **Low Risk**: unused, ineffassign, goconst, lll, revive (documentation only)
- **Medium Risk**: errcheck, gosec (security), gocritic, gochecknoinits, dupl
- **High Risk**: gocyclo, funlen (requires significant refactoring, may need comprehensive testing)

## Notes

- All fixes will maintain backward compatibility
- No functionality changes, only code quality improvements
- Some refactoring may improve performance (extracted functions, early returns)
- Documentation improvements will enhance code maintainability
