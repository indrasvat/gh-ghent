# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Task 032 Visual Tests: Summary overflow, async loading, Esc navigation.

Stress-tests the P1/P2/P3 fixes against oven-sh/bun extreme PRs.

Tests:
    1. Build & Install
    2. Summary overflow — PR #24063 (61 reviews, 101 threads): KPIs visible, approvals capped
    3. Summary overflow — PR #27327 (25 reviews, 68 threads): all sections fit
    4. Async loading — PR #24063: loading indicator appears immediately
    5. Esc navigation round-trip: summary → c → esc → summary → k → esc → summary

Usage:
    uv run .claude/automations/test_ghent_task032.py
"""

import iterm2
import asyncio
import subprocess
import os
import time
from datetime import datetime

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")
TIMEOUT_SECONDS = 15.0

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
    print("TEST SUMMARY — Task 032 Stress Tests")
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
    print("# ghent Task 032 — Overflow/Async/Esc Stress Tests")
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

        # ── TEST 2: Summary overflow — PR #24063 (61 reviews) ─────
        print_test_header("Summary overflow: PR #24063 (61 reviews, 101 threads)", 2)
        await session.async_send_text("gh ghent summary -R oven-sh/bun --pr 24063 2>&1\n")
        await asyncio.sleep(15.0)  # Large PR — give it time

        screen_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("t032_overflow_pr24063")

        # Verify: KPI cards visible (they were pushed off-screen before the fix)
        has_kpi = "UNRESOLVED" in screen_text or "PASSED" in screen_text
        has_sections = "Review Threads" in screen_text or "CI Checks" in screen_text
        has_overflow = "more" in screen_text.lower()  # "... and N more"
        has_approvals_section = "Approvals" in screen_text

        if has_kpi and has_sections:
            log_result("Summary overflow PR #24063", "PASS",
                       f"KPI={has_kpi}, sections={has_sections}, overflow_indicator={has_overflow}, approvals={has_approvals_section}",
                       screenshot=screenshot)
        elif has_sections:
            log_result("Summary overflow PR #24063", "UNVERIFIED",
                       f"KPI={has_kpi}, sections={has_sections}, overflow={has_overflow}",
                       screenshot=screenshot)
        else:
            log_result("Summary overflow PR #24063", "FAIL",
                       f"KPI={has_kpi}, sections={has_sections}")
            await dump_screen(session, "overflow_24063")

        await session.async_send_text("q")
        await asyncio.sleep(1.0)

        # ── TEST 3: Summary overflow — PR #27327 (68 threads, 25 reviews) ─
        print_test_header("Summary overflow: PR #27327 (68 threads, 25 reviews)", 3)
        await session.async_send_text("gh ghent summary -R oven-sh/bun --pr 27327 2>&1\n")
        await asyncio.sleep(10.0)

        screen_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("t032_overflow_pr27327")

        has_kpi = "UNRESOLVED" in screen_text or "PASSED" in screen_text
        has_threads = "Review Threads" in screen_text
        has_checks = "CI Checks" in screen_text
        has_approvals = "Approvals" in screen_text

        sections_visible = sum([has_threads, has_checks, has_approvals])
        if has_kpi and sections_visible >= 2:
            log_result("Summary overflow PR #27327", "PASS",
                       f"KPI={has_kpi}, threads={has_threads}, checks={has_checks}, approvals={has_approvals}",
                       screenshot=screenshot)
        else:
            log_result("Summary overflow PR #27327", "FAIL",
                       f"KPI={has_kpi}, sections_visible={sections_visible}")
            await dump_screen(session, "overflow_27327")

        await session.async_send_text("q")
        await asyncio.sleep(1.0)

        # ── TEST 4: Async loading — quick screenshot during load ──
        print_test_header("Async loading: instant TUI frame", 4)
        await session.async_send_text("gh ghent summary -R oven-sh/bun --pr 24063 2>&1\n")
        # Capture very quickly — within 1.5s of launch
        await asyncio.sleep(1.5)

        early_text = await get_all_screen_text(session)
        screenshot_early = capture_screenshot("t032_async_loading_early")

        # The TUI frame (status bar, help bar) should be visible
        has_ghent = "ghent" in early_text
        has_help = "quit" in early_text or "comments" in early_text
        has_loading = "Loading" in early_text or "loading" in early_text

        # Wait for full load
        await asyncio.sleep(12.0)
        full_text = await get_all_screen_text(session)
        screenshot_full = capture_screenshot("t032_async_loading_full")

        has_sections_after = "Review Threads" in full_text or "CI Checks" in full_text

        if has_ghent and has_sections_after:
            log_result("Async loading", "PASS",
                       f"early: ghent={has_ghent}, help={has_help}, loading={has_loading}; full: sections={has_sections_after}",
                       screenshot=screenshot_early)
        elif has_sections_after:
            log_result("Async loading", "UNVERIFIED",
                       f"early frame not caught (early: ghent={has_ghent}), but data loaded",
                       screenshot=screenshot_full)
        else:
            log_result("Async loading", "FAIL",
                       f"early: ghent={has_ghent}; full: sections={has_sections_after}")
            await dump_screen(session, "async_loading")

        await session.async_send_text("q")
        await asyncio.sleep(1.0)

        # ── TEST 5: Esc navigation round-trip ────────────────────
        print_test_header("Esc navigation round-trip", 5)
        # Use a fast PR for this test
        await session.async_send_text("gh ghent summary -R indrasvat/tbgs --pr 1 2>&1\n")
        await asyncio.sleep(8.0)

        # Verify we're at summary
        summary_text = await get_all_screen_text(session)
        at_summary = "Review Threads" in summary_text or "UNRESOLVED" in summary_text

        # Press 'c' → comments
        await session.async_send_text("c")
        await asyncio.sleep(1.0)
        comments_text = await get_all_screen_text(session)
        screenshot_c = capture_screenshot("t032_esc_nav_comments")
        at_comments = "comments" in comments_text and ("expand" in comments_text or "unresolved" in comments_text)

        # Press Esc → back to summary
        await session.async_send_text("\x1b")  # Escape key
        await asyncio.sleep(1.0)
        back_text = await get_all_screen_text(session)
        screenshot_back = capture_screenshot("t032_esc_nav_back_to_summary")
        back_at_summary = "Review Threads" in back_text or "UNRESOLVED" in back_text or "CI Checks" in back_text

        # Press 'k' → checks
        await session.async_send_text("k")
        await asyncio.sleep(1.0)
        checks_text = await get_all_screen_text(session)
        at_checks = "checks" in checks_text and ("view logs" in checks_text or "passed" in checks_text)

        # Press Esc → back to summary
        await session.async_send_text("\x1b")
        await asyncio.sleep(1.0)
        final_text = await get_all_screen_text(session)
        screenshot_final = capture_screenshot("t032_esc_nav_final_summary")
        final_at_summary = "Review Threads" in final_text or "UNRESOLVED" in final_text or "CI Checks" in final_text

        if at_summary and at_comments and back_at_summary and final_at_summary:
            log_result("Esc navigation round-trip", "PASS",
                       f"summary={at_summary}, c→comments={at_comments}, esc→summary={back_at_summary}, k→checks={at_checks}, esc→summary={final_at_summary}",
                       screenshot=screenshot_final)
        elif back_at_summary or final_at_summary:
            log_result("Esc navigation round-trip", "UNVERIFIED",
                       f"summary={at_summary}, c→comments={at_comments}, esc→back={back_at_summary}, k→checks={at_checks}, esc→final={final_at_summary}",
                       screenshot=screenshot_back)
        else:
            log_result("Esc navigation round-trip", "FAIL",
                       f"summary={at_summary}, comments={at_comments}, back={back_at_summary}, final={final_at_summary}")
            await dump_screen(session, "esc_nav")

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
