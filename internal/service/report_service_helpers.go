package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"treblesurf-backend/internal/model"
	"treblesurf-backend/internal/repository"

	"github.com/rwcarlsen/goexif/exif"
)

const (
	mediaTypeBoth  = "both"
	mediaTypeImage = "image"
	mediaTypeVideo = "video"
)

func (s *ReportService) getUserAndValidate(ctx context.Context, userEmail string) (*model.User, error) {
	user, err := s.userService.GetByEmail(ctx, userEmail)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	if user.UUID == "" {
		return nil, fmt.Errorf("user does not have a UUID")
	}
	return user, nil
}

func parseReportDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Now()
	}

	parsedDate, err := time.Parse("2006-01-02 15:04:05", dateStr)
	if err != nil {
		slog.Warn("failed to parse report date, using current time", slog.String("date", dateStr), slog.Any("error", err))
		return time.Now()
	}

	return parsedDate
}

func extractDateFromImageData(imageData []byte, currentTime time.Time) time.Time {
	exifData, err := exif.Decode(bytes.NewReader(imageData))
	if err != nil {
		return currentTime
	}

	dateTime, err := exifData.DateTime()
	if err != nil {
		return currentTime
	}

	slog.Debug("extracted EXIF time from image", slog.Time("time", dateTime))
	return dateTime
}

func decodeBase64ImageData(base64String string) ([]byte, error) {
	// Handle data URIs by removing the prefix
	if strings.HasPrefix(base64String, "data:") {
		commaIndex := strings.Index(base64String, ",")
		if commaIndex != -1 {
			base64String = base64String[commaIndex+1:]
		}
	}

	return base64.StdEncoding.DecodeString(base64String)
}

func addReportFieldsToReport(
	report *model.SurfReport,
	surfSize, windAmount, windDirection, consistency, quality, messiness string,
) {
	if surfSize != "" {
		report.SurfSize = surfSize
	}
	if windAmount != "" {
		report.WindAmount = windAmount
	}
	if windDirection != "" {
		report.WindDirection = windDirection
	}
	if consistency != "" {
		report.Consistency = consistency
	}
	if quality != "" {
		report.Quality = quality
	}
	if messiness != "" {
		report.Messiness = messiness
	}
}

func buildWebSocketMessage(
	country, region, spot, userName, userUUID, imageKey, videoKey, mediaType string,
	iosValidated bool,
	reportFields map[string]string,
	currentTime time.Time,
) map[string]interface{} {
	data := map[string]interface{}{
		"country":      country,
		"region":       region,
		"spot":         spot,
		"reporter":     userName,
		"reportedBy":   userUUID,
		"imageKey":     imageKey,
		"videoKey":     videoKey,
		"mediaType":    mediaType,
		"iosValidated": iosValidated,
		"reportTime":   currentTime.Format(time.RFC3339),
	}

	// Add report fields
	for key, value := range reportFields {
		data[key] = value
	}

	return map[string]interface{}{
		"action": "new_report",
		"data":   data,
	}
}

func (s *ReportService) broadcastReportMessage(
	ctx context.Context,
	country, region, spot string,
	message map[string]interface{},
) {
	subscribers, err := s.getSpotSubscribers(ctx, country, region, spot)
	if err != nil {
		slog.Warn("failed to get subscribers", slog.Any("error", err))
		return
	}
	if len(subscribers) == 0 {
		return
	}

	go func() {
	broadcastCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		s.broadcastToUsers(broadcastCtx, subscribers, message)
	}()
}

func (s *ReportService) createBaseReport(
	countryRegionSpot, dateReported, userEmail, userName, userUUID string,
	currentTime time.Time,
	mediaType string,
	iosValidated bool,
) *model.SurfReport {
	normalizedTime := currentTime.UTC()
	return &model.SurfReport{
		CountryRegionSpot: countryRegionSpot,
		DateReported:      dateReported,
		Timestamp:         normalizedTime,
		CreatedAt:         normalizedTime,
		UpdatedAt:         normalizedTime,
		UserEmail:         userEmail,
		Reporter:          userName,
		Time:              normalizedTime.Format(time.RFC3339),
		ReportedBy:        userUUID,
		MediaType:         mediaType,
		IOSValidated:      iosValidated,
	}
}

