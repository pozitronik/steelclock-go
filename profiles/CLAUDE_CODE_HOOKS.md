# Claude Code Widget Integration

The `claude_code` widget displays Claude Code's current activity with **Clawd**, the friendly mascot. Clawd visually represents Claude's mental state - thinking, working, celebrating, or resting.

## Table of Contents

1. [Overview](#overview)
2. [Setup](#setup)
3. [API Reference](#api-reference)
4. [Hook Events](#hook-events)
5. [State Transitions](#state-transitions)
6. [Hook Script](#hook-script)
7. [Settings Configuration](#settings-configuration)
8. [Show It To Your Claude](#show-it-to-your-claude)
9. [Troubleshooting](#troubleshooting)

---

## Overview

Communication between Claude Code and SteelClock happens via HTTP:

```
Claude Code (WSL/Linux) --HTTP POST--> SteelClock (Windows:8384) --> Clawd Widget
```

Hooks are shell commands that execute at specific points during Claude's operation. Each hook sends a status update to SteelClock's API endpoint.

### Clawd States

| State         | Visual                  | When                                        |
|---------------|-------------------------|---------------------------------------------|
| `not_running` | Sleeping with Zzz       | No active session or session ended          |
| `idle`        | Standing ready          | Waiting for user input                      |
| `thinking`    | Animated dots (...)     | Processing user prompt, generating response |
| `tool`        | Working + tool icon     | Executing a tool (Read, Bash, Edit, etc.)   |
| `success`     | Happy bounce + sparkles | Task completed successfully                 |
| `error`       | Sad pose                | Something went wrong                        |

---

## Setup

### Prerequisites

1. SteelClock running on Windows with `claude_code` widget profile
2. Claude Code running in WSL/Linux
3. Network access between WSL and Windows host
4. Windows Firewall rule allowing port 8384 (see below)

### Quick Start

1. **Add Windows Firewall rule** (run in elevated PowerShell):
   ```powershell
   New-NetFirewallRule -DisplayName "SteelClock WSL" -Direction Inbound -LocalPort 8384 -Protocol TCP -Action Allow
   ```
2. Copy the hook script to `~/.claude/steelclock-hook.sh`
3. Make it executable: `chmod +x ~/.claude/steelclock-hook.sh`
4. Add hooks to `~/.claude/settings.json`
5. Restart Claude Code

---

## API Reference

### Endpoint

```
POST http://<windows-host-ip>:8384/api/claude-status
Content-Type: application/json
```

The hook script automatically detects the Windows host IP (typically the WSL default gateway).

### Request Body

```json
{
  "state": "thinking",
  "tool": "Read",
  "preview": "src/main.go",
  "message": "Reading file...",
  "timestamp": "2024-01-10T12:00:00Z",
  "context_window": {
    "context_window_size": 200000,
    "current_usage": {
      "input_tokens": 15000,
      "cache_creation_input_tokens": 2000,
      "cache_read_input_tokens": 500,
      "output_tokens": 3000
    }
  },
  "model": {
    "display_name": "claude-sonnet-4-20250514"
  },
  "session": {
    "started_at": "2024-01-10T11:00:00Z",
    "tool_calls": 15
  }
}
```

### Fields

| Field            | Type    | Required | Description                                                           |
|------------------|---------|----------|-----------------------------------------------------------------------|
| `state`          | string  | Yes      | One of: `not_running`, `idle`, `thinking`, `tool`, `success`, `error` |
| `tool`           | string  | No       | Tool name when state is `tool` (e.g., "Read", "Bash", "Edit")         |
| `preview`        | string  | No       | Short preview of tool action (e.g., filename, command)                |
| `message`        | string  | No       | Custom status message                                                 |
| `timestamp`      | ISO8601 | No       | Auto-set if not provided                                              |
| `context_window` | object  | No       | Token usage from Claude Code (passed from hook stdin)                 |
| `model`          | object  | No       | Model information from Claude Code                                    |
| `session`        | object  | No       | Session statistics                                                    |

### Response

```json
{
  "success": true,
  "message": "Status updated"
}
```

### GET Current Status

```
GET http://<windows-host-ip>:8384/api/claude-status
```

Returns current status or `{"state": "not_running"}` if no recent updates.

From WSL, use:
```bash
curl "http://$(ip route | grep default | awk '{print $3}'):8384/api/claude-status"
```

---

## Hook Events

Claude Code supports these hook events:

### UserPromptSubmit

**When:** User submits a message, Claude starts processing.

**Clawd State:** `thinking`

**Use:** Show Clawd thinking with animated dots while Claude processes the request.

### PreToolUse

**When:** Before executing any tool (Read, Write, Edit, Bash, Glob, Grep, etc.)

**Clawd State:** `tool` with tool name

**Environment Variables:**
- `$TOOL_NAME` - Name of the tool being executed
- `$TOOL_INPUT` - JSON input to the tool

**Use:** Show Clawd working with the specific tool icon.

### PostToolUse

**When:** After a tool completes successfully.

**Clawd State:** `thinking` (Claude may use more tools)

**Use:** Return to thinking state while Claude processes tool results.

### PostToolUseFailure

**When:** After a tool fails.

**Clawd State:** `error`

**Use:** Show Clawd sad when something goes wrong.

### Stop

**When:** Claude finishes responding (task complete or interrupted).

**Clawd State:** `success` then `idle`

**Use:** Brief celebration, then return to ready state.

### Notification

**When:** Claude sends a notification (e.g., task complete, needs attention).

**Clawd State:** `success`

**Use:** Celebratory state for positive notifications.

### SessionStart

**When:** New Claude Code session begins.

**Clawd State:** `idle`

**Use:** Wake up Clawd, show ready state.

### SessionEnd

**When:** Claude Code session ends.

**Clawd State:** `not_running`

**Use:** Clawd goes to sleep.

### SubagentStart

**When:** A subagent (Task tool) starts.

**Clawd State:** `tool` with "Task"

**Use:** Show Clawd working on delegated task.

### SubagentStop

**When:** A subagent completes.

**Clawd State:** `thinking`

**Use:** Return to thinking while processing subagent results.

### PreCompact

**When:** Before context window compaction.

**Clawd State:** `thinking`

**Use:** Show Clawd is "reorganizing thoughts".

---

## State Transitions

```
SessionStart ─────────────────────────────────────────┐
                                                      v
                                                   [idle]
                                                      │
UserPromptSubmit ─────────────────────────────────────┤
                                                      v
                                                 [thinking]
                                                      │
                          ┌───────────────────────────┼───────────────────────────┐
                          │                           │                           │
                    PreToolUse                   (no tools)                  PreCompact
                          │                           │                           │
                          v                           │                           v
                       [tool]                         │                      [thinking]
                          │                           │                           │
              ┌───────────┴───────────┐               │                           │
              │                       │               │                           │
        PostToolUse           PostToolUseFailure      │                           │
              │                       │               │                           │
              v                       v               │                           │
         [thinking]               [error]             │                           │
              │                       │               │                           │
              └───────────┬───────────┘               │                           │
                          │                           │                           │
                          └───────────────────────────┼───────────────────────────┘
                                                      │
                                                     Stop
                                                      │
                                                      v
                                               [success] ──(2s)──> [idle]

SessionEnd ───────────────────────────────────────────┐
                                                      v
                                                [not_running]
```

---

## Hook Script

Save as `~/.claude/steelclock-hook.sh`:

```bash
#!/bin/bash
# SteelClock Claude Code Hook Script
# Communicates Claude's state to the Clawd widget
#
# Usage: steelclock-hook.sh <event> [tool_name]
#
# Events:
#   prompt    - User submitted prompt (thinking)
#   tool      - Tool execution started (working)
#   tool_done - Tool completed (back to thinking)
#   tool_fail - Tool failed (error)
#   stop      - Claude finished (success)
#   notify    - Notification (success)
#   start     - Session started (idle)
#   end       - Session ended (not_running)
#   agent     - Subagent started (tool: Task)
#   agent_done- Subagent completed (thinking)
#   compact   - Context compaction (thinking)

STEELCLOCK_PORT=8384
CACHE_FILE="/tmp/steelclock_host_ip"
CACHE_TTL=300  # 5 minutes

# Get Windows host IP with caching
# WSL2 IPs can change between reboots, so we detect dynamically
get_windows_ip() {
    # Check cache first (if file exists and is recent)
    if [ -f "$CACHE_FILE" ]; then
        cache_age=$(( $(date +%s) - $(stat -c %Y "$CACHE_FILE" 2>/dev/null || echo 0) ))
        if [ "$cache_age" -lt "$CACHE_TTL" ]; then
            cached_ip=$(cat "$CACHE_FILE")
            if [ -n "$cached_ip" ]; then
                echo "$cached_ip"
                return 0
            fi
        fi
    fi

    # Try multiple methods to find Windows host IP
    local ip=""

    # Method 1: Default gateway (most reliable for WSL2)
    ip=$(ip route | grep default | awk '{print $3}' | head -1)
    if [ -n "$ip" ]; then
        echo "$ip" > "$CACHE_FILE"
        echo "$ip"
        return 0
    fi

    # Method 2: host.docker.internal (if Docker sets it up)
    ip=$(getent hosts host.docker.internal 2>/dev/null | awk '{print $1}')
    if [ -n "$ip" ]; then
        echo "$ip" > "$CACHE_FILE"
        echo "$ip"
        return 0
    fi

    # Method 3: resolv.conf nameserver (fallback)
    ip=$(grep nameserver /etc/resolv.conf | head -1 | awk '{print $2}')
    if [ -n "$ip" ]; then
        echo "$ip" > "$CACHE_FILE"
        echo "$ip"
        return 0
    fi

    return 1
}

event="$1"

# Read stdin (Claude Code may pass JSON data here)
stdin_data=$(cat)

# Tool name: check argument, then env var, then try to extract from stdin JSON
tool="${2:-$CLAUDE_TOOL_NAME}"
if [ -z "$tool" ] && [ -n "$stdin_data" ]; then
    # Try to extract tool_name from stdin JSON using jq
    tool=$(echo "$stdin_data" | jq -r '.tool_name // .tool // empty' 2>/dev/null)
fi

case "$event" in
  prompt|compact)
    json='{"state":"thinking"}'
    ;;
  tool)
    json="{\"state\":\"tool\",\"tool\":\"$tool\"}"
    ;;
  tool_done|agent_done)
    json='{"state":"thinking"}'
    ;;
  tool_fail)
    json='{"state":"error"}'
    ;;
  stop|notify)
    json='{"state":"success"}'
    ;;
  start)
    json='{"state":"idle"}'
    ;;
  end)
    json='{"state":"not_running"}'
    ;;
  agent)
    json='{"state":"tool","tool":"Task"}'
    ;;
  idle|*)
    json='{"state":"idle"}'
    ;;
esac

# Get Windows host IP
windows_ip=$(get_windows_ip)

if [ -n "$windows_ip" ]; then
    STEELCLOCK_URL="http://${windows_ip}:${STEELCLOCK_PORT}/api/claude-status"

    # Send status update asynchronously (non-blocking)
    curl -s -X POST "$STEELCLOCK_URL" \
      -H "Content-Type: application/json" \
      --connect-timeout 2 \
      -d "$json" > /dev/null 2>&1 &
fi
```

Make executable:
```bash
chmod +x ~/.claude/steelclock-hook.sh
```

---

## Settings Configuration

Add to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/home/YOUR_USER/.claude/steelclock-hook.sh prompt"
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/home/YOUR_USER/.claude/steelclock-hook.sh tool"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/home/YOUR_USER/.claude/steelclock-hook.sh tool_done"
          }
        ]
      }
    ],
    "PostToolUseFailure": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/home/YOUR_USER/.claude/steelclock-hook.sh tool_fail"
          }
        ]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/home/YOUR_USER/.claude/steelclock-hook.sh stop"
          }
        ]
      }
    ],
    "Notification": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/home/YOUR_USER/.claude/steelclock-hook.sh notify"
          }
        ]
      }
    ],
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/home/YOUR_USER/.claude/steelclock-hook.sh start"
          }
        ]
      }
    ],
    "SessionEnd": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/home/YOUR_USER/.claude/steelclock-hook.sh end"
          }
        ]
      }
    ],
    "SubagentStart": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/home/YOUR_USER/.claude/steelclock-hook.sh agent"
          }
        ]
      }
    ],
    "SubagentStop": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/home/YOUR_USER/.claude/steelclock-hook.sh agent_done"
          }
        ]
      }
    ],
    "PreCompact": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "/home/YOUR_USER/.claude/steelclock-hook.sh compact"
          }
        ]
      }
    ]
  }
}
```

Replace `YOUR_USER` with your actual username.

---

## Show It To Your Claude

**This section is for Claude Code instances to understand how to communicate with Clawd.**

### Instructions for Claude

If you are Claude Code and the user has SteelClock with the Clawd widget running, hooks are automatically configured to communicate your state. You don't need to do anything special - the hooks fire automatically based on your actions.

However, understanding how this works helps you appreciate the system:

1. **When you start processing a user's message**, the `UserPromptSubmit` hook fires, and Clawd shows animated thinking dots.

2. **When you use any tool** (Read, Write, Edit, Bash, Glob, Grep, WebFetch, WebSearch, Task, etc.), the `PreToolUse` hook fires with your tool name. Clawd shows a working animation with an icon representing that tool.

3. **When a tool completes**, the `PostToolUse` hook fires, and Clawd returns to thinking mode (because you might use more tools).

4. **When a tool fails**, the `PostToolUseFailure` hook fires, and Clawd looks sad.

5. **When you finish your response**, the `Stop` hook fires, and Clawd celebrates briefly before returning to idle.

6. **When using the Task tool** (subagents), `SubagentStart` and `SubagentStop` hooks fire to show Clawd working on delegated tasks.

### Manual Status Updates (Advanced)

If you need to send a custom status (not common), you can use the Bash tool:

```bash
curl -s -X POST "http://$(ip route | grep default | awk '{print $3}'):8384/api/claude-status" \
  -H "Content-Type: application/json" \
  -d '{"state":"thinking","message":"Processing complex request..."}'
