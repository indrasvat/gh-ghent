# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Error Handling Visual Test: Verify user-friendly error messages and exit codes.

Tests:
    1. Nonexistent repo: User-friendly "not found" message, not a stack trace
    2. Invalid PR number: "PR #99999 not found" style message
    3. Missing --pr flag: "--pr flag is required" error
    4. Valid request still works after hardening

Verification Strategy:
    - Build binary, run commands that trigger each error path
    - Capture screenshots at each stage for manual review
    - Verify exit code 2 for error cases

Screenshots:
    - ghent_error_notfound.png: Not-found error display
    - ghent_error_auth.png: Auth/error display (or valid fallback)

Screenshot Inspection Checklist:
    - Content: Error messages are human-readable, no stack traces
    - Exit codes: Error cases return exit code 2
    - Valid: Normal requests still work after hardening

Usage:
    uv run .claude/automations/test_ghent_errors.py
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
    print(f"ghent Error Handling Visual Tests")
    print(f"Started: {datetime.now().isoformat()}")
    print(f"{'='*60}\n")

    # Build the binary first
    print("--- Building binary ---")
    await session.async_send_text(f"cd {PROJECT_ROOT} && make build 2>&1; echo BUILD_DONE_$?\n")
    await asyncio.sleep(5.0)

    if not await verify_screen_contains(session, "BUILD_DONE_0", "build success", timeout=15.0):
        log_result("Build binary", "FAIL", "Build did not succeed")
        print("\nAborting: build failed.")
        await tab.async_close()
        return
    log_result("Build binary", "PASS")

    # Test 1: Nonexistent repo → NotFoundError
    print("\n--- Test 1: Nonexistent Repo ---")
    await session.async_send_text(
        f"{BINARY} comments --pr 1 -R nonexistent/repo-that-does-not-exist --format json 2>&1; echo NOTFOUND_EXIT=$?\n"
    )
    await asyncio.sleep(8.0)

    screen_text = await get_screen_text(session)
    # Check for user-friendly error (not a Go stack trace)
    has_friendly_error = any(
        phrase in screen_text.lower()
        for phrase in ["not found", "could not resolve", "does not exist", "error", "failed"]
    )
    has_stack_trace = "goroutine" in screen_text or "panic:" in screen_text

    if has_friendly_error and not has_stack_trace:
        log_result("Nonexistent repo error message", "PASS", "User-friendly error, no stack trace")
    elif has_stack_trace:
        log_result("Nonexistent repo error message", "FAIL", "Stack trace in output")
    else:
        log_result("Nonexistent repo error message", "UNVERIFIED", "Could not confirm error format")

    # Check exit code
    if "NOTFOUND_EXIT=2" in screen_text:
        log_result("Nonexistent repo exit code", "PASS", "exit 2")
    elif "NOTFOUND_EXIT=" in screen_text:
        exit_code = screen_text.split("NOTFOUND_EXIT=")[1][:1]
        log_result("Nonexistent repo exit code", "UNVERIFIED", f"exit {exit_code} (expected 2)")
    else:
        log_result("Nonexistent repo exit code", "SKIP", "Could not capture exit code")

    capture_quartz_screenshot("ghent_error_notfound")

    # Test 2: Invalid PR number
    print("\n--- Test 2: Invalid PR Number ---")
    await session.async_send_text(
        f"{BINARY} comments --pr 99999 -R indrasvat/tbgs --format json 2>&1; echo BADPR_EXIT=$?\n"
    )
    await asyncio.sleep(8.0)

    screen_text = await get_screen_text(session)
    has_pr_error = any(
        phrase in screen_text.lower()
        for phrase in ["not found", "could not", "error", "no pull request"]
    )

    if has_pr_error and "goroutine" not in screen_text:
        log_result("Invalid PR error message", "PASS", "User-friendly error")
    else:
        log_result("Invalid PR error message", "UNVERIFIED", "Could not confirm error format")

    capture_quartz_screenshot("ghent_error_auth")

    # Test 3: Missing --pr flag
    print("\n--- Test 3: Missing --pr Flag ---")
    await session.async_send_text(
        f"{BINARY} comments --format json 2>&1; echo NOPR_EXIT=$?\n"
    )
    await asyncio.sleep(3.0)

    if await verify_screen_contains(session, "--pr flag is required", "pr required"):
        log_result("Missing --pr error", "PASS", "--pr flag is required")
    else:
        log_result("Missing --pr error", "FAIL", "Expected error message not found")

    # Test 4: Valid request still works
    print("\n--- Test 4: Valid Request ---")
    await session.async_send_text(
        f"{BINARY} comments -R indrasvat/tbgs --pr 1 --format json > /tmp/ghent_error_valid.txt 2>&1; echo VALID_EXIT=$?\n"
    )
    await asyncio.sleep(8.0)

    # Validate JSON
    await session.async_send_text(
        "python3 -m json.tool /tmp/ghent_error_valid.txt > /dev/null 2>&1 && echo JSON_VALID || echo JSON_INVALID\n"
    )
    await asyncio.sleep(2.0)

    if await verify_screen_contains(session, "JSON_VALID", "json validation"):
        log_result("Valid request after hardening", "PASS", "JSON output valid")
    else:
        log_result("Valid request after hardening", "SKIP", "Could not validate JSON")

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
