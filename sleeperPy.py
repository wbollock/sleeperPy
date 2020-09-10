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

# Variables - Adjust Sleeper Username Here
#username = "KingDedede"
#username = "puffplants"
username = "Jz904"

# https://github.com/abhinavk99/espn-borischentiers/blob/master/src/js/espn-borischentiers.js
# qb = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_QB.txt"
# rbStd = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_RB.txt"
# rbHalfPPR = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_RB-HALF.txt"
# rbPPR = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_RB-PPR.txt"
# wrSTD = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_WR.txt"
# wrHalfPPR = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_WR-HALF.txt"
# wrPPR = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_WR-PPR.txt"
# teSTD = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_TE.txt"
# teHALFPPR = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_TE-HALF.txt"
# tePPR = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_TE-PPR.txt"
# dst = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_DST.txt"
# k = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_K.txt"



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

# def scoringMode(scoring):
#     i = 0 
#     mode = []
#     while i < len(scoring):
#         if scoring[i] == 1.0:
#             mode.append("PPR")
#         elif scoring[i] == 0.5:
#             mode.append("HPPR")
#         elif scoring[i] == 0.0:
#             mode.append("STD")
#         i = 1 + 1
#     return mode


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
scoring = []
i = 0

while i < len(data):
    jsonDict = data[i]
    #print(jsonDict)
    leagues.append(jsonDict['league_id'])
    leagueNames.append(jsonDict['name'])
    scoring.append(jsonDict['scoring_settings']['rec'])
    i = i + 1

#print(scoring)
#list, e.g [1.0, 1.0]



# set scoring mode



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


# Print roster from each league, bench and starters
i = 0
bench = []


while i < len(starters):
    # for each league, do:
    print("")
    print("League" + ": " + str(leagueNames[i]))
    print("")
    # mode = scoringMode(scoring)
    #print(scoring)
    #print(mode)

    # figure out what lists to use
    # god i dont think this should be here
    qbBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_QB.txt"
    dstBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_DST.txt"
    kBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_K.txt"
    
    if scoring[i] == 1.0:
        rbBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_RB-PPR.txt"
        wrBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_WR-PPR.txt"
        teBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_TE-PPR.txt"    
    elif scoring[i] == 0.5:
        rbBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_RB-HALF.txt"
        wrBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_WR-HALF.txt"
        teBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_TE-HALF.txt"
    elif scoring[i] == 0.0:
        rbBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_RB.txt"
        wrBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_WR.txt"
        teBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_TE.txt"


    r = requests.get(rbBoris)
    data = r.text
    tierListRB = data.splitlines()

    # tierListRB[0] = Tier 1: Christian McCaffrey, Ezekiel Elliott

    r = requests.get(wrBoris)
    data = r.text
    tierListWR = data.splitlines()

    r = requests.get(teBoris)
    data = r.text
    tierListTE = data.splitlines()

    r = requests.get(qbBoris)
    data = r.text
    tierListQB = data.splitlines()

    r = requests.get(kBoris)
    data = r.text
    tierListK = data.splitlines()

    r = requests.get(dstBoris)
    data = r.text
    tierListDST = data.splitlines()

    # hey this works nicely
    # tfw when python has no switch/case
    

    bench = Diff(players[i], starters[i])
    
    # list starters
    tierSum = 0
    print("Starters:")
    for key in playerData:
        # key is definitely the ids
        j = 0
        tier = 99
        while j < len(starters[i]):
            if key == starters[i][j]:
                # one player from each loop... add tiers here i guess?
                fName = playerData[starters[i][j]]['first_name']  
                lName = playerData[starters[i][j]]['last_name']
                pos = playerData[starters[i][j]]['position']
                fullName = fName + " " + lName

                # iterate through tierlists based on pos
                # DEF, WR, TE, K, RB
                #print(len(tierListQB))
                
                # len is amount of tiers
                if pos == "QB":
                    q = 0
                    while q < len(tierListQB):
                        if fullName in tierListQB[q]:
                            tier = q
                        q = q + 1

                if pos == "RB":
                    q = 0
                    while q < len(tierListRB):
                        if fullName in tierListRB[q]:
                            tier = q
                        q = q + 1

                if pos == "WR":
                    q = 0
                    while q < len(tierListWR):
                        if fullName in tierListWR[q]:
                            tier = q
                        q = q + 1

                if pos == "K":
                    q = 0
                    while q < len(tierListK):
                        if fullName in tierListK[q]:
                            tier = q
                        q = q + 1   

                if pos == "DEF":
                    q = 0
                    while q < len(tierListDST):
                        if fullName in tierListDST[q]:
                            tier = q
                        q = q + 1     

                if pos == "TE":
                    q = 0
                    while q < len(tierListTE):
                        if fullName in tierListTE[q]:
                            tier = q
                        q = q + 1   

                
                tier = tier + 1
                tierSum = tier + tierSum

                
                print(fName + " " + lName + " " + "[" +  "Tier " + str(tier) + "]")
            j = j + 1

    
    print("\nAverage Tier of Starters is (lower is better): " + str(tierSum / (len(starters[i]))))
    # bench
    print("\nBench:")
    for key in playerData:
        # key is definitely the ids
        # should iterate through 5
        j = 0
        while j < len(bench):
            if key == bench[j]:
                fName = playerData[bench[j]]['first_name']
                lName = playerData[bench[j]]['last_name']
                pos = playerData[bench[j]]['position']
                fullName = fName + " " + lName

                if pos == "QB":
                    q = 0
                    while q < len(tierListQB):
                        if fullName in tierListQB[q]:
                            tier = q
                        q = q + 1

                if pos == "RB":
                    q = 0
                    while q < len(tierListRB):
                        if fullName in tierListRB[q]:
                            tier = q
                        q = q + 1

                if pos == "WR":
                    q = 0
                    while q < len(tierListWR):
                        if fullName in tierListWR[q]:
                            tier = q
                        q = q + 1

                if pos == "K":
                    q = 0
                    while q < len(tierListK):
                        if fullName in tierListK[q]:
                            tier = q
                        q = q + 1   

                if pos == "DEF":
                    q = 0
                    while q < len(tierListDST):
                        if fullName in tierListDST[q]:
                            tier = q
                        q = q + 1     

                if pos == "TE":
                    q = 0
                    while q < len(tierListTE):
                        if fullName in tierListTE[q]:
                            tier = q
                        q = q + 1   

                tier = tier + 1

                

                print(fName + " " + lName + " " + "[" +  "Tier " + str(tier) + "]")
            j = j + 1

    
    # end of main while    
    i = i + 1

