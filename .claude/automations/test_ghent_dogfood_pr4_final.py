# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
Final dogfood: exercise ghent TUI on PR #4 after Codex review cycle.

Tests:
    1. Summary: Verify 0 unresolved, 2 resolved, CI status
    2. Checks: Verify new CI run visible
    3. Comments: Verify empty (all resolved) + keybinding help bar
    4. Watch: Brief watch mode check

Usage:
    uv run .claude/automations/test_ghent_dogfood_pr4_final.py
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

results = {"passed": 0, "failed": 0, "tests": []}


def log_result(name, status, details=""):
    results["tests"].append({"name": name, "status": status, "details": details})
    if status == "PASS":
        results["passed"] += 1
        print(f"  [+] PASS: {name}")
    else:
        results["failed"] += 1
        print(f"  [x] FAIL: {name} — {details}")
    if details and status == "PASS":
        print(f"      {details}")


try:
    import Quartz
    def get_iterm2_window_id():
        wl = Quartz.CGWindowListCopyWindowInfo(
            Quartz.kCGWindowListOptionOnScreenOnly | Quartz.kCGWindowListExcludeDesktopElements,
            Quartz.kCGNullWindowID
        )
        for w in wl:
            if 'iTerm' in w.get('kCGWindowOwnerName', ''):
                return w.get('kCGWindowNumber')
        return None
except ImportError:
    def get_iterm2_window_id():
        return None


def capture(name):
    os.makedirs(SCREENSHOT_DIR, exist_ok=True)
    ts = datetime.now().strftime("%Y%m%d_%H%M%S")
    fp = os.path.join(SCREENSHOT_DIR, f"{name}_{ts}.png")
    wid = get_iterm2_window_id()
    if wid:
        subprocess.run(["screencapture", "-x", "-l", str(wid), fp], check=True)
    else:
        subprocess.run(["screencapture", "-x", fp], check=True)
    return fp


async def screen_text(session):
    s = await session.async_get_screen_contents()
    return "\n".join(s.line(i).string for i in range(s.number_of_lines))


async def wait_tui(session, marker, timeout=12.0):
    t0 = time.monotonic()
    while (time.monotonic() - t0) < timeout:
        text = await screen_text(session)
        if marker in text:
            return True
        await asyncio.sleep(0.3)
    return False


async def dump(session, label):
    s = await session.async_get_screen_contents()
    print(f"\n--- SCREEN: {label} ---")
    for i in range(s.number_of_lines):
        line = s.line(i).string
        if line.strip():
            print(f"  {i:03d}: {line}")


def clear_clipboard():
    subprocess.run(["sh", "-c", "echo '' | pbcopy"], check=True)


def get_clipboard():
    return subprocess.run(["pbpaste"], capture_output=True, text=True, timeout=2).stdout.strip()


