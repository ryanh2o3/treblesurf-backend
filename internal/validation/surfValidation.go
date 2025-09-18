package validation

// IsValidSwellSize validates if a swell size is valid
func IsValidSwellSize(swellSize string) bool {
	validSizes := []string{"flat", "0-0.5", "0.5-1", "1-1.5", "1.5-2.5", "2.5+"}
	return contains(validSizes, swellSize)
}

func IsValidSurfSize(swellSize string) bool {
	validSizes := []string{"flat", "knee-waist", "chest-shoulder", "head-high", "overhead", "double-overhead"}
	return contains(validSizes, swellSize)
}

// IsValidWindAmount validates if a wind amount is valid
func IsValidWindAmount(windAmount string) bool {
	validAmounts := []string{"light", "moderate", "strong", "very-strong"}
	return contains(validAmounts, windAmount)
}

// IsValidWindDirection validates if a wind direction is valid
func IsValidWindDirection(windDirection string) bool {
	validDirections := []string{"onshore", "offshore", "cross-shore", "no-wind"}
	return contains(validDirections, windDirection)
}

// IsValidSurfConditions validates if surf conditions are valid
func IsValidSurfConditions(surfConditions string) bool {
	validConditions := []string{"mushy", "average", "okay", "good", "excellent"}
	return contains(validConditions, surfConditions)
}

// IsValidSurfDifficulty validates if surf difficulty is valid
func IsValidSurfDifficulty(surfDifficulty string) bool {
	validDifficulties := []string{"setty", "consistent", "inconsistent", "sporadic"}
	return contains(validDifficulties, surfDifficulty)
}

// IsValidMessiness validates if messiness is valid
func IsValidMessiness(messiness string) bool {
	validMessiness := []string{"clean", "slight-chop", "choppy", "messy"}
	return contains(validMessiness, messiness)
}

// contains checks if a slice contains a specific item
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}