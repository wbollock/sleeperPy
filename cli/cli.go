package cli

import (
	"flag"
	"fmt"
	"os"
)

// Context holds CLI execution context
type Context struct {
	JSON    bool
	Debug   bool
	NoCache bool
	Args    []string
}

// Run executes CLI commands
func Run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}

	// Parse CLI flags
	fs := flag.NewFlagSet("cli", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "Output JSON")
	debug := fs.Bool("debug", false, "Enable debug logging")
	noCache := fs.Bool("cache-off", false, "Disable caching")
	fs.Parse(args[1:])

	// Create CLI context
	ctx := &Context{
		JSON:    *jsonOutput,
		Debug:   *debug,
		NoCache: *noCache,
		Args:    fs.Args(),
	}

	// Route to command
	command := args[0]
	switch command {
	case "user":
		return cmdUser(ctx)
	case "league":
		return cmdLeague(ctx)
	case "tiers":
		return cmdTiers(ctx)
	case "dynasty-values":
		return cmdDynastyValues(ctx)
	case "player":
		return cmdPlayer(ctx)
	case "test":
		return cmdTest(ctx)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		return 1
	}
}

func printUsage() {
	fmt.Println(`SleeperPy CLI

Usage:
  sleeperPy cli <command> [args] [flags]

Commands:
  user <username>              Fetch user's leagues
  league <league_id> <user>    Analyze specific league
  tiers <format>               Fetch Boris Chen tiers (ppr, half-ppr, standard, superflex)
  dynasty-values               Fetch KTC dynasty values
  player <name>                Look up player
  test                         Run integration tests

Flags:
  --json        Output JSON (machine readable)
  --debug       Enable debug logging
  --cache-off   Disable caching for testing

Examples:
  sleeperPy cli user wbollock
  sleeperPy cli league 123456789 wbollock --json
  sleeperPy cli tiers ppr
  sleeperPy cli test --debug`)
}
