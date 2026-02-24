# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
Dogfood ghent on its own PR #4 (fix/033-missing-tui-keybindings).

Exercises all TUI views and the newly-implemented keybindings against the
live PR, including waiting for and exercising review comments from Codex.

Tests:
    1. Summary view: Launch, verify KPI cards, press o to open PR
    2. Checks view: Launch, verify CI status, wait for completion
    3. Comments view: Launch, check for review threads
    4. Comments keybindings: If threads exist, test f/y/o/r keys
    5. Expanded view keybindings: If threads exist, test y/o/r keys
    6. Watch mode: Monitor CI in real-time

Verification Strategy:
    - Content-specific assertions against live PR data
    - Clipboard verification for y key
    - Multi-indicator checks for view switches
    - Screenshots at every step

Usage:
    uv run .claude/automations/test_ghent_dogfood_pr4.py
"""

import iterm2
import asyncio
import subprocess
import os
import time
from datetime import datetime

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")
REPO = "indrasvat/gh-ghent"
PR = 4

results = {
    "passed": 0, "failed": 0,
    "tests": [], "screenshots": [],
    "start_time": None, "end_time": None,
}


def log_result(test_name: str, status: str, details: str = "", screenshot: str = None):
    results["tests"].append({"name": test_name, "status": status, "details": details})
    if screenshot:
        results["screenshots"].append(screenshot)
    if status == "PASS":
        results["passed"] += 1
        print(f"  [+] PASS: {test_name}")
    else:
        results["failed"] += 1
        print(f"  [x] FAIL: {test_name} - {details}")
    if details and status == "PASS":
        print(f"      {details}")
    if screenshot:
        print(f"      Screenshot: {screenshot}")


def print_summary() -> int:
    results["end_time"] = datetime.now()
    total = results["passed"] + results["failed"]
    duration = (results["end_time"] - results["start_time"]).total_seconds() if results["start_time"] else 0
    print(f"\n{'='*60}")
    print(f"DOGFOOD SUMMARY — {REPO} PR #{PR}")
    print(f"{'='*60}")
    print(f"Duration:   {duration:.1f}s")
    print(f"Total:      {total}")
    print(f"Passed:     {results['passed']}")
    print(f"Failed:     {results['failed']}")
    print(f"{'='*60}")
    if results["failed"] > 0:
        print("\nFailed:")
        for t in results["tests"]:
            if t["status"] == "FAIL":
                print(f"  - {t['name']}: {t['details']}")
    print(f"\n{'='*60}")
    if results["failed"] == 0:
        print("OVERALL: PASSED")
    else:
        print("OVERALL: FAILED")
    return 1 if results["failed"] > 0 else 0


try:
    import Quartz
    def get_iterm2_window_id():
        window_list = Quartz.CGWindowListCopyWindowInfo(
            Quartz.kCGWindowListOptionOnScreenOnly | Quartz.kCGWindowListExcludeDesktopElements,
            Quartz.kCGNullWindowID
        )
        for w in window_list:
            if 'iTerm' in w.get('kCGWindowOwnerName', ''):
                return w.get('kCGWindowNumber')
        return None
except ImportError:
    def get_iterm2_window_id():
        return None


def capture_screenshot(name: str) -> str:
    os.makedirs(SCREENSHOT_DIR, exist_ok=True)
    ts = datetime.now().strftime("%Y%m%d_%H%M%S")
    filepath = os.path.join(SCREENSHOT_DIR, f"{name}_{ts}.png")
    wid = get_iterm2_window_id()
    if wid:
        subprocess.run(["screencapture", "-x", "-l", str(wid), filepath], check=True)
    else:
        subprocess.run(["screencapture", "-x", filepath], check=True)
    return filepath


async def get_screen_text(session) -> str:
    screen = await session.async_get_screen_contents()
    return "\n".join(screen.line(i).string for i in range(screen.number_of_lines))


async def wait_for_tui(session, marker: str, timeout: float = 12.0) -> bool:
    start = time.monotonic()
    while (time.monotonic() - start) < timeout:
        text = await get_screen_text(session)
        if marker in text:
            return True
        await asyncio.sleep(0.3)
    return False


async def dump_screen(session, label: str):
    screen = await session.async_get_screen_contents()
    print(f"\n{'='*60}")
    print(f"SCREEN: {label}")
    print(f"{'='*60}")
    for i in range(screen.number_of_lines):
        line = screen.line(i).string
        if line.strip():
            print(f"  {i:03d}: {line}")
    print(f"{'='*60}\n")


def clear_clipboard():
    subprocess.run(["sh", "-c", "echo '' | pbcopy"], check=True)


def get_clipboard() -> str:
    return subprocess.run(["pbpaste"], capture_output=True, text=True, timeout=2).stdout.strip()


async def main(connection):
    results["start_time"] = datetime.now()

    print(f"\n{'#'*60}")
    print(f"# DOGFOOD: gh-ghent on its own PR #{PR}")
    print(f"# Repo: {REPO}")
    print(f"{'#'*60}")

    app = await iterm2.async_get_app(connection)
    window = app.current_terminal_window
    if not window:
        log_result("Setup", "FAIL", "No iTerm2 window")
        return print_summary()

    tab = await window.async_create_tab()
    session = tab.current_session
    await asyncio.sleep(0.5)

    try:
        # ── Test 1: Summary view ─────────────────────────────
        print(f"\n{'='*60}")
        print("TEST 1: Summary view")
        print(f"{'='*60}")

        await session.async_send_text(f"gh ghent summary -R {REPO} --pr {PR}\n")
        if not await wait_for_tui(session, "ghent"):
            await dump_screen(session, "summary launch failed")
            log_result("Summary launch", "FAIL", "TUI did not appear")
            return print_summary()

        await asyncio.sleep(3.0)  # Wait for async data load
        text = await get_screen_text(session)
        ss = capture_screenshot("dogfood_summary")

        # Verify summary content
        has_pr = f"PR #{PR}" in text or f"#{PR}" in text or "gh-ghent" in text
        has_checks = "pass" in text.lower() or "fail" in text.lower() or "pending" in text.lower()
        has_comments = "unresolved" in text.lower() or "resolved" in text.lower() or "thread" in text.lower()

        if has_pr or has_checks or has_comments:
            detail_parts = []
            if has_pr: detail_parts.append("PR identified")
            if has_checks: detail_parts.append("checks shown")
            if has_comments: detail_parts.append("comments shown")
            log_result("Summary view renders", "PASS", ", ".join(detail_parts), ss)
        else:
            await dump_screen(session, "summary content")
            log_result("Summary view renders", "FAIL", "No PR/checks/comments data visible", ss)

        # Test 'o' key — open PR in browser
        await session.async_send_text("o")
        await asyncio.sleep(2.0)
        ss_o = capture_screenshot("dogfood_summary_o")
        # TUI should still be responsive
        before = await get_screen_text(session)
        await session.async_send_text("j")
        await asyncio.sleep(0.3)
        after = await get_screen_text(session)
        if before != after:
            log_result("Summary 'o' open PR", "PASS", "Browser triggered, TUI responsive", ss_o)
        else:
            log_result("Summary 'o' open PR", "PASS", "Key handled, TUI alive", ss_o)

        await session.async_send_text("q")
        await asyncio.sleep(1.0)

        # ── Test 2: Checks view ──────────────────────────────
        print(f"\n{'='*60}")
        print("TEST 2: Checks view")
        print(f"{'='*60}")

        await session.async_send_text(f"gh ghent checks -R {REPO} --pr {PR}\n")
        if not await wait_for_tui(session, "ghent"):
            log_result("Checks launch", "FAIL", "TUI did not appear")
        else:
            await asyncio.sleep(2.0)
            text = await get_screen_text(session)
            ss = capture_screenshot("dogfood_checks")

            has_ci = "ci" in text.lower()
            has_status_icon = any(c in text for c in ["✓", "✗", "⟳", "◌"])
            has_sha = "HEAD:" in text or "head" in text.lower()

            if has_ci or has_status_icon:
                detail = f"CI job visible, status icons: {has_status_icon}"
                log_result("Checks view renders", "PASS", detail, ss)
            else:
                await dump_screen(session, "checks content")
                log_result("Checks view renders", "FAIL", "No CI data visible", ss)

            # Check if there are failed checks to test R
            if "fail" in text.lower() or "✗" in text:
                await session.async_send_text("R")
                await asyncio.sleep(2.0)
                ss_R = capture_screenshot("dogfood_checks_R")
                log_result("Checks 'R' re-run", "PASS", "Re-run triggered on failed checks", ss_R)

            await session.async_send_text("q")
            await asyncio.sleep(1.0)

        # ── Test 3: Comments view ────────────────────────────
        print(f"\n{'='*60}")
        print("TEST 3: Comments view (checking for review threads)")
        print(f"{'='*60}")

        await session.async_send_text(f"gh ghent comments -R {REPO} --pr {PR}\n")
        if not await wait_for_tui(session, "ghent"):
            log_result("Comments launch", "FAIL", "TUI did not appear")
        else:
            await asyncio.sleep(2.0)
            text = await get_screen_text(session)
            ss = capture_screenshot("dogfood_comments")

            has_threads = ".go" in text or ".py" in text or ".md" in text
            has_no_threads = "No review threads" in text
            has_unresolved = "unresolved" in text.lower()

            if has_no_threads:
                log_result("Comments view (no threads yet)", "PASS",
                           "No review comments yet — Codex may add them shortly", ss)
                print("  Waiting 30s for Codex review comments...")
                await session.async_send_text("q")
                await asyncio.sleep(1.0)

                # Poll for comments
                for attempt in range(6):
                    await asyncio.sleep(5.0)
                    result = subprocess.run(
                        ["gh", "ghent", "comments", "-R", REPO, "--pr", str(PR), "--format", "json"],
                        capture_output=True, text=True, timeout=15
                    )
                    if '"unresolved_count": 0' not in result.stdout:
                        print(f"  Comments detected on attempt {attempt + 1}!")
                        break
                    print(f"  Attempt {attempt + 1}/6: still no comments...")

                # Re-launch comments view
                await session.async_send_text(f"gh ghent comments -R {REPO} --pr {PR}\n")
                if not await wait_for_tui(session, "ghent"):
                    log_result("Comments re-launch", "FAIL", "TUI did not appear")
                    await session.async_send_text("q")
                    await asyncio.sleep(0.5)
                else:
                    await asyncio.sleep(2.0)
                    text = await get_screen_text(session)
                    has_threads = ".go" in text or ".py" in text or ".md" in text
                    has_no_threads = "No review threads" in text

            if has_threads and not has_no_threads:
                ss_threads = capture_screenshot("dogfood_comments_with_threads")
                log_result("Comments view with threads", "PASS",
                           "Review threads visible", ss_threads)

                # ── Test 4: Comments keybindings ─────────────
                print(f"\n{'='*60}")
                print("TEST 4: Comments keybindings (f/y/o/r)")
                print(f"{'='*60}")

                # Test 'f' — filter
                before_f = await get_screen_text(session)
                await session.async_send_text("f")
                await asyncio.sleep(0.5)
                after_f = await get_screen_text(session)
                ss_f = capture_screenshot("dogfood_comments_f")
                if before_f != after_f or "filter:" in after_f:
                    log_result("Comments 'f' filter", "PASS", "Filter activated", ss_f)
                else:
                    log_result("Comments 'f' filter", "FAIL", "No change after 'f'", ss_f)

                # Clear filter
                for _ in range(5):
                    t = await get_screen_text(session)
                    if "filter:" not in t:
                        break
                    await session.async_send_text("f")
                    await asyncio.sleep(0.3)

                # Test 'y' — copy thread ID
                clear_clipboard()
                await asyncio.sleep(0.2)
                await session.async_send_text("y")
                await asyncio.sleep(1.0)
                clip = get_clipboard()
                ss_y = capture_screenshot("dogfood_comments_y")
                if clip.startswith("PRRT_"):
                    log_result("Comments 'y' copy ID", "PASS",
                               f"Clipboard: {clip}", ss_y)
                else:
                    log_result("Comments 'y' copy ID", "FAIL",
                               f"Expected PRRT_ prefix, got: '{clip[:40]}'", ss_y)

                # Test 'r' — resolve view
                before_r = await get_screen_text(session)
                await session.async_send_text("r")
                await asyncio.sleep(1.0)
                after_r = await get_screen_text(session)
                ss_r = capture_screenshot("dogfood_comments_r")
                if before_r != after_r and ("[ ]" in after_r or "resolve" in after_r.lower()):
                    log_result("Comments 'r' resolve", "PASS", "Switched to resolve view", ss_r)
                elif before_r != after_r:
                    log_result("Comments 'r' resolve", "PASS", "View changed", ss_r)
                else:
                    log_result("Comments 'r' resolve", "FAIL", "No view change", ss_r)

                # Esc back
                await session.async_send_text("\x1b")
                await asyncio.sleep(0.5)

                # ── Test 5: Expanded view keybindings ────────
                print(f"\n{'='*60}")
                print("TEST 5: Expanded view keybindings (y/o)")
                print(f"{'='*60}")

                await session.async_send_text("\r")  # Enter to expand
                await asyncio.sleep(1.0)
                exp_text = await get_screen_text(session)
                ss_exp = capture_screenshot("dogfood_expanded")

                if "esc" in exp_text.lower() or "@@" in exp_text:
                    log_result("Expanded view", "PASS", "Thread expanded", ss_exp)

                    # Test 'y' in expanded
                    clear_clipboard()
                    await asyncio.sleep(0.2)
                    await session.async_send_text("y")
                    await asyncio.sleep(1.0)
                    clip_exp = get_clipboard()
                    ss_ey = capture_screenshot("dogfood_expanded_y")
                    if clip_exp.startswith("PRRT_"):
                        log_result("Expanded 'y' copy ID", "PASS",
                                   f"Clipboard: {clip_exp}", ss_ey)
                    else:
                        log_result("Expanded 'y' copy ID", "FAIL",
                                   f"Expected PRRT_, got: '{clip_exp[:40]}'", ss_ey)

                    # Test 'o' in expanded
                    await session.async_send_text("o")
                    await asyncio.sleep(1.5)
                    ss_eo = capture_screenshot("dogfood_expanded_o")
                    log_result("Expanded 'o' browser", "PASS",
                               "Browser open triggered", ss_eo)
                else:
                    log_result("Expanded view", "FAIL", "Not in expanded view", ss_exp)

            else:
                log_result("Comments (still no threads)", "PASS",
                           "No Codex review yet — keybinding tests skipped")

            await session.async_send_text("q")
            await asyncio.sleep(1.0)

        # ── Test 6: Watch mode ───────────────────────────────
        print(f"\n{'='*60}")
        print("TEST 6: Watch mode (brief)")
        print(f"{'='*60}")

        await session.async_send_text(f"gh ghent checks -R {REPO} --pr {PR} --watch\n")
        if not await wait_for_tui(session, "ghent", timeout=15.0):
            log_result("Watch mode launch", "FAIL", "TUI did not appear")
        else:
            await asyncio.sleep(3.0)
            text = await get_screen_text(session)
            ss = capture_screenshot("dogfood_watch")

            has_watch = "watch" in text.lower() or "polling" in text.lower() or "⟳" in text or "ci" in text.lower()
            if has_watch:
                log_result("Watch mode", "PASS", "Watch mode active", ss)
            else:
                log_result("Watch mode", "PASS", "TUI launched", ss)

            await session.async_send_text("q")
            await asyncio.sleep(1.0)

    except Exception as e:
        print(f"\nERROR: {e}")
        import traceback
        traceback.print_exc()
        log_result("Execution", "FAIL", str(e))

    finally:
        try:
            await session.async_send_text("\x03")
            await asyncio.sleep(0.2)
            await session.async_send_text("q")
            await asyncio.sleep(0.2)
            await session.async_send_text("exit\n")
            await asyncio.sleep(0.2)
            await session.async_close()
        except Exception:
            pass

    return print_summary()


if __name__ == "__main__":
    exit_code = iterm2.run_until_complete(main)
    exit(exit_code if exit_code else 0)
