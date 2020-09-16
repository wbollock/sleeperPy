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
import sys 



# Variables 


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


# ISSUES/TODO
# TODO: not taking account into flex, e.g noah fant tier 5 better than desean tier 7
# TODO: needs functions
# TODO: waiver wire suggestions would be great, especially for DST/K. If player on WW is higher tier, mention it.
# TODO: sorting by tier would be cool
# TODO: converting.txt file to HTML would be prudent, and mobile-friendly. also dark mode i can't stand this shit
# TODO: add average tier of opponent vs average tier of you

# BUGS:
# Average Tier doesn't always work
# https://wboll.dev/sleeperPy/tiers/tiers_StuMac.txt
# Average is 13.5
# https://wboll.dev/sleeperPy/tiers/tiers_Nr2016.txt
# Average 13.8

# API
# https://docs.sleeper.app/
# grab userid from username
# curl "https://api.sleeper.app/v1/user/<username>"
# curl "https://api.sleeper.app/v1/user/<user_id>"
#curl "https://api.sleeper.app/v1/user/puffplants"
# https://api.sleeper.app/v1/user/<user_id>/leagues/<sport>/<season>
# get all leagues for user
# curl "https://api.sleeper.app/v1/user/470054939452764160/leagues/nfl/2020"

# Tiers
# https://github.com/abhinavk99/espn-borischentiers/blob/master/src/js/espn-borischentiers.js

# Web Arguments
# total arguments 
n = len(sys.argv) 
if n < 2:
    print("Error: please enter your Sleeper username.")
elif n > 2:
    print("Error: Too many arguments. Please type your sleeper username.")




#username = "KingDedede"
#username = "puffplants"
#username = "Jz904"

username = str(sys.argv[1])


tiersFilename = "tiers_" + username + ".txt"
tiersFilepath = "tiers/" + tiersFilename

# open text file for writing
if not os.path.isdir('tiers/'):
    os.mkdir('tiers/')
sys.stdout = open(tiersFilepath, "w")
os.chmod(tiersFilepath, 0o666)


# Get USERID
url = f"https://api.sleeper.app/v1/user/{username}"

r = requests.get(url)

data = r.json()

try:
    userid = data['user_id']
except TypeError:
    print("Sorry, invalid Sleeper username. Please try again.")
    sys.exit()
# dict type

# Get all leagues for user
url = f"https://api.sleeper.app/v1/user/{userid}/leagues/{sport}/{year}"

r = requests.get(url)

data = r.json()
# list type

leagues = []
leagueNames = []
scoring = []
i = 0

for d in data:
    leagues.append(d['league_id'])
    leagueNames.append(d['name'])
    scoring.append(d['scoring_settings']['rec'])


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
for league in leagues:
    # print league name
    #print("League" + ": " + str(leagueNames[i]))

    # Get current roster
    url = f"https://api.sleeper.app/v1/league/{league}/rosters"
    r = requests.get(url)
    data = r.json()
    # data is a list here
    # length is number of teams
    for d in data:
        # for every team in league, do
        if d['owner_id'] == userid:
            # then this is current user
            starters.append(d['starters'])
            players.append(d['players'])
            # shit but multiple leagues



# for each league
# print player names, cross reference with playerData
# playerData is a dict
# players/starters are lists

#print(players[0][1])
# second value from first list of players
# first list of players, 15

# Print roster from each league, bench and starters

i = 0
bench = []
print("SleeperPy: Boris Chen Tiers for Sleeper Leagues\n")
print("Note:")
#print("Tier 100 means player is not properly ranked on Boris Chen.")
print("[X] indicates the tier of the player. Lower is better.")
print("\nUsername:", username)

