package controller

import (
	"log/slog"
	"net/http"
	"strconv"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// ModerationController handles admin moderation endpoints.
type ModerationController struct {
	moderationService *service.ModerationService
}

// NewModerationController creates a new ModerationController.
func NewModerationController(moderationService *service.ModerationService) *ModerationController {
	return &ModerationController{
		moderationService: moderationService,
	}
}

// parsePagination extracts limit and offset from query parameters.
func parsePagination(ctx *gin.Context) (limit, offset int) {
	limit = 50
	offset = 0

	if l := ctx.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if o := ctx.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	return limit, offset
}

// ListReports handles GET /api/admin/reports.
func (c *ModerationController) ListReports(ctx *gin.Context) {
	limit, offset := parsePagination(ctx)

	reports, err := c.moderationService.GetPendingReports(ctx.Request.Context(), limit, offset)
	if err != nil {
		slog.ErrorContext(ctx.Request.Context(), "failed to get reports", slog.Any("error", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve reports"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"reports": reports,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetReport handles GET /api/admin/reports/:id.
func (c *ModerationController) GetReport(ctx *gin.Context) {
	reportID := ctx.Param("id")
	if reportID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Report ID is required"})
		return
	}

	report, err := c.moderationService.GetReportDetails(ctx.Request.Context(), reportID)
	if err != nil {
		slog.ErrorContext(ctx.Request.Context(), "failed to get report", slog.Any("error", err))
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Report not found"})
		return
	}

	// Get actions for this report
	actions, err := c.moderationService.GetReportActions(ctx.Request.Context(), reportID)
	if err != nil {
		slog.WarnContext(ctx.Request.Context(), "failed to get report actions", slog.Any("error", err))
		actions = []*model.ModerationAction{}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"report":  report,
		"actions": actions,
	})
}

// ResolveReport handles POST /api/admin/reports/:id/resolve.
func (c *ModerationController) ResolveReport(ctx *gin.Context) {
	reportID := ctx.Param("id")
	if reportID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Report ID is required"})
		return
	}

	email, exists := ctx.Get("email")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	adminEmail, ok := email.(string)
	if !ok || adminEmail == "" {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication"})
		return
	}

	// Parse request body
	var req model.ResolveReportRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if req.Resolution == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Resolution is required"})
		return
	}

	input := service.ResolveReportInput{
		ReportID:   reportID,
		Resolution: req.Resolution,
		Notes:      req.Notes,
		AdminID:    adminEmail,
	}

	if err := c.moderationService.ResolveReport(ctx.Request.Context(), input); err != nil {
		slog.ErrorContext(ctx.Request.Context(), "failed to resolve report", slog.Any("error", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve report"})
		return
	}

	slog.InfoContext(
		ctx.Request.Context(),
		"report resolved",
		slog.String("report_id", reportID),
		slog.String("admin", adminEmail),
		slog.String("resolution", req.Resolution),
	)

	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Report resolved successfully",
	})
}

// ListActions handles GET /api/admin/moderation-actions.
func (c *ModerationController) ListActions(ctx *gin.Context) {
	limit, offset := parsePagination(ctx)

	actions, err := c.moderationService.GetModerationHistory(ctx.Request.Context(), limit, offset)
	if err != nil {
		slog.ErrorContext(ctx.Request.Context(), "failed to get actions", slog.Any("error", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve actions"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"actions": actions,
		"limit":   limit,
		"offset":  offset,
	})
}

