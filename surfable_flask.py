import flask
from flask import Flask, render_template, send_from_directory, jsonify, request
import os
from flask_cors import CORS
from api.spotForecast import *
# Create the application.
APP = flask.Flask(__name__, static_folder='static')
CORS(APP, resources={r"/api/*": {"origins": "*"}})

@APP.route('/../../frontend/dist/<path:filename>')
def serve_vue_dist(filename):
    dist_path = os.path.join(APP.root_path, 'frontend', 'dist')  # Adjust the path accordingly
    return send_from_directory(dist_path, filename)

@APP.route('/')
def index():
    """ Displays the index page accessible at '/'
    """
    return flask.render_template('index.html')

#uses the getCOordinates function to get the latitude and longitude of a location
@APP.route("/api/location")
def get_location():
    region = request.args.get("region")
    spot = request.args.get("spot")
    country = request.args.get("country")
    coordinates = getCoordinates(spot, region, country)
    return jsonify(coordinates)

@APP.route("/api/locationInfo")
def get_location_info():
    region = request.args.get("region")
    spot = request.args.get("spot")
    country = request.args.get("country")
    locationInfo = getLocationInfo(spot, region, country)
    return jsonify(locationInfo)

@APP.route("/api/buoyLocationInfo")
def get_buoyLocationInfo():
    locationInfo = buoyLocationInfo()
    return jsonify(locationInfo)

#uses the getSpots function to get all the spots from a region and country
@APP.route("/api/spots")
def get_spots():
    region = request.args.get("region")
    country = request.args.get("country")
    spots = getSpots(region, country)
    return jsonify(spots)

#uses the getRegion function to get all the regions from a country
@APP.route("/api/regions")
def get_regions():
    country = request.args.get("country")
    regions = getRegions(country)
    return jsonify(regions)

#function that retrievs the weather data from the database for a given location using the spotForecast.py file
@APP.route("/api/forecast")
def get_forecast():
    spot = request.args.get("spot")
    region = request.args.get("region")
    country = request.args.get("country")

    forecast = getSpotForecast(spot, region, country)
    return jsonify(forecast)

@APP.route("/api/currentConditions")
def get_currentConditions():
    spot = request.args.get("spot")
    region = request.args.get("region")
    country = request.args.get("country")

    forecast = getCurrentWeather(spot, region, country)
    return jsonify(forecast)

@APP.route("/api/beforeAfterTide")
def get_beforeAfterTide():
    locationName = request.args.get("locationName")

    tides = getBeforeAfterTides(locationName)
    return jsonify(tides)

@APP.route("/api/tideExtremes")
def get_tideExtremes():
    locationName = request.args.get("locationName")
    start = request.args.get("start")
    tides = getDayTides(locationName, start)
    return jsonify(tides)

@APP.route("/api/regionForecast")
def get_regionForecast():
    region = request.args.get("region")
    country = request.args.get("country")

    forecast = getRegionForecast(region, country)
    return jsonify(forecast)

@APP.route("/api/getLiveBuoyData")
def get_liveBuoyData():
    data = getLiveBuoyData()
    return jsonify(data)

@APP.route("/api/getSingleBuoyData")
def get_singleBuoyData():
    buoyName = request.args.get("buoyName")
    data = getSingleBuoyData(buoyName)
    return jsonify(data)

@APP.route("/api/individualBuoyLocation")
def get_individualBuoyLocation():
    buoyName = request.args.get("buoyName")
    data = individualBuoyLocationInfo(buoyName)
    return jsonify(data)

@APP.route("/api/getLast24BuoyData")
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

@APP.route("/api/getTodaySpotReports")
def get_todaySpotReports():
    spot = request.args.get("spot")
    region = request.args.get("region")
    country = request.args.get("country")
    data = retrieveTodaysSurfReports(country, region, spot)
    return jsonify(data)


if __name__ == '__main__':
    APP.debug=True
    APP.run()
