package controller

import (
	"errors"
	"log/slog"
	"net/http"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// ContentReportController handles content report endpoints.
type ContentReportController struct {
	reportService *service.ContentReportService
	userService   *service.UserService
}

// NewContentReportController creates a new ContentReportController.
func NewContentReportController(
	reportService *service.ContentReportService,
	userService *service.UserService,
) *ContentReportController {
	return &ContentReportController{
		reportService: reportService,
		userService:   userService,
	}
}

// SubmitReport handles POST /api/reports/submit.
func (c *ContentReportController) SubmitReport(ctx *gin.Context) {
	user, ok := c.getAuthenticatedUser(ctx)
	if !ok {
		return
	}

	req, ok := c.parseReportRequest(ctx)
	if !ok {
		return
	}

	// Submit the report
	input := service.SubmitReportInput{
		SurfReportID: req.SurfReportID,
		Reason:       req.Reason,
		Description:  req.Description,
		ReporterID:   user.UUID,
	}

	report, err := c.reportService.SubmitReport(ctx.Request.Context(), input)
	if err != nil {
		c.handleSubmitError(ctx, err)
		return
	}

	slog.InfoContext(
		ctx.Request.Context(),
		"report submitted",
		slog.String("report_id", report.ID),
		slog.String("user_id", user.UUID),
	)

	ctx.JSON(http.StatusOK, model.SubmitReportResponse{
		Success:  true,
		Message:  "Report submitted successfully",
		ReportID: report.ID,
	})
}

// getAuthenticatedUser extracts the authenticated user from context.
func (c *ContentReportController) getAuthenticatedUser(ctx *gin.Context) (*model.User, bool) {
	email, exists := ctx.Get("email")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Authentication required",
		})
		return nil, false
	}

	emailStr, ok := email.(string)
	if !ok || emailStr == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Authentication required",
		})
		return nil, false
	}

	user, err := c.userService.GetByEmail(ctx.Request.Context(), emailStr)
	if err != nil {
		slog.ErrorContext(ctx.Request.Context(), "failed to get user", slog.Any("error", err))
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "User not found",
		})
		return nil, false
	}
	return user, true
}

// parseReportRequest parses and validates the report submission request.
func (c *ContentReportController) parseReportRequest(ctx *gin.Context) (*model.SubmitReportRequest, bool) {
	var req model.SubmitReportRequest
	if bindErr := ctx.ShouldBindJSON(&req); bindErr != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + bindErr.Error(),
		})
		return nil, false
	}

	if req.SurfReportID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: missing surfReportId",
		})
		return nil, false
	}

	if req.Reason == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: missing reason",
		})
		return nil, false
	}
	return &req, true
}

func (c *ContentReportController) handleSubmitError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrRateLimitExceeded):
		ctx.JSON(http.StatusTooManyRequests, gin.H{
			"success": false,
			"message": "Rate limit exceeded. Please try again later.",
		})
	case errors.Is(err, service.ErrInvalidReportReason):
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid reason. Must be one of: Inappropriate Content, Spam, " +
				"Offensive Language, Not Surf Related, Other",
		})
	case errors.Is(err, service.ErrDescriptionTooLong):
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Description exceeds maximum length of 500 characters",
		})
	case errors.Is(err, service.ErrSurfReportNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Surf report not found",
		})
	default:
		slog.ErrorContext(ctx.Request.Context(), "failed to submit report", slog.Any("error", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to submit report",
		})
	}
}
