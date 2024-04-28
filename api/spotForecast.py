import datetime
import os
import json
import firebase_admin
import arrow
from firebase_admin import credentials, initialize_app, db


# Initialize the Firebase Admin SDK with the secret value
if os.getenv('ENV') == 'production':
    cred = os.environ.get('DATABASE_ACCESS')
else:
    json_path = os.path.abspath('keys/surfeable-firebase-adminsdk-jsap5-c3dd0bd252.json')
    # Initialize the Firebase Admin SDK
    cred = credentials.Certificate(json_path)
firebase_admin.initialize_app(cred, {'databaseURL': 'https://surfeable-default-rtdb.europe-west1.firebasedatabase.app/'})

def getRegions(countryName):
    # get all the regions from the database for a specific country and store it in a variable
    ref = db.reference('Location')
    locations = ref.get()

    
    regions = []
    for country, regions_dict in locations.items():
        if country == countryName:
            for region, spots_dict in regions_dict.items():
                regions.append(region)

    return regions

def getCoordinates(spotName, regionName, countryName):
    # get the latitude and longitude from the database for a specific spot and region name and store it in a variable
    ref = db.reference(f'Location/{countryName}/{regionName}')
    locations = ref.get()
    spot = next((spot for spot in locations.values() if spot['Name'] == spotName), None)
    coordinates = [spot['Latitude'], spot['Longitude']] if spot else None
    return coordinates

def getLocationInfo(spotName, regionName, countryName):
    ref = db.reference(f'Location/{countryName}/{regionName}')
    locations = ref.get()
    spot = next((spot for spot in locations.values() if spot['Name'] == spotName), None)
    return spot

def buoyLocationInfo():
    ref = db.reference('BuoyLocations')
    locations = ref.get()
    return locations

def individualBuoyLocationInfo(buoyName):
    ref = db.reference(f'BuoyLocations')
    locations = ref.get()
    buoy = next((buoy for buoy in locations.values() if buoy['Name'] == buoyName), None)
    return buoy

#function to get all the spots from the region and country provided
def getSpots(regionName, countryName):
    # get all the spots from the database for a specific region and country and store it in a variable
    ref = db.reference(f'Location/{countryName}/{regionName}')
    spots_dict = ref.get()
    spots = []
    for spot in spots_dict.values():
        spots.append(spot['Name'])
    return spots

def getSpotForecast(spotName, regionName, countryName):
    # get the most recent forecast data for the spot in the region of the country
    query = db.reference('WeatherData').order_by_key().limit_to_last(1).get()
    if query:
        # Get the key of the most recent top-level object
        top_level_key = list(query.keys())[0]
        # Navigate to the country, region, and spot within the top-level object
        query = db.reference(f"WeatherData/{top_level_key}/{countryName}/{regionName}/{spotName}").get()
    
    # Filter out any data where dateForecastedFor is before now
    now = datetime.datetime.now()
    for key in list(query.keys()):
        if datetime.datetime.strptime(query[key]['dateForecastedFor'], '%Y-%m-%d %H:%M:%S') < now:
            del query[key]
    
    # return the forecast
    return query

def getRegionForecast(regionName, countryName):
    # get the most recent forecast data for the spot in the region of the country
    query = db.reference('WeatherData').order_by_key().limit_to_last(1).get()
    if query:
        # Get the key of the most recent top-level object
        top_level_key = list(query.keys())[0]
        # Navigate to the country, region, and spot within the top-level object
        query = db.reference(f"WeatherData/{top_level_key}/{countryName}/{regionName}").get()
    
    # Filter out any data where dateForecastedFor is before now
    now = datetime.datetime.now()
    # Iterate over each region
    for region_key in list(query.keys()):
        region = query[region_key]
        # Iterate over each spot in the region
        for spot_key in list(region.keys()):
            spot = region[spot_key]

            # If the 'dateForecastedFor' of the spot is before 'now', remove it
            if datetime.datetime.strptime(spot['dateForecastedFor'], '%Y-%m-%d %H:%M:%S') < now:
                del region[spot_key]
    
    # return the forecast
    return query


#method that retrieves current weather data for a location which will be the most recent data (forecastDate) for a location and then the dateForecastedFor that is just after now. location will be in as location ID
def getCurrentWeather(spotName, regionName, countryName):

    # get the most recent weather data for the spot in the region of the country
    query = db.reference('WeatherData').order_by_key().limit_to_last(1).get()
    if query:
        top_level_key = list(query.keys())[0]
        query = db.reference(f"WeatherData/{top_level_key}/{countryName}/{regionName}/{spotName}").get()

    now = datetime.datetime.now()
    for key in list(query.keys()):
        if datetime.datetime.strptime(query[key]['dateForecastedFor'], '%Y-%m-%d %H:%M:%S') < now:
            del query[key]

    query = list(query.values())[0]
    return query

def getCurrentTides(locationName):
    #todays date only
    today = datetime.datetime.today().strftime('%Y-%m-%d')
    #yesterdays date
    yesterday = (datetime.datetime.today() - datetime.timedelta(days=1)).strftime('%Y-%m-%d') 
    #tomorrows date
    tomorrow = (datetime.datetime.today() + datetime.timedelta(days=1)).strftime('%Y-%m-%d') 

    queryToday = db.reference(f'Tides/Extremes/{locationName}/{today}').get()
    queryYesterday = db.reference(f'Tides/Extremes/{locationName}/{yesterday}').get()
    queryTomorrow = db.reference(f'Tides/Extremes/{locationName}/{tomorrow}').get()
    query = {**queryYesterday, **queryToday, **queryTomorrow}
    return query



