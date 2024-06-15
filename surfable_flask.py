import flask
from flask import Flask, render_template, send_from_directory, jsonify, request
import os
from flask_cors import CORS
from api.spotForecast import *
# Create the application.
APP = flask.Flask(__name__, static_folder='static')
CORS(APP, resources={r"/api/*": {"origins": "*"}})

#uses the getCOordinates function to get the latitude and longitude of a location
@APP.route("/api/location", methods=['GET'])
def get_location():
    region = request.args.get("region")
    spot = request.args.get("spot")
    country = request.args.get("country")
    coordinates = getCoordinates(spot, region, country)
    return jsonify(coordinates)

#get location info for a given spot
@APP.route("/api/locationInfo", methods=['GET'])
def get_location_info():
    region = request.args.get("region")
    spot = request.args.get("spot")
    country = request.args.get("country")
    locationInfo = getLocationInfo(spot, region, country)
    return jsonify(locationInfo)

@APP.route("/api/buoyLocationInfo", methods=['GET'])
def get_buoyLocationInfo():
    locationInfo = buoyLocationInfo()
    return jsonify(locationInfo)

#uses the getSpots function to get all the spots from a region and country
@APP.route("/api/spots", methods=['GET'])
def get_spots():
    region = request.args.get("region")
    country = request.args.get("country")
    spots = getSpots(region, country)
    return jsonify(spots)

#uses the getRegion function to get all the regions from a country
@APP.route("/api/regions", methods=['GET'])
def get_regions():
    country = request.args.get("country")
    regions = getRegions(country)
    return jsonify(regions)

#function that retrievs the weather data from the database for a given location using the spotForecast.py file
@APP.route("/api/forecast", methods=['GET'])
def get_forecast():
    spot = request.args.get("spot")
    region = request.args.get("region")
    country = request.args.get("country")

    forecast = getSpotForecast(spot, region, country)
    return jsonify(forecast)

@APP.route("/api/currentConditions", methods=['GET'])
def get_currentConditions():
    spot = request.args.get("spot")
    region = request.args.get("region")
    country = request.args.get("country")

    forecast = getCurrentWeather(spot, region, country)
    return jsonify(forecast)

@APP.route("/api/beforeAfterTide", methods=['GET'])
def get_beforeAfterTide():
    locationName = request.args.get("locationName")

    tides = getBeforeAfterTides(locationName)
    return jsonify(tides)

@APP.route("/api/tideExtremes", methods=['GET'])
def get_tideExtremes():
    locationName = request.args.get("locationName")
    start = request.args.get("start")
    tides = getDayTides(locationName, start)
    return jsonify(tides)

@APP.route("/api/regionForecast", methods=['GET'])
def get_regionForecast():
    region = request.args.get("region")
    country = request.args.get("country")

    forecast = getRegionForecast(region, country)
    return jsonify(forecast)

@APP.route("/api/getLiveBuoyData", methods=['GET'])
def get_liveBuoyData():
    data = getLiveBuoyData()
    return jsonify(data)

@APP.route("/api/getSingleBuoyData", methods=['GET'])
def get_singleBuoyData():
    buoyName = request.args.get("buoyName")
    data = getSingleBuoyData(buoyName)
    return jsonify(data)

@APP.route("/api/individualBuoyLocation", methods=['GET'])
def get_individualBuoyLocation():
    buoyName = request.args.get("buoyName")
    data = individualBuoyLocationInfo(buoyName)
    return jsonify(data)

@APP.route("/api/getLast24BuoyData", methods=['GET'])
def get_last24BuoyData():
    buoyName = request.args.get("buoyName")
    data = getLast24HoursBuoyData(buoyName)
    return jsonify(data)

@APP.route("/api/submitSurfReport", methods=['POST'])
def submitSurfReport():
    data = request.json
    result = submitCurrentSurfReport(data)
    if(result != "Report submitted successfully"):
        return result, 400
    else:
        return 'Surf report submitted', 200

#retrieve todays surf reports for a given spot
@APP.route("/api/getTodaySpotReports", methods=['GET'])
def get_todaySpotReports():
    spot = request.args.get("spot")
    region = request.args.get("region")
    country = request.args.get("country")
    data = retrieveTodaysSurfReports(country, region, spot)
    return jsonify(data)

@APP.route("/api/listSpotsForecast", methods=['GET'])
def get_listSpotsForecast():
    spots = request.args.get("spots")
    region = request.args.get("region")
    country = request.args.get("country")
    data = getListSpotsForecast(spots, region, country)
    return jsonify(data)

@APP.route("/api/getMultipleBuoyData", methods=['GET'])
def get_multipleBuoyData():
    buoys= request.args.get("buoys")
    data = getMultipleBuoyData(buoys)
    return jsonify(data)

if __name__ == '__main__':
    APP.debug=True
    APP.run()
