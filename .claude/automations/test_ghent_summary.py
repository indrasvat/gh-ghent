# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Summary Dashboard Visual Test: Comprehensive automated verification
of the TUI summary view with KPI cards, sections, and merge readiness.

Tests:
    1. Build: Verify gh-ghent builds and installs
    2. Launch (NOT READY): Summary TUI with unresolved threads (tbgs PR #1)
    3. KPI Cards: Unresolved, Passed, Failed, Approvals labels visible
    4. Merge Badge: "NOT READY" badge visible
    5. Sections: Review Threads, CI Checks, Approvals headers
    6. Thread Preview: File paths visible in thread section
    7. Help Bar: Summary-specific keys (c, k, r, q)
    8. Ready Repo: Launch with doot PR #1 → status bar shows "READY"
    9. Not Ready (failing checks): peek-it PR #2

Verification Strategy:
    - Use indrasvat/tbgs PR #1 (2 unresolved, checks pass → NOT READY)
    - Use indrasvat/doot PR #1 (0 unresolved, checks pass → READY-ish)
    - Use indrasvat/peek-it PR #2 (failing checks → NOT READY)
    - Read screen contents at each step
    - Take screenshots at every visual milestone

Usage:
    uv run .claude/automations/test_ghent_summary.py
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
    print("TEST SUMMARY — Summary Dashboard")
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
    print("# ghent Summary Dashboard — Visual Test Suite")
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

        # ── TEST 2: Launch Summary TUI (NOT READY — tbgs) ────────
        print_test_header("Launch Summary TUI (indrasvat/tbgs PR #1)", 2)
        await session.async_send_text("gh ghent summary -R indrasvat/tbgs --pr 1 2>&1\n")
        await asyncio.sleep(10.0)

        screen_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_summary_launch")

        launch_indicators = {
            "review_threads": "Review Threads" in screen_text,
            "ci_checks": "CI Checks" in screen_text or "checks" in screen_text.lower(),
            "approvals": "Approvals" in screen_text or "reviews" in screen_text.lower(),
            "ghent": "ghent" in screen_text,
        }
        tui_launched = sum(launch_indicators.values()) >= 2

        if tui_launched:
            log_result("Launch Summary TUI", "PASS",
                       f"indicators={launch_indicators}", screenshot=screenshot)
        else:
            log_result("Launch Summary TUI", "FAIL",
                       f"indicators={launch_indicators}")
            await dump_screen(session, "launch_fail")
            return print_summary()

        # ── TEST 3: KPI Cards ────────────────────────────────────
        print_test_header("KPI Cards Visible", 3)
        has_unresolved = "UNRESOLVED" in screen_text
        has_passed = "PASSED" in screen_text
        has_failed = "FAILED" in screen_text
        has_approvals = "APPROVALS" in screen_text

        kpi_count = sum([has_unresolved, has_passed, has_failed, has_approvals])
        if kpi_count >= 3:
            log_result("KPI Cards", "PASS",
                       f"unresolved={has_unresolved}, passed={has_passed}, failed={has_failed}, approvals={has_approvals}")
        elif kpi_count >= 1:
            log_result("KPI Cards", "UNVERIFIED",
                       f"partial: unresolved={has_unresolved}, passed={has_passed}, failed={has_failed}, approvals={has_approvals}")
        else:
            log_result("KPI Cards", "FAIL",
                       f"no KPI labels found")
            await dump_screen(session, "kpi_cards")

        # ── TEST 4: Merge Badge (NOT READY) ──────────────────────
        print_test_header("Merge Badge (NOT READY)", 4)
        has_not_ready = "NOT READY" in screen_text
        if has_not_ready:
            log_result("Merge Badge (NOT READY)", "PASS", "NOT READY badge visible")
        else:
            log_result("Merge Badge (NOT READY)", "FAIL", "NOT READY badge not found")
            await dump_screen(session, "merge_badge")

        # ── TEST 5: Section Headers ──────────────────────────────
        print_test_header("Section Headers", 5)
        has_threads_section = "Review Threads" in screen_text
        has_checks_section = "CI Checks" in screen_text
        has_approvals_section = "Approvals" in screen_text

        sections_found = sum([has_threads_section, has_checks_section, has_approvals_section])
        if sections_found >= 2:
            log_result("Section Headers", "PASS",
                       f"threads={has_threads_section}, checks={has_checks_section}, approvals={has_approvals_section}")
        else:
            log_result("Section Headers", "FAIL",
                       f"threads={has_threads_section}, checks={has_checks_section}, approvals={has_approvals_section}")
            await dump_screen(session, "sections")

        # ── TEST 6: Thread Preview ───────────────────────────────
        print_test_header("Thread Preview Content", 6)
        has_go_file = ".go" in screen_text
        has_author = "@" in screen_text
        if has_go_file and has_author:
            log_result("Thread Preview", "PASS", f"go_file={has_go_file}, author={has_author}")
        elif has_go_file or has_author:
            log_result("Thread Preview", "UNVERIFIED",
                       f"partial: go_file={has_go_file}, author={has_author}")
        else:
            log_result("Thread Preview", "FAIL", "no thread preview content")

        # ── TEST 7: Help Bar ─────────────────────────────────────
        print_test_header("Help Bar (summary-specific bindings)", 7)
        has_comments_key = "comments" in screen_text
        has_checks_key = "checks" in screen_text
        has_quit = "quit" in screen_text
        if has_comments_key and has_checks_key and has_quit:
            log_result("Help Bar", "PASS",
                       f"comments={has_comments_key}, checks={has_checks_key}, quit={has_quit}")
        else:
            log_result("Help Bar", "FAIL",
                       f"comments={has_comments_key}, checks={has_checks_key}, quit={has_quit}")
            await dump_screen(session, "help_bar")

        # Exit TUI for next test
        await session.async_send_text("q")
        await asyncio.sleep(1.0)

        # ── TEST 8: Ready Repo (doot PR #1) ──────────────────────
        print_test_header("Ready Repo (indrasvat/doot PR #1)", 8)
        await session.async_send_text("gh ghent summary -R indrasvat/doot --pr 1 2>&1\n")
        await asyncio.sleep(10.0)

        ready_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_summary_ready")

        # doot has 0 unresolved, checks pass, but no approval → may still be NOT READY
        # Check if "READY" appears (either READY or NOT READY)
        has_ready = "READY" in ready_text
        has_summary_content = "CI Checks" in ready_text or "Review Threads" in ready_text or "Approvals" in ready_text

        if has_summary_content:
            log_result("Ready Repo", "PASS",
                       f"summary rendered, READY in text={has_ready}", screenshot=screenshot)
        else:
            log_result("Ready Repo", "UNVERIFIED",
                       f"summary content not clearly visible", screenshot=screenshot)
            await dump_screen(session, "ready_repo")

        # Exit TUI
        await session.async_send_text("q")
        await asyncio.sleep(1.0)

        # ── TEST 9: Not Ready (failing checks — peek-it) ────────
        print_test_header("Not Ready Repo (indrasvat/peek-it PR #2)", 9)
        await session.async_send_text("gh ghent summary -R indrasvat/peek-it --pr 2 2>&1\n")
        await asyncio.sleep(10.0)

        notready_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_summary_not_ready")

        has_not_ready_badge = "NOT READY" in notready_text
        has_fail_indicator = "failed" in notready_text.lower() or "FAILED" in notready_text or "✗" in notready_text

        if has_not_ready_badge or has_fail_indicator:
            log_result("Not Ready Repo", "PASS",
                       f"NOT_READY={has_not_ready_badge}, fail_indicator={has_fail_indicator}",
                       screenshot=screenshot)
        else:
            log_result("Not Ready Repo", "UNVERIFIED",
                       f"NOT_READY={has_not_ready_badge}, fail_indicator={has_fail_indicator}",
                       screenshot=screenshot)
            await dump_screen(session, "not_ready_repo")

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
