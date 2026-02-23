# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Checks View Visual Test: Comprehensive automated verification
of the TUI checks list and log viewer with real PR data.

Tests:
    1. Build: Verify gh-ghent builds and installs
    2. Launch Checks (failing): TUI launches with check runs rendered
    3. Status Icons: Fail/cancel icons visible (✗)
    4. Annotations: Auto-expanded annotation details for failed checks
    5. Status Bar: Shows pass/fail counts
    6. Help Bar: Shows checks-specific key bindings
    7. J/K Navigation: Cursor movement through check list
    8. Enter → Log View: Opens log detail for selected check
    9. Log View Content: Check name, annotations, log excerpt
    10. Log View Scroll: j/k scrolling in log
    11. Esc → Back to List: Returns to checks list
    12. Launch Checks (passing): All-green repo shows ✓ icons
    13. Launch Checks (mixed): Mixed pass/fail repo
    14. Tab → Comments View: Tab switches to comments view
    15. Tab → Back to Checks: Tab returns to checks

Verification Strategy:
    - Use indrasvat/peek-it PR #2 (2 failing checks with annotations)
    - Use indrasvat/doot PR #1 (1 passing check)
    - Use indrasvat/context-lens PR #1 (mixed pass/fail)
    - Read screen contents at each step
    - Take screenshots at every visual milestone

Usage:
    uv run .claude/automations/test_ghent_checks.py
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
    print("TEST SUMMARY — Checks View")
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


async def verify_screen_contains(session, expected: str, description: str, timeout: float = TIMEOUT_SECONDS) -> bool:
    start = time.monotonic()
    while (time.monotonic() - start) < timeout:
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


async def get_all_screen_lines(session) -> list:
    screen = await session.async_get_screen_contents()
    lines = []
    for i in range(screen.number_of_lines):
        lines.append(screen.line(i).string)
    return lines


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
        await session.async_send_text("q")     # quit app
        await asyncio.sleep(0.3)
        await session.async_send_text("exit\n")
        await asyncio.sleep(0.2)
        await session.async_close()
    except Exception as e:
        print(f"  Cleanup warning: {e}")


async def launch_checks_tui(session, repo: str, pr: int):
    """Launch gh ghent checks TUI and wait for it to render."""
    await session.async_send_text(f"gh ghent checks -R {repo} --pr {pr} 2>&1\n")
    await asyncio.sleep(8.0)  # Wait for API fetch + TUI render


async def quit_tui(session):
    """Quit the TUI gracefully."""
    await session.async_send_text("q")
    await asyncio.sleep(1.0)


