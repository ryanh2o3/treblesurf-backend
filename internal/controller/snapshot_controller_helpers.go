package controller

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// parseTimestamp parses timestamp from form data with multiple format support.
func parseTimestamp(c *gin.Context) (time.Time, error) {
	timestampStr := c.PostForm("timestamp")
	if timestampStr == "" {
		return time.Now(), nil
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.999999", // Python's isoformat()
		"2006-01-02T15:04:05",        // isoformat without microseconds
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		timestamp, err := time.Parse(format, timestampStr)
		if err == nil {
			return timestamp, nil
		}
	}

	return time.Time{}, http.ErrNotSupported
}

// validateImageFile validates that the uploaded file is an image.
func validateImageFile(contentType string) bool {
	if contentType == "" {
		return false
	}
	// Check for common image types or if it starts with "image/"
	return contentType == "image/jpeg" ||
		contentType == "image/png" ||
		contentType == "image/gif" ||
		contentType == "image/webp" ||
		(len(contentType) >= 5 && contentType[:6] == "image/")
}
