package service

import (
	"context"
	"fmt"
	"log/slog"

	"treblesurf-backend/internal/model"
)

func (s *ReportService) SubmitSurfReport(
	ctx context.Context,
	report *model.ReportWithImage,
	userEmail, userName string,
) error {
	user, err := s.getUserAndValidate(ctx, userEmail)
	if err != nil {
		return err
	}

	currentTime := parseReportDate(report.Date)
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", report.Country, report.Region, report.Spot)

	// Process image data if provided
	s3KeyReport, err := s.processBase64Image(ctx, report.ImageData, report.Date, countryRegionSpot, user.UUID, &currentTime)
	if err != nil {
		return err
	}

	dateReported := fmt.Sprintf("%s_%s", currentTime, user.UUID)
	reportItem := s.createBaseReport(
		countryRegionSpot, dateReported, userEmail, userName, user.UUID, currentTime, "image", false,
	)
	addReportFieldsToReport(
		reportItem, report.SurfSize, report.WindAmount, report.WindDirection,
		report.Consistency, report.Quality, report.Messiness,
	)

	if s3KeyReport != "" {
		reportItem.ImageKey = s3KeyReport
	}

	if err := s.storeReport(ctx, reportItem); err != nil {
		return err
	}

	reportFields := map[string]string{
		"quality":       report.Quality,
		"surfSize":      report.SurfSize,
		"windAmount":    report.WindAmount,
		"windDirection": report.WindDirection,
		"messiness":     report.Messiness,
		"consistency":   report.Consistency,
	}
	message := buildWebSocketMessage(
		report.Country, report.Region, report.Spot, userName, user.UUID,
		s3KeyReport, "", "image", false, reportFields, currentTime,
	)
	s.broadcastReportMessage(report.Country, report.Region, report.Spot, message)

	return nil
}

func (s *ReportService) SubmitSurfReportWithS3Image(
	ctx context.Context,
	report *model.ReportWithS3Image,
	userEmail string,
	userName string,
) error {
	user, err := s.getUserAndValidate(ctx, userEmail)
	if err != nil {
		return err
	}

	currentTime := parseReportDate(report.Date)
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", report.Country, report.Region, report.Spot)
	dateReported := fmt.Sprintf("%s_%s", currentTime, user.UUID)

	reportItem := s.createBaseReport(
		countryRegionSpot, dateReported, userEmail, userName, user.UUID, currentTime, "image", false,
	)
	addReportFieldsToReport(
		reportItem, report.SurfSize, report.WindAmount, report.WindDirection,
		report.Consistency, report.Quality, report.Messiness,
	)

	s3KeyReport, err := s.processS3ImageForReport(ctx, report.ImageKey, reportItem)
	if err != nil {
		return err
	}

	if err := s.storeReportWithCleanup(ctx, reportItem, s3KeyReport); err != nil {
		return err
	}

	reportFields := map[string]string{
		"quality":       report.Quality,
		"surfSize":      report.SurfSize,
		"windAmount":    report.WindAmount,
		"windDirection": report.WindDirection,
		"messiness":     report.Messiness,
		"consistency":   report.Consistency,
	}
	message := buildWebSocketMessage(
		report.Country, report.Region, report.Spot, userName, user.UUID,
		s3KeyReport, "", "image", false, reportFields, currentTime,
	)
	s.broadcastReportMessage(report.Country, report.Region, report.Spot, message)

	return nil
}

// SubmitSurfReportWithIOSValidation submits a surf report that has been validated
// using iOS Vision framework.
func (s *ReportService) SubmitSurfReportWithIOSValidation(
	ctx context.Context,
	report *model.ReportWithIOSValidation,
	userEmail string,
	userName string,
) error {
	user, err := s.getUserAndValidate(ctx, userEmail)
	if err != nil {
		return err
	}

	currentTime := parseReportDate(report.Date)
	countryRegionSpot := fmt.Sprintf("%s_%s_%s", report.Country, report.Region, report.Spot)
	dateReported := fmt.Sprintf("%s_%s", currentTime, user.UUID)
	mediaType := determineMediaType(report.ImageKey != "", report.VideoKey != "")

	reportItem := s.createBaseReport(
		countryRegionSpot, dateReported, userEmail, userName, user.UUID, currentTime, mediaType, report.IOSValidated,
	)
	addReportFieldsToReport(
		reportItem, report.SurfSize, report.WindAmount, report.WindDirection,
		report.Consistency, report.Quality, report.Messiness,
	)

	s3KeyReport, videoKeyReport := s.processIOSMediaKeys(report.ImageKey, report.VideoKey, reportItem)

	if err := s.storeReport(ctx, reportItem); err != nil {
		return err
	}

	reportFields := map[string]string{
		"quality":       report.Quality,
		"surfSize":      report.SurfSize,
		"windAmount":    report.WindAmount,
		"windDirection": report.WindDirection,
		"messiness":     report.Messiness,
		"consistency":   report.Consistency,
	}
	message := buildWebSocketMessage(
		report.Country, report.Region, report.Spot, userName, user.UUID,
		s3KeyReport, videoKeyReport, mediaType, report.IOSValidated, reportFields, currentTime,
	)
	s.broadcastReportMessage(report.Country, report.Region, report.Spot, message)

	return nil
}

func (s *ReportService) getSpotSubscribers(_, _, _ string) ([]string, error) {
	// TODO: Implement spot subscribers retrieval
	// For now, return empty list
	return []string{}, nil
}

func (s *ReportService) broadcastToUsers(subscribers []string, _ interface{}) {
	// TODO: Implement user broadcasting
	// For now, just log the message
	slog.Info("broadcasting message to subscribers", slog.Int("count", len(subscribers)))
}
