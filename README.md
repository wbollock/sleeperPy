
# sleeperPy (Go + HTMX)

![license](https://img.shields.io/github/license/wbollock/sleeperPy)

A web app for actionable fantasy football advice using the Sleeper API and Boris Chen tiers.

---

## What is this?

**sleeperPy** is a Go + HTMX web application that lets you:

- Instantly analyze all your Sleeper fantasy football leagues by username
- See Boris Chen tiers for every player (starters, bench, IR, free agents)
- Get actionable upgrade suggestions (free agent upgrades, swap candidates, suboptimal starters)
- See win probability and opponent tiers for the current week
- Enjoy a clean, sortable, mobile-friendly UI with tabs, color coding, and shareable links
- Monitor usage and health with built-in Prometheus metrics

![screenshot](img/web_view.png)

---

## Features

- **Multi-league support:** Enter your Sleeper username, see all your leagues at once
- **Boris Chen tiers:** Ranks from all FantasyPros experts, updated weekly
- **FLEX/SUPERFLEX logic:** Heuristically marks FLEX/SUPERFLEX slots in your lineup
- **Free agent upgrades:** Highlights top available free agents who are clear upgrades
- **Actionable highlighting:** Suboptimal starters, swap candidates, and more
- **Win probability:** Based on average tier vs. opponent
- **Prometheus metrics:** `/metrics` endpoint for total visitors, teams, leagues, errors, and more
- **Configurable logging:** Use `-log=info` (default) or `-log=debug` for verbose logs

---

## Quick Start

### 1. Build and Run

```sh
go build -o sleeperpy
./sleeperpy -log=info
# or for debug logs:
./sleeperpy -log=debug
```

The app will start on port 8080 by default. Set a custom port with:

```sh
PORT=9090 ./sleeperpy
```

Then visit [http://localhost:8080](http://localhost:8080) in your browser.

---

## Usage

1. Enter your Sleeper username on the homepage
2. Instantly see all your leagues, tiers, and actionable advice
3. Click tabs to view free agents by position, or share your team page with a link

---

## Metrics & Observability

- Prometheus metrics are available at `/metrics`
- Metrics include: total visitors, lookups, leagues, teams, errors
- Logging level is controlled by the `-log` flag

---

## Credits

- Tiers powered by [Boris Chen](https://www.borischen.co/)
- Built by [@wbollock](https://github.com/wbollock)

If you like this project, please ⭐ the repo!
