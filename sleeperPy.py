#!/usr/bin/env python3
# sleeperPy
# Description: used to fetch user's current roster, across multiple leagues, and compare
# to http://www.borischen.co/ tiers

import requests
import json
import os, time
from pathlib import Path

# Variables
username = "puffplants"

# Variables - not user adjustable
sport = "nfl"
year = "2020"
playersFile = "players.txt"


# Functions
def file_age(filepath):
    return time.time() - os.path.getmtime(filepath)


# API
# https://docs.sleeper.app/

# grab userid from username
# curl "https://api.sleeper.app/v1/user/<username>"
# curl "https://api.sleeper.app/v1/user/<user_id>"

#curl "https://api.sleeper.app/v1/user/puffplants"


# https://api.sleeper.app/v1/user/<user_id>/leagues/<sport>/<season>

# get all leagues for user
# curl "https://api.sleeper.app/v1/user/470054939452764160/leagues/nfl/2020"


# Get USERID
url = "https://api.sleeper.app/v1/user/"
url = url + username

r = requests.get(url)

data = r.json()
userid = data['user_id']
# dict type

# Get all leagues for user
url = "https://api.sleeper.app/v1/user/"
url = url + userid + "/" + "leagues/" + sport + "/" + year

r = requests.get(url)

data = r.json()
# list type

leagues = []
leagueNames = []
i = 0

while i < len(data):
    jsonDict = data[i]
    leagues.append(jsonDict['league_id'])
    leagueNames.append(jsonDict['name'])
    i = i + 1

# leagues, e.g 
# ['603501445962080256', '597557922544807936']


# Get current players of user for * leagues
# GET https://api.sleeper.app/v1/league/<league_id>/rosters

# first, fetch all players so I can cross reference IDs

# from sleeper:
# You should save this information on your own servers as this is not intended to be called every time you need to look up players due to the filesize being close to 5MB in size.
# You do not need to call this endpoint more than once per day.

if Path(playersFile).is_file():
    # if file exists but older than 1 day, recreate
    seconds = file_age(playersFile)
    if seconds > 86400:
        print("Downloading player data.")
        with open(playersFile, 'w') as outfile:
            url = "https://api.sleeper.app/v1/players/nfl"
            r = requests.get(url)
            json.dump(r.json(), outfile)
else:
    # if file doesn't exist at all
    print("Downloading player data.")
    with open(playersFile, 'w') as outfile:
        url = "https://api.sleeper.app/v1/players/nfl"
        r = requests.get(url)
        json.dump(r.json(), outfile)

# read from players file
with open(playersFile) as json_file:
    data = json.load(json_file)

# data is a dict here

# print roster for both leagues:

i = 0

while i < len(leagues):
    # print league name
    print("League" + ": " + str(leagueNames[i]))
    url = "https://api.sleeper.app/v1/league/" + leagues[i] + "/rosters"
    # iterate through players for length of roster
    # len(players)
    #for 
    i = i + 1