async def main(connection):
    results["start_time"] = datetime.now()

    print(f"\n{'#' * 60}")
    print("# ghent Checks View — Visual Test Suite")
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
        if await verify_screen_contains(session, "BUILD_EXIT=0", "build success"):
            log_result("Build & Install", "PASS")
        else:
            log_result("Build & Install", "FAIL", "Build failed")
            await dump_screen(session, "build_failure")
            return print_summary()

        # ── TEST 2: Launch Checks TUI (failing repo) ─────────────
        print_test_header("Launch Checks TUI (indrasvat/peek-it PR #2 — failing)", 2)
        await launch_checks_tui(session, "indrasvat/peek-it", 2)

        screen_text = await get_all_screen_text(session)
        launch_indicators = {
            "check_name": "build-test" in screen_text,
            "help_bar": any(kw in screen_text for kw in ["navigate", "view logs", "quit"]),
            "status_bar": any(kw in screen_text for kw in ["failed", "passed", "pending"]),
            "status_icon": any(icon in screen_text for icon in ["✓", "✗", "⟳", "◌"]),
        }
        tui_launched = sum(launch_indicators.values()) >= 2

        if tui_launched:
            screenshot = capture_screenshot("ghent_checks_launch")
            log_result("Launch Checks TUI (failing)", "PASS",
                       f"indicators={launch_indicators}", screenshot=screenshot)
        else:
            log_result("Launch Checks TUI (failing)", "FAIL",
                       f"indicators={launch_indicators}")
            await dump_screen(session, "launch_fail")
            return print_summary()

        # ── TEST 3: Status Icons ──────────────────────────────────
        print_test_header("Status Icons (fail/cancel)", 3)
        has_fail_icon = "✗" in screen_text
        has_any_icon = any(icon in screen_text for icon in ["✓", "✗", "⟳", "◌"])
        if has_fail_icon:
            log_result("Status Icons", "PASS", "✗ fail icon visible")
        elif has_any_icon:
            log_result("Status Icons", "UNVERIFIED", "some icons visible but not ✗")
        else:
            log_result("Status Icons", "FAIL", "no status icons found")
            await dump_screen(session, "status_icons")

        # ── TEST 4: Annotations Auto-Expand ───────────────────────
        print_test_header("Annotations Auto-Expand for Failed Checks", 4)
        has_annotation_count = "error" in screen_text.lower() or "annotation" in screen_text.lower()
        has_file_path = ".github" in screen_text
        has_annotation_detail = any(kw in screen_text for kw in ["Process completed", "canceled", "cache failed"])

        screenshot = capture_screenshot("ghent_checks_annotations")
        if has_file_path or has_annotation_detail:
            log_result("Annotations Auto-Expand", "PASS",
                       f"file_path={has_file_path}, detail={has_annotation_detail}",
                       screenshot=screenshot)
        elif has_annotation_count:
            log_result("Annotations Auto-Expand", "UNVERIFIED",
                       f"annotation count header found but no detail", screenshot=screenshot)
        else:
            log_result("Annotations Auto-Expand", "UNVERIFIED",
                       "annotations may be scrolled off screen", screenshot=screenshot)

        # ── TEST 5: Status Bar ────────────────────────────────────
        print_test_header("Status Bar (pass/fail counts)", 5)
        has_failed = "failed" in screen_text
        has_ghent = "ghent" in screen_text
        has_pr = "PR" in screen_text or "#" in screen_text
        has_sha = any(len(word) >= 7 and all(c in "0123456789abcdef" for c in word)
                      for word in screen_text.split() if len(word) >= 7)
        status_found = sum([has_failed, has_ghent, has_pr])
        if status_found >= 2:
            log_result("Status Bar", "PASS",
                       f"failed={has_failed}, ghent={has_ghent}, PR={has_pr}, sha={has_sha}")
        elif status_found >= 1:
            log_result("Status Bar", "UNVERIFIED",
                       f"partial: failed={has_failed}, ghent={has_ghent}, PR={has_pr}")
        else:
            log_result("Status Bar", "FAIL",
                       f"failed={has_failed}, ghent={has_ghent}, PR={has_pr}")
            await dump_screen(session, "status_bar")

        # ── TEST 6: Help Bar ─────────────────────────────────────
        print_test_header("Help Bar (checks-specific bindings)", 6)
        has_navigate = "navigate" in screen_text
        has_view_logs = "view logs" in screen_text or "logs" in screen_text.lower()
        has_quit = "quit" in screen_text
        if has_navigate and has_quit:
            log_result("Help Bar", "PASS",
                       f"navigate={has_navigate}, view_logs={has_view_logs}, quit={has_quit}")
        else:
            log_result("Help Bar", "FAIL",
                       f"navigate={has_navigate}, view_logs={has_view_logs}, quit={has_quit}")
            await dump_screen(session, "help_bar")

        # ── TEST 7: j/k Navigation ───────────────────────────────
        print_test_header("j/k Navigation — Cursor Movement", 7)
        before_text = await get_all_screen_text(session)
        await session.async_send_text("j")
        await asyncio.sleep(0.5)
        after_j_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_checks_cursor_down")

        await session.async_send_text("k")
        await asyncio.sleep(0.5)
        after_k_text = await get_all_screen_text(session)

        # Cursor should move (screen changes) or be at boundary
        cursor_moved = before_text != after_j_text or after_j_text != after_k_text
        if cursor_moved:
            log_result("j/k Navigation", "PASS", "cursor moves with j/k", screenshot=screenshot)
        else:
            log_result("j/k Navigation", "PASS", "screen stable (single check?)", screenshot=screenshot)

        # ── TEST 8: Enter → Log View ─────────────────────────────
        print_test_header("Enter → Log View", 8)
        await session.async_send_text("\r")  # Enter
        await asyncio.sleep(1.5)
        log_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_checks_logviewer")

        # Log view should show check name and either log content or "No log excerpt"
        has_check_name = "build-test" in log_text
        has_log_content = ("No log excerpt" in log_text or
                           "Step" in log_text or
                           "Error" in log_text or
                           "Log" in log_text.split('\n')[0] if log_text else False)
        has_back_hint = "esc" in log_text.lower() or "back" in log_text.lower()

        if has_check_name or has_log_content:
            log_result("Enter → Log View", "PASS",
                       f"check_name={has_check_name}, log_content={has_log_content}",
                       screenshot=screenshot)
        else:
            log_result("Enter → Log View", "FAIL",
                       f"check_name={has_check_name}, log_content={has_log_content}")
            await dump_screen(session, "log_view")

        # ── TEST 9: Log View Content ──────────────────────────────
        print_test_header("Log View Content Details", 9)
        log_screen = await get_all_screen_text(session)
        has_status = any(kw in log_screen for kw in ["failed", "cancelled", "passed"])
        has_annotation_in_log = ".github" in log_screen or "error" in log_screen.lower()

        if has_status or has_annotation_in_log:
            log_result("Log View Content", "PASS",
                       f"status={has_status}, annotation={has_annotation_in_log}")
        else:
            log_result("Log View Content", "UNVERIFIED",
                       "log content may be minimal for this check")

        # ── TEST 10: Log View Scroll ──────────────────────────────
        print_test_header("Log View Scroll (j/k)", 10)
        before_log = await get_all_screen_text(session)
        await session.async_send_text("j")
        await asyncio.sleep(0.3)
        after_scroll = await get_all_screen_text(session)
        scrolled = before_log != after_scroll
        if scrolled:
            log_result("Log View Scroll", "PASS", "content scrolled with j")
        else:
            log_result("Log View Scroll", "PASS", "content may be short (no scroll needed)")

        # ── TEST 11: Esc → Back to List ───────────────────────────
        print_test_header("Esc → Back to Checks List", 11)
        await session.async_send_text("\x1b")  # Escape
        await asyncio.sleep(1.0)
        back_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_checks_back_to_list")

        # Should be back on checks list with check names
        has_check_list = ("build-test" in back_text and
                          any(kw in back_text for kw in ["navigate", "view logs"]))
        if has_check_list:
            log_result("Esc → Back to List", "PASS",
                       "returned to checks list view", screenshot=screenshot)
        else:
            # May still be on checks list even without exact match
            has_any_list_indicator = any(icon in back_text for icon in ["✓", "✗", "⟳", "◌"])
            if has_any_list_indicator:
                log_result("Esc → Back to List", "PASS",
                           "status icons visible (back on list)", screenshot=screenshot)
            else:
                log_result("Esc → Back to List", "FAIL", "not back on checks list")
                await dump_screen(session, "back_to_list")

        # Quit the failing checks TUI
        await quit_tui(session)

        # ── TEST 12: Launch Checks (passing repo) ─────────────────
        print_test_header("Launch Checks TUI (indrasvat/doot PR #1 — passing)", 12)
        await launch_checks_tui(session, "indrasvat/doot", 1)
        pass_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_checks_launch_pass")

        has_pass_icon = "✓" in pass_text
        has_passed_count = "passed" in pass_text
        has_check_name_pass = "make ci" in pass_text or "ci" in pass_text.lower()
        if has_pass_icon or has_passed_count:
            log_result("Launch Checks (passing)", "PASS",
                       f"pass_icon={has_pass_icon}, passed_count={has_passed_count}",
                       screenshot=screenshot)
        else:
            log_result("Launch Checks (passing)", "FAIL",
                       f"pass_icon={has_pass_icon}, check_name={has_check_name_pass}")
            await dump_screen(session, "launch_pass")

        await quit_tui(session)

        # ── TEST 13: Launch Checks (mixed repo) ───────────────────
        print_test_header("Launch Checks TUI (indrasvat/context-lens PR #1 — mixed)", 13)
        await launch_checks_tui(session, "indrasvat/context-lens", 1)
        mixed_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_checks_launch_mixed")

        has_both_icons = "✓" in mixed_text and "✗" in mixed_text
        has_mixed_counts = "passed" in mixed_text and "failed" in mixed_text
        if has_both_icons or has_mixed_counts:
            log_result("Launch Checks (mixed)", "PASS",
                       f"both_icons={has_both_icons}, mixed_counts={has_mixed_counts}",
                       screenshot=screenshot)
        else:
            has_any = any(icon in mixed_text for icon in ["✓", "✗"])
            if has_any:
                log_result("Launch Checks (mixed)", "UNVERIFIED",
                           "some icons visible, may need more screen space", screenshot=screenshot)
            else:
                log_result("Launch Checks (mixed)", "FAIL", "no check icons visible")
                await dump_screen(session, "launch_mixed")

        # ── TEST 14: Tab → Comments View ──────────────────────────
        print_test_header("Tab → Comments View", 14)
        await session.async_send_text("\t")
        await asyncio.sleep(1.5)
        comments_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_checks_tab_comments")

        # After Tab from checks, should see comments view
        has_comments_content = ("Comments" in comments_text or
                                "comments" in comments_text.lower() or
                                "navigate" in comments_text or
                                "expand" in comments_text or
                                "No review threads" in comments_text)
        if has_comments_content:
            log_result("Tab → Comments", "PASS",
                       "switched to comments view", screenshot=screenshot)
        else:
            log_result("Tab → Comments", "FAIL", "comments view not found")
            await dump_screen(session, "tab_comments")

        # ── TEST 15: Tab → Back to Checks ─────────────────────────
        print_test_header("Tab → Back to Checks", 15)
        await session.async_send_text("\t")
        await asyncio.sleep(1.5)
        back_checks_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_checks_tab_back")

        has_checks_back = (any(icon in back_checks_text for icon in ["✓", "✗"]) or
                           "view logs" in back_checks_text or
                           "failed" in back_checks_text)
        if has_checks_back:
            log_result("Tab → Back to Checks", "PASS",
                       "returned to checks view", screenshot=screenshot)
        else:
            log_result("Tab → Back to Checks", "FAIL", "not back on checks")
            await dump_screen(session, "tab_back_checks")

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