func (s *ReportService) storeReport(ctx context.Context, report *model.SurfReport) error {
	if report == nil {
		return fmt.Errorf("report is nil")
	}
	if err := s.reportRepo.Create(ctx, report); err != nil {
		return fmt.Errorf("failed to store report: %w", err)
	}

	slog.Debug("stored surf report")
	return nil
}

func (s *ReportService) processBase64Image(
	ctx context.Context,
	imageDataStr, dateStr, countryRegionSpot, userUUID string,
	currentTime *time.Time,
) (string, error) {
	if imageDataStr == "" {
		return "", nil
	}

	decoded, err := decodeBase64ImageData(imageDataStr)
	if err != nil {
		return "", model.ErrInvalidImageData
	}

	// Extract EXIF date if no date provided
	if dateStr == "" {
		*currentTime = extractDateFromImageData(decoded, *currentTime)
	}

	// Validate image using Rekognition
	valid, err := s.validateImageWithRekognition(decoded)
	if err != nil {
		return "", err
	}
	if !valid {
		return "", model.ErrImageNotSurfRelated
	}

	imageKey := fmt.Sprintf(
		"surf-reports/%s/%s_%s.jpg",
		countryRegionSpot,
		currentTime.UTC().Format("2006-01-02T15:04:05Z"),
		userUUID,
	)
	s3Key, err := s.uploadImageToS3(ctx, decoded, imageKey)
	if err != nil {
		return "", model.NewImageValidationError(err, "failed to upload image")
	}

	return s3Key, nil
}

func determineMediaType(hasImage, hasVideo bool) string {
	switch {
	case hasImage && hasVideo:
		return mediaTypeBoth
	case hasImage:
		return mediaTypeImage
	case hasVideo:
		return mediaTypeVideo
	default:
		return "none"
	}
}

func (s *ReportService) processS3ImageForReport(
	ctx context.Context,
	imageKey string,
	report *model.SurfReport,
) (string, error) {
	if imageKey == "" {
		return "", nil
	}

	imageData, err := s.mediaRepo.Download(ctx, imageKey)
	if err != nil {
		slog.Warn("failed to retrieve pre-uploaded image", slog.String("key", imageKey), slog.Any("error", err))
		if delErr := s.mediaRepo.Delete(ctx, imageKey); delErr != nil {
			slog.Warn("failed to cleanup image", slog.String("key", imageKey), slog.Any("error", delErr))
		}
		return "", model.ErrImageRetrievalFailed
	}

	valid, err := s.validateImageWithRekognition(imageData)
	if err != nil {
		slog.Warn("failed to validate pre-uploaded image", slog.String("key", imageKey), slog.Any("error", err))
		if delErr := s.mediaRepo.Delete(ctx, imageKey); delErr != nil {
			slog.Warn("failed to cleanup image", slog.String("key", imageKey), slog.Any("error", delErr))
		}
		return "", err
	}
	if !valid {
		slog.Warn("pre-uploaded image failed validation, deleting", slog.String("key", imageKey))
		if delErr := s.mediaRepo.Delete(ctx, imageKey); delErr != nil {
			slog.Warn("failed to cleanup image", slog.String("key", imageKey), slog.Any("error", delErr))
		}
		return "", model.ErrImageNotSurfRelated
	}

	if report != nil {
		report.ImageKey = imageKey
	}
	return imageKey, nil
}

func (s *ReportService) storeReportWithCleanup(ctx context.Context, report *model.SurfReport, s3Key string) error {
	err := s.storeReport(ctx, report)
	if err != nil && s3Key != "" {
		slog.Warn("database insertion failed, cleaning up image", slog.String("key", s3Key))
		if delErr := s.mediaRepo.Delete(ctx, s3Key); delErr != nil {
			slog.Warn("failed to cleanup image", slog.String("key", s3Key), slog.Any("error", delErr))
		}
	}
	return err
}

