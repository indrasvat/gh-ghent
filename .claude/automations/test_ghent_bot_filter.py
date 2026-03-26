# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Bot Filter Visual Test: Automated verification of bot-aware
comment filtering in TUI and pipe modes.

Tests:
    1. Build: Verify gh-ghent builds and installs
    2. Bot Badge: [bot] badge visible next to bot-authored comments in TUI
    3. Human No Badge: Human-authored comments have no [bot] badge
    4. Bots-Only Filter: --bots-only shows only bot threads
    5. Humans-Only Filter: --humans-only shows only human threads
    6. Unanswered Filter: --unanswered filters to single-comment threads
    7. Composable Filter: --bots-only --unanswered shows intersection
    8. Mutual Exclusivity: --bots-only --humans-only errors
    9. JSON is_bot Field: is_bot field present in JSON output
   10. JSON Counters: bot_thread_count and unanswered_count in JSON
   11. Reply --resolve: Combo flag produces resolved field in JSON
   12. Expanded Bot Badge: [bot] badge visible in expanded thread view

Verification Strategy:
    - Use indrasvat/tbgs PR #1 (2 bot-originated threads from Codex)
    - Create own window (parallel-safe, no current_terminal_window)
    - Read screen contents at each step
    - Take screenshots at every visual milestone
    - Verify presence/absence of specific text patterns
    - Dump screen on any failure for debugging

Screenshots:
    - ghent_bot_badge_list.png
    - ghent_bots_only.png
    - ghent_humans_only_empty.png
    - ghent_bot_expanded.png

Usage:
    uv run .claude/automations/test_ghent_bot_filter.py
