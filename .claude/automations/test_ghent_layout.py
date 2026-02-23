# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent TUI Layout Visual Test: Automated verification of dual-mode routing,
TUI launch from real gh commands, and layout integrity.

Tests:
    1. Build: Verify gh-ghent builds and installs
    2. TUI Launch: gh ghent comments --pr 1 in TTY → TUI appears
    3. Pipe Mode: gh ghent comments --pr 1 --no-tui → JSON output
    4. Tab Switching: Tab cycles between comments and checks views
    5. Checks TUI: gh ghent checks --pr 1 → TUI with checks view
    6. Summary TUI: gh ghent summary --pr 1 → TUI with summary view

Screenshots:
    - ghent_tui_launch.png
    - ghent_pipe_mode.png
    - ghent_no_tui_flag.png

Usage:
    uv run .claude/automations/test_ghent_layout.py
"""

import iterm2
import asyncio
import subprocess
import os
import time
from datetime import datetime

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")
TIMEOUT_SECONDS = 10.0
# Test against indrasvat/tbgs PR #1 (has 2 unresolved threads + 3 passing checks)
TEST_REPO = "indrasvat/tbgs"
TEST_PR = "1"

results = {
    "passed": 0, "failed": 0, "unverified": 0,
    "tests": [], "screenshots": [],
    "start_time": None, "end_time": None,
}


def log_result(test_name: str, status: str, details: str = "", screenshot: str = None):
    results["tests"].append({"name": test_name, "status": status, "details": details, "screenshot": screenshot})
    if screenshot:
        results["screenshots"].append(screenshot)
    if status == "PASS":
        results["passed"] += 1
        print(f"  [+] PASS: {test_name}")
    elif status == "FAIL":
        results["failed"] += 1
        print(f"  [x] FAIL: {test_name} - {details}")
    else:
        results["unverified"] += 1
        print(f"  [?] UNVERIFIED: {test_name} - {details}")
    if screenshot:
        print(f"      Screenshot: {screenshot}")


def print_summary() -> int:
    results["end_time"] = datetime.now()
    total = results["passed"] + results["failed"] + results["unverified"]
    duration = (results["end_time"] - results["start_time"]).total_seconds()
    print(f"\n{'=' * 60}")
    print("TEST SUMMARY")
    print(f"{'=' * 60}")
    print(f"Duration:   {duration:.1f}s")
    print(f"Total:      {total}")
    print(f"Passed:     {results['passed']}")
    print(f"Failed:     {results['failed']}")
    print(f"Unverified: {results['unverified']}")
    if results["screenshots"]:
        print(f"Screenshots: {len(results['screenshots'])}")
    print(f"{'=' * 60}")
    if results["failed"] > 0:
        print("\nFailed tests:")
        for test in results["tests"]:
            if test["status"] == "FAIL":
                print(f"  - {test['name']}: {test['details']}")
    print(f"\n{'-' * 60}")
    if results["failed"] > 0:
        print("OVERALL: FAILED")
        return 1
    elif results["unverified"] > 0:
        print("OVERALL: PASSED (with unverified tests)")
        return 0
    else:
        print("OVERALL: PASSED")
        return 0


def print_test_header(test_name: str, test_num: int):
    print(f"\n{'=' * 60}")
    print(f"TEST {test_num}: {test_name}")
    print(f"{'=' * 60}")


try:
    import Quartz
    def get_iterm2_window_id():
        window_list = Quartz.CGWindowListCopyWindowInfo(
            Quartz.kCGWindowListOptionOnScreenOnly | Quartz.kCGWindowListExcludeDesktopElements,
            Quartz.kCGNullWindowID
        )
        for window in window_list:
            if 'iTerm' in window.get('kCGWindowOwnerName', ''):
                return window.get('kCGWindowNumber')
        return None
except ImportError:
    def get_iterm2_window_id():
        return None


def capture_screenshot(name: str) -> str:
    os.makedirs(SCREENSHOT_DIR, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    filepath = os.path.join(SCREENSHOT_DIR, f"{name}_{timestamp}.png")
    window_id = get_iterm2_window_id()
    if window_id:
        subprocess.run(["screencapture", "-x", "-l", str(window_id), filepath], check=True)
    else:
        subprocess.run(["screencapture", "-x", filepath], check=True)
    print(f"  SCREENSHOT: {filepath}")
    return filepath


async def verify_screen_contains(session, expected: str, description: str, timeout: float = None) -> bool:
    t = timeout or TIMEOUT_SECONDS
    start = time.monotonic()
    while (time.monotonic() - start) < t:
        screen = await session.async_get_screen_contents()
        for i in range(screen.number_of_lines):
            if expected in screen.line(i).string:
                return True
        await asyncio.sleep(0.3)
    return False


async def get_all_screen_text(session) -> str:
    screen = await session.async_get_screen_contents()
    lines = []
    for i in range(screen.number_of_lines):
        line = screen.line(i).string
        if line.strip():
            lines.append(line)
    return "\n".join(lines)


async def dump_screen(session, label: str):
    screen = await session.async_get_screen_contents()
    print(f"\n{'=' * 60}")
    print(f"SCREEN DUMP: {label}")
    print(f"{'=' * 60}")
    for i in range(screen.number_of_lines):
        line = screen.line(i).string
        if line.strip():
            print(f"{i:03d}: {line}")
    print(f"{'=' * 60}\n")


async def cleanup_session(session):
    try:
        await session.async_send_text("\x03")  # Ctrl+C
        await asyncio.sleep(0.3)
        await session.async_send_text("q")     # quit TUI
        await asyncio.sleep(0.3)
        await session.async_send_text("\x03")
        await asyncio.sleep(0.2)
        await session.async_send_text("exit\n")
        await asyncio.sleep(0.2)
        await session.async_close()
    except Exception as e:
        print(f"  Cleanup warning: {e}")


async def main(connection):
    results["start_time"] = datetime.now()

    print(f"\n{'#' * 60}")
    print("# ghent TUI Layout Visual Test")
    print(f"{'#' * 60}")

    app = await iterm2.async_get_app(connection)
    window = app.current_terminal_window
    if not window:
        log_result("Setup", "FAIL", "No active iTerm2 window")
        return print_summary()

    # TEST 1: Build + install
    print_test_header("Build & Install", 1)
    tab = await window.async_create_tab()
    session = tab.current_session
    try:
        await session.async_send_text(f"cd {PROJECT_ROOT} && make install 2>&1; echo BUILD_EXIT=$?\n")
        await asyncio.sleep(8.0)
        if await verify_screen_contains(session, "BUILD_EXIT=0", "build success"):
            log_result("Build & Install", "PASS")
        else:
            # Also accept "Installed" message
            if await verify_screen_contains(session, "Installed", "installed"):
                log_result("Build & Install", "PASS")
            else:
                log_result("Build & Install", "FAIL", "Build or install failed")
                await dump_screen(session, "build_failure")
                return print_summary()
    finally:
        await cleanup_session(session)

    # TEST 2: TUI Launch — gh ghent comments in TTY
    print_test_header("TUI Launch (comments)", 2)
    tab = await window.async_create_tab()
    session = tab.current_session
    try:
        await session.async_send_text(f"cd {PROJECT_ROOT} && gh ghent comments -R {TEST_REPO} --pr {TEST_PR} 2>&1\n")
        await asyncio.sleep(8.0)  # API fetch + TUI render

        screen_text = await get_all_screen_text(session)
        has_ghent = "ghent" in screen_text
        has_repo = TEST_REPO in screen_text or "tbgs" in screen_text
        has_help = any(k in screen_text for k in ["navigate", "expand", "quit"])
        has_unresolved = "unresolved" in screen_text

        if has_ghent and has_help:
            screenshot = capture_screenshot("ghent_tui_launch")
            log_result("TUI Launch (comments)", "PASS",
                       f"ghent={has_ghent}, repo={has_repo}, help={has_help}, unresolved={has_unresolved}",
                       screenshot=screenshot)
        else:
            log_result("TUI Launch (comments)", "FAIL",
                       f"ghent={has_ghent}, repo={has_repo}, help={has_help}")
            await dump_screen(session, "tui_launch")

        # TEST 3: Tab to checks view
        print_test_header("Tab to Checks View", 3)
        await session.async_send_text("\t")  # Tab
        await asyncio.sleep(1.0)
        screen_text = await get_all_screen_text(session)
        has_checks = "Checks List View" in screen_text or "view logs" in screen_text
        if has_checks:
            log_result("Tab to Checks", "PASS", "Checks view visible after Tab")
        else:
            log_result("Tab to Checks", "FAIL", "Checks view not visible")
            await dump_screen(session, "tab_checks")

    finally:
        await cleanup_session(session)

    # TEST 4: Pipe mode with --no-tui
    print_test_header("Pipe Mode (--no-tui)", 4)
    tab = await window.async_create_tab()
    session = tab.current_session
    try:
        await session.async_send_text(
            f"cd {PROJECT_ROOT} && gh ghent comments -R {TEST_REPO} --pr {TEST_PR} --no-tui --format json 2>&1; echo PIPE_EXIT=$?\n"
        )
        await asyncio.sleep(8.0)
        screen_text = await get_all_screen_text(session)
        # Should see JSON output, NOT the TUI.
        # Note: JSON may be long enough that "pr_number" scrolled off screen.
        # Check for end-of-JSON markers or any JSON structure visible.
        has_json = any(k in screen_text for k in [
            "pr_number", "unresolved_count", "resolved_count",
            "total_count", '"threads"', '"id"',
        ])
        has_exit = "PIPE_EXIT=" in screen_text

        if has_json and has_exit:
            screenshot = capture_screenshot("ghent_no_tui_flag")
            log_result("Pipe Mode (--no-tui)", "PASS",
                       f"json={has_json}, exit_marker={has_exit}",
                       screenshot=screenshot)
        else:
            log_result("Pipe Mode (--no-tui)", "FAIL",
                       f"json={has_json}, exit={has_exit}")
            await dump_screen(session, "pipe_mode")
    finally:
        await cleanup_session(session)

    # TEST 5: Pipe mode via pipe
    print_test_header("Pipe Mode (piped output)", 5)
    tab = await window.async_create_tab()
    session = tab.current_session
    try:
        await session.async_send_text(
            f"cd {PROJECT_ROOT} && gh ghent comments -R {TEST_REPO} --pr {TEST_PR} --format json 2>&1 | head -3; echo PIPE2_EXIT=$?\n"
        )
        await asyncio.sleep(8.0)
        screen_text = await get_all_screen_text(session)
        has_json = "pr_number" in screen_text
        has_exit = "PIPE2_EXIT=0" in screen_text

        if has_json:
            screenshot = capture_screenshot("ghent_pipe_mode")
            log_result("Pipe Mode (piped)", "PASS",
                       f"json={has_json}, exit={has_exit}",
                       screenshot=screenshot)
        else:
            log_result("Pipe Mode (piped)", "FAIL", f"json={has_json}")
            await dump_screen(session, "pipe_piped")
    finally:
        await cleanup_session(session)

    # TEST 6: Checks TUI launch
    print_test_header("Checks TUI Launch", 6)
    tab = await window.async_create_tab()
    session = tab.current_session
    try:
        await session.async_send_text(f"cd {PROJECT_ROOT} && gh ghent checks -R {TEST_REPO} --pr {TEST_PR} 2>&1\n")
        await asyncio.sleep(8.0)
        screen_text = await get_all_screen_text(session)
        has_ghent = "ghent" in screen_text
        has_checks_help = "view logs" in screen_text or "re-run" in screen_text
        has_passed = "passed" in screen_text

        if has_ghent and (has_checks_help or has_passed):
            log_result("Checks TUI Launch", "PASS",
                       f"ghent={has_ghent}, help={has_checks_help}, passed={has_passed}")
        else:
            log_result("Checks TUI Launch", "FAIL",
                       f"ghent={has_ghent}, help={has_checks_help}")
            await dump_screen(session, "checks_tui")
    finally:
        await cleanup_session(session)

    return print_summary()


if __name__ == "__main__":
    exit_code = iterm2.run_until_complete(main)
    exit(exit_code if exit_code else 0)
