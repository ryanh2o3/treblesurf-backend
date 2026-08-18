package model

const (
	PushTypeNewReport = "new_report"
	PushTypeGoodSurf  = "good_surf"

	DeviceEnvironmentSandbox    = "sandbox"
	DeviceEnvironmentProduction = "production"
)

// DeviceToken is an APNs device token registered by a signed-in iOS client.
type DeviceToken struct {
	UserUUID    string `json:"userUuid" dynamodbav:"user_uuid"`
	Token       string `json:"token" dynamodbav:"token"`
	Platform    string `json:"platform" dynamodbav:"platform"`
	Environment string `json:"environment" dynamodbav:"environment"`
	UpdatedAt   string `json:"updatedAt" dynamodbav:"updated_at"`
}

// SpotAlertSubscription is a persistent per-spot push watch (not a live WebSocket connection).
type SpotAlertSubscription struct {
	SpotID                  string `json:"spotId" dynamodbav:"spot_id"`
	UserUUID                string `json:"userUuid" dynamodbav:"user_uuid"`
	Country                 string `json:"country" dynamodbav:"country"`
	Region                  string `json:"region" dynamodbav:"region"`
	Spot                    string `json:"spot" dynamodbav:"spot"`
	ReportsEnabled          bool   `json:"reportsEnabled" dynamodbav:"reports_enabled"`
	GoodSurfEnabled         bool   `json:"goodSurfEnabled" dynamodbav:"good_surf_enabled"`
	LastGoodSurfNotifiedKey string `json:"lastGoodSurfNotifiedKey,omitempty" dynamodbav:"last_good_surf_notified_key,omitempty"`
	UpdatedAt               string `json:"updatedAt" dynamodbav:"updated_at"`
}

// PushPayload is the APNs alert plus deep-link fields.
type PushPayload struct {
	Title   string
	Body    string
	Type    string
	Country string
	Region  string
	Spot    string
}

// DeviceTokenRequest is the iOS body for registering or removing a device token.
type DeviceTokenRequest struct {
	Token       string `json:"token"`
	Environment string `json:"environment"`
}

// SpotAlertRequest is the iOS body for enabling/disabling alerts on a spot.
type SpotAlertRequest struct {
	ReportsEnabled  bool `json:"reportsEnabled"`
	GoodSurfEnabled bool `json:"goodSurfEnabled"`
}

// NotificationPreferencesResponse is returned by GET /notification/preferences.
type NotificationPreferencesResponse struct {
	Spots []SpotAlertPreference `json:"spots"`
}

// SpotAlertPreference is one watched spot as shown in Settings.
type SpotAlertPreference struct {
	Country         string `json:"country"`
	Region          string `json:"region"`
	Spot            string `json:"spot"`
	ReportsEnabled  bool   `json:"reportsEnabled"`
	GoodSurfEnabled bool   `json:"goodSurfEnabled"`
}