def getBeforeAfterTides(locationName):

    tides = getCurrentTides(locationName)
    # Get current time
    now = arrow.now()

    # Initialize previous and next tides
    prev_tide = None
    next_tide = None

    # Iterate over tides
    for tide_key, tide in tides.items():
        tide_time = arrow.get(tide[1])  # Assuming tide time is in this format

        # Update previous tide if tide time is before current time and closer than previous tide
        if tide_time < now and (prev_tide is None or tide_time > arrow.get(prev_tide[1])):
            prev_tide = tide

        # Update next tide if tide time is after current time and closer than next tide
        if tide_time > now and (next_tide is None or tide_time < arrow.get(next_tide[1])):
            next_tide = tide

    # Now prev_tide and next_tide hold the tides either side of the current time
    return(prev_tide, next_tide)

def getDayTides(locationName, startDay):
    tide_data = {}
    start = arrow.get(startDay)
    end = start.shift(days=10)

    while start <= end:
        day = start.format('YYYY-MM-DD')
        query = db.reference(f'Tides/Extremes/{locationName}/{day}').get()
        tide_data[day] = query
        start = start.shift(days=1)

    return tide_data

def getLiveBuoyData():
    # get the most recent buoy data from the database
    query = db.reference('BuoyData/M4').order_by_key().limit_to_last(1).get()
    query2 = db.reference('BuoyData/Blackstones').order_by_key().limit_to_last(1).get()
    query3 = db.reference('BuoyData/West Hebrides').order_by_key().limit_to_last(1).get()
    query4 = db.reference('BuoyData/M2').order_by_key().limit_to_last(1).get()
    query5 = db.reference('BuoyData/M3').order_by_key().limit_to_last(1).get()
    query6 = db.reference('BuoyData/M5').order_by_key().limit_to_last(1).get()
    query7 = db.reference('BuoyData/M6').order_by_key().limit_to_last(1).get()
    # Get the last key in the dictionary
    last_key = list(query.keys())[-1]
    last_key2 = list(query2.keys())[-1]
    last_key3 = list(query3.keys())[-1]
    last_key4 = list(query4.keys())[-1]
    last_key5 = list(query5.keys())[-1]
    last_key6 = list(query6.keys())[-1]
    last_key7 = list(query7.keys())[-1]


    # Get the value of the last key
    value = query[last_key]
    value2 = query2[last_key2]
    value3 = query3[last_key3]
    value4 = query4[last_key4]
    value5 = query5[last_key5]
    value6 = query6[last_key6]
    value7 = query7[last_key7]

    return [value, value2, value3, value4, value5, value6, value7]

def getSingleBuoyData(buoyName):
    # get the most recent buoy data from the database
    query = db.reference(f'BuoyData/{buoyName}').order_by_key().limit_to_last(1).get()
    # Get the last key in the dictionary
    last_key = list(query.keys())[-1]
    # Get the value of the last key
    value = query[last_key]
    return value

def getLast24HoursBuoyData(buoyName):
    # get the most recent buoy data from the database
    query = db.reference(f'BuoyData/{buoyName}').order_by_key().limit_to_last(24).get()
    return query

def submitCurrentSurfReport(report):
    # get the most recent buoy data from the database
    if(report['swellSize'] != 'flat' and report['swellSize'] != '0-0.5' and report['swellSize'] != '0.5-1' and report['swellSize'] != '1-1.5' and report['swellSize'] != '1.5-2.5' and report['swellSize'] != '2.5+'):
        return "Invalid swell size"
    elif(report['windAmount'] != 'calm' and report['windAmount'] != 'light' and report['windAmount'] != 'moderate' and report['windAmount'] != 'strong'):
        return "Invalid wind speed"
    elif(report['windDirection'] != 'glassy' and report['windDirection'] != 'offshore' and report['windDirection'] != 'cross' and report['windDirection'] != 'onshore'):
        return "Invalid wind direction"
    elif(report['surfConditions'] != 'clean' and report['surfConditions'] != 'messy' and report['surfConditions'] != 'okay'):
        return "Invalid surf conditions"
    elif(report['surfDifficulty'] != 'lulls' and report['surfDifficulty'] != 'consistent' and report['surfDifficulty'] != 'relentless'):
        return "Invalid surf difficulty"
    else:
        currentTime = datetime.datetime.now()
        currentDate = currentTime.strftime("%Y-%m-%d")
        surfReport = {
            'swellSize': report['swellSize'],
            'windAmount': report['windAmount'],
            'windDirection': report['windDirection'],
            'surfConditions': report['surfConditions'],
            'surfDifficulty': report['surfDifficulty'],
            'time': str(currentTime)
        }
        print(surfReport)
        ref = db.reference(f"SurfReports/{report['country']}/{report['region']}/{report['spot']}/{currentDate}")
        ref.push(surfReport)
        return "Report submitted successfully"


def retrieveTodaysSurfReports(countryName, regionName, spotName):
    # get all the surf reports from the database for a specific spot in the region of the country and store it in a variable
    currentTime = datetime.datetime.now()
    currentDate = currentTime.strftime("%Y-%m-%d")
    ref = db.reference(f'SurfReports/{countryName}/{regionName}/{spotName}/{currentDate}')
    #loop through the reports and store them in a list
    reports = ref.get()
    return reports