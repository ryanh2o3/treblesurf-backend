import json
import boto3
from decimal import Decimal

class DecimalEncoder(json.JSONEncoder):
    def default(self, o):
        if isinstance(o, Decimal):
            if o % 1 > 0:
                return float(o)
            else:
                return float(o) 
        return super(DecimalEncoder, self).default(o)

# Initialize the DynamoDB client
dynamodb = boto3.resource('dynamodb')

# Get the table
table = dynamodb.Table('LocationData')

def lambda_handler(event, context):
    # Parse the body of the request
    body = event.get('queryStringParameters', '{}')

    #retrieve country, region, and spot from the body
    country = body.get('country', '')
    region = body.get('region', '')
    spot = body.get('spot', '')

    # Create the primary key
    primary_key = f"{country}/{region}/{spot}"

    # Get the item
    response = table.get_item(
        Key={
            'country_region_spot': primary_key
        }
    )

    # Get the item
    item = response.get('Item', {})

    # Convert the item to JSON using the custom encoder
    location = json.dumps(item, cls=DecimalEncoder)
    
    return location