#!/usr/bin/env python3
# sleeperPy

# REQUIREMENTS:
# pip3 install bs4
# if apache is root, sudo pip3 install bs4

# ISSUES/TODO
# TODO: not taking account into flex, e.g noah fant tier 5 better than desean tier 7
# TODO: waiver wire suggestions would be great, especially for DST/K. If player on WW is higher tier, mention it.
# looks like i would have to curl "https://api.sleeper.app/v1/players/nfl/trending/add" and keep ~top 5
# then determine if that player is rostered in any team, seperately
# TODO: easy one, unranked player doesn't count towards the tier. maybe make them tier 12?


# API
# https://docs.sleeper.app/

# Bugs:
# Craig Stevens
# https://www.reddit.com/r/fantasyfootball/comments/jszktk/like_boris_chen_tiers_and_use_sleeper_simple_tool/gc2mqa7/
# has to be *not* having a player in a position, like no kicker

# players.txt perm issue
# check apache log in next few days, might've fixed it
# PHP Warning:  file_put_contents(players.txt): failed to open stream: Permission denied in /var/www/html/sleeperPy/index.php on line 40, referer: https://wboll.dev/sleeperPy/


# Tiers
# https://github.com/abhinavk99/espn-borischentiers/blob/master/src/js/espn-borischentiers.js



# TODO: lmao clean up these imports
import requests
import json
import os, time
from pathlib import Path
import sys 
import fileinput
from shutil import copyfile
from urllib.request import urlopen
import re
from bs4 import BeautifulSoup
from datetime import datetime
from operator import itemgetter
import logging, time
from logging.handlers import RotatingFileHandler
from pymongo import MongoClient
import os.path
from os import path
from pathlib import Path
import calendar
import urllib.request, json 

# Variables
sport = "nfl"
# current year, e.g 2020
year = datetime.now().strftime('%Y')
# template file
htmlFile  = "tiers.php"
# logging
loggingFile = "sleeperPy.log"
playersFile = "players.json"

dbName="sleeperPy"
collectionName="players"



# Functions
def mongoConnect():
    # connect to mongodb
    client = MongoClient()
    # default host/port
    db = client[dbName]
    collection = db[collectionName]
    return db,collection

def mongoImport():
    playersPath = Path(playersFile)

    url = "https://api.sleeper.app/v1/players/nfl"
    today = datetime.today()

    print("mongoimport --db " + dbName + " --collection " + collectionName +" --file " + playersFile)
    exit
    # see if we need a new players.json
    if playersPath.is_file():
        # 8 is last modified
        t = os.stat(playersPath)[8] 
        filetime = today - datetime.fromtimestamp(t) 

        if int(filetime.seconds) > 1:
            # if it's last been modified longer than a day ago, download new players.txt
            with urllib.request.urlopen(url) as url:
                data = json.loads(url.read().decode())
                with open(playersFile, 'w') as f:
                    json.dump(data,f)
                    f.close()

            # clean up existing data
            collection.delete_many({})
            # import new stuff
            os.system("mongoimport --db " + dbName + " --collection " + collectionName +" --file " + playersFile)

    # if file doesn't exist at all"
    else:
        with urllib.request.urlopen(url) as url:
                data = json.loads(url.read().decode())
                with open(playersFile, 'w') as f:
                    json.dump(data,f)
                    f.close()

                # clean up existing data
                collection.delete_many({})
                # import new stuff
                os.system("mongoimport --db " + dbName + " --collection " + collectionName +" --file " + playersFile)

    print("nothing")

def Diff(li1, li2):
    # used to find "bench" by finding the difference between total team and starters
    li_dif = [i for i in li1 + li2 if i not in li1 or i not in li2] 
    return li_dif

def sortLists(list1, list2):
    # list1 should be tiers, list2 players
    # sort player list and tier list together, keeping values
    # sort by lowest value (highest tier) first
    if type(list1) == "None" or type(list2) == "None":
        return list1, list2

    if len(list1) == 0 or len(list2) == 0:
        return list1, list2

    # force tiers to ints
    list1 = [int(i) for i in list1]
    # reverse=True
    # damn this works!!!
    # https://stackoverflow.com/questions/9764298/how-to-sort-two-lists-which-reference-each-other-in-the-exact-same-way
    list1, list2 = (list(t) for t in zip(*sorted(zip(list1, list2),key=itemgetter(0))))
    return list1, list2