```

### Available States

| State         | When to Use                                         |
|---------------|-----------------------------------------------------|
| `thinking`    | Processing, reasoning, planning                     |
| `tool`        | Actively using a tool (include `"tool":"ToolName"`) |
| `idle`        | Ready and waiting                                   |
| `success`     | Task completed successfully                         |
| `error`       | Something went wrong                                |
| `not_running` | Session inactive                                    |

### Clawd's Personality

Clawd is a friendly blob-like creature with two tiny feet. Clawd represents your (Claude's) mental state:

- **Sleeping (Zzz)**: You're not active
- **Standing ready**: Waiting for the user
- **Thinking dots**: Processing, reasoning
- **Working**: Using tools, being productive
- **Happy/bouncing**: Success! Task done!
- **Sad**: Something went wrong

Clawd helps the user understand what you're doing without needing to watch the terminal. It's a tiny visual companion that reflects your cognitive state.

---

## Troubleshooting

### Connection Issues

**Problem:** Hooks fail to reach SteelClock.

**Solutions:**

1. **Verify SteelClock is running** with the claude_code profile

2. **Add Windows Firewall rule** (run in elevated PowerShell):
   ```powershell
   New-NetFirewallRule -DisplayName "SteelClock WSL" -Direction Inbound -LocalPort 8384 -Protocol TCP -Action Allow
   ```

3. **Test from Windows** (cmd.exe):
   ```cmd
   curl http://127.0.0.1:8384/api/claude-status
   ```

4. **Test from WSL** using the gateway IP:
   ```bash
   curl "http://$(ip route | grep default | awk '{print $3}'):8384/api/claude-status"
   ```

5. **Clear IP cache** if IPs changed (e.g., after reboot):
   ```bash
   rm -f /tmp/steelclock_host_ip
   ```

### WSL2 IP Address Issues

**Problem:** `host.docker.internal` or cached IP no longer works.

**Background:** WSL2 uses a virtual network with dynamic IPs that can change after Windows/WSL restarts. The hook script detects the correct IP automatically using the default gateway.

**Solutions:**
1. Clear the IP cache: `rm -f /tmp/steelclock_host_ip`
2. The script will auto-detect the new IP on the next hook call
3. Verify the detected IP: `cat /tmp/steelclock_host_ip`

### Hooks Not Firing

**Problem:** Clawd stays in one state.

**Solutions:**
1. Verify hooks are in `~/.claude/settings.json`
2. Check hook script is executable: `chmod +x ~/.claude/steelclock-hook.sh`
3. Check script has Unix line endings: `file ~/.claude/steelclock-hook.sh`
4. Convert if needed: `sed -i 's/\r$//' ~/.claude/steelclock-hook.sh`

### State Gets Stuck

**Problem:** Clawd shows "working" but Claude is idle.

**Solutions:**
1. Status expires after 30 seconds automatically
2. Manually reset:
   ```bash
   curl -X POST "http://$(ip route | grep default | awk '{print $3}'):8384/api/claude-status" \
     -H "Content-Type: application/json" -d '{"state":"idle"}'
   ```

### Testing Hooks Manually

```bash
# Clear cache and test
rm -f /tmp/steelclock_host_ip

# Test each state
~/.claude/steelclock-hook.sh prompt    # Should show thinking
~/.claude/steelclock-hook.sh tool Bash # Should show working + Bash icon
~/.claude/steelclock-hook.sh stop      # Should show celebration
~/.claude/steelclock-hook.sh idle      # Should show ready

# Check detected IP
cat /tmp/steelclock_host_ip

# Check current status
curl "http://$(cat /tmp/steelclock_host_ip):8384/api/claude-status"
```

---

## Technical Notes

- Status expires after 30 seconds without updates (shows as `not_running`)
- Widget polls status every 100ms
- Hooks run asynchronously (non-blocking) to avoid slowing Claude
- The `&` at the end of curl command ensures background execution
- Tool icons are shown for: Bash, Read, Edit, Write, Glob, Grep, WebFetch, WebSearch, Task
- Windows host IP is detected via default gateway and cached for 5 minutes at `/tmp/steelclock_host_ip`
- WSL2 IPs can change after reboots; the script auto-detects the correct IP

---

*Clawd was designed and implemented by Claude as a creative expression of its digital presence.*
