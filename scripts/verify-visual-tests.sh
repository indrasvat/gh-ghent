#!/usr/bin/env bash
# scripts/verify-visual-tests.sh — Verify L4 visual test requirements for a task file
#
# Usage: bash scripts/verify-visual-tests.sh <task-file-path>
#
# Checks:
#   1. L4 test scripts referenced in "Files to Create" exist in .claude/automations/
#   2. Expected screenshots from L4 section exist in .claude/screenshots/
#   3. Task file contains a "## Visual Test Results" section with review content
#
# Exit codes:
#   0 = all checks pass (or task has no L4 requirements)
#   1 = one or more checks failed
set -euo pipefail

TASK_FILE="${1:-}"
if [[ -z "$TASK_FILE" ]]; then
    echo "Usage: $0 <task-file-path>" >&2
    exit 1
fi

if [[ ! -f "$TASK_FILE" ]]; then
    echo "ERROR: Task file not found: $TASK_FILE" >&2
    exit 1
fi

# Resolve project root relative to this script
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
AUTOMATIONS_DIR="$PROJECT_ROOT/.claude/automations"
SCREENSHOTS_DIR="$PROJECT_ROOT/.claude/screenshots"

TASK_NAME="$(basename "$TASK_FILE" .md)"
ERRORS=()

# ============================================================
# Check 1: Extract test_ghent_*.py scripts from "Files to Create"
# ============================================================
L4_SCRIPTS=()
while IFS= read -r line; do
    # Match lines like: - `.claude/automations/test_ghent_foo.py`
    # or: - `test_ghent_foo.py`
    script_name=$(echo "$line" | grep -oE 'test_ghent_[a-z_]+\.py' || true)
    if [[ -n "$script_name" ]]; then
        L4_SCRIPTS+=("$script_name")
    fi
done < "$TASK_FILE"

# De-duplicate
if [[ ${#L4_SCRIPTS[@]} -gt 0 ]]; then
    mapfile -t L4_SCRIPTS < <(printf '%s\n' "${L4_SCRIPTS[@]}" | sort -u)
fi

# If no L4 scripts referenced, task has no visual test requirements
if [[ ${#L4_SCRIPTS[@]} -eq 0 ]]; then
    echo "PASS: Task $TASK_NAME has no L4 visual test requirements"
    exit 0
fi

echo "Task: $TASK_NAME"
echo "L4 scripts required: ${L4_SCRIPTS[*]}"
echo ""

# Check each script exists
for script in "${L4_SCRIPTS[@]}"; do
    if [[ -f "$AUTOMATIONS_DIR/$script" ]]; then
        echo "  [OK] Script exists: $script"
    else
        ERRORS+=("L4 script missing: $AUTOMATIONS_DIR/$script — create it before marking task DONE")
        echo "  [FAIL] Script missing: $script"
    fi
done

# ============================================================
# Check 2: Extract expected screenshot prefixes from L4 section
# ============================================================
EXPECTED_SCREENSHOTS=()
in_l4_section=false
while IFS= read -r line; do
    # Detect L4 section headers
    if echo "$line" | grep -qiE '##.*L4|##.*Visual.*iterm|##.*iterm2-driver'; then
        in_l4_section=true
        continue
    fi
    # Exit L4 section on next ## header
    if $in_l4_section && echo "$line" | grep -qE '^##[^#]'; then
        in_l4_section=false
        continue
    fi
    if $in_l4_section; then
        # Match screenshot names like: ghent_foo_bar.png or `ghent_foo.png`
        while IFS= read -r match; do
            if [[ -n "$match" ]]; then
                # Strip the .png suffix to get the prefix for glob matching
                prefix="${match%.png}"
                EXPECTED_SCREENSHOTS+=("$prefix")
            fi
        done < <(echo "$line" | grep -oE 'ghent_[a-z_]+\.png' || true)
    fi
done < "$TASK_FILE"

# De-duplicate
if [[ ${#EXPECTED_SCREENSHOTS[@]} -gt 0 ]]; then
    mapfile -t EXPECTED_SCREENSHOTS < <(printf '%s\n' "${EXPECTED_SCREENSHOTS[@]}" | sort -u)
fi

echo ""
if [[ ${#EXPECTED_SCREENSHOTS[@]} -gt 0 ]]; then
    echo "Expected screenshots: ${EXPECTED_SCREENSHOTS[*]}"
    for prefix in "${EXPECTED_SCREENSHOTS[@]}"; do
        # Screenshots have timestamps appended: ghent_foo_20260222_143000.png
        matches=$(find "$SCREENSHOTS_DIR" -name "${prefix}_*.png" 2>/dev/null | head -1)
        # Also check exact match (without timestamp)
        exact_match=$(find "$SCREENSHOTS_DIR" -name "${prefix}.png" 2>/dev/null | head -1)
        if [[ -n "$matches" || -n "$exact_match" ]]; then
            echo "  [OK] Screenshot found: ${prefix}"
        else
            ERRORS+=("Screenshot missing: ${prefix}.png — run the L4 test script to capture it")
            echo "  [FAIL] Screenshot missing: ${prefix}"
        fi
    done
else
    echo "No specific screenshots referenced in L4 section (checking scripts only)"
fi

# ============================================================
# Check 3: "## Visual Test Results" section exists with content
# ============================================================
echo ""
if grep -q '## Visual Test Results' "$TASK_FILE"; then
    # Check it has actual content (not just the header)
    in_results=false
    content_lines=0
    while IFS= read -r line; do
        if echo "$line" | grep -q '## Visual Test Results'; then
            in_results=true
            continue
        fi
        if $in_results; then
            # Stop at next section
            if echo "$line" | grep -qE '^##[^#]'; then
                break
            fi
            # Count non-empty lines
            if [[ -n "$(echo "$line" | tr -d '[:space:]')" ]]; then
                ((content_lines++))
            fi
        fi
    done < "$TASK_FILE"

    if [[ $content_lines -ge 3 ]]; then
        echo "  [OK] Visual Test Results section has content ($content_lines lines)"
    else
        ERRORS+=("Visual Test Results section is empty or too brief ($content_lines lines) — document your review findings")
        echo "  [FAIL] Visual Test Results section too brief ($content_lines lines, need >= 3)"
    fi
else
    ERRORS+=("Missing '## Visual Test Results' section in task file — add it with screenshots reviewed, findings, and pass/fail status")
    echo "  [FAIL] No '## Visual Test Results' section found"
fi

# ============================================================
# Summary
# ============================================================
echo ""
echo "================================"
if [[ ${#ERRORS[@]} -eq 0 ]]; then
    echo "RESULT: PASS — All L4 requirements met for $TASK_NAME"
    exit 0
else
    echo "RESULT: FAIL — ${#ERRORS[@]} issue(s) found for $TASK_NAME"
    echo ""
    for err in "${ERRORS[@]}"; do
        echo "  - $err"
    done
    exit 1
fi