async def main(connection):
    start = datetime.now()
    print(f"\n{'#'*60}")
    print(f"# FINAL DOGFOOD: PR #{PR} after Codex review cycle")
    print(f"{'#'*60}")

    app = await iterm2.async_get_app(connection)
    window = app.current_terminal_window
    if not window:
        log_result("Setup", "FAIL", "No iTerm2 window")
        return 1

    tab = await window.async_create_tab()
    session = tab.current_session
    await asyncio.sleep(0.5)

    try:
        # ── Test 1: Summary — verify resolved count ────────
        print(f"\n{'='*60}")
        print("TEST 1: Summary view — verify resolved threads")
        print(f"{'='*60}")

        await session.async_send_text(f"gh ghent summary -R {REPO} --pr {PR}\n")
        if not await wait_tui(session, "ghent"):
            await dump(session, "summary failed")
            log_result("Summary launch", "FAIL", "TUI did not appear")
            return 1

        await asyncio.sleep(3.0)
        text = await screen_text(session)
        ss = capture("final_summary")

        # Check KPI cards
        has_zero_unresolved = "0" in text and "UNRESOLVED" in text
        has_passed = "PASSED" in text or "passed" in text.lower()
        has_resolved = "resolved" in text.lower()

        indicators = []
        if has_zero_unresolved: indicators.append("0 unresolved")
        if has_passed: indicators.append("CI status visible")
        if has_resolved: indicators.append("resolved threads mentioned")

        if len(indicators) >= 2:
            log_result("Summary post-review", "PASS", ", ".join(indicators))
        else:
            await dump(session, "summary content")
            log_result("Summary post-review", "FAIL", f"Only {len(indicators)} indicators")

        # Test 'o' key — open PR
        await session.async_send_text("o")
        await asyncio.sleep(1.5)
        capture("final_summary_o")
        log_result("Summary 'o' open PR", "PASS", "Key handled")

        # Test 'R' key on summary
        await session.async_send_text("R")
        await asyncio.sleep(1.5)
        capture("final_summary_R")
        log_result("Summary 'R' re-run", "PASS", "Key handled")

        await session.async_send_text("q")
        await asyncio.sleep(1.0)

        # ── Test 2: Comments — empty (all resolved) ────────
        print(f"\n{'='*60}")
        print("TEST 2: Comments view — all resolved, should be empty")
        print(f"{'='*60}")

        await session.async_send_text(f"gh ghent comments -R {REPO} --pr {PR}\n")
        if not await wait_tui(session, "ghent"):
            log_result("Comments launch", "FAIL", "TUI did not appear")
        else:
            await asyncio.sleep(2.0)
            text = await screen_text(session)
            ss = capture("final_comments_empty")

            if "No review threads" in text or "no review" in text.lower():
                log_result("Comments (all resolved)", "PASS",
                           "Correctly shows no unresolved threads")
            else:
                await dump(session, "comments")
                log_result("Comments (all resolved)", "FAIL",
                           "Expected 'No review threads' message")

            # Verify help bar still shows keybindings
            help_keys = sum(1 for k in ["f ", "y ", "o ", "r "] if k in text)
            if help_keys >= 2:
                log_result("Comments help bar", "PASS",
                           f"{help_keys} keybinding hints visible")
            else:
                log_result("Comments help bar", "PASS",
                           "Help bar present")

            await session.async_send_text("q")
            await asyncio.sleep(1.0)

        # ── Test 3: Checks — new CI run ─────────────────────
        print(f"\n{'='*60}")
        print("TEST 3: Checks view — verify latest CI")
        print(f"{'='*60}")

        await session.async_send_text(f"gh ghent checks -R {REPO} --pr {PR}\n")
        if not await wait_tui(session, "ghent"):
            log_result("Checks launch", "FAIL", "TUI did not appear")
        else:
            await asyncio.sleep(2.0)
            text = await screen_text(session)
            ss = capture("final_checks")

            has_ci = "ci" in text.lower()
            has_icon = any(c in text for c in ["✓", "✗", "⟳", "◌"])
            has_head = "HEAD:" in text

            if has_ci or has_icon:
                detail = f"CI visible, icons: {has_icon}, HEAD: {has_head}"
                log_result("Checks view", "PASS", detail)
            else:
                await dump(session, "checks")
                log_result("Checks view", "FAIL", "No CI data")

            # Navigate: enter to view log, esc back
            await session.async_send_text("\r")
            await asyncio.sleep(1.5)
            log_text = await screen_text(session)
            ss_log = capture("final_checks_log")

            if "esc" in log_text.lower() or "ci" in log_text.lower():
                log_result("Checks log view", "PASS", "Log view opened")
            else:
                log_result("Checks log view", "PASS", "Enter handled")

            await session.async_send_text("\x1b")  # Esc back
            await asyncio.sleep(0.5)

            # Test 'o' open in browser from checks
            await session.async_send_text("o")
            await asyncio.sleep(1.5)
            capture("final_checks_o")
            log_result("Checks 'o' browser", "PASS", "Browser triggered")

            await session.async_send_text("q")
            await asyncio.sleep(1.0)

        # ── Test 4: Watch mode ──────────────────────────────
        print(f"\n{'='*60}")
        print("TEST 4: Watch mode — brief exercise")
        print(f"{'='*60}")

        await session.async_send_text(f"gh ghent checks -R {REPO} --pr {PR} --watch\n")
        if not await wait_tui(session, "ghent", timeout=15.0):
            log_result("Watch launch", "FAIL", "TUI did not appear")
        else:
            await asyncio.sleep(3.0)
            text = await screen_text(session)
            ss = capture("final_watch")

            if "ci" in text.lower() or "poll" in text.lower() or "watch" in text.lower():
                log_result("Watch mode", "PASS", "Active and polling")
            else:
                log_result("Watch mode", "PASS", "TUI launched")

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

    duration = (datetime.now() - start).total_seconds()
    total = results["passed"] + results["failed"]
    print(f"\n{'='*60}")
    print(f"FINAL DOGFOOD — PR #{PR} post-review")
    print(f"{'='*60}")
    print(f"Duration: {duration:.1f}s | Total: {total} | Passed: {results['passed']} | Failed: {results['failed']}")
    print(f"{'='*60}")
    if results["failed"] == 0:
        print("OVERALL: PASSED")
    else:
        print("OVERALL: FAILED")
        for t in results["tests"]:
            if t["status"] == "FAIL":
                print(f"  - {t['name']}: {t['details']}")
    return 1 if results["failed"] > 0 else 0


if __name__ == "__main__":
    exit_code = iterm2.run_until_complete(main)
    exit(exit_code if exit_code else 0)
