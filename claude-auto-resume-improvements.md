# Claude Auto-Resume: Future Improvements

## Current Status (v2.8.0)

**Working:**
- ✅ Auto-resumes on rate limits, timeouts, errors
- ✅ Tmux integration for watchable sessions
- ✅ Auto-accepts bypass permissions prompt
- ✅ PTY for real-time output capture
- ✅ Summary log file (high-level actions only)
- ✅ Heartbeat monitoring (kills stuck processes)
- ✅ Multiple session tracking
- ✅ Graceful stop with --stop
- ✅ Force kill with --force-stop

**Issues:**
- ⚠️ Tmux output can be messy (ANSI codes, cursor positioning from Claude TUI)
- ⚠️ Sometimes shows 1-2 chars per line (cursor positioning artifacts)
- ⚠️ Log file summary patterns could be more comprehensive

## Future Improvements

### 1. Better Output Rendering (High Priority)

**Problem:** Claude's TUI uses heavy cursor positioning, making raw PTY capture messy.

**Solutions to Try:**
- Use `script` command instead of PTY (better terminal emulation)
- Use `tmux` pipes instead of PTY (tmux handles rendering)
- Parse Claude's structured output format if available
- Use `unbuffer` from expect package
- Capture to file and `tail -f` instead of live streaming

**Example with script:**
```python
cmd = ["script", "-qec", f"claude {args}", "/dev/null"]
# script handles TTY better than raw PTY
```

### 2. Structured Log Output (Medium Priority)

**Current:** Regex pattern matching for summary lines
**Better:** Parse Claude's actual tool calls and responses

**Approach:**
- Check if Claude has a `--output-format json` or similar
- Parse tool call blocks explicitly
- Extract: tool name, file paths, commands, results
- Format as clean summary: `[READ] file.py`, `[EDIT] file.py:123`, `[BASH] git commit`

### 3. Better Bypass Prompt Handling (Low Priority)

**Current:** Auto-sends Down Arrow + Enter after 3s delay
**Better:** Check Claude CLI for environment variable or config to skip prompt entirely

**Try:**
- `CLAUDE_BYPASS_PERMISSIONS_PROMPT=1` env var (if it exists)
- `~/.claude/settings.json` additional settings
- Contact Anthropic about headless mode

### 4. Session Resume on Script Restart (Medium Priority)

**Problem:** If the wrapper script crashes, Claude session is lost
**Solution:** Save session ID to file, allow resuming with `claude -c`

**Implementation:**
```python
# Save session ID
STATUS_FILE["claude_session_id"] = session_id

# On restart, check if session exists
if old_session_id:
    cmd.append("-c")  # Continue flag
```

### 5. Multiple Concurrent Sessions (Low Priority)

**Current:** One session at a time
**Better:** Support multiple named sessions

**Usage:**
```bash
./claude-auto-resume --name feature1 "implement feature 1"
./claude-auto-resume --name feature2 "implement feature 2"
./claude-auto-resume --list  # Show all sessions
./claude-auto-resume --attach feature1  # Attach to specific session
```

### 6. Better Progress Tracking (Medium Priority)

**Current:** Line counts, heartbeats
**Better:** Parse actual progress from Claude

**Track:**
- Current file being worked on
- Current task/phase
- Completion percentage (if Claude reports it)
- Estimated time remaining
- Files modified count
- Commands run count

### 7. Notification System (Low Priority)

**When to Notify:**
- Rate limit hit (long wait ahead)
- Error encountered (may need manual intervention)
- Session completed (all done!)
- Timeout/stuck detected

**Methods:**
- Desktop notification (`notify-send` on Linux)
- Email (configurable SMTP)
- Slack webhook
- Discord webhook
- SMS (Twilio)

### 8. Web Dashboard (Low Priority)

**Instead of tmux/tail, provide web UI:**
- Real-time status view
- Live log streaming
- Session control (stop/restart)
- Historical runs
- Success rate statistics

**Tech Stack:**
- Python Flask/FastAPI
- WebSocket for real-time updates
- Simple HTML/CSS/JS frontend
- Run on localhost:8080

### 9. Better Error Recovery (Medium Priority)

**Current:** Retries with -c flag on any error
**Better:** Smart error detection and recovery

**Error Types:**
- Network errors → wait and retry
- Permission errors → warn user
- File not found → create directory structure
- Syntax errors → log and continue (don't retry same thing)

### 10. Configuration File (Low Priority)

**Current:** Hardcoded constants in script
**Better:** `~/.claude-auto-resume.conf`

**Configurable:**
```ini
[general]
max_runs = 100
heartbeat_interval = 300
subprocess_timeout = 3600

[notifications]
enabled = true
methods = desktop,email
email_to = user@example.com

[logging]
summary_only = true
keep_last_n_logs = 10

[model]
default = claude-sonnet-4-5-20250929
fallback = claude-haiku-4-5
```

## Quick Wins (Do These First)

1. **Try `script` command** instead of PTY for better output (30 min)
2. **Add --quiet flag** to reduce wrapper noise in tmux (10 min)
3. **Better summary patterns** in log file (30 min)
4. **Save session ID** for crash recovery (20 min)
5. **Desktop notifications** for rate limits (15 min)

## Known Limitations

1. **Claude TUI is interactive** - designed for human viewing, not programmatic capture
2. **No official headless mode** - Claude CLI expects a terminal
3. **ANSI rendering is complex** - full emulation would require terminal emulator library
4. **Rate limits are per-account** - can't work around them, only wait

## Alternative Approaches

### Approach A: Use Claude API Instead of CLI
- More control, structured responses
- No TUI rendering issues
- Requires API key and separate implementation
- Loses CLI features (skills, hooks, etc.)

### Approach B: Screen Recording
- Use `asciinema` or similar to record session
- Playback later for review
- Still doesn't solve real-time viewing issue
- But provides perfect historical record

### Approach C: Dual Mode
- Foreground: Full Claude TUI for interactive work
- Background: API mode for autonomous tasks
- Best of both worlds
- More complex implementation

## References

- Claude CLI docs: https://code.claude.com/docs
- PTY programming: `man pty`, `man script`
- tmux scripting: `man tmux`, tmux pipe-pane
- Expect/unbuffer: `man unbuffer`
- Terminal emulation: xterm.js, blessed-contrib

## Notes

- Current version (v2.8.0) is "good enough" for autonomous long-running tasks
- Tmux output may be messy but log file provides clean summary
- Focus should be on making log file excellent rather than perfect tmux output
- Claude CLI is designed for humans, not automation - some messiness is expected
