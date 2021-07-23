# sleeperPy

![license](https://img.shields.io/github/license/wbollock/sleeperPy) ![users](https://img.shields.io/badge/users-1000%2B-blue)



A web application to allow users to easily parse multiple [Sleeper](https://sleeper.app/) Fantasy Football leagues using [Boris Chen's](http://www.borischen.co/) tiers (accumulated from all FantasyPros.com experts).

This includes comparisons of the current weeks opponent's tiers, vaguely predicting whether you will win or lose.

If you use Sleeper for fantasy football, try it on my [personal website](https://wboll.dev/sleeperPy/). Only Sleeper Fantasy Football is supported. No ESPN or Yahoo for now.



![one team](img/web_view.png) 




## Features

* Utilizes the Sleeper API for multiple leagues. Simply enter your Sleeper username, and retrieve all player's associated Boris Chen tiers.
* Multiple scoring types accounted for, e.g standard, 0.5 PPR and PPR. 
* Shows the current tier of each player, dividing starters and bench.
* Represents each league in simple HTML tables.
* Shows the tiers for your current week's opponents, vaguely predicting whether you will win or lose.

## Usage


### Command Line

```
python3 sleeperPy.py <username>
```

Then find your tiers:

```
cat tiers/tiers_$username.html
```

### Web

Requirements:

 * php 7+
 * python3
 * pip3 - bs4, pymongo (TODO: requirements.txt for easy pip3 installs)
 * mongodb (default database is "sleeperPy", collection "players")
 * mongodb-tools (recommended)