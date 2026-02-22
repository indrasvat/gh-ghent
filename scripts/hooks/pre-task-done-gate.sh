#!/usr/bin/env bash
# scripts/hooks/pre-task-done-gate.sh — Claude Code PreToolUse hook for Edit|Write
#
# Intercepts Edit/Write operations on task files that change status to DONE,
# and verifies L4 visual test requirements are met before allowing the change.
#
# Claude Code hook protocol:
#   stdin:  JSON {"tool_name":"Edit","tool_input":{"file_path":"...","new_string":"..."}}
#           or   {"tool_name":"Write","tool_input":{"file_path":"...","content":"..."}}
#   stdout: JSON {"decision":"block","reason":"..."} to deny (exit 2)
#   exit 0: allow the tool use
#   exit 2: block the tool use (stdout reason shown to agent)
set -uo pipefail

# Read hook input from stdin
INPUT=$(cat)

# Extract tool name
TOOL_NAME=$(echo "$INPUT" | grep -o '"tool_name"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/"tool_name"[[:space:]]*:[[:space:]]*"//' | sed 's/"$//')

# Extract file path
FILE_PATH=$(echo "$INPUT" | grep -o '"file_path"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/"file_path"[[:space:]]*:[[:space:]]*"//' | sed 's/"$//')

# Only gate on task files
if ! echo "$FILE_PATH" | grep -qE 'docs/tasks/[0-9]+-.*\.md$'; then
    exit 0
fi

# Check if the edit/write is setting status to DONE
SETS_DONE=false
if [[ "$TOOL_NAME" == "Edit" ]]; then
    # For Edit: check new_string for "Status: DONE"
    NEW_STRING=$(echo "$INPUT" | grep -o '"new_string"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/"new_string"[[:space:]]*:[[:space:]]*"//' | sed 's/"$//')
    if echo "$NEW_STRING" | grep -qiE 'Status:.*DONE'; then
        SETS_DONE=true
    fi
elif [[ "$TOOL_NAME" == "Write" ]]; then
    # For Write: check content for "Status: DONE"
    # Content can be very large; check if it contains the status change
    if echo "$INPUT" | grep -qiE 'Status:.*DONE'; then
        SETS_DONE=true
    fi
fi

if ! $SETS_DONE; then
    exit 0  # Not setting status to DONE — allow
fi

# Resolve project root
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
VERIFY_SCRIPT="$PROJECT_ROOT/scripts/verify-visual-tests.sh"

if [[ ! -f "$VERIFY_SCRIPT" ]]; then
    echo '{"decision":"block","reason":"verify-visual-tests.sh not found — cannot verify L4 requirements before marking DONE"}' >&2
    exit 2
fi

TASK_NAME="$(basename "$FILE_PATH" .md)"

# Run verification
output=$(bash "$VERIFY_SCRIPT" "$FILE_PATH" 2>&1) || {
    # Extract specific errors
    errors=$(echo "$output" | grep '^\s*-' | tr '\n' ' ' | head -c 500)
    REASON="Cannot mark $TASK_NAME as DONE — L4 visual test requirements not met. $errors. Run the iterm2-driver test scripts, review screenshots, and add '## Visual Test Results' section first."
    echo "{\"decision\":\"block\",\"reason\":\"$REASON\"}"
    exit 2
}

# Verification passed — allow the edit
exit 0
