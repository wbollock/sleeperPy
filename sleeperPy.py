#!/usr/bin/env python3
# sleeperPy

import requests
import json
import os, time
from pathlib import Path
import sys 
import fileinput
from shutil import copyfile

# Variables
sport = "nfl"
year = "2020"
playersFile = "players.txt"
htmlFile  = "tiers.html"


# Functions

# used to determine age of players.txt, if > 1 day then download it again
def file_age(filepath):
    return time.time() - os.path.getmtime(filepath)

# used to find "bench" by finding the difference between total team and starters
def Diff(li1, li2): 
    li_dif = [i for i in li1 + li2 if i not in li1 or i not in li2] 
    return li_dif

# used to generate HTML with lists of players and their matching tiers
def printTiers(playerList, tierList, pos):
    outputList = []
    if (len(playerList) > 0):
        print("<tr>")
        print("<th>" + pos + "</th>")
        print("<th>Tier</th>")
        print("</tr>")
        for x in range(len(playerList)):
            outputList.append("<tr>" + "<td>" + playerList[x] + "</td>" + "<td>" + tierList[x] + "</td>" + "</tr>")
    return outputList

def createTiers(tierListPos, fullName, posPlayerList, posTierList, tier):
    for q in range(len(tierListPos)):
        # tierListPos[q] means: Tier 1: Lamar Jackson, Dak Prescott, Patrick Mahomes II
        # iterates through each line of tier
        if fullName in tierListPos[q]:
            tier = q + 1
            posPlayerList.append(f"{fullName}")
            posTierList.append(f"{tier}")

    return posPlayerList, posTierList, tier

def createUnranked(tierListPos, fullName):
    flag = False
    if any(fullName in word for word in tierListPos):
        flag = True
    if flag == False:
        return (f"{fullName}")
    else:
        return "ranked"


# ISSUES/TODO
# TODO: not taking account into flex, e.g noah fant tier 5 better than desean tier 7
# TODO: waiver wire suggestions would be great, especially for DST/K. If player on WW is higher tier, mention it.
# TODO: sorting by tier would be cool
# TODO: add average tier of opponent vs average tier of you


# API
# https://docs.sleeper.app/


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


tiersFilename = "tiers_" + username + ".html"
tiersFilepath = "tiers/" + tiersFilename

# mkdir if not exists
if not os.path.isdir('tiers/'):
    os.mkdir('tiers/')

# delete file if exists already
if os.path.exists(tiersFilepath):
  os.remove(tiersFilepath)


# copy template html file to user specific file
copyfile(htmlFile, tiersFilepath )
os.chmod(tiersFilepath, 0o666)
sys.stdout = open(tiersFilepath, "a")


# Get USERID
url = f"https://api.sleeper.app/v1/user/{username}"
r = requests.get(url)
data = r.json()

# attempt to see if we can get any data from sleeper for inputted username
try:
    userid = data['user_id']
except TypeError:
    print("Sorry, invalid Sleeper username. Please try again.")
    sys.exit()


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


i = 0
bench = []

print("<h5>Username: " + username + "</h5>")
print("</div>")
print("</div>")
print("<div class=\"flex-container container\">")

