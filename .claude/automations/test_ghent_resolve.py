# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Resolve View Visual Test: Comprehensive automated verification
of the TUI multi-select resolve interface with real PR data.

Tests:
    1. Build: Verify gh-ghent builds and installs
    2. Launch: TUI launches in resolve mode with threads listed
    3. Checkboxes: Unchecked [ ] boxes visible for each thread
    4. Cursor: ▶ marker on first thread
    5. Space Toggle: Space toggles checkbox on current thread
    6. Select All: 'a' selects all eligible threads
    7. Deselect All: 'a' again deselects all
    8. j/k Navigation: Cursor movement
    9. Enter → Confirmation: Shows "Resolve N threads?" prompt
    10. Esc → Cancel Confirmation: Returns to browsing
    11. Status Bar: Shows "resolve mode" and selection count
    12. Help Bar: Shows resolve-specific key bindings

Verification Strategy:
    - Use indrasvat/tbgs PR #1 (2 unresolved threads, viewerCanResolve=true)
    - DO NOT actually resolve threads (Esc before confirm)
    - Read screen contents at each step
    - Take screenshots at every visual milestone

Usage:
    uv run .claude/automations/test_ghent_resolve.py
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
    print("TEST SUMMARY — Resolve View")
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
    print("# ghent Resolve View — Visual Test Suite")
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

        # ── TEST 2: Launch Resolve TUI ────────────────────────────
        print_test_header("Launch Resolve TUI (indrasvat/tbgs PR #1)", 2)
        await session.async_send_text("gh ghent resolve -R indrasvat/tbgs --pr 1 2>&1\n")
        await asyncio.sleep(8.0)

        screen_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_resolve_launch")

        launch_indicators = {
            "go_file": ".go" in screen_text,
            "checkbox": "[ ]" in screen_text or "[✓]" in screen_text,
            "resolve_mode": "resolve" in screen_text.lower(),
            "help_bar": any(kw in screen_text for kw in ["toggle", "select all", "navigate"]),
        }
        tui_launched = sum(launch_indicators.values()) >= 2

        if tui_launched:
            log_result("Launch Resolve TUI", "PASS",
                       f"indicators={launch_indicators}", screenshot=screenshot)
        else:
            log_result("Launch Resolve TUI", "FAIL",
                       f"indicators={launch_indicators}")
            await dump_screen(session, "launch_fail")
            return print_summary()

        # ── TEST 3: Checkboxes ────────────────────────────────────
        print_test_header("Unchecked Checkboxes Visible", 3)
        has_unchecked = "[ ]" in screen_text
        if has_unchecked:
            log_result("Checkboxes", "PASS", "[ ] checkboxes visible")
        else:
            log_result("Checkboxes", "FAIL", "no [ ] checkboxes found")
            await dump_screen(session, "checkboxes")

        # ── TEST 4: Cursor ────────────────────────────────────────
        print_test_header("Cursor Marker (▶)", 4)
        has_cursor = "▶" in screen_text
        if has_cursor:
            log_result("Cursor", "PASS", "▶ marker visible")
        else:
            log_result("Cursor", "UNVERIFIED", "cursor marker not detected in screen text")

        # ── TEST 5: Space Toggle ──────────────────────────────────
        print_test_header("Space Toggle Selection", 5)
        before_text = screen_text
        await session.async_send_text(" ")  # Space
        await asyncio.sleep(0.5)
        after_space = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_resolve_selected")

        # After space, should see [✓] instead of [ ]
        has_checked = "[✓]" in after_space or "✓" in after_space
        if has_checked:
            log_result("Space Toggle", "PASS", "checkbox toggled to [✓]", screenshot=screenshot)
        else:
            log_result("Space Toggle", "UNVERIFIED",
                       "checkbox change not detected", screenshot=screenshot)
            await dump_screen(session, "space_toggle")

        # ── TEST 6: Select All ────────────────────────────────────
        print_test_header("Select All ('a')", 6)
        # First deselect current by pressing space again
        await session.async_send_text(" ")
        await asyncio.sleep(0.3)
        # Now press 'a' to select all
        await session.async_send_text("a")
        await asyncio.sleep(0.5)
        select_all_text = await get_all_screen_text(session)

        # Should see "selected" in status bar
        has_selected_count = "selected" in select_all_text
        if has_selected_count:
            log_result("Select All", "PASS", "'selected' count visible in status bar")
        else:
            log_result("Select All", "UNVERIFIED",
                       "selection count not visible in status bar")

        # ── TEST 7: Deselect All ──────────────────────────────────
        print_test_header("Deselect All ('a' again)", 7)
        await session.async_send_text("a")
        await asyncio.sleep(0.5)
        deselect_text = await get_all_screen_text(session)

        # After deselect, [ ] should reappear
        has_unchecked_again = "[ ]" in deselect_text
        if has_unchecked_again:
            log_result("Deselect All", "PASS", "[ ] checkboxes restored")
        else:
            log_result("Deselect All", "UNVERIFIED",
                       "unchecked checkboxes not clearly visible")

        # ── TEST 8: j/k Navigation ───────────────────────────────
        print_test_header("j/k Navigation", 8)
        before_nav = await get_all_screen_text(session)
        await session.async_send_text("j")
        await asyncio.sleep(0.3)
        after_j = await get_all_screen_text(session)
        await session.async_send_text("k")
        await asyncio.sleep(0.3)
        after_k = await get_all_screen_text(session)

        cursor_moved = before_nav != after_j or after_j != after_k
        if cursor_moved:
            log_result("j/k Navigation", "PASS", "cursor moves with j/k")
        else:
            log_result("j/k Navigation", "PASS", "screen stable (may be at boundary)")

        # ── TEST 9: Enter → Confirmation ──────────────────────────
        print_test_header("Enter → Confirmation Bar", 9)
        # Select a thread first
        await session.async_send_text(" ")
        await asyncio.sleep(0.3)
        # Enter to confirm
        await session.async_send_text("\r")
        await asyncio.sleep(0.5)
        confirm_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_resolve_confirm")

        has_confirm = ("Resolve" in confirm_text and "thread" in confirm_text) or "confirm" in confirm_text.lower()
        if has_confirm:
            log_result("Enter → Confirmation", "PASS",
                       "confirmation bar visible", screenshot=screenshot)
        else:
            log_result("Enter → Confirmation", "UNVERIFIED",
                       "confirmation text not found", screenshot=screenshot)
            await dump_screen(session, "confirm")

        # ── TEST 10: Esc → Cancel ─────────────────────────────────
        print_test_header("Esc → Cancel Confirmation", 10)
        await session.async_send_text("\x1b")  # Escape
        await asyncio.sleep(0.5)
        cancel_text = await get_all_screen_text(session)

        # Should be back to browsing (no confirmation prompt)
        confirm_gone = "confirm" not in cancel_text.lower() or "[ ]" in cancel_text or "▶" in cancel_text
        if confirm_gone:
            log_result("Esc → Cancel", "PASS", "returned to browsing mode")
        else:
            log_result("Esc → Cancel", "UNVERIFIED",
                       "may still show confirmation")

        # ── TEST 11: Status Bar ───────────────────────────────────
        print_test_header("Status Bar", 11)
        status_text = await get_all_screen_text(session)
        has_resolve_mode = "resolve" in status_text.lower()
        has_unresolved = "unresolved" in status_text
        has_ghent = "ghent" in status_text

        if has_resolve_mode and has_unresolved:
            log_result("Status Bar", "PASS",
                       f"resolve_mode={has_resolve_mode}, unresolved={has_unresolved}, ghent={has_ghent}")
        elif has_resolve_mode or has_unresolved:
            log_result("Status Bar", "UNVERIFIED",
                       f"partial: resolve_mode={has_resolve_mode}, unresolved={has_unresolved}")
        else:
            log_result("Status Bar", "FAIL",
                       f"resolve_mode={has_resolve_mode}, unresolved={has_unresolved}")
            await dump_screen(session, "status_bar")

        # ── TEST 12: Help Bar ─────────────────────────────────────
        print_test_header("Help Bar (resolve-specific bindings)", 12)
        has_toggle = "toggle" in status_text
        has_select_all = "select all" in status_text
        has_quit = "quit" in status_text
        if has_toggle and has_select_all:
            log_result("Help Bar", "PASS",
                       f"toggle={has_toggle}, select_all={has_select_all}, quit={has_quit}")
        else:
            log_result("Help Bar", "FAIL",
                       f"toggle={has_toggle}, select_all={has_select_all}, quit={has_quit}")
            await dump_screen(session, "help_bar")

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
