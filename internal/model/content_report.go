package model

import "time"

// ReportReason represents valid reasons for content reports.
type ReportReason string

const (
	ReportReasonInappropriate  ReportReason = "Inappropriate Content"
	ReportReasonSpam           ReportReason = "Spam"
	ReportReasonOffensive      ReportReason = "Offensive Language"
	ReportReasonNotSurfRelated ReportReason = "Not Surf Related"
	ReportReasonOther          ReportReason = "Other"
)

// ValidReportReasons contains all valid report reasons.
var ValidReportReasons = []ReportReason{
	ReportReasonInappropriate,
	ReportReasonSpam,
	ReportReasonOffensive,
	ReportReasonNotSurfRelated,
	ReportReasonOther,
}

// IsValidReportReason checks if a reason string is valid.
func IsValidReportReason(reason string) bool {
	for _, r := range ValidReportReasons {
		if string(r) == reason {
			return true
		}
	}
	return false
}

// ReportStatus represents the status of a content report.
type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusReviewing ReportStatus = "reviewing"
	ReportStatusResolved  ReportStatus = "resolved"
	ReportStatusDismissed ReportStatus = "dismissed"
)

// ResolutionType represents the resolution outcome of a report.
type ResolutionType string

const (
	ResolutionContentRemoved ResolutionType = "content_removed"
	ResolutionUserWarned     ResolutionType = "user_warned"
	ResolutionUserBanned     ResolutionType = "user_banned"
	ResolutionNoAction       ResolutionType = "no_action"
	ResolutionFalseReport    ResolutionType = "false_report"
)

// ContentReport represents a user-submitted report about content.
type ContentReport struct {
	ReviewedAt      time.Time `json:"reviewed_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	ID              string    `json:"id"`
	SurfReportID    string    `json:"surf_report_id"`
	ReporterUserID  string    `json:"reporter_user_id"`
	Reason          string    `json:"reason"`
	Description     string    `json:"description,omitempty"`
	Status          string    `json:"status"`
	ReviewedBy      string    `json:"reviewed_by,omitempty"`
	Resolution      string    `json:"resolution,omitempty"`
	ResolutionNotes string    `json:"resolution_notes,omitempty"`
}

// ModerationActionType represents types of moderation actions.
type ModerationActionType string

const (
	ActionContentRemoved ModerationActionType = "content_removed"
	ActionUserWarned     ModerationActionType = "user_warned"
	ActionUserSuspended  ModerationActionType = "user_suspended"
	ActionUserBanned     ModerationActionType = "user_banned"
	ActionFalseReport    ModerationActionType = "false_report"
)

// TargetType represents the type of entity being moderated.
type TargetType string

const (
	TargetTypeReport TargetType = "report"
	TargetTypeUser   TargetType = "user"
)

// ModerationAction represents an action taken by a moderator.
type ModerationAction struct {
	CreatedAt   time.Time `json:"created_at"`
	ID          string    `json:"id"`
	ReportID    string    `json:"report_id"`
	ActionType  string    `json:"action_type"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	PerformedBy string    `json:"performed_by"`
	Notes       string    `json:"notes,omitempty"`
}

// SubmitReportRequest represents the request body for submitting a report.
type SubmitReportRequest struct {
	SurfReportID string `json:"surfReportId" binding:"required"`
	Reason       string `json:"reason" binding:"required"`
	Description  string `json:"description"`
}

// SubmitReportResponse represents the response for a successful report submission.
type SubmitReportResponse struct {
	Message  string `json:"message"`
	ReportID string `json:"reportId"`
	Success  bool   `json:"success"`
}

// ResolveReportRequest represents the request body for resolving a report.
type ResolveReportRequest struct {
	Resolution string `json:"resolution" binding:"required"`
	Notes      string `json:"notes"`
}
