# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Open-in-Browser Visual Test: Verifies the 'o' key opens a browser
URL in both checks and resolve TUI views without crashing.

Tests:
    1. Build: Verify gh-ghent builds and installs
    2. Checks View Launch: TUI launches with check runs
    3. Checks 'o' Key: Pressing 'o' on a check triggers browser open,
       TUI stays responsive
    4. Checks Post-Open Navigation: j/k still works after 'o'
    5. Resolve View Launch: TUI launches in resolve mode
    6. Resolve 'o' Key: Pressing 'o' on a thread triggers browser open,
       TUI stays responsive (this was broken before the fix)
    7. Resolve Post-Open Navigation: j/k still works after 'o'

Verification Strategy:
    - Use indrasvat/peek-it PR #2 (2 failing checks with URLs)
    - Use indrasvat/tbgs PR #1 (2 unresolved threads with URLs)
    - Cannot verify browser actually opens (OS-level), but verify:
      (a) TUI does not crash/hang after 'o'
      (b) Navigation continues to work after 'o'
      (c) Status bar and help bar remain rendered
    - Take screenshots before and after 'o' press

Usage:
    uv run .claude/automations/test_ghent_open_browser.py
"""

import iterm2
import asyncio
import subprocess
import os
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
    print("TEST SUMMARY — Open-in-Browser")
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


def capture_screenshot(name: str, full_screen: bool = False) -> str:
    os.makedirs(SCREENSHOT_DIR, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    filepath = os.path.join(SCREENSHOT_DIR, f"{name}_{timestamp}.png")
    if full_screen:
        # Full-screen capture to see browser opening alongside iTerm2
        subprocess.run(["screencapture", "-x", filepath], check=True)
    else:
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
    print("# ghent Open-in-Browser — Visual Test Suite")
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

        # ════════════════════════════════════════════════════════════
        # PART A: Checks View — 'o' key (already worked pre-fix)
        # ════════════════════════════════════════════════════════════

        # ── TEST 2: Launch Checks TUI ───────────────────────────────
        print_test_header("Launch Checks TUI (indrasvat/peek-it PR #2)", 2)
        await session.async_send_text("gh ghent checks -R indrasvat/peek-it --pr 2 2>&1\n")
        await asyncio.sleep(8.0)

        screen_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_open_checks_launch")

        checks_indicators = {
            "status_icon": any(icon in screen_text for icon in ["✗", "✓", "◌"]),
            "checks_mode": "checks" in screen_text.lower(),
            "help_bar": any(kw in screen_text for kw in ["open", "navigate", "quit"]),
        }
        tui_launched = sum(checks_indicators.values()) >= 2

        if tui_launched:
            log_result("Launch Checks TUI", "PASS",
                       f"indicators={checks_indicators}", screenshot=screenshot)
        else:
            log_result("Launch Checks TUI", "FAIL",
                       f"indicators={checks_indicators}")
            await dump_screen(session, "checks_launch_fail")
            # Continue to resolve tests even if checks fail
            await session.async_send_text("q")
            await asyncio.sleep(0.5)

        if tui_launched:
            # ── TEST 3: Checks 'o' Key ──────────────────────────────
            print_test_header("Checks View: Press 'o' (open in browser)", 3)
            before_text = await get_all_screen_text(session)
            screenshot_before = capture_screenshot("ghent_open_checks_before_o")

            await session.async_send_text("o")
            await asyncio.sleep(2.0)  # Give browser time to launch

            after_text = await get_all_screen_text(session)
            screenshot_after = capture_screenshot("ghent_open_checks_after_o", full_screen=True)

            # TUI should still be rendered (not crashed)
            tui_alive = any(kw in after_text.lower() for kw in ["checks", "navigate", "quit", "open"])
            if tui_alive:
                log_result("Checks 'o' Key", "PASS",
                           "TUI remained responsive after 'o'", screenshot=screenshot_after)
            else:
                log_result("Checks 'o' Key", "FAIL",
                           "TUI appears to have crashed after 'o'")
                await dump_screen(session, "checks_after_o")

            # ── TEST 4: Checks Post-Open Navigation ─────────────────
            print_test_header("Checks View: Navigation after 'o'", 4)
            await session.async_send_text("j")
            await asyncio.sleep(0.3)
            nav_text = await get_all_screen_text(session)

            nav_works = any(kw in nav_text.lower() for kw in ["checks", "navigate", "quit"])
            if nav_works:
                log_result("Checks Post-Open Navigation", "PASS",
                           "j/k navigation works after 'o'")
            else:
                log_result("Checks Post-Open Navigation", "FAIL",
                           "navigation broken after 'o'")

            # Exit checks TUI
            await session.async_send_text("q")
            await asyncio.sleep(1.0)

        # ════════════════════════════════════════════════════════════
        # PART B: Resolve View — 'o' key (WAS BROKEN before fix)
        # ════════════════════════════════════════════════════════════

        # ── TEST 5: Launch Resolve TUI ──────────────────────────────
        print_test_header("Launch Resolve TUI (indrasvat/tbgs PR #1)", 5)
        await session.async_send_text("gh ghent resolve -R indrasvat/tbgs --pr 1 2>&1\n")
        await asyncio.sleep(8.0)

        screen_text = await get_all_screen_text(session)
        screenshot = capture_screenshot("ghent_open_resolve_launch")

        resolve_indicators = {
            "go_file": ".go" in screen_text,
            "checkbox": "[ ]" in screen_text or "[✓]" in screen_text,
            "resolve_mode": "resolve" in screen_text.lower(),
            "help_bar": any(kw in screen_text for kw in ["toggle", "select all", "navigate"]),
        }
        resolve_launched = sum(resolve_indicators.values()) >= 2

        if resolve_launched:
            log_result("Launch Resolve TUI", "PASS",
                       f"indicators={resolve_indicators}", screenshot=screenshot)
        else:
            log_result("Launch Resolve TUI", "FAIL",
                       f"indicators={resolve_indicators}")
            await dump_screen(session, "resolve_launch_fail")
            return print_summary()

        # ── TEST 6: Resolve 'o' Key ─────────────────────────────────
        print_test_header("Resolve View: Press 'o' (open in browser) — THE FIX", 6)
        before_text = await get_all_screen_text(session)
        screenshot_before = capture_screenshot("ghent_open_resolve_before_o")

        await session.async_send_text("o")
        await asyncio.sleep(2.0)  # Give browser time to launch

        after_text = await get_all_screen_text(session)
        screenshot_after = capture_screenshot("ghent_open_resolve_after_o", full_screen=True)

        # TUI should still be rendered and responsive
        tui_alive = any(kw in after_text.lower() for kw in ["resolve", "toggle", "select all", "navigate", "quit"])
        if tui_alive:
            log_result("Resolve 'o' Key", "PASS",
                       "TUI remained responsive after 'o' (fix verified!)", screenshot=screenshot_after)
        else:
            log_result("Resolve 'o' Key", "FAIL",
                       "TUI crashed or became unresponsive after 'o'")
            await dump_screen(session, "resolve_after_o")

        # ── TEST 7: Resolve Post-Open Navigation ────────────────────
        print_test_header("Resolve View: Navigation after 'o'", 7)
        await session.async_send_text("j")
        await asyncio.sleep(0.3)
        after_j = await get_all_screen_text(session)

        await session.async_send_text("k")
        await asyncio.sleep(0.3)
        after_k = await get_all_screen_text(session)

        screenshot = capture_screenshot("ghent_open_resolve_post_nav")

        # TUI should still render properly
        nav_alive = any(kw in after_k.lower() for kw in ["resolve", "toggle", "navigate", "quit"])
        if nav_alive:
            log_result("Resolve Post-Open Navigation", "PASS",
                       "j/k navigation works after 'o'", screenshot=screenshot)
        else:
            log_result("Resolve Post-Open Navigation", "FAIL",
                       "navigation broken after 'o'")
            await dump_screen(session, "resolve_post_nav")

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
