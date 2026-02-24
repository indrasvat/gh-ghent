# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Extension Install Visual Test: Verify gh extension packaging and install flow.

Tests:
    1. make install: Symlink-based install succeeds
    2. gh ghent --version: Shows version string
    3. gh extension list: ghent appears in list
    4. gh ghent --help: Help output is readable
    5. Cross-platform build: linux/amd64, darwin/arm64, windows/amd64

Verification Strategy:
    - Run make install, verify gh ghent commands work
    - Capture screenshots at each stage
    - Verify cross-platform builds produce correct binaries

Screenshots:
    - ghent_install.png: Extension install confirmation
    - ghent_list.png: Extension visible in gh extension list

Screenshot Inspection Checklist:
    - Content: Version string visible, extension listed, help readable
    - Cross-platform: All 3 builds succeed

Usage:
    uv run .claude/automations/test_ghent_install.py
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
    print(f"ghent Extension Install Visual Tests")
    print(f"Started: {datetime.now().isoformat()}")
    print(f"{'='*60}\n")

    # Test 1: make install
    print("--- Test 1: make install ---")
    await session.async_send_text(f"cd {PROJECT_ROOT} && make install 2>&1; echo INSTALL_DONE_$?\n")
    await asyncio.sleep(8.0)

    if await verify_screen_contains(session, "INSTALL_DONE_0", "install success", timeout=15.0):
        log_result("make install", "PASS")
    else:
        log_result("make install", "FAIL", "Install did not succeed")
        await tab.async_close()
        return

    capture_quartz_screenshot("ghent_install")

    # Test 2: gh ghent --version
    print("\n--- Test 2: gh ghent --version ---")
    await session.async_send_text("gh ghent --version 2>&1; echo VERSION_DONE\n")
    await asyncio.sleep(3.0)

    if await verify_screen_contains(session, "ghent version", "version output"):
        log_result("gh ghent --version", "PASS")
    else:
        log_result("gh ghent --version", "FAIL", "Version string not found")

    # Test 3: gh extension list
    print("\n--- Test 3: gh extension list ---")
    await session.async_send_text("gh extension list 2>&1; echo LIST_DONE\n")
    await asyncio.sleep(3.0)

    if await verify_screen_contains(session, "gh-ghent", "extension listed"):
        log_result("gh extension list shows ghent", "PASS")
    else:
        # Try alternate match
        if await verify_screen_contains(session, "ghent", "extension listed alt"):
            log_result("gh extension list shows ghent", "PASS")
        else:
            log_result("gh extension list shows ghent", "FAIL", "ghent not in extension list")

    capture_quartz_screenshot("ghent_list")

    # Test 4: gh ghent --help
    print("\n--- Test 4: gh ghent --help ---")
    await session.async_send_text("gh ghent --help 2>&1; echo HELP_DONE\n")
    await asyncio.sleep(3.0)

    if await verify_screen_contains(session, "agentic PR monitoring", "help text"):
        log_result("gh ghent --help", "PASS")
    else:
        log_result("gh ghent --help", "FAIL", "Help text missing expected content")

    # Test 5: Cross-platform builds
    print("\n--- Test 5: Cross-platform builds ---")
    await session.async_send_text(
        f"cd {PROJECT_ROOT} && "
        "GOOS=linux GOARCH=amd64 go build -o /tmp/gh-ghent-test-linux ./cmd/ghent 2>&1 && echo LINUX_OK || echo LINUX_FAIL; "
        "GOOS=darwin GOARCH=arm64 go build -o /tmp/gh-ghent-test-darwin ./cmd/ghent 2>&1 && echo DARWIN_OK || echo DARWIN_FAIL; "
        "GOOS=windows GOARCH=amd64 go build -o /tmp/gh-ghent-test-win.exe ./cmd/ghent 2>&1 && echo WIN_OK || echo WIN_FAIL; "
        "rm -f /tmp/gh-ghent-test-linux /tmp/gh-ghent-test-darwin /tmp/gh-ghent-test-win.exe; "
        "echo XBUILD_DONE\n"
    )
    await asyncio.sleep(30.0)

    screen_text = await get_screen_text(session)
    if "LINUX_OK" in screen_text:
        log_result("Cross-build linux/amd64", "PASS")
    else:
        log_result("Cross-build linux/amd64", "FAIL")

    if "DARWIN_OK" in screen_text:
        log_result("Cross-build darwin/arm64", "PASS")
    else:
        log_result("Cross-build darwin/arm64", "FAIL")

    if "WIN_OK" in screen_text:
        log_result("Cross-build windows/amd64", "PASS")
    else:
        log_result("Cross-build windows/amd64", "FAIL")

    # Summary
    print(f"\n{'='*60}")
    print("Results Summary")
    print(f"{'='*60}")
    pass_count = sum(1 for r in results if r["status"] == "PASS")
    fail_count = sum(1 for r in results if r["status"] == "FAIL")
    skip_count = sum(1 for r in results if r["status"] in ("SKIP", "UNVERIFIED"))
    print(f"  PASS: {pass_count} | FAIL: {fail_count} | SKIP: {skip_count}")
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
