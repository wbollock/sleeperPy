#!/usr/bin/env python3
# sleeperPy
# Description: used to fetch user's current roster, across multiple leagues, and compare
# to http://www.borischen.co/ tiers

# Main Goal: use boris chen tiers to see if you should sub out players based on boris chen tier
# possibly with trending/WW players too

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

def Diff(li1, li2): 
    li_dif = [i for i in li1 + li2 if i not in li1 or i not in li2] 
    return li_dif


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
    playerData = json.load(json_file)

# data is a dict here

# print roster for both leagues:

i = 0
starters = []
players = []
# get players and starters for user team in each league
while i < len(leagues):
    # print league name
    #print("League" + ": " + str(leagueNames[i]))

    # Get current roster
    url = "https://api.sleeper.app/v1/league/" + leagues[i] + "/rosters"
    r = requests.get(url)
    data = r.json()
    # data is a list here
    #print(data)
    # print(len(data))
    # length is number of teams
    j = 0
    while j < len(data):
        jsonDict = data[j]
        # for every team in league, do

        if jsonDict['owner_id'] == userid:
            # then this is current user
            starters.append(jsonDict['starters'])
            players.append(jsonDict['players'])
            # shit but multiple leagues
        # end of nested while loop
        j = j + 1
    # end of main while loop
    i = i + 1



# for each league
# print player names, cross reference with playerData
# playerData is a dict
# players/starters are lists

#print(players[0][1])
# second value from first list of players
#print(players)
#print(len(players[0]))
# first list of players, 15
i = 0




# all players
# while i < len(players):
#     print("")
#     print("League" + ": " + str(leagueNames[i]))
#     print("")
#     for key in playerData:
#         # key is definitely the ids
#         j = 0
#         while j < len(players[i]):
#             if key == players[i][j]:
#                 print(playerData[players[i][j]]['first_name'] + " " + playerData[players[i][j]]['last_name'])
#             j = j + 1
#     # end of main while        
#     i = i + 1

# starters
i = 0
bench = []
while i < len(starters):
    print("")
    print("League" + ": " + str(leagueNames[i]))
    print("")

    bench = Diff(players[i], starters[i])
    # print(players[i])
    # print(starters[i])
    
    # list starters
    print("Starters:")
    for key in playerData:
        # key is definitely the ids
        j = 0
        while j < len(starters[i]):
            if key == starters[i][j]:
                print(playerData[starters[i][j]]['first_name'] + " " + playerData[starters[i][j]]['last_name'])
            j = j + 1
         
    # bench
    print("\nBench:")
    for key in playerData:
        # key is definitely the ids
        # should iterate through 5
        j = 0
        while j < len(bench):
            if key == bench[j]:
                print(playerData[bench[j]]['first_name'] + " " + playerData[bench[j]]['last_name'])
            j = j + 1

    # end of main while    
    i = i + 1

