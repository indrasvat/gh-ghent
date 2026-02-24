#!/usr/bin/env bash
# scripts/hooks/pre-push-visual-gate.sh — Claude Code PreToolUse hook for Bash
#
# Intercepts `git push` commands and verifies that all IN PROGRESS tasks
# have completed their L4 visual test requirements before allowing the push.
#
# Claude Code hook protocol:
#   stdin:  JSON {"tool_name":"Bash","tool_input":{"command":"git push ..."}}
#   stdout: JSON {"decision":"block","reason":"..."} to deny (exit 2)
#   exit 0: allow the tool use
#   exit 2: block the tool use (stdout reason shown to agent)
set -uo pipefail

# Read hook input from stdin
INPUT=$(cat)

# Extract the command being run
COMMAND=$(echo "$INPUT" | grep -o '"command"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/"command"[[:space:]]*:[[:space:]]*"//' | sed 's/"$//')

# Only gate on git push commands
if ! echo "$COMMAND" | grep -qE '\bgit\b.*\bpush\b'; then
    exit 0
fi

# Resolve project root
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TASKS_DIR="$PROJECT_ROOT/docs/tasks"
VERIFY_SCRIPT="$PROJECT_ROOT/scripts/verify-visual-tests.sh"

if [[ ! -d "$TASKS_DIR" ]]; then
    exit 0  # No tasks directory — allow push
fi

if [[ ! -f "$VERIFY_SCRIPT" ]]; then
    echo '{"decision":"block","reason":"verify-visual-tests.sh not found — cannot verify L4 requirements"}' >&2
    exit 2
fi

# Find tasks with Status: IN PROGRESS
FAILED_TASKS=()
REASONS=()

for task_file in "$TASKS_DIR"/*.md; do
    [[ -f "$task_file" ]] || continue

    # Check if task is IN PROGRESS
    if ! grep -q 'Status: IN PROGRESS' "$task_file"; then
        continue
    fi

    task_name="$(basename "$task_file" .md)"

    # Run verification
    output=$(bash "$VERIFY_SCRIPT" "$task_file" 2>&1) || {
        FAILED_TASKS+=("$task_name")
        # Extract the specific errors
        errors=$(echo "$output" | grep '^\s*-' | head -3)
        REASONS+=("$task_name: $errors")
    }
done

if [[ ${#FAILED_TASKS[@]} -eq 0 ]]; then
    exit 0  # All tasks pass — allow push
fi

# Build denial message
TASK_LIST=$(printf ', %s' "${FAILED_TASKS[@]}")
TASK_LIST="${TASK_LIST:2}"  # Remove leading ", "

REASON="L4 visual test requirements not met for IN PROGRESS task(s): ${TASK_LIST}. Run the iterm2-driver test scripts, review screenshots, and add a '## Visual Test Results' section to each task file before pushing."

# Block the push
echo "{\"decision\":\"block\",\"reason\":\"$REASON\"}"
exit 2