while i < len(starters):
    # for each league, do:
    print("\n#######################")
    print(f"League: {leagueNames[i]}")
    print("#######################\n")
    

    starterList = []
    benchList = []

    qbStarterList = []
    rbStarterList = []
    wrStarterList = []
    teStarterList = []
    dstStarterList = []
    kStarterList = []

    qbBenchList = []
    rbBenchList = []
    wrBenchList = []
    teBenchList = []
    dstBenchList = []
    kBenchList = []
    # mode = scoringMode(scoring)

    # figure out what lists to use
    # god i dont think this should be here
    qbBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_QB.txt"
    dstBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_DST.txt"
    kBoris = "https://s3-us-west-1.amazonaws.com/fftiers/out/text_K.txt"

    scoring_to_text_map = {1.0: '-PPR', 0.5: '-HALF', 0.0: ''}
    rbBoris = f"https://s3-us-west-1.amazonaws.com/fftiers/out/text_RB{scoring_to_text_map[scoring[i]]}.txt"
    wrBoris = f"https://s3-us-west-1.amazonaws.com/fftiers/out/text_WR{scoring_to_text_map[scoring[i]]}.txt"
    teBoris = f"https://s3-us-west-1.amazonaws.com/fftiers/out/text_TE{scoring_to_text_map[scoring[i]]}.txt"


    if scoring[i] == 1.0:
        print("Scoring Type: PPR\n")
    elif scoring[i] == 0.5:
        print("Scoring Type: Half PPR\n")
    elif scoring[i] == 0.0:
        print("Scoring Type: Standard\n")


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
    print("**Starters:**")
    for key in playerData:
        # key is definitely the ids
        j = 0
        tier = 0
        
        while j < len(starters[i]):
            if key == starters[i][j]:
                # one player from each loop... add tiers here i guess?
                fName = playerData[starters[i][j]]['first_name']  
                lName = playerData[starters[i][j]]['last_name']
                pos = playerData[starters[i][j]]['position']
                fullName = fName + " " + lName

                # iterate through tierlists based on pos
                # DEF, WR, TE, K, RB, QB
            
                # len is amount of tiers
                
                if pos == "QB":
                    for q in range(len(tierListQB)):
                        if fullName in tierListQB[q]:
                            tier = q + 1
                            qbStarterList.append(f"{fullName} [{tier}]")
                    tier = tier + 1


                if pos == "RB":
                    for q in range(len(tierListRB)):
                        if fullName in tierListRB[q]:
                            tier = q + 1
                            rbStarterList.append(f"{fullName} [{tier}]")

                if pos == "WR":
                    for q in range(len(tierListWR)):
                        if fullName in tierListWR[q]:
                            tier = q + 1
                            wrStarterList.append(f"{fullName} [{tier}]")

                if pos == "K":
                    for q in range(len(tierListK)):
                        if fullName in tierListK[q]:
                            tier = q + 1
                            kStarterList.append(f"{fullName} [{tier}]")

                if pos == "DEF":
                    for q in range(len(tierListDST)):
                        if fullName in tierListDST[q]:
                            tier = q + 1
                            dstStarterList.append(f"{fullName} [{tier}]")

                if pos == "TE":
                    for q in range(len(tierListTE)):
                        if fullName in tierListTE[q]:
                            tier = q + 1
                            teStarterList.append(f"{fullName} [{tier}]")
                tierSum = tier + tierSum
                tier = tier + 1
                starterList.append(f"{fName} {lName} [{pos}] [Tier {tier}]")

            j = j + 1

    y = 0

    print("--QB---")
    print(*qbStarterList, sep = "\n")

    print("--WR---")
    print(*wrStarterList, sep = "\n")

    print("--RB---")
    print(*rbStarterList, sep = "\n")

    print("--TE---")
    print(*teStarterList, sep = "\n")

    print("--DST---")
    print(*dstStarterList, sep = "\n")

    print("--K---")
    print(*kStarterList, sep = "\n")
            

    tierSum = tierSum - 1
    print(f"\nAverage Tier of Starters is: {round(tierSum / (len(starters[i])),3)}")
    # bench
    print("\nBench:")
    for key in playerData:
        # key is definitely the ids
        # should iterate through 5
        for b in bench:
            if key == b:
                fName = playerData[b]['first_name']
                lName = playerData[b]['last_name']
                pos = playerData[b]['position']
                fullName = fName + " " + lName

                if pos == "QB":
                    for q in range(len(tierListQB)):
                        if fullName in tierListQB[q]:
                            tier = q + 1
                            qbBenchList.append(f"{fullName} [{tier}]")

                if pos == "RB":
                    for q in range(len(tierListRB)):
                        if fullName in tierListRB[q]:
                            tier = q + 1
                            rbBenchList.append(f"{fullName} [{tier}]")

                if pos == "WR":
                    for q in range(len(tierListWR)):
                        if fullName in tierListWR[q]:
                            tier = q + 1
                            wrBenchList.append(f"{fullName} [{tier}]")

                if pos == "K":
                    for q in range(len(tierListK)):
                        if fullName in tierListK[q]:
                            tier = q + 1
                            kBenchList.append(f"{fullName} [{tier}]")

                if pos == "DEF":
                    for q in range(len(tierListDST)):
                        if fullName in tierListDST[q]:
                            tier = q + 1
                            dstBenchList.append(f"{fullName} [{tier}]")

                if pos == "TE":
                    for q in range(len(tierListTE)):
                        if fullName in tierListTE[q]:
                            tier = q + 1
                            teBenchList.append(f"{fullName} [{tier}]")

                tier = tier + 1
                benchList.append(f"{fName} {lName} [{pos}] [Tier {tier}]")

    y = 0
    
    if len(qbBenchList) > 0:
        print("\n--QB---")
        print(*qbBenchList, sep = "\n")

    if len(wrBenchList) > 0:
        print("\n--WR---")
        print(*wrBenchList, sep = "\n")

    if len(rbBenchList) > 0:
        print("\n--RB---")
        print(*rbBenchList, sep = "\n")

    if len(teBenchList) > 0:
        print("\n--TE---")
        print(*teBenchList, sep = "\n")

    if len(dstBenchList) > 0:
        print("\n--DST---")
        print(*dstBenchList, sep = "\n")

    if len(kBenchList) > 0:
        print("\n--WR---")
        print(*kBenchList, sep = "\n")

    #print(data)
    
    # end of main while    
    i = i + 1

