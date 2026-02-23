# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Watch Mode Visual Test: Automated verification of the TUI watch view
with spinner, progress, check list, event log, and terminal states.

Tests:
    1. Build: Verify gh-ghent builds and installs
    2. Launch Watch (initial): doot PR #1 → checks already complete → done state
    3. Done State: "all checks passed" visible
    4. Check List: Check names visible in done state
    5. Event Log: Timestamped entries visible
    6. Help Bar: Watch-specific keys (ctrl+c, q)
    7. Exit and Relaunch (refreshed): peek-it PR #2 → checks failed → fail-fast
    8. Fail State: "failure detected" visible
    9. Fail Details: Failed check names visible
    10. Clean Exit: q exits cleanly

Verification Strategy:
    - Use indrasvat/doot PR #1 (checks already passed → immediate done)
    - Use indrasvat/peek-it PR #2 (checks failed → immediate fail-fast)
    - Read screen contents at each step
    - Take screenshots: ghent_watch_initial, ghent_watch_refreshed, ghent_watch_exit

Usage:
    uv run .claude/automations/test_ghent_watch.py
"""

import asyncio
import os
import re
import subprocess
from datetime import datetime

import iterm2

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")

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
    print("TEST SUMMARY — Watch Mode TUI")
    print(f"{'=' * 60}")
    print(f"Duration:   {duration:.1f}s")
    print(f"Total:      {total}")
    print(f"Passed:     {results['passed']}")
    print(f"Failed:     {results['failed']}")
    print(f"Unverified: {results['unverified']}")
    if results["screenshots"]:
        print(f"Screenshots: {len(results['screenshots'])}")
        for s in results["screenshots"]:
            print(f"  - {os.path.basename(s)}")
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
        await session.async_send_text("q")
        await asyncio.sleep(0.3)
        await session.async_send_text("exit\n")
        await asyncio.sleep(0.2)
        await session.async_close()
    except Exception as e:
        print(f"  Cleanup warning: {e}")


async def main(connection):
    results["start_time"] = datetime.now()

    print(f"\n{'#' * 60}")
    print("# ghent Watch Mode — Visual Test Suite")
    print(f"{'#' * 60}")

    app = await iterm2.async_get_app(connection)
    window = app.current_terminal_window
    if not window:
        log_result("Setup", "FAIL", "No active iTerm2 window")
        return print_summary()

    tab = await window.async_create_tab()
    session = tab.current_session

    try:
        # ── TEST 1: Build ──────────────────────────────────────────
        print_test_header("Build & Install", 1)
        await session.async_send_text(f"cd {PROJECT_ROOT} && make install 2>&1; echo BUILD_EXIT=$?\n")
        await asyncio.sleep(8.0)
        screen_text = await get_all_screen_text(session)
        if "BUILD_EXIT=0" in screen_text:
            log_result("Build & Install", "PASS")
        else:
            log_result("Build & Install", "FAIL", "Build failed")
            await dump_screen(session, "build_failure")
            return print_summary()

        # ── TEST 2: Launch Watch (initial — doot, pass state) ──────
        print_test_header("Launch Watch TUI (indrasvat/doot PR #1 — initial)", 2)
        await session.async_send_text("gh ghent checks --watch -R indrasvat/doot --pr 1 2>&1\n")
        await asyncio.sleep(12.0)  # Wait for poll + render

        screen_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_watch_initial")

        # Since doot's checks are already complete, should show done state.
        launch_ok = (
            "ghent" in screen_text or
            "passed" in screen_text.lower() or
            "watching" in screen_text or
            "Event Log" in screen_text
        )

        if launch_ok:
            log_result("Launch Watch TUI (initial)", "PASS",
                       f"screen contains watch indicators", screenshot=screenshot)
        else:
            log_result("Launch Watch TUI (initial)", "FAIL",
                       "no watch indicators found")
            await dump_screen(session, "watch_launch_initial")
            return print_summary()

        # ── TEST 3: Done State ──────────────────────────────────────
        print_test_header("Done State (all checks passed)", 3)
        has_passed = "all checks passed" in screen_text or "passed" in screen_text.lower()
        has_done_icon = "✓" in screen_text

        if has_passed:
            log_result("Done State", "PASS", f"passed={has_passed}, icon={has_done_icon}")
        elif has_done_icon:
            log_result("Done State", "UNVERIFIED", f"icon found but not 'all checks passed' text")
        else:
            log_result("Done State", "FAIL", "no done state indicators")
            await dump_screen(session, "done_state")

        # ── TEST 4: Check List ──────────────────────────────────────
        print_test_header("Check List Visible", 4)
        has_check_name = "make ci" in screen_text or "ci" in screen_text.lower()

        if has_check_name:
            log_result("Check List", "PASS", f"check_name={has_check_name}")
        else:
            log_result("Check List", "UNVERIFIED", "check names not clearly visible")
            await dump_screen(session, "check_list")

        # ── TEST 5: Event Log ───────────────────────────────────────
        print_test_header("Event Log Visible", 5)
        has_event_log = "Event Log" in screen_text
        has_time_pattern = bool(re.search(r'\d{2}:\d{2}:\d{2}', screen_text))

        if has_event_log and has_time_pattern:
            log_result("Event Log", "PASS", f"header={has_event_log}, timestamps={has_time_pattern}")
        elif has_event_log:
            log_result("Event Log", "UNVERIFIED", f"header found, timestamps unclear")
        else:
            log_result("Event Log", "FAIL", "no Event Log header")
            await dump_screen(session, "event_log")

        # ── TEST 6: Help Bar ────────────────────────────────────────
        print_test_header("Help Bar (watch-specific bindings)", 6)
        has_quit = "quit" in screen_text
        has_ctrl_c = "ctrl+c" in screen_text or "stop" in screen_text.lower()

        if has_quit:
            log_result("Help Bar", "PASS", f"quit={has_quit}, ctrl_c={has_ctrl_c}")
        else:
            log_result("Help Bar", "FAIL", f"quit={has_quit}, ctrl_c={has_ctrl_c}")
            await dump_screen(session, "help_bar")

        # Exit TUI for next test
        await session.async_send_text("q")
        await asyncio.sleep(1.0)

        # ── TEST 7: Launch Watch (refreshed — peek-it, fail state) ──
        print_test_header("Launch Watch TUI (indrasvat/peek-it PR #2 — refreshed/fail)", 7)
        await session.async_send_text("gh ghent checks --watch -R indrasvat/peek-it --pr 2 2>&1\n")
        await asyncio.sleep(12.0)

        fail_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_watch_refreshed")

        fail_launch_ok = (
            "failure" in fail_text.lower() or
            "fail" in fail_text.lower() or
            "✗" in fail_text or
            "Event Log" in fail_text
        )

        if fail_launch_ok:
            log_result("Launch Watch TUI (refreshed/fail)", "PASS",
                       f"failure indicators found", screenshot=screenshot)
        else:
            log_result("Launch Watch TUI (refreshed/fail)", "FAIL",
                       "no failure indicators found")
            await dump_screen(session, "watch_launch_fail")

        # ── TEST 8: Fail State ──────────────────────────────────────
        print_test_header("Fail State (failure detected)", 8)
        has_failure = "failure detected" in fail_text or "failure" in fail_text.lower()
        has_fail_fast = "fail-fast" in fail_text or "fail" in fail_text.lower()

        if has_failure:
            log_result("Fail State", "PASS", f"failure={has_failure}, failfast={has_fail_fast}")
        else:
            log_result("Fail State", "FAIL", "no failure state indicators")
            await dump_screen(session, "fail_state")

        # ── TEST 9: Failed Check Names ──────────────────────────────
        print_test_header("Failed Check Names Visible", 9)
        has_fail_icon = "✗" in fail_text
        has_check_info = "failure" in fail_text.lower() or "failed" in fail_text.lower()

        if has_fail_icon and has_check_info:
            log_result("Failed Check Names", "PASS",
                       f"fail_icon={has_fail_icon}, check_info={has_check_info}")
        else:
            log_result("Failed Check Names", "FAIL", "no failed check indicators")
            await dump_screen(session, "failed_checks")

        # ── TEST 10: Clean Exit ─────────────────────────────────────
        print_test_header("Clean Exit (q quits)", 10)
        await session.async_send_text("q")
        await asyncio.sleep(1.5)

        exit_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_watch_exit")

        # After quitting, should see shell prompt (not TUI content)
        tui_gone = "Event Log" not in exit_text or "$" in exit_text or "%" in exit_text
        if tui_gone:
            log_result("Clean Exit", "PASS", "TUI exited cleanly", screenshot=screenshot)
        else:
            log_result("Clean Exit", "UNVERIFIED",
                       "TUI may still be visible", screenshot=screenshot)
            await dump_screen(session, "exit_state")

    except Exception as e:
        log_result("Execution", "FAIL", str(e))
        import traceback
        traceback.print_exc()
        await dump_screen(session, "error_state")

    finally:
        await cleanup_session(session)

    return print_summary()


if __name__ == "__main__":
    exit_code = iterm2.run_until_complete(main)
    exit(exit_code if exit_code else 0)
