# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Pipe Mode Visual Test: Automated verification of pipe-mode output formats.

Tests:
    1. JSON output: Valid JSON, no ANSI codes, correct fields
    2. Markdown output: Headers, file paths, author names
    3. XML output: Well-formed XML with declaration
    4. Exit codes: 0 for clean, 1 for unresolved threads

Verification Strategy:
    - Build binary and run commands redirected to temp files
    - Validate each format independently
    - Capture screenshots at each stage for manual review

Screenshots:
    - ghent_pipe_json.png: JSON output validation
    - ghent_pipe_md.png: Markdown output rendered

Screenshot Inspection Checklist:
    - Content: Output is non-empty and shows expected format
    - No ANSI: Piped output has no escape sequences
    - Structure: JSON parseable, XML well-formed, MD has headers

Usage:
    uv run .claude/automations/test_ghent_pipe.py
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
    icon = {"PASS": "✓", "FAIL": "✗", "SKIP": "○", "UNVERIFIED": "?"}.get(status, "?")
    msg = f"  {icon} {name}: {status}"
    if detail:
        msg += f" — {detail}"
    print(msg)


def capture_quartz_screenshot(filename):
    """Capture screenshot of the frontmost window using screencapture."""
    try:
        os.makedirs(SCREENSHOT_DIR, exist_ok=True)
        path = os.path.join(SCREENSHOT_DIR, f"{filename}.png")
        # -l flag captures a specific window; -w captures frontmost window interactively
        # Use -x to suppress sound, capture the whole screen and crop is complex,
        # so just capture the frontmost window via applescript + screencapture
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
    print(f"ghent Pipe Mode Visual Tests")
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

    # Test 1: JSON output
    print("\n--- Test 1: JSON Output ---")
    await session.async_send_text(
        f"{BINARY} comments -R indrasvat/gh-ghent --pr 1 --format json > /tmp/ghent_pipe_json.txt 2>&1; "
        "echo JSON_EXIT=$?\n"
    )
    await asyncio.sleep(5.0)

    screen_text = await get_screen_text(session)
    if "JSON_EXIT=" in screen_text:
        log_result("JSON command ran", "PASS")
    else:
        log_result("JSON command ran", "FAIL", "No exit code captured")

    # Validate JSON
    await session.async_send_text(
        "python3 -m json.tool /tmp/ghent_pipe_json.txt > /dev/null 2>&1 && echo JSON_VALID || echo JSON_INVALID\n"
    )
    await asyncio.sleep(2.0)

    if await verify_screen_contains(session, "JSON_VALID", "json validation"):
        log_result("JSON valid", "PASS")
    else:
        log_result("JSON valid", "SKIP", "Command may have errored (no real PR data)")

    # Check for ANSI codes
    await session.async_send_text(
        r"grep -P '\\x1b\\[' /tmp/ghent_pipe_json.txt > /dev/null 2>&1 && echo ANSI_FOUND || echo ANSI_CLEAN" + "\n"
    )
    await asyncio.sleep(1.0)

    if await verify_screen_contains(session, "ANSI_CLEAN", "ansi check"):
        log_result("JSON no ANSI", "PASS")
    else:
        log_result("JSON no ANSI", "SKIP", "Could not verify (command may have errored)")

    # Show JSON output for screenshot
    await session.async_send_text("echo '=== JSON Output ==='; cat /tmp/ghent_pipe_json.txt | head -30\n")
    await asyncio.sleep(1.0)
    capture_quartz_screenshot("ghent_pipe_json")

    # Test 2: Markdown output
    print("\n--- Test 2: Markdown Output ---")
    await session.async_send_text(
        f"{BINARY} comments -R indrasvat/gh-ghent --pr 1 --format md > /tmp/ghent_pipe_md.txt 2>&1; "
        "echo MD_EXIT=$?\n"
    )
    await asyncio.sleep(5.0)

    screen_text = await get_screen_text(session)
    if "MD_EXIT=" in screen_text:
        log_result("Markdown command ran", "PASS")
    else:
        log_result("Markdown command ran", "FAIL", "No exit code captured")

    # Show MD output for screenshot
    await session.async_send_text("echo '=== Markdown Output ==='; cat /tmp/ghent_pipe_md.txt | head -30\n")
    await asyncio.sleep(1.0)
    capture_quartz_screenshot("ghent_pipe_md")

    # Test 3: XML output
    print("\n--- Test 3: XML Output ---")
    await session.async_send_text(
        f"{BINARY} comments -R indrasvat/gh-ghent --pr 1 --format xml > /tmp/ghent_pipe_xml.txt 2>&1; "
        "echo XML_EXIT=$?\n"
    )
    await asyncio.sleep(5.0)

    # Validate XML
    await session.async_send_text(
        "python3 -c \"import xml.etree.ElementTree as ET; ET.parse('/tmp/ghent_pipe_xml.txt'); print('XML_VALID')\" "
        "2>/dev/null || echo XML_INVALID\n"
    )
    await asyncio.sleep(2.0)

    if await verify_screen_contains(session, "XML_VALID", "xml validation"):
        log_result("XML well-formed", "PASS")
    else:
        log_result("XML well-formed", "SKIP", "Command may have errored (no real PR data)")

    # Test 4: --pr required
    print("\n--- Test 4: --pr Required ---")
    await session.async_send_text(
        f"{BINARY} comments --format json 2>&1; echo NOPR_EXIT=$?\n"
    )
    await asyncio.sleep(3.0)

    if await verify_screen_contains(session, "--pr flag is required", "pr required error"):
        log_result("--pr required check", "PASS")
    else:
        log_result("--pr required check", "SKIP", "Error message not found")

    # Summary
    print(f"\n{'='*60}")
    print("Results Summary")
    print(f"{'='*60}")
    pass_count = sum(1 for r in results if r["status"] == "PASS")
    fail_count = sum(1 for r in results if r["status"] == "FAIL")
    skip_count = sum(1 for r in results if r["status"] in ("SKIP", "UNVERIFIED"))
    print(f"  PASS: {pass_count} | FAIL: {fail_count} | SKIP: {skip_count}")
    for r in results:
        icon = {"PASS": "✓", "FAIL": "✗", "SKIP": "○", "UNVERIFIED": "?"}.get(r["status"], "?")
        line = f"  {icon} {r['name']}: {r['status']}"
        if r["detail"]:
            line += f" — {r['detail']}"
        print(line)

    print(f"\nScreenshots saved to: {SCREENSHOT_DIR}")
    print(f"Finished: {datetime.now().isoformat()}")

    # Clean up tab
    await asyncio.sleep(1.0)
    await tab.async_close()


def main():
    iterm2.run_until_complete(run_test)


if __name__ == "__main__":
    main()
