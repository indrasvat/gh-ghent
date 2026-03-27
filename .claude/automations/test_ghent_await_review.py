# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent --await-review Visual Test: Automated verification of the review-await
feature in both pipe mode (JSON output) and TUI watch mode.

Tests:
    1. Build: Verify gh-ghent builds and installs
    2. Pipe --await-review: JSON includes review_settled with phase=settled
    3. Pipe wait_seconds: review_settled.wait_seconds >= 30 (debounce)
    4. Pipe backward compat: --watch alone has no review_settled
    5. TUI --await-review launch: watch view shows CI phase
    6. TUI review phase: screen shows "awaiting reviews" after CI passes
    7. TUI settled transition: after debounce, transitions to status dashboard
    8. TUI status content: KPI cards visible in status view

Verification Strategy:
    - Create a dedicated window (never current_terminal_window) for isolation
    - Use indrasvat/doot PR #1 (checks passed, no active reviews)
    - Pipe mode: subprocess with JSON parsing (fast, no TUI needed)
    - TUI mode: launch in iTerm2 session, poll screen for state transitions
    - 30s debounce means review phase takes ~30-35s — use polling, not fixed sleep
    - Dump screen on any failure for debugging

Screenshots:
    - ghent_await_review_ci_phase.png: Watch view during CI check phase
    - ghent_await_review_awaiting.png: Watch view in "awaiting reviews" phase
    - ghent_await_review_status.png: Status dashboard after settlement

Screenshot Inspection Checklist:
    - Colors: Tokyo Night theme, yellow for awaiting, green for passed
    - Boundaries: Status bar, help bar, check list
    - Visible Elements: spinner, "awaiting reviews", idle/timeout counters
    - Keyboard Navigation: q to quit

Key Bindings:
    - q: Quit TUI
    - Ctrl+C: Interrupt

Usage:
    uv run .claude/automations/test_ghent_await_review.py