while i < len(starters):
    # for each league, do:

    starterList,benchList = [],[]

    # ur = unranked
    qbStarterList,rbStarterList,wrStarterList,teStarterList,dstStarterList,kStarterList,urStarterList = [],[],[],[],[],[],[]

    qbTierList,rbTierList,wrTierList,dstTierList,teTierList,kTierList = [],[],[],[],[],[]
    
    qbBenchList,rbBenchList,wrBenchList,teBenchList,dstBenchList,kBenchList,urBenchList = [],[],[],[],[],[],[]

    qbTierBenchList,rbTierBenchList,wrTierBenchList,teTierBenchList,dstTierBenchList,kTierBenchList = [],[],[],[],[],[]
     
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
    
   
    
    print("<table class=\"table-fill\">")
    if scoring[i] == 1.0:
        print(f"<th class=\"league-title\" colspan=\"2\">League: {leagueNames[i]} (PPR) | Starters</th>")
    elif scoring[i] == 0.5:
        print(f"<th class=\"league-title\" colspan=\"2\">League: {leagueNames[i]} (Half PPR) | Starters</th>")
    elif scoring[i] == 0.0:
        print(f"<th class=\"league-title\" colspan=\"2\">League: {leagueNames[i]} (Standard) | Starters</th>")

    print("</tr>")


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
    
    for key in playerData:
        # key is definitely the ids
        j = 0
        tier = 0
        
        while j < len(starters[i]):
            if key == starters[i][j]:
                # one player from each loop... 
                fName = playerData[starters[i][j]]['first_name']  
                lName = playerData[starters[i][j]]['last_name']
                pos = playerData[starters[i][j]]['position']
                fullName = fName + " " + lName

                # iterate through tierlists based on pos
                # DEF, WR, TE, K, RB, QB
            
                # len is amount of tiers
                # Kenny Murphy credit for flags
                if pos == "QB":
                    qbStarterList, qbTierList, tier = createTiers(tierListQB,fullName,qbStarterList,qbTierList,tier)
                    if createUnranked(tierListQB,fullName) != "ranked":
                        urStarterList.append(createUnranked(tierListQB,fullName))
        
                if pos == "RB":
                    rbStarterList, rbTierList, tier = createTiers(tierListRB,fullName,rbStarterList,rbTierList,tier)
                    if createUnranked(tierListRB,fullName) != "ranked":
                        urStarterList.append(createUnranked(tierListRB,fullName))
              
                if pos == "WR":
                    wrStarterList, wrTierList, tier = createTiers(tierListWR,fullName,wrStarterList,wrTierList,tier)
                    if createUnranked(tierListWR,fullName) != "ranked":
                        urStarterList.append(createUnranked(tierListWR,fullName))

                if pos == "K":
                    kStarterList, kTierList, tier = createTiers(tierListK,fullName,kStarterList,kTierList,tier)
                    if createUnranked(tierListK,fullName) != "ranked":
                        urStarterList.append(createUnranked(tierListK,fullName))

                if pos == "DEF":
                    dstStarterList, dstTierList, tier = createTiers(tierListDST,fullName,dstStarterList,dstTierList,tier)
                    if createUnranked(tierListDST,fullName) != "ranked":
                        urStarterList.append(createUnranked(tierListDST,fullName))

                if pos == "TE":
                    teStarterList, teTierList, tier = createTiers(tierListTE,fullName,teStarterList,teTierList,tier)
                    if createUnranked(tierListTE,fullName) != "ranked":
                        urStarterList.append(createUnranked(tierListTE,fullName))
                    

                tierSum = tier + tierSum
                tier = tier + 1
                #starterList.append(f"{fName} {lName} [{pos}] [Tier {tier}]")
                
            j = j + 1
            
                

    y = 0

    
    # returns list with HTML to print to page
    outputList = printTiers(qbStarterList, qbTierList, "QB")
    for x in range(len(outputList)):
        print(outputList[x])
    outputList = printTiers(rbStarterList, rbTierList, "RB")
    for x in range(len(outputList)):
        print(outputList[x])
    outputList = printTiers(wrStarterList, wrTierList, "WR")
    for x in range(len(outputList)):
        print(outputList[x])
    outputList = printTiers(teStarterList, teTierList, "TE")
    for x in range(len(outputList)):
        print(outputList[x])
    outputList = printTiers(kStarterList, kTierList, "K")
    for x in range(len(outputList)):
        print(outputList[x])
    outputList = printTiers(dstStarterList, dstTierList, "DST")
    for x in range(len(outputList)):
        print(outputList[x])

    if (len(urStarterList) > 0):
        print("<tr>")
        print("<th colspan=\"2\" style=\"text-align: center\">Not Ranked</th>")
        print("</tr>")
        for x in range(len(urStarterList)):
            print("<tr>")
            print("<td>" + urStarterList[x] + "</td>")
            print("</tr>")

            

    tierSum = tierSum - 1
    print(f"<tr><td colspan=\"2\" style=\"text-align: center\">Average Tier {round(tierSum / (len(starters[i])),3)}</td></tr>")
    # bench
    print("<br>")
    print("<br>")
    for key in playerData:
        # key is definitely the ids
        # should iterate through 5
        # total tier unused in bench but might as well have it
        tier = 0
        for b in bench:
            if key == b:
                fName = playerData[b]['first_name']
                lName = playerData[b]['last_name']
                pos = playerData[b]['position']
                fullName = fName + " " + lName
                
                if pos == "QB":
                    qbBenchList, qbTierBenchList, tier = createTiers(tierListQB,fullName,qbBenchList,qbTierBenchList,tier)
                    if createUnranked(tierListQB,fullName) != "ranked":
                        urBenchList.append(createUnranked(tierListQB,fullName))
        
                if pos == "RB":
                    rbBenchList, rbTierBenchList, tier = createTiers(tierListRB,fullName,rbBenchList,rbTierBenchList,tier)
                    if createUnranked(tierListRB,fullName) != "ranked":
                        urBenchList.append(createUnranked(tierListRB,fullName))
              
                if pos == "WR":
                    wrBenchList, wrTierBenchList, tier = createTiers(tierListWR,fullName,wrBenchList,wrTierBenchList,tier)
                    if createUnranked(tierListWR,fullName) != "ranked":
                        urBenchList.append(createUnranked(tierListWR,fullName))

                if pos == "K":
                    kBenchList, kTierBenchList, tier = createTiers(tierListK,fullName,kBenchList,kTierBenchList,tier)
                    if createUnranked(tierListK,fullName) != "ranked":
                        urBenchList.append(createUnranked(tierListK,fullName))

                if pos == "DEF":
                    dstBenchList, dstTierBenchList, tier = createTiers(tierListDST,fullName,dstBenchList,dstTierBenchList,tier)
                    if createUnranked(tierListDST,fullName) != "ranked":
                        urBenchList.append(createUnranked(tierListDST,fullName))

                if pos == "TE":
                    teBenchList, teTierBenchList, tier = createTiers(tierListTE,fullName,teBenchList,teTierBenchList,tier)
                    if createUnranked(tierListTE,fullName) != "ranked":
                        urBenchList.append(createUnranked(tierListTE,fullName))
               

                tier = tier + 1
                

    y = 0
    
    print("<tr>")
    print("<th colspan=\"2\" style=\"text-align:center;\">Bench</th>")
    print("</tr>")

    
    outputList = printTiers(qbBenchList, qbTierBenchList, "QB")
    for x in range(len(outputList)):
        print(outputList[x])
    outputList = printTiers(rbBenchList, rbTierBenchList, "RB")
    for x in range(len(outputList)):
        print(outputList[x])
    outputList = printTiers(wrBenchList, wrTierBenchList, "WR")
    for x in range(len(outputList)):
        print(outputList[x])
    outputList = printTiers(teBenchList, teTierBenchList, "TE")
    for x in range(len(outputList)):
        print(outputList[x])
    outputList = printTiers(kBenchList, kTierBenchList, "K")
    for x in range(len(outputList)):
        print(outputList[x])
    outputList = printTiers(dstBenchList, dstTierBenchList, "DST")
    for x in range(len(outputList)):
        print(outputList[x])

    if (len(urBenchList) > 0):
        print("<tr>")
        print("<th colspan=\"2\" style=\"text-align: center\">Not Ranked</th>")
        print("</tr>")
        for x in range(len(urBenchList)):
            print("<tr>")
            print("<td colspan=\"2\">" + urBenchList[x] + "</td>")
            print("</tr>")

    # end of all output for league
    print("</table>")
    # end of main while    
    i = i + 1

print("</div>")
print("<br>")
print("</body>")
print("</html>")


