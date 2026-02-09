# Auto-Resume Script Fixes

## Problems Found

1. **Processes stuck for 40+ hours** - Claude processes had no output for 140,000+ seconds
2. **Timeout not enforced** - `SUBPROCESS_TIMEOUT` constant was defined but never used
3. **No auto-resume on stuck processes** - Script didn't detect and restart stuck sessions
4. **No context compaction handling** - Didn't detect when Claude hit session limits
5. **No auto-continue mode** - Never switched to `-c` flag to resume sessions

## Fixes Applied

### 1. Actual Timeout Enforcement
```python
def heartbeat_logger():
    # Check for timeout - kill if stuck too long
    if elapsed > SUBPROCESS_TIMEOUT:
        log(f"[TIMEOUT] No output for {int(elapsed)}s (timeout: {SUBPROCESS_TIMEOUT}s)")
        log(f"[TIMEOUT] Killing stuck subprocess PID {proc[0].pid}...")
        proc[0].kill()
```

Now the heartbeat thread **actually kills** stuck processes after 1 hour of no output.

### 2. Context Compaction Detection
```python
def check_context_compaction(output):
    """Check if output indicates context window compaction or session limit."""
    patterns = [
        r"context.*window.*full",
        r"compacting.*context",
        r"session.*limit.*reached",
        r"message.*history.*truncated",
        r"conversation.*too.*long",
    ]
```

Detects when Claude hits session limits and automatically switches to continue mode.

### 3. Stuck Process Detection
```python
def check_stuck_or_timeout(ret_code, output):
    """Check if Claude got stuck or timed out."""
    # Exit code -9 means killed by SIGKILL (our timeout killer)
    if ret_code == -9 or ret_code == 137:
        return True
```

Detects when process was killed by timeout and automatically restarts with `-c`.

### 4. Auto-Continue Mode
```python
# After first successful run, always use continue flag
if not current_continue:
    log("[SESSION] Switching to continue mode (-c) for subsequent runs...")
    current_continue = True
```

After the first run completes, **all subsequent runs use `-c`** to continue the session.

### 5. Error Recovery
```python
if ret_code != 0:
    log(f"[ERROR] Will retry with continue flag (-c) in case session needs resuming...")
    current_continue = True
    run_number -= 1  # Retry with continue flag
    time.sleep(10)
    continue
```

Any error now triggers a retry with continue mode after 10 seconds.

## Why --fg Mode?

The script runs in **foreground mode by default when called with --fg**, but when you run it WITHOUT --fg, it:

1. **Spawns a background process** with `--fg`
2. **Detaches** it from your terminal
3. **Redirects** stdout/stderr to the log file
4. **Returns immediately** so you can use your terminal

The `--fg` in your `ps` output is **normal and expected** - it means the background wrapper is working correctly.

## How to Use (Correct Way)

### Background Mode (Recommended)
```bash
./claude-auto-resume "Implement all features in plan/ directory"
```

This will:
- Start in background
- Return immediately
- Keep running forever
- Log to `.claude-auto-resume/session.log`

### Check Status
```bash
./claude-auto-resume --status
```

### Follow Logs
```bash
tail -f .claude-auto-resume/session.log
```

### Stop Gracefully
```bash
./claude-auto-resume --stop
```

### Force Stop (If Stuck)
```bash
./claude-auto-resume --force-stop
```

## Testing the Fixes

Run this to test:

```bash
# Clean up old sessions
./claude-auto-resume --force-stop

# Start fresh session
./claude-auto-resume "Implement all features in plan/ directory"

# Follow logs in another terminal
tail -f .claude-auto-resume/session.log
```

## What Changed

1. **Timeout enforcement**: Kills stuck processes after 1 hour
2. **Auto-continue**: Switches to `-c` flag after first run
3. **Context detection**: Detects session limits and resumes
4. **Error recovery**: Retries with continue flag on any error
5. **Stuck detection**: Detects killed processes and restarts

## Expected Behavior

- First run: Uses original prompt
- Subsequent runs: Uses `-c` to continue
- On timeout: Kills process, restarts with `-c`
- On error: Waits 10s, restarts with `-c`
- On context limit: Immediately restarts with `-c`
- On rate limit: Waits until reset, then continues

The script will **never get stuck for 2 days again**!