func (s *ReportService) processIOSMediaKeys(
	imageKey, videoKey string,
	report *model.SurfReport,
) (s3KeyReport, videoKeyReport string) {
	if imageKey != "" {
		slog.Debug("iOS validated report with image", slog.String("key", imageKey))
		if report != nil {
			report.ImageKey = imageKey
		}
		s3KeyReport = imageKey
	}

	if videoKey != "" {
		slog.Debug("iOS validated report with video", slog.String("key", videoKey))
		if report != nil {
			report.VideoKey = videoKey
		}
		videoKeyReport = videoKey
	}

	return s3KeyReport, videoKeyReport
}

// convertReportsToMaps converts report structs into response-friendly maps.
func (s *ReportService) convertReportsToMaps(
	reports []*model.SurfReport,
) ([]map[string]interface{}, error) {
	out := make([]map[string]interface{}, 0, len(reports))
	for _, report := range reports {
		if report == nil {
			continue
		}
		item, err := json.Marshal(report)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal report to json: %w", err)
		}
		var reportMap map[string]interface{}
		if err := json.Unmarshal(item, &reportMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal report json: %w", err)
		}
		out = append(out, reportMap)
	}
	return out, nil
}

// normalizeSpotReports normalizes report maps by removing sensitive fields and adding defaults.
func (s *ReportService) normalizeSpotReports(reports []map[string]interface{}) {
	for _, report := range reports {
		delete(report, "user_email") // Remove sensitive field

		// Ensure new fields have defaults if missing
		setDefaultIfMissing(report, "video_key", "")
		setDefaultIfMissing(report, "media_type", mediaTypeImage)
		setDefaultIfMissing(report, "ios_validated", false)

		// Ensure all required fields have defaults
		setDefaultIfMissing(report, "consistency", "")
		setDefaultIfMissing(report, "messiness", "")
		setDefaultIfMissing(report, "quality", "")
		setDefaultIfMissing(report, "surf_size", "")
		setDefaultIfMissing(report, "wind_amount", "")
		setDefaultIfMissing(report, "wind_direction", "")
		setDefaultIfMissing(report, "reporter", "Anonymous")
	}
}

// setDefaultIfMissing sets a default value for a key if it doesn't exist in the map.
func setDefaultIfMissing(m map[string]interface{}, key string, defaultValue interface{}) {
	if _, exists := m[key]; !exists {
		m[key] = defaultValue
	}
}

// queryCurrentForecast queries for the most recent forecast data.
func (s *ReportService) queryCurrentForecast(ctx context.Context, spotID string) (*model.ForecastDataPoint, error) {
	currentTime := time.Now().Add(-1 * time.Hour)
	forecasts, err := s.forecastDataRepo.QuerySince(ctx, spotID, currentTime, 1)
	if err != nil {
		return nil, err
	}
	if len(forecasts) == 0 {
		return nil, nil
	}
	return forecasts[0], nil
}

// queryHistoricalForecast queries for forecast data looking backwards up to 24 hours.
//
//nolint:unparam // Error return maintained for API consistency
func (s *ReportService) queryHistoricalForecast(ctx context.Context, spotID string) (*model.ForecastDataPoint, error) {
	currentTime := time.Now()

	for i := 1; i <= 24; i++ {
		pastTime := currentTime.Add(-time.Duration(i) * time.Hour)
		forecasts, err := s.forecastDataRepo.QuerySince(ctx, spotID, pastTime, 1)
		if err == nil && len(forecasts) > 0 {
			return forecasts[0], nil
		}
	}

	return nil, nil
}

// extractWindData extracts wind speed and direction from forecast data.
func (s *ReportService) extractWindData(forecast *model.ForecastDataPoint) (windSpeed, windDirection float64, err error) {
	if forecast == nil || forecast.Data == nil {
		return 0, 0, fmt.Errorf("invalid forecast data structure")
	}

	windSpeed = extractFloatFromData(forecast.Data, "windSpeed")
	windDirection = extractFloatFromData(forecast.Data, "windDirection")

	return windSpeed, windDirection, nil
}

// extractFloatFromData extracts a float value from data map, handling both float64 and string types.
func extractFloatFromData(data map[string]interface{}, key string) float64 {
	if val, ok := data[key].(float64); ok {
		return val
	}
	if valStr, ok := data[key].(string); ok {
		if val, err := strconv.ParseFloat(valStr, 64); err == nil {
			return val
		}
	}
	return 0
}
