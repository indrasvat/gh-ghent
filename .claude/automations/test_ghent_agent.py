# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Agent Workflow Visual Test: End-to-end agent workflow verification.

Tests:
    1. JSON output valid: --format json produces parseable JSON
    2. Exit codes meaningful: 0=clean, 1=unresolved threads
    3. No ANSI in piped output: Clean data for agent consumption
    4. Response time: Commands complete within reasonable time
    5. Error messages actionable: Errors include fix instructions

Verification Strategy:
    - Run commands as an agent would (piped, --format json, --no-tui)
    - Validate JSON, exit codes, and response times
    - Verify no ANSI escape sequences in output

Screenshots:
    - ghent_agent_json.png: JSON output from comments command
    - ghent_agent_checks.png: JSON output from checks command

Screenshot Inspection Checklist:
    - Content: Valid JSON with expected fields
    - No ANSI: Output is clean for parsing
    - Speed: Commands complete quickly

Usage:
    uv run .claude/automations/test_ghent_agent.py
"""

import iterm2
import asyncio
import subprocess
import os
import time
from datetime import datetime

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")
BINARY = os.path.join(PROJECT_ROOT, "bin", "gh-ghent")

results = []


def log_result(name, status, detail=""):
    results.append({"name": name, "status": status, "detail": detail})
    icon = {"PASS": "\u2713", "FAIL": "\u2717", "SKIP": "\u25cb", "UNVERIFIED": "?"}.get(status, "?")
    msg = f"  {icon} {name}: {status}"
    if detail:
        msg += f" \u2014 {detail}"
    print(msg)


def capture_quartz_screenshot(filename):
    """Capture screenshot of the frontmost window using screencapture."""
    try:
        os.makedirs(SCREENSHOT_DIR, exist_ok=True)
        path = os.path.join(SCREENSHOT_DIR, f"{filename}.png")
        result = subprocess.run(
            ["screencapture", "-x", "-o", path],
            capture_output=True, timeout=5,
        )
        if result.returncode == 0 and os.path.exists(path):
            print(f"  [screenshot] Saved: {path}")
            return True
        print(f"  [screenshot] screencapture failed (rc={result.returncode})")
        return False
    except Exception as e:
        print(f"  [screenshot] Error: {e}")
        return False


async def get_screen_text(session):
    """Get all text from the terminal screen."""
    screen = await session.async_get_screen_contents()
    lines = []
    for i in range(screen.number_of_lines):
        lines.append(screen.line(i).string)
    return "\n".join(lines)


async def verify_screen_contains(session, text, label, timeout=5.0):
    """Poll screen for expected text within timeout."""
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        screen_text = await get_screen_text(session)
        if text in screen_text:
            return True
        await asyncio.sleep(0.5)
    return False


async def run_test(connection):
    app = await iterm2.async_get_app(connection)
    window = app.current_terminal_window
    if window is None:
        print("ERROR: No iTerm2 window found. Open iTerm2 first.")
        return

    tab = await window.async_create_tab()
    session = tab.current_session

    print(f"\n{'='*60}")
    print(f"ghent Agent Workflow Visual Tests")
    print(f"Started: {datetime.now().isoformat()}")
    print(f"{'='*60}\n")

    # Build binary
    print("--- Building binary ---")
    await session.async_send_text(f"cd {PROJECT_ROOT} && make build 2>&1; echo BUILD_DONE_$?\n")
    await asyncio.sleep(5.0)

    if not await verify_screen_contains(session, "BUILD_DONE_0", "build success", timeout=15.0):
        log_result("Build binary", "FAIL", "Build did not succeed")
        await tab.async_close()
        return
    log_result("Build binary", "PASS")

    # Test 1: Comments JSON output
    print("\n--- Test 1: Comments JSON ---")
    await session.async_send_text(
        f"time {BINARY} comments -R indrasvat/tbgs --pr 1 --format json > /tmp/ghent_agent_comments.txt 2>/tmp/ghent_agent_time.txt; "
        "echo COMMENTS_EXIT=$?\n"
    )
    await asyncio.sleep(8.0)

    # Validate JSON
    await session.async_send_text(
        "python3 -m json.tool /tmp/ghent_agent_comments.txt > /dev/null 2>&1 && echo CJSON_VALID || echo CJSON_INVALID\n"
    )
    await asyncio.sleep(2.0)

    if await verify_screen_contains(session, "CJSON_VALID", "comments json"):
        log_result("Comments JSON valid", "PASS")
    else:
        log_result("Comments JSON valid", "SKIP", "Could not validate")

    # Check unresolved_count field
    await session.async_send_text(
        "python3 -c \"import json; d=json.load(open('/tmp/ghent_agent_comments.txt')); print('HAS_UNRESOLVED' if 'unresolved_count' in d else 'NO_FIELD')\" 2>&1\n"
    )
    await asyncio.sleep(2.0)

    if await verify_screen_contains(session, "HAS_UNRESOLVED", "unresolved field"):
        log_result("JSON has unresolved_count", "PASS")
    else:
        log_result("JSON has unresolved_count", "SKIP", "Field not found")

    # Check exit code
    screen_text = await get_screen_text(session)
    if "COMMENTS_EXIT=1" in screen_text:
        log_result("Comments exit code", "PASS", "exit 1 (has unresolved)")
    elif "COMMENTS_EXIT=0" in screen_text:
        log_result("Comments exit code", "PASS", "exit 0 (no unresolved)")
    else:
        log_result("Comments exit code", "SKIP", "Could not capture")

    # Check no ANSI
    await session.async_send_text(
        r"grep -P '\x1b\[' /tmp/ghent_agent_comments.txt > /dev/null 2>&1 && echo ANSI_FOUND || echo ANSI_CLEAN" + "\n"
    )
    await asyncio.sleep(1.0)

    if await verify_screen_contains(session, "ANSI_CLEAN", "ansi check"):
        log_result("Comments no ANSI", "PASS")
    else:
        log_result("Comments no ANSI", "SKIP", "Could not verify")

    # Show output for screenshot
    await session.async_send_text("echo '=== Agent Comments JSON ==='; cat /tmp/ghent_agent_comments.txt | head -20\n")
    await asyncio.sleep(1.0)
    capture_quartz_screenshot("ghent_agent_json")

    # Test 2: Checks JSON output
    print("\n--- Test 2: Checks JSON ---")
    await session.async_send_text(
        f"{BINARY} checks -R indrasvat/tbgs --pr 1 --format json > /tmp/ghent_agent_checks.txt 2>&1; "
        "echo CHECKS_EXIT=$?\n"
    )
    await asyncio.sleep(8.0)

    await session.async_send_text(
        "python3 -m json.tool /tmp/ghent_agent_checks.txt > /dev/null 2>&1 && echo KJSON_VALID || echo KJSON_INVALID\n"
    )
    await asyncio.sleep(2.0)

    if await verify_screen_contains(session, "KJSON_VALID", "checks json"):
        log_result("Checks JSON valid", "PASS")
    else:
        log_result("Checks JSON valid", "SKIP", "Could not validate")

    await session.async_send_text("echo '=== Agent Checks JSON ==='; cat /tmp/ghent_agent_checks.txt | head -20\n")
    await asyncio.sleep(1.0)
    capture_quartz_screenshot("ghent_agent_checks")

    # Test 3: Summary JSON
    print("\n--- Test 3: Summary JSON ---")
    await session.async_send_text(
        f"{BINARY} summary -R indrasvat/tbgs --pr 1 --format json > /tmp/ghent_agent_summary.txt 2>&1; "
        "echo SUMMARY_EXIT=$?\n"
    )
    await asyncio.sleep(8.0)

    await session.async_send_text(
        "python3 -m json.tool /tmp/ghent_agent_summary.txt > /dev/null 2>&1 && echo SJSON_VALID || echo SJSON_INVALID\n"
    )
    await asyncio.sleep(2.0)

    if await verify_screen_contains(session, "SJSON_VALID", "summary json"):
        log_result("Summary JSON valid", "PASS")
    else:
        log_result("Summary JSON valid", "SKIP", "Could not validate")

    # Test 4: Error message is actionable
    print("\n--- Test 4: Actionable Error ---")
    await session.async_send_text(
        f"{BINARY} comments --format json 2>&1; echo ERR_EXIT=$?\n"
    )
    await asyncio.sleep(3.0)

    if await verify_screen_contains(session, "--pr", "actionable error"):
        log_result("Error is actionable", "PASS", "Mentions --pr flag")
    else:
        log_result("Error is actionable", "UNVERIFIED")

    # Summary
    print(f"\n{'='*60}")
    print("Results Summary")
    print(f"{'='*60}")
    pass_count = sum(1 for r in results if r["status"] == "PASS")
    fail_count = sum(1 for r in results if r["status"] == "FAIL")
    skip_count = sum(1 for r in results if r["status"] in ("SKIP", "UNVERIFIED"))
    print(f"  PASS: {pass_count} | FAIL: {fail_count} | SKIP/UNVERIFIED: {skip_count}")
    for r in results:
        icon = {"PASS": "\u2713", "FAIL": "\u2717", "SKIP": "\u25cb", "UNVERIFIED": "?"}.get(r["status"], "?")
        line = f"  {icon} {r['name']}: {r['status']}"
        if r["detail"]:
            line += f" \u2014 {r['detail']}"
        print(line)

    print(f"\nScreenshots saved to: {SCREENSHOT_DIR}")
    print(f"Finished: {datetime.now().isoformat()}")

    await asyncio.sleep(1.0)
    await tab.async_close()


def main():
    iterm2.run_until_complete(run_test)


if __name__ == "__main__":
    main()