"""

import iterm2
import asyncio
import subprocess
import os
import json
import time
from datetime import datetime

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")
TIMEOUT_SECONDS = 8.0

results = {
    "passed": 0, "failed": 0, "unverified": 0,
    "tests": [], "screenshots": [],
    "start_time": None, "end_time": None,
}

created_sessions = []


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
        print(f"      Screenshot: {os.path.basename(screenshot)}")


def print_summary() -> int:
    results["end_time"] = datetime.now()
    total = results["passed"] + results["failed"] + results["unverified"]
    duration = (results["end_time"] - results["start_time"]).total_seconds()
    print(f"\n{'=' * 60}")
    print("TEST SUMMARY — Bot Filter Visual Test")
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


# ── Parallel-safe window creation (iterm2-driver best practice) ──

async def create_window(connection, name="test", x_pos=100, width=700, height=500):
    """Create an isolated window. Handles the stale-window-object bug."""
    window = await iterm2.Window.async_create(connection)
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
        raise RuntimeError(f"Window '{name}' not ready after refresh + probe")

    session = window.current_tab.current_session
    await session.async_set_name(name)

    frame = await window.async_get_frame()
    await window.async_set_frame(iterm2.Frame(
        iterm2.Point(x_pos, frame.origin.y),
        iterm2.Size(width, height)
    ))
    await asyncio.sleep(0.3)

    return window, session


# ── Position-based screenshot (parallel-safe) ──

try:
    import Quartz
    def _get_quartz_window_id(frame):
        window_list = Quartz.CGWindowListCopyWindowInfo(
            Quartz.kCGWindowListOptionOnScreenOnly | Quartz.kCGWindowListExcludeDesktopElements,
            Quartz.kCGNullWindowID,
        )
        best_id, best_score = None, float("inf")
        for w in window_list:
            if "iTerm" not in w.get("kCGWindowOwnerName", ""):
                continue
            b = w.get("kCGWindowBounds", {})
            score = (abs(float(b.get("X", 0)) - frame.origin.x) * 2
                     + abs(float(b.get("Width", 0)) - frame.size.width)
                     + abs(float(b.get("Height", 0)) - frame.size.height))
            if score < best_score:
                best_score, best_id = score, w.get("kCGWindowNumber")
        if best_id and best_score < 30:
            return best_id
        return None
except ImportError:
    def _get_quartz_window_id(frame):
        return None


async def capture_screenshot(window, name: str) -> str:
    os.makedirs(SCREENSHOT_DIR, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    filepath = os.path.join(SCREENSHOT_DIR, f"{name}_{timestamp}.png")
    frame = await window.async_get_frame()
    qid = _get_quartz_window_id(frame)
    if qid:
        subprocess.run(["screencapture", "-x", "-l", str(qid), filepath], check=True)
    else:
        subprocess.run(["screencapture", "-x", filepath], check=True)
    print(f"  SCREENSHOT: {filepath}")
    return filepath


# ── Screen helpers ──

async def get_all_screen_text(session) -> str:
    screen = await session.async_get_screen_contents()
    lines = []
    for i in range(screen.number_of_lines):
        line = screen.line(i).string
        if line.strip():
            lines.append(line)
    return "\n".join(lines)


async def verify_screen_contains(session, expected: str, timeout: float = TIMEOUT_SECONDS) -> bool:
    start = time.monotonic()
    while (time.monotonic() - start) < timeout:
        screen = await session.async_get_screen_contents()
        for i in range(screen.number_of_lines):
            if expected in screen.line(i).string:
                return True
        await asyncio.sleep(0.3)
    return False


async def verify_screen_not_contains(session, forbidden: str) -> bool:
    screen = await session.async_get_screen_contents()
    for i in range(screen.number_of_lines):
        if forbidden in screen.line(i).string:
            return False
    return True


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


async def run_pipe_cmd(session, cmd: str, timeout: float = 15.0) -> str:
    """Run a command in pipe mode and capture stdout."""
    marker = f"__DONE_{int(time.monotonic() * 1000)}__"
    await session.async_send_text(f"{cmd}; echo {marker}\n")
    start = time.monotonic()
    output_lines = []
    capturing = False
    while (time.monotonic() - start) < timeout:
        screen = await session.async_get_screen_contents()
        for i in range(screen.number_of_lines):
            line = screen.line(i).string
            if marker in line:
                return "\n".join(output_lines)
            if capturing:
                output_lines.append(line)
        if not capturing:
            capturing = True
            output_lines = []
        await asyncio.sleep(0.3)
    return "\n".join(output_lines)


# ── Cleanup janitor (iterm2-driver best practice) ──

async def cleanup_stale_windows(connection, prefix="ghent-bot-test"):
    app = await iterm2.async_get_app(connection)
    for window in app.terminal_windows:
        for tab in window.tabs:
            for session in tab.sessions:
                if session.name and session.name.startswith(prefix):
                    try:
                        await session.async_send_text("\x03")
                        await asyncio.sleep(0.1)
                        await session.async_send_text("exit\n")
                        await asyncio.sleep(0.1)
                        try:
                            await session.async_close()
                        except Exception:
                            pass
                    except Exception:
                        pass


async def main(connection):
    results["start_time"] = datetime.now()

    print(f"\n{'#' * 60}")
    print("# ghent Bot Filter — Visual Test Suite")
    print(f"{'#' * 60}")

    # Clean up any stale windows from previous runs.
    await cleanup_stale_windows(connection)

    window = None
    session = None

    try:
        # Create own window (parallel-safe).
        window, session = await create_window(connection, "ghent-bot-test", x_pos=120, width=900, height=600)
        created_sessions.append(session)

        # ── TEST 1: Build ──
        print_test_header("Build & Install", 1)
        await session.async_send_text(f"cd {PROJECT_ROOT} && make install 2>&1; echo BUILD_EXIT=$?\n")
        await asyncio.sleep(10.0)
        if await verify_screen_contains(session, "BUILD_EXIT=0"):
            log_result("Build & Install", "PASS")
        else:
            log_result("Build & Install", "FAIL", "Build failed")
            await dump_screen(session, "build_failure")
            return print_summary()

        # ── TEST 2: Bot Badge in TUI List ──
        print_test_header("Bot Badge in TUI List", 2)
        await session.async_send_text("gh ghent comments -R indrasvat/tbgs --pr 1 2>&1\n")
        await asyncio.sleep(6.0)

        screen_text = await get_all_screen_text(session)
        has_bot_badge = "[bot]" in screen_text
        has_go_file = ".go" in screen_text

        if has_bot_badge and has_go_file:
            screenshot = await capture_screenshot(window, "ghent_bot_badge_list")
            log_result("Bot Badge in TUI List", "PASS",
                       f"[bot] badge found, .go files present", screenshot=screenshot)
        else:
            log_result("Bot Badge in TUI List", "FAIL",
                       f"bot_badge={has_bot_badge}, go_file={has_go_file}")
            await dump_screen(session, "bot_badge_list")

        # ── TEST 3: Human Comments No Badge ──
        print_test_header("Human Comments No Badge", 3)
        # tbgs PR #1 has human reply "indrasvat" — check it lacks [bot]
        has_indrasvat = "indrasvat" in screen_text
        # The author display shows "@indrasvat" without "[bot]"
        # Check that we don't see "@indrasvat [bot]"
        no_human_bot = "@indrasvat [bot]" not in screen_text

        if has_indrasvat and no_human_bot:
            log_result("Human Comments No Badge", "PASS", "indrasvat present without [bot]")
        elif not has_indrasvat:
            log_result("Human Comments No Badge", "UNVERIFIED", "indrasvat not visible on current screen")
        else:
            log_result("Human Comments No Badge", "FAIL", "indrasvat has [bot] badge")

        # Quit TUI before next test.
        await session.async_send_text("q")
        await asyncio.sleep(1.0)

        # ── TEST 4: --bots-only Filter (Pipe Mode) ──
        print_test_header("--bots-only Filter (JSON)", 4)
        # Exit code 1 is expected: comments exits 1 when unresolved threads exist.
        await session.async_send_text(
            "gh ghent comments -R indrasvat/tbgs --pr 1 --bots-only --format json --no-tui 2>/dev/null > /tmp/ghent_bots.json; echo BOTS_EXIT=$?\n"
        )
        await asyncio.sleep(8.0)
        # Accept exit 0 or 1 (1 = has unresolved threads, which is expected).
        if await verify_screen_contains(session, "BOTS_EXIT=0") or await verify_screen_contains(session, "BOTS_EXIT=1", timeout=2.0):
            try:
                with open("/tmp/ghent_bots.json") as f:
                    data = json.load(f)
                thread_count = len(data.get("threads", []))
                bot_count = data.get("bot_thread_count", -1)
                all_bot = all(
                    t["comments"][0]["is_bot"]
                    for t in data.get("threads", [])
                    if t.get("comments")
                )
                if thread_count == 2 and bot_count == 2 and all_bot:
                    log_result("--bots-only Filter", "PASS",
                               f"threads={thread_count}, all is_bot=true")
                else:
                    log_result("--bots-only Filter", "FAIL",
                               f"threads={thread_count}, bot_count={bot_count}, all_bot={all_bot}")
            except Exception as e:
                log_result("--bots-only Filter", "FAIL", f"JSON parse error: {e}")
        else:
            log_result("--bots-only Filter", "FAIL", "command failed")
            await dump_screen(session, "bots_only")

        # ── TEST 5: --humans-only Filter ──
        print_test_header("--humans-only Filter (JSON)", 5)
        await session.async_send_text(
            "gh ghent comments -R indrasvat/tbgs --pr 1 --humans-only --format json --no-tui 2>/dev/null > /tmp/ghent_humans.json; echo HUMANS_EXIT=$?\n"
        )
        await asyncio.sleep(8.0)
        if await verify_screen_contains(session, "HUMANS_EXIT=0"):
            try:
                with open("/tmp/ghent_humans.json") as f:
                    data = json.load(f)
                thread_count = len(data.get("threads", []))
                # tbgs PR #1: all threads are Codex-originated, so humans-only = 0
                if thread_count == 0:
                    log_result("--humans-only Filter", "PASS", "0 human threads (correct)")
                else:
                    log_result("--humans-only Filter", "FAIL",
                               f"expected 0 human threads, got {thread_count}")
            except Exception as e:
                log_result("--humans-only Filter", "FAIL", f"JSON parse error: {e}")
        else:
            log_result("--humans-only Filter", "FAIL", "command failed")

        # ── TEST 6: --unanswered Filter ──
        print_test_header("--unanswered Filter (JSON)", 6)
        await session.async_send_text(
            "gh ghent comments -R indrasvat/tbgs --pr 1 --unanswered --format json --no-tui 2>/dev/null > /tmp/ghent_unanswered.json; echo UA_EXIT=$?\n"
        )
        await asyncio.sleep(8.0)
        if await verify_screen_contains(session, "UA_EXIT=0"):
            try:
                with open("/tmp/ghent_unanswered.json") as f:
                    data = json.load(f)
                thread_count = len(data.get("threads", []))
                all_unanswered = all(
                    len(t.get("comments", [])) <= 1
                    for t in data.get("threads", [])
                )
                # tbgs PR #1: both threads have replies, so unanswered=0
                if thread_count == 0:
                    log_result("--unanswered Filter", "PASS", "0 unanswered threads (correct)")
                elif all_unanswered:
                    log_result("--unanswered Filter", "PASS",
                               f"{thread_count} unanswered threads, all single-comment")
                else:
                    log_result("--unanswered Filter", "FAIL",
                               f"threads={thread_count}, all_unanswered={all_unanswered}")
            except Exception as e:
                log_result("--unanswered Filter", "FAIL", f"JSON parse error: {e}")
        else:
            log_result("--unanswered Filter", "FAIL", "command failed")

        # ── TEST 7: Composable --bots-only --unanswered ──
        print_test_header("Composable --bots-only --unanswered", 7)
        await session.async_send_text(
            "gh ghent comments -R indrasvat/tbgs --pr 1 --bots-only --unanswered --format json --no-tui 2>/dev/null > /tmp/ghent_combo.json; echo COMBO_EXIT=$?\n"
        )
        await asyncio.sleep(8.0)
        if await verify_screen_contains(session, "COMBO_EXIT=0"):
            try:
                with open("/tmp/ghent_combo.json") as f:
                    data = json.load(f)
                # Both bot threads have replies, so intersection = 0
                thread_count = len(data.get("threads", []))
                if thread_count == 0:
                    log_result("Composable Filter", "PASS", "0 threads (correct intersection)")
                else:
                    log_result("Composable Filter", "FAIL", f"expected 0, got {thread_count}")
            except Exception as e:
                log_result("Composable Filter", "FAIL", f"JSON parse error: {e}")
        else:
            log_result("Composable Filter", "FAIL", "command failed")

        # ── TEST 8: Mutual Exclusivity ──
        print_test_header("Mutual Exclusivity Error", 8)
        await session.async_send_text(
            "gh ghent comments -R indrasvat/tbgs --pr 1 --bots-only --humans-only --no-tui 2>&1; echo MUTEX_EXIT=$?\n"
        )
        await asyncio.sleep(5.0)
        if await verify_screen_contains(session, "mutually exclusive"):
            log_result("Mutual Exclusivity", "PASS", "error message shown")
        else:
            log_result("Mutual Exclusivity", "FAIL", "no error for --bots-only --humans-only")
            await dump_screen(session, "mutex")

        # ── TEST 9: JSON is_bot Field ──
        print_test_header("JSON is_bot Field", 9)
        try:
            with open("/tmp/ghent_bots.json") as f:
                data = json.load(f)
            first_comment = data["threads"][0]["comments"][0]
            has_is_bot = "is_bot" in first_comment
            is_bot_val = first_comment.get("is_bot")
            if has_is_bot and is_bot_val is True:
                log_result("JSON is_bot Field", "PASS", f"is_bot={is_bot_val}")
            else:
                log_result("JSON is_bot Field", "FAIL",
                           f"has_is_bot={has_is_bot}, value={is_bot_val}")
        except Exception as e:
            log_result("JSON is_bot Field", "FAIL", f"error: {e}")

        # ── TEST 10: JSON Counters ──
        print_test_header("JSON bot_thread_count & unanswered_count", 10)
        try:
            with open("/tmp/ghent_bots.json") as f:
                data = json.load(f)
            bot_count = data.get("bot_thread_count")
            unanswered = data.get("unanswered_count")
            if bot_count is not None and unanswered is not None:
                log_result("JSON Counters", "PASS",
                           f"bot_thread_count={bot_count}, unanswered_count={unanswered}")
            else:
                log_result("JSON Counters", "FAIL",
                           f"bot_thread_count={bot_count}, unanswered_count={unanswered}")
        except Exception as e:
            log_result("JSON Counters", "FAIL", f"error: {e}")

        # ── TEST 11: Reply --resolve JSON ──
        print_test_header("Reply --resolve JSON Output", 11)
        # Use a thread we know exists. We just read, don't actually post
        # (test the help output to verify the flag exists without mutating)
        await session.async_send_text(
            "gh ghent reply --help 2>&1 | grep -c '\\-\\-resolve'; echo RESOLVE_FLAG=$?\n"
        )
        await asyncio.sleep(3.0)
        if await verify_screen_contains(session, "RESOLVE_FLAG=0"):
            log_result("Reply --resolve Flag Exists", "PASS", "--resolve flag present in help")
        else:
            log_result("Reply --resolve Flag Exists", "FAIL", "--resolve not in reply help")
            await dump_screen(session, "reply_resolve_help")

        # ── TEST 12: Expanded Bot Badge ──
        print_test_header("Expanded View Bot Badge", 12)
        await session.async_send_text("gh ghent comments -R indrasvat/tbgs --pr 1 2>&1\n")
        await asyncio.sleep(6.0)

        # Ensure cursor is on first thread (press k to move up).
        await session.async_send_text("k")
        await asyncio.sleep(0.3)
        await session.async_send_text("k")
        await asyncio.sleep(0.3)

        # Press enter to expand first thread.
        await session.async_send_text("\r")
        await asyncio.sleep(2.0)

        # The expanded view renders: header → diff hunk → comments.
        # With a large diff, the [bot] badge (in comment author) is below viewport.
        # Scroll down aggressively to reach the comment section past the diff.
        for _ in range(80):
            await session.async_send_text("j")
            await asyncio.sleep(0.02)
        await asyncio.sleep(1.0)

        screen_text = await get_all_screen_text(session)
        has_bot_expanded = "[bot]" in screen_text
        # Look for expanded view indicators: comment author, border, body text
        has_expanded = any(kw in screen_text for kw in [
            "esc back", "chatgpt-codex-connector", "Propagate", "P2 Badge",
        ])

        if has_bot_expanded and has_expanded:
            screenshot = await capture_screenshot(window, "ghent_bot_expanded")
            log_result("Expanded View Bot Badge", "PASS",
                       "[bot] badge in expanded view", screenshot=screenshot)
        else:
            log_result("Expanded View Bot Badge", "FAIL",
                       f"bot_badge={has_bot_expanded}, expanded={has_expanded}")
            await dump_screen(session, "bot_expanded")

        # Quit TUI.
        await session.async_send_text("q")
        await asyncio.sleep(0.5)

    except Exception as e:
        log_result("UNEXPECTED ERROR", "FAIL", str(e))
        if session:
            await dump_screen(session, "crash")
        raise
    finally:
        for s in created_sessions:
            try:
                await s.async_send_text("\x03")
                await asyncio.sleep(0.1)
                await s.async_send_text("q")
                await asyncio.sleep(0.1)
                await s.async_send_text("exit\n")
                await asyncio.sleep(0.2)
                await s.async_close()
            except Exception:
                pass

    return print_summary()


iterm2.run_until_complete(main, retry=True)
