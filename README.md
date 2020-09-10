# sleeperPy

A program to allow users to easily parse their [Sleeper](https://sleeper.app/) Fantasy Football team using [Boris Chen's](http://www.borischen.co/) tiers (accumulated from all FantasyPros.com experts).

![one team](img/one_team.png)

## Features

* Utilizies the Sleeper APISupport for multiple leagues. Simply enter your Sleeper username.
* Multiple scoring types accounted for, standard, 0.5 PPR and PPR. 
* Shows the current tier of each player, dividing starters and bench.

## Usage

Edit your username into the `username` variable, e.g:

```
username = "FooBarSleeper"
```

Then run the program:

```
python3 sleeperPy.py
```

### Multiple League Support

![two teams](img/two_teams.png)

### Considerations

Team formatting is not great right now. Code is pretty horrible too, lots of copying and pasting. Definitely not clean code. But it works.