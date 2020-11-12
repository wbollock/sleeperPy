# sleeperPy

A program to allow users to easily parse their [Sleeper](https://sleeper.app/) Fantasy Football team using [Boris Chen's](http://www.borischen.co/) tiers (accumulated from all FantasyPros.com experts).

Try it on my [personal website](https://wboll.dev/sleeperPy/). Try the username "puffplants" if you don't have a Sleeper FF team.

![one team](img/web_view.png)

Only Sleeper Fantasy Football is supported.

## Features

* Utilizes the Sleeper API for multiple leagues. Simply enter your Sleeper username, and get player's associated Boris Chen tiers.
* Multiple scoring types accounted for, standard, 0.5 PPR and PPR. 
* Shows the current tier of each player, dividing starters and bench.
* Represents each league in simple HTML tables.

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
 * pip3 install bs4

```
git clone https://github.com/wbollock/sleeperPy.git
sudo chmod tiers 774
```

Make sure your web server has permissions for the `tiers` folder, `sleeperPy.py`, and `index.php`.
