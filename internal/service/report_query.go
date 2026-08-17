package service

import (
	"context"
	"fmt"
)

func (s *ReportService) GetTodaysSurfReports(
	ctx context.Context,
	countryName, regionName, spotName string,
) ([]map[string]interface{}, error) {
	return s.GetSpotSurfReports(ctx, countryName, regionName, spotName, 1)
}

// GetSpotSurfReports retrieves surf reports for a specific spot with pagination support.
// limit: maximum number of reports to return (0 for all).
// lastEvaluatedKey: for pagination, provide the last key from previous query.
func (s *ReportService) GetSpotSurfReports(
	ctx context.Context,
	countryName, regionName, spotName string,
	limit int,
) ([]map[string]interface{}, error) {
	reports, err := s.reportRepo.GetBySpot(ctx, countryName, regionName, spotName, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSurfReportsQuery, err)
	}

	reportMaps, err := s.convertReportsToMaps(reports)
	if err != nil {
		return nil, err
	}

	s.normalizeSpotReports(reportMaps)
	return reportMaps, nil
}
