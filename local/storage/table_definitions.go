package storage

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// tableDefinition holds the definition for a DynamoDB table.
type tableDefinition struct {
	name       string
	keySchema  []*dynamodb.KeySchemaElement
	attributes []*dynamodb.AttributeDefinition
}

// getAllTableDefinitions returns all table definitions for local development.
func getAllTableDefinitions() []tableDefinition {
	return []tableDefinition{
		newLocationDataTable(),
		newSurfForecastsTable(),
		newBuoyDataTable(),
		newBuoyLocationsTable(),
		newSurfReportsTable(),
		newUsersTable(),
		newSpotSnapshotsTable(),
		newStreamRequestsTable(),
		newAPIKeysTable(),
	}
}

// newLocationDataTable returns the definition for the LocationData table.
func newLocationDataTable() tableDefinition {
	return tableDefinition{
		name: "LocationData",
		keySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("country_region_spot"),
				KeyType:       aws.String("HASH"),
			},
		},
		attributes: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("country_region_spot"),
				AttributeType: aws.String("S"),
			},
		},
	}
}

// newSurfForecastsTable returns the definition for the surf_forecasts table (PK includes source, numeric SK).
func newSurfForecastsTable() tableDefinition {
	return tableDefinition{
		name: "surf_forecasts",
		keySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("spot_id"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("timestamp_ts"),
				KeyType:       aws.String("RANGE"),
			},
		},
		attributes: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("spot_id"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("timestamp_ts"),
				AttributeType: aws.String("N"),
			},
		},
	}
}

// newBuoyDataTable returns the definition for the BuoyData table.
func newBuoyDataTable() tableDefinition {
	return tableDefinition{
		name: "BuoyData",
		keySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("region_buoy"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("dataDateTime"),
				KeyType:       aws.String("RANGE"),
			},
		},
		attributes: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("region_buoy"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("dataDateTime"),
				AttributeType: aws.String("S"),
			},
		},
	}
}

// newBuoyLocationsTable returns the definition for the BuoyLocations table.
func newBuoyLocationsTable() tableDefinition {
	return tableDefinition{
		name: "BuoyLocations",
		keySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("region_buoy"),
				KeyType:       aws.String("HASH"),
			},
		},
		attributes: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("region_buoy"),
				AttributeType: aws.String("S"),
			},
		},
	}
}

// newSurfReportsTable returns the definition for the SurfReports table.
func newSurfReportsTable() tableDefinition {
	return tableDefinition{
		name: "SurfReports",
		keySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("country_region_spot"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("dateReported"),
				KeyType:       aws.String("RANGE"),
			},
		},
		attributes: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("country_region_spot"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("dateReported"),
				AttributeType: aws.String("S"),
			},
		},
	}
}

// newUsersTable returns the definition for the Users table.
func newUsersTable() tableDefinition {
	return tableDefinition{
		name: "Users",
		keySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("email"),
				KeyType:       aws.String("HASH"),
			},
		},
		attributes: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("email"),
				AttributeType: aws.String("S"),
			},
		},
	}
}

// newSpotSnapshotsTable returns the definition for the SpotSnapshots table.
func newSpotSnapshotsTable() tableDefinition {
	return tableDefinition{
		name: "SpotSnapshots",
		keySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("spot_id"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("timestamp"),
				KeyType:       aws.String("RANGE"),
			},
		},
		attributes: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("spot_id"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("timestamp"),
				AttributeType: aws.String("S"),
			},
		},
	}
}

// newStreamRequestsTable returns the definition for the StreamRequests table.
func newStreamRequestsTable() tableDefinition {
	return tableDefinition{
		name: "StreamRequests",
		keySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("spot_id"),
				KeyType:       aws.String("HASH"),
			},
		},
		attributes: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("spot_id"),
				AttributeType: aws.String("S"),
			},
		},
	}
}

// newAPIKeysTable returns the definition for the ApiKeys table.
func newAPIKeysTable() tableDefinition {
	return tableDefinition{
		name: "ApiKeys",
		keySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("key_id"),
				KeyType:       aws.String("HASH"),
			},
		},
		attributes: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("key_id"),
				AttributeType: aws.String("S"),
			},
		},
	}
}

