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

| State | Visual | When |
|-------|--------|------|
| `not_running` | Sleeping with Zzz | No active session or session ended |
| `idle` | Standing ready | Waiting for user input |
| `thinking` | Animated dots (...) | Processing user prompt, generating response |
| `tool` | Working + tool icon | Executing a tool (Read, Bash, Edit, etc.) |
| `success` | Happy bounce + sparkles | Task completed successfully |
| `error` | Sad pose | Something went wrong |

---

## Setup

### Prerequisites

1. SteelClock running on Windows with `claude_code` widget profile
2. Claude Code running in WSL/Linux
3. Network access between WSL and Windows host

### Quick Start

1. Copy the hook script to `~/.claude/steelclock-hook.sh`
2. Make it executable: `chmod +x ~/.claude/steelclock-hook.sh`
3. Add hooks to `~/.claude/settings.json`
4. Restart Claude Code

---

## API Reference

### Endpoint

```
POST http://host.docker.internal:8384/api/claude-status
Content-Type: application/json
```

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

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `state` | string | Yes | One of: `not_running`, `idle`, `thinking`, `tool`, `success`, `error` |
| `tool` | string | No | Tool name when state is `tool` (e.g., "Read", "Bash", "Edit") |
| `preview` | string | No | Short preview of tool action (e.g., filename, command) |
| `message` | string | No | Custom status message |
| `timestamp` | ISO8601 | No | Auto-set if not provided |
| `context_window` | object | No | Token usage from Claude Code (passed from hook stdin) |
| `model` | object | No | Model information from Claude Code |
| `session` | object | No | Session statistics |

### Response

```json
{
  "success": true,
  "message": "Status updated"
}
```

### GET Current Status

```
GET http://host.docker.internal:8384/api/claude-status
```

Returns current status or `{"state": "not_running"}` if no recent updates.

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
# Communicates Claude's state and token usage to the Clawd widget
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

STEELCLOCK_URL="http://host.docker.internal:8384/api/claude-status"

event="$1"
tool="${2:-}"

# Read hook input from stdin (contains context_window data)
input=$(cat)

# Extract context_window and model from hook input using jq
context_window=$(echo "$input" | jq -c '.context_window // empty' 2>/dev/null)
model=$(echo "$input" | jq -c '.model // empty' 2>/dev/null)

# Build base state JSON
case "$event" in
  prompt|compact)
    state="thinking"
    ;;
  tool)
    state="tool"
    ;;
  tool_done|agent_done)
    state="thinking"
    ;;
  tool_fail)
    state="error"
    ;;
  stop|notify)
    state="success"
    ;;
  start)
    state="idle"
    ;;
  end)
    state="not_running"
    ;;
  agent)
    state="tool"
    tool="Task"
    ;;
  idle|*)
    state="idle"
    ;;
esac

# Build JSON with optional context_window and model
json="{\"state\":\"$state\""
[ -n "$tool" ] && json="$json,\"tool\":\"$tool\""
[ -n "$context_window" ] && json="$json,\"context_window\":$context_window"
[ -n "$model" ] && json="$json,\"model\":$model"
json="$json}"

# Send status update asynchronously (non-blocking)
curl -s -X POST "$STEELCLOCK_URL" \
  -H "Content-Type: application/json" \
  -d "$json" > /dev/null 2>&1 &
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
            "command": "/home/YOUR_USER/.claude/steelclock-hook.sh tool $TOOL_NAME"
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
curl -s -X POST "http://host.docker.internal:8384/api/claude-status" \
  -H "Content-Type: application/json" \
  -d '{"state":"thinking","message":"Processing complex request..."}'
```

### Available States

| State | When to Use |
|-------|-------------|
| `thinking` | Processing, reasoning, planning |
| `tool` | Actively using a tool (include `"tool":"ToolName"`) |
| `idle` | Ready and waiting |
| `success` | Task completed successfully |
| `error` | Something went wrong |
| `not_running` | Session inactive |

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
1. Verify SteelClock is running with the claude_code profile
2. Check Windows Firewall allows connections on port 8384
3. Test connection: `curl http://host.docker.internal:8384/api/claude-status`
4. If `host.docker.internal` doesn't work, try:
   - `$(wsl.exe hostname -I | tr -d '\r' | awk '{print $1}')`
   - Check `/etc/resolv.conf` nameserver

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
2. Manually reset: `curl -X POST http://host.docker.internal:8384/api/claude-status -d '{"state":"idle"}'`

### Testing Hooks Manually

```bash
# Test each state
~/.claude/steelclock-hook.sh prompt    # Should show thinking
~/.claude/steelclock-hook.sh tool Bash # Should show working + Bash icon
~/.claude/steelclock-hook.sh stop      # Should show celebration
~/.claude/steelclock-hook.sh idle      # Should show ready

# Check current status
curl http://host.docker.internal:8384/api/claude-status
```

---

## Technical Notes

- Status expires after 30 seconds without updates (shows as `not_running`)
- Widget polls status every 100ms
- Hooks run asynchronously (non-blocking) to avoid slowing Claude
- The `&` at the end of curl command ensures background execution
- Tool icons are shown for: Bash, Read, Edit, Write, Glob, Grep, WebFetch, WebSearch, Task

---

*Clawd was designed and implemented by Claude as a creative expression of its digital presence.*