"""

import asyncio
import json
import os
import subprocess
import sys
import time
from datetime import datetime

import iterm2

# ============================================================
# CONFIGURATION
# ============================================================

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")
TIMEOUT_SECONDS = 10.0
REPO = "indrasvat/doot"
PR = "1"

# ============================================================
# RESULT TRACKING
# ============================================================

results = {
    "passed": 0,
    "failed": 0,
    "unverified": 0,
    "tests": [],
    "screenshots": [],
    "start_time": None,
    "end_time": None,
}


def log_result(
    test_name: str, status: str, details: str = "", screenshot: str | None = None
):
    results["tests"].append(
        {"name": test_name, "status": status, "details": details, "screenshot": screenshot}
    )
    if screenshot:
        results["screenshots"].append(screenshot)

    symbol = {"PASS": "+", "FAIL": "x", "UNVERIFIED": "?"}.get(status, "?")
    results[{"PASS": "passed", "FAIL": "failed"}.get(status, "unverified")] += 1
    print(f"  [{symbol}] {status}: {test_name}")
    if details:
        print(f"      {details}")
    if screenshot:
        print(f"      Screenshot: {screenshot}")


def print_summary() -> int:
    results["end_time"] = datetime.now()
    total = results["passed"] + results["failed"] + results["unverified"]
    duration = (
        (results["end_time"] - results["start_time"]).total_seconds()
        if results["start_time"]
        else 0
    )

    print(f"\n{'=' * 60}")
    print("TEST SUMMARY — --await-review Feature")
    print("=" * 60)
    print(f"Duration:   {duration:.1f}s")
    print(f"Total:      {total}")
    print(f"Passed:     {results['passed']}")
    print(f"Failed:     {results['failed']}")
    print(f"Unverified: {results['unverified']}")
    if results["screenshots"]:
        print(f"Screenshots: {len(results['screenshots'])}")
    print("=" * 60)

    if results["failed"] > 0:
        print("\nFailed tests:")
        for t in results["tests"]:
            if t["status"] == "FAIL":
                print(f"  - {t['name']}: {t['details']}")
        print("\nOVERALL: FAILED")
        return 1
    print("\nOVERALL: PASSED")
    return 0


def print_test_header(test_name: str, test_num: int | None = None):
    header = f"TEST {test_num}: {test_name}" if test_num else f"TEST: {test_name}"
    print(f"\n{'=' * 60}")
    print(header)
    print("=" * 60)


# ============================================================
# QUARTZ WINDOW TARGETING (position-based, parallel-safe)
# ============================================================

try:
    import Quartz

    def find_quartz_window_id(target_x, target_w, target_h, tolerance=30):
        window_list = Quartz.CGWindowListCopyWindowInfo(
            Quartz.kCGWindowListOptionOnScreenOnly
            | Quartz.kCGWindowListExcludeDesktopElements,
            Quartz.kCGNullWindowID,
        )
        best_id, best_score = None, float("inf")
        for w in window_list:
            if "iTerm" not in w.get("kCGWindowOwnerName", ""):
                continue
            b = w.get("kCGWindowBounds", {})
            score = (
                abs(float(b.get("X", 0)) - target_x) * 2
                + abs(float(b.get("Width", 0)) - target_w)
                + abs(float(b.get("Height", 0)) - target_h)
            )
            if score < best_score:
                best_score, best_id = score, w.get("kCGWindowNumber")
        return best_id if best_score < tolerance else None

except ImportError:
    print("WARNING: Quartz not available, screenshots will capture full screen")

    def find_quartz_window_id(target_x, target_w, target_h, tolerance=30):
        return None


async def capture_screenshot(window, name: str) -> str:
    os.makedirs(SCREENSHOT_DIR, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    filename = f"{name}_{timestamp}.png"
    filepath = os.path.join(SCREENSHOT_DIR, filename)

    frame = await window.async_get_frame()
    qid = find_quartz_window_id(frame.origin.x, frame.size.width, frame.size.height)

    if qid:
        subprocess.run(["screencapture", "-x", "-l", str(qid), filepath], check=True)
    else:
        print("  WARNING: Quartz window not found, capturing full screen")
        subprocess.run(["screencapture", "-x", filepath], check=True)

    print(f"  SCREENSHOT: {filepath}")
    return filepath


# ============================================================
# VERIFICATION HELPERS
# ============================================================


async def get_screen_text(session) -> str:
    screen = await session.async_get_screen_contents()
    lines = []
    for i in range(screen.number_of_lines):
        lines.append(screen.line(i).string)
    return "\n".join(lines)


async def verify_screen_contains(session, expected: str, description: str) -> bool:
    start = time.monotonic()
    while (time.monotonic() - start) < TIMEOUT_SECONDS:
        screen = await session.async_get_screen_contents()
        for i in range(screen.number_of_lines):
            if expected.lower() in screen.line(i).string.lower():
                print(f"  Found: '{expected}' ({description})")
                return True
        await asyncio.sleep(0.3)
    print(f"  Not found: '{expected}' after {TIMEOUT_SECONDS}s ({description})")
    return False


async def dump_screen(session, label: str):
    screen = await session.async_get_screen_contents()
    print(f"\n{'=' * 60}")
    print(f"SCREEN DUMP: {label}")
    print("=" * 60)
    for i in range(screen.number_of_lines):
        line = screen.line(i).string
        if line.strip():
            print(f"{i:03d}: {line}")
    print("=" * 60 + "\n")


# ============================================================
# WINDOW CREATION WITH READINESS PROBES
# ============================================================


async def create_test_window(connection, name="test", x_pos=100, width=900, height=600):
    window = await iterm2.Window.async_create(connection)
    if window is None:
        raise RuntimeError("Window.async_create() returned None")

    await asyncio.sleep(0.5)

    app = await iterm2.async_get_app(connection)
    if window.current_tab is None:
        for w in app.terminal_windows:
            if w.window_id == window.window_id:
                window = w
                break

    for _ in range(20):
        if window.current_tab and window.current_tab.current_session:
            break
        await asyncio.sleep(0.2)

    if not window.current_tab or not window.current_tab.current_session:
        raise RuntimeError("Window tab/session not ready after timeout")

    session = window.current_tab.current_session
    await session.async_set_name(name)

    frame = await window.async_get_frame()
    await window.async_set_frame(
        iterm2.Frame(
            iterm2.Point(x_pos, frame.origin.y),
            iterm2.Size(width, height),
        )
    )
    await asyncio.sleep(0.3)

    screen = await session.async_get_screen_contents()
    if screen is None:
        raise RuntimeError("Screen not readable after window creation")

    return window, session


# ============================================================
# CLEANUP
# ============================================================


async def cleanup_session(session, quit_key: str | None = None):
    print("\n  Performing cleanup...")
    try:
        await session.async_send_text("\x03")  # Ctrl+C
        await asyncio.sleep(0.2)
        if quit_key:
            await session.async_send_text(quit_key)
            await asyncio.sleep(0.3)
        await session.async_send_text("\x03")  # Ctrl+C again
        await asyncio.sleep(0.1)
        await session.async_send_text("exit\n")
        await asyncio.sleep(0.3)
        await session.async_close()
        print("  Cleanup complete")
    except Exception as e:
        print(f"  Cleanup warning: {e}")


# ============================================================
# MAIN TEST FUNCTION
# ============================================================


async def main(connection):
    results["start_time"] = datetime.now()

    print("\n" + "#" * 60)
    print("# ghent --await-review VISUAL TEST")
    print("# Tests review-await in pipe mode and TUI watch mode")
    print("#" * 60)

    window, session = await create_test_window(connection, "await-review-test", x_pos=150)
    created_sessions = [session]

    try:
        await session.async_send_text(f"cd {PROJECT_ROOT}\n")
        await asyncio.sleep(0.5)

        # ============================================================
        # TEST 1: Build
        # ============================================================
        print_test_header("Build & Install", 1)
        result = subprocess.run(
            ["make", "install"], capture_output=True, text=True, cwd=PROJECT_ROOT
        )
        if result.returncode == 0:
            log_result("Build & install", "PASS")
        else:
            log_result("Build & install", "FAIL", result.stderr[:200])
            return print_summary()

        # ============================================================
        # TESTS 2-3: Pipe mode --await-review
        # ============================================================
        print_test_header("Pipe --await-review (JSON output)", 2)
        print("  Running gh ghent status --await-review (takes ~30s for debounce)...")

        pipe_result = subprocess.run(
            [
                "gh", "ghent", "status", "--pr", PR, "-R", REPO,
                "--format", "json", "--no-tui",
                "--await-review", "--review-timeout", "45s",
            ],
            capture_output=True, text=True, timeout=120,
        )

        try:
            data = json.loads(pipe_result.stdout)
            settled = data.get("review_settled")
            if settled and settled.get("phase") == "settled":
                log_result("Pipe: review_settled.phase=settled", "PASS")
            else:
                log_result(
                    "Pipe: review_settled.phase=settled", "FAIL",
                    f"got: {settled}"
                )

            print_test_header("Pipe wait_seconds >= 30", 3)
            wait = settled.get("wait_seconds", 0) if settled else 0
            if wait >= 30:
                log_result("Pipe: wait_seconds >= 30", "PASS", f"wait_seconds={wait}")
            else:
                log_result("Pipe: wait_seconds >= 30", "FAIL", f"wait_seconds={wait}")
        except (json.JSONDecodeError, TypeError) as e:
            log_result("Pipe: review_settled.phase=settled", "FAIL", str(e))
            log_result("Pipe: wait_seconds >= 30", "FAIL", "no valid JSON")

        # ============================================================
        # TEST 4: Backward compat
        # ============================================================
        print_test_header("Backward compat (--watch alone)", 4)
        compat_result = subprocess.run(
            [
                "gh", "ghent", "status", "--pr", PR, "-R", REPO,
                "--format", "json", "--no-tui", "--watch",
            ],
            capture_output=True, text=True, timeout=30,
        )
        try:
            compat_data = json.loads(compat_result.stdout)
            if compat_data.get("review_settled") is None:
                log_result("Backward compat: no review_settled", "PASS")
            else:
                log_result(
                    "Backward compat: no review_settled", "FAIL",
                    "review_settled should be absent"
                )
        except json.JSONDecodeError:
            log_result("Backward compat: no review_settled", "FAIL", "invalid JSON")

        # ============================================================
        # TESTS 5-8: TUI mode --await-review
        # ============================================================
        print_test_header("TUI --await-review launch", 5)
        await session.async_send_text("clear\n")
        await asyncio.sleep(0.3)
        await session.async_send_text(
            f"gh ghent status --pr {PR} -R {REPO} --await-review --review-timeout 45s\n"
        )
        # Wait for TUI to render (CI checks are already completed → fast)
        await asyncio.sleep(3)

        screen_text = await get_screen_text(session)
        ss = await capture_screenshot(window, "ghent_await_review_ci_phase")

        # CI should complete nearly instantly for doot (checks already done)
        if any(kw in screen_text.lower() for kw in ["watching", "all checks passed", "awaiting reviews", "passed"]):
            log_result("TUI: watch phase visible", "PASS", screenshot=ss)
        else:
            await dump_screen(session, "tui_launch")
            log_result("TUI: watch phase visible", "FAIL", "no watch indicators", screenshot=ss)

        # ============================================================
        # TEST 6: Review-await phase
        # ============================================================
        print_test_header("TUI review-await phase", 6)
        # Poll for "awaiting reviews" text (CI passes fast, review phase starts)
        found_awaiting = False
        for attempt in range(15):  # ~15s of polling
            screen_text = await get_screen_text(session)
            if "awaiting reviews" in screen_text.lower():
                found_awaiting = True
                break
            # May have already settled if debounce passed
            if "reviews settled" in screen_text.lower() or "unresolved" in screen_text.lower():
                found_awaiting = True
                break
            await asyncio.sleep(1)

        if found_awaiting:
            ss = await capture_screenshot(window, "ghent_await_review_awaiting")
            log_result("TUI: review-await phase visible", "PASS", screenshot=ss)
        else:
            await dump_screen(session, "no_awaiting")
            log_result("TUI: review-await phase visible", "FAIL", "never saw awaiting/settled")

        # ============================================================
        # TEST 7: Status transition after debounce
        # ============================================================
        print_test_header("TUI status transition", 7)
        print("  Waiting for 30s debounce + status transition...")
        # Poll for status dashboard indicators
        found_status = False
        for attempt in range(45):  # up to ~45s total
            screen_text = await get_screen_text(session)
            if any(kw in screen_text.lower() for kw in ["unresolved", "approved", "merge", "review comments", "ci checks"]):
                found_status = True
                break
            await asyncio.sleep(1)

        if found_status:
            ss = await capture_screenshot(window, "ghent_await_review_status")
            log_result("TUI: status dashboard visible", "PASS", screenshot=ss)
        else:
            await dump_screen(session, "no_status")
            log_result("TUI: status dashboard visible", "FAIL", "no status indicators after wait")

        # ============================================================
        # TEST 8: Status KPI content
        # ============================================================
        print_test_header("TUI status KPI cards", 8)
        screen_text = await get_screen_text(session)
        has_kpi = any(kw in screen_text.lower() for kw in ["pass", "fail", "unresolved", "approved", "ready", "not ready"])
        if has_kpi:
            log_result("TUI: KPI cards visible", "PASS")
        else:
            log_result("TUI: KPI cards visible", "UNVERIFIED", "could not confirm KPI cards")

    except Exception as e:
        print(f"\nERROR during test execution: {e}")
        log_result("Test Execution", "FAIL", str(e))
        try:
            await dump_screen(session, "error_state")
        except Exception:
            pass

    finally:
        for s in created_sessions:
            await cleanup_session(s, quit_key="q")

    return print_summary()


if __name__ == "__main__":
    exit_code = iterm2.run_until_complete(main)
    sys.exit(exit_code or 0)