def printTiers(playerList, tierList, pos):
    # used to generate HTML with lists of players and their matching tiers
    outputList = []
    #playerList, tierList = sortLists(playerList, tierList)
    tierList, playerList = sortLists(tierList, playerList)
    # print(*tierList)
    # print(*playerList)
    if (len(playerList) > 0):
        print("<tr>")
        print("<th>" + pos + "</th>")
        print("<th>Tier</th>")
        print("</tr>")
        for x in range(len(playerList)):
            outputList.append("<tr>" + "<td>" + playerList[x] + "</td>" + "<td>" + str(tierList[x]) + "</td>" + "</tr>")
    return outputList

def validateBoris(tierListPos):
    # fixing inconsistencies as i find them between boris chen and sleeper player names
    # ok this works at least but i think i have to edit the list
    # BorisName, Sleeper Name
    tierListPos = [w.replace('D.K. Metcalf', 'DK Metcalf') for w in tierListPos]
    tierListPos = [w.replace('Jeff Wilson Jr.', 'Jeffery Wilson') for w in tierListPos]
    tierListPos = [w.replace('JaMycal Hasty', 'Jamycal Hasty') for w in tierListPos]
    return tierListPos

def createTiers(tierListPos, fullName, posPlayerList, posTierList, tier):  
    # find the players name in a tier list, when found also note their tier
    tierListPos = validateBoris(tierListPos)
    for q in range(len(tierListPos)):
    # tierListPos[q] means: Tier 1: Lamar Jackson, Dak Prescott, Patrick Mahomes II
    # iterates through each line of tier
        if fullName in tierListPos[q]:
            tier = q + 1
            posPlayerList.append(f"{fullName}")
            posTierList.append(f"{tier}")

    return posPlayerList, posTierList, tier


def createUnranked(tierListPos, fullName):
    # go through entire tier list for a position, if player name not in any of them, they are not ranked
    flag = False
    tierListPos = validateBoris(tierListPos)
    if any(fullName in word for word in tierListPos):
        flag = True
    if flag == False:
        return (f"{fullName}")
    else:
        return "ranked"

# temporarily retired in favor of shitty calendar math
# def currentWeek():
#     # somehow return current NFL week, e.g week 4
#     # im screwed if this changes
#     url = "https://www.espn.com/nfl/lines"
#     page = urlopen(url)
#     html = page.read().decode("utf-8")
#     soup = BeautifulSoup(html, "html.parser")
#     # god this regex sucks
#     page = soup.get_text()
#     # should work for weeks 10-17 too
#     #pattern = "Week [1-9]|[1-9][0-9]"
#     pattern = "(Week [0-9][1-9])|(Week [1-9][0-9])|(Week [1-9])"
#     week = re.search(pattern, page)
#     week = [int(i) for i in str(week.group()).split() if i.isdigit()]
#     return week[0]

def currentWeek():
    # if the start of the NFL season isn't ~week 36 of the year, change that value

    # between like August -> September 9th (Thursday) (2021), it's week 1. 
    # Final games in week 1 would be 13th.. so september 14th - 21st is week 2.. 22nd - 29th is week 3
    # ending on Jan 9th, 2020, a Sunday
    currentWeek = datetime.today().isocalendar()[1]
    # if july 17 is week 28.. then september 9th would be week 37... and january 9th 2022 is week.. 2 of 2022

    # lmao, $currentWeek - 36 is technically right. but man that's nasty to change in the future
    currentNFLWeek = currentWeek - 36
    # if it's postseason, cap at 16. wait this'll screwup for 2020
    if currentNFLWeek > 16:
        currentNFLWeek == 16
    # a beautiful elif for week 1/2 of 2022, the last week.. ugh
    elif currentWeek == 1 | 2:
        currentNFLWeek == 16
    # basically offseason, if negative weeks then 
    else:
        currentNFLWeek = max(1,currentNFLWeek)

        

    return currentNFLWeek

# Rotating application logging I stole from the Internet
def create_rotating_log(path):

    logger = logging.getLogger("Rotating Log")
    logger.setLevel(logging.DEBUG)
    
    # add a rotating handler
    handler = RotatingFileHandler(path, maxBytes=2000000,
                                  backupCount=5)
    logger.addHandler(handler)
    
    # logging.basicConfig(stream=sys.stdout, level=logging.DEBUG)
        
    log_file = path
    
    root = logging.getLogger()
    root.setLevel(logging.DEBUG)

    # handler = logging.StreamHandler(sys.stdout)
    handler.setLevel(logging.DEBUG)
    formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
    handler.setFormatter(formatter)
    root.addHandler(handler)
    

create_rotating_log(loggingFile)
db, collection = mongoConnect()
mongoImport()

# logger = logging.getLogger()



# get username argument
n = len(sys.argv) 
if n < 2:
    print("Error: please enter your Sleeper username.")
elif n > 2:
    print("Error: Too many arguments. Please type your sleeper username.")

username = str(sys.argv[1])


tiersFilename = "tiers_" + username + ".php"
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

# data is a dict here
# print roster for both leagues:

i = 0
starters = []
players = []
# roster_id = []
# matchup_id = []

oppStarters = []
oppPlayers = []



print("<h5>Username: " + username + " - Week " + str(currentWeek()) + "</h5>")
print("<div class=\"buttonholder\"><form action=\"\" method=\"post\"> <input type=\"submit\" name=\"submit\" value=\"Refresh Tiers\" /></form>")
print("</div>")
print("</div>")
print("</div>")
print("<div class=\"flex-container container\">")


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
            roster_id = d['roster_id']
    
    week = currentWeek()
    url = f"https://api.sleeper.app/v1/league/{league}/matchups/{week}"
    r = requests.get(url)
    data = r.json()

    for d in data:
        if roster_id == d['roster_id']:
            # if player matchup is found via roster id
            matchup_id = d['matchup_id'] 
            # get matchup id
           
    
    for d in data:
        if matchup_id == d['matchup_id']:
                if roster_id != d['roster_id']:
                    # if you found a matching matchup ID but NOT matching roster_id, must be opponent
                    oppStarters.append(d['starters'])
                    oppPlayers.append(d['players'])


    
    bench = []

    starterList,benchList = [],[]

    # ur = unranked
    qbStarterList,rbStarterList,wrStarterList,teStarterList,dstStarterList,kStarterList,urStarterList = [],[],[],[],[],[],[]

    qbTierList,rbTierList,wrTierList,dstTierList,teTierList,kTierList = [],[],[],[],[],[]
    
    qbBenchList,rbBenchList,wrBenchList,teBenchList,dstBenchList,kBenchList,urBenchList = [],[],[],[],[],[],[]

    qbTierBenchList,rbTierBenchList,wrTierBenchList,teTierBenchList,dstTierBenchList,kTierBenchList = [],[],[],[],[],[]
     
    # mode = scoringMode(scoring)

    # figure out what lists to use
    # below 3 are guranteed regardless of scoring
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
    # tfw when python has no switch/case
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
    
    # get bench by subtracting all players by starters
    bench = Diff(players[i], starters[i])
    
    tierSum = 0

    # Time Start
    # from here to end of for loop by far slowest part of program, ~2.7 seconds
    # start_time = datetime.now() 
    # CODE HERE
    # time_elapsed = datetime.now() - start_time 

    # print("Time1")
    # print('Time elapsed (hh:mm:ss.ms) {}'.format(time_elapsed))

    # before: iterated through *every* key in players.txt
    # if that key HAPPENED to match their key, starters[i][j], then do shit
    # that sucks horribly

    # instead, just loop through their teams' players
        # instead, just loop through their teams' players

    j = 0
    tier = 0
    # key = {}
    
    # looping through starters[i] to get all IDs of starters, e.g 4881
    while j < len(starters[i]):
        for key in collection.distinct(str(starters[i][j])):
            fName = key['first_name']  
            lName = key['last_name']
            pos = key['position']
            fullName = fName + " " + lName

            # iterate through tierlists based on pos
            # DEF, WR, TE, K, RB, QB
        
            # len is amount of tiers
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
        j = j + 1
    
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

    # Sometimes goes out of bounds and grabs "Craig Stevens"
    badList = ['Craig Stevens']
    urStarterList = [element for element in urStarterList if element not in badList]
    if (len(urStarterList) > 0):
        print("<tr>")
        print("<th colspan=\"2\" style=\"text-align: center\">Not Ranked</th>")
        print("</tr>")
        #urStarterList.remove("Craig Stevens")
        for x in range(len(urStarterList)):
            print("<tr>")
            print("<td colspan=\"2\">" + urStarterList[x] + "</td>")
            print("</tr>")

    # Time End, 2.7 seconds

    qbOppTierList,rbOppTierList,wrOppTierList,dstOppTierList,teOppTierList,kOppTierList = [],[],[],[],[],[]
    qbOppList,rbOppList,wrOppList,teOppList,dstOppList,kOppList,urOppList = [],[],[],[],[],[],[]
    tierOppSum = 0
    # Calculate Opponent Tiers
    # not the best test, but if oppStarters has no length, don't do any of this
    if oppStarters:
        j = 0
        tier = 0
        # key = {}
        try:
            # looping through starters[i] to get all IDs of starters, e.g 4881
            while j < len(oppStarters[i]):
                for key in collection.distinct(str(oppStarters[i][j])):
                    fName = key['first_name']  
                    lName = key['last_name']
                    pos = key['position']
                    fullName = fName + " " + lName

                    if pos == "QB":
                        qbOppList, qbOppTierList, tier = createTiers(tierListQB,fullName,qbOppList,qbOppTierList,tier)
                    if pos == "RB":
                        rbOppList, rbOppTierList, tier = createTiers(tierListRB,fullName,rbOppList,rbOppTierList,tier)
                    if pos == "WR":
                        wrOppList, wrOppTierList, tier = createTiers(tierListWR,fullName,wrOppList,wrOppTierList,tier)
                    if pos == "K":
                        kOppList, kOppTierList, tier = createTiers(tierListK,fullName,kOppList,kOppTierList,tier)
                    if pos == "DEF":
                        dstOppList, dstOppTierList, tier = createTiers(tierListDST,fullName,dstOppList,dstOppTierList,tier)
                    if pos == "TE":
                        teOppList, teOppTierList, tier = createTiers(tierListTE,fullName,teOppList,teOppTierList,tier)

                    tierOppSum = tier + tierOppSum
                    tier = tier + 1
                j = j + 1
        except IndexError:
            avgTier = round(tierSum / (len(starters[i])),2)
            print(f"<tr><td colspan=\"2\" style=\"text-align: center\">Average Tier {avgTier}</td></tr>")
            break
        try:
            tierOppSum = tierOppSum - 1
            tierSum = tierSum - 1
            avgTier = round(tierSum / (len(starters[i])),2)
            avgOppTier = round(tierOppSum / (len(oppStarters[i])),2)
                
            # &#127942; = trophy
            # &#128201; = down line
            # &#128528; = neutral face
            # higher tier is worse
            if avgTier < avgOppTier:
                print(f"<tr><td colspan=\"2\" style=\"text-align: center\">&#127942; Average Tier {avgTier}</td></tr>")
                print(f"<tr><td colspan=\"2\" style=\"text-align: center\">&#128201; Opponent Average Tier {avgOppTier}</td></tr>")
            elif avgOppTier < avgTier:
                print(f"<tr><td colspan=\"2\" style=\"text-align: center\">&#128201; Average Tier {avgTier}</td></tr>")
                print(f"<tr><td colspan=\"2\" style=\"text-align: center\">&#127942; Opponent Average Tier {avgOppTier}</td></tr>")
            elif avgOppTier == avgTier:
                print(f"<tr><td colspan=\"2\" style=\"text-align: center\">&#128528; Average Tier {avgTier}</td></tr>")
                print(f"<tr><td colspan=\"2\" style=\"text-align: center\">&#128528; Opponent Average Tier {avgOppTier}</td></tr>")
            else:
                print(f"<tr><td colspan=\"2\" style=\"text-align: center\">Average Tier {avgTier}</td></tr>")
                print(f"<tr><td colspan=\"2\" style=\"text-align: center\">Opponent Average Tier {avgOppTier}</td></tr>")
        except IndexError:
            # issues with leagues without opponents
            print("")
    else:
        # basically if i cant get opponent, do plain old tier averages
        avgTier = round(tierSum / (len(starters[i])),2)
        print(f"<tr><td colspan=\"2\" style=\"text-align: center\">Average Tier {avgTier}</td></tr>")

    # bench
    print("<br>")
    print("<br>")
    
    tier = 0
    for b in bench:
        for key in collection.distinct(str(b)):
            fName = key['first_name']  
            lName = key['last_name']
            pos = key['position']
            fullName = fName + " " + lName
            
            if pos == "QB":
                qbBenchList, qbTierBenchList, tier = createTiers(tierListQB,fullName,qbBenchList,qbTierBenchList,tier)
                if createUnranked(tierListQB,fullName) != "ranked":
                    urBenchList.append(createUnranked(tierListQB,fullName))
    
            if pos == "RB":
                #rbStarterList, rbTierList, tier = createTiers(tierListRB,fullName,rbStarterList,rbTierList,tier)
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

    # Sometimes goes out of bounds and grabs "Craig Stevens"
    badList = ['Craig Stevens']
    urBenchList = [element for element in urBenchList if element not in badList]
    if (len(urBenchList) > 0):
        print("<tr>")
        print("<th colspan=\"2\" style=\"text-align: center\">Not Ranked</th>")
        print("</tr>")
        # urBenchList.remove("Craig Stevens")
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


