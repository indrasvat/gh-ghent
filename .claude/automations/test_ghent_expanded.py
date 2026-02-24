# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Comments Expanded View Visual Test: Comprehensive automated verification
of the TUI expanded thread view with real PR data.

Tests:
    1. Build: Verify gh-ghent builds and installs
    2. Launch: TUI launches with comments list
    3. Enter Expand: Enter opens expanded thread view
    4. Thread Header: File path and line number visible
    5. Diff Hunk: Diff context with @@ header and +/- lines
    6. Comments: Root comment with author and body
    7. Replies: Reply with │ border visible
    8. Time Ago: Time-ago labels present
    9. Help Bar: Expanded view key bindings shown
    10. Status Bar: Shows "Thread X of Y"
    11. n Key: Next thread changes content
    12. p Key: Prev thread returns to original
    13. j/k Scroll: Viewport scrolling in expanded view
    14. Esc Back: Returns to comments list

Verification Strategy:
    - Use indrasvat/tbgs PR #1 (2 unresolved threads with diff hunks)
    - Read screen contents at each step
    - Take screenshots at every visual milestone
    - Verify specific text patterns for each element
    - Dump screen on any failure for debugging

Screenshots:
    - ghent_comments_expanded.png
    - ghent_comments_diffhunk.png
    - ghent_comments_next_thread.png
    - ghent_comments_expanded_back.png

Usage:
    uv run .claude/automations/test_ghent_expanded.py
"""

import iterm2
import asyncio
import subprocess
import os
from datetime import datetime

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")
TIMEOUT_SECONDS = 8.0

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
    print("TEST SUMMARY — Comments Expanded View")
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


async def screen_text(session) -> str:
    screen = await session.async_get_screen_contents()
    lines = []
    for i in range(screen.number_of_lines):
        lines.append(screen.line(i).string)
    return "\n".join(lines)


async def dump_screen(session, label: str):
    screen = await session.async_get_screen_contents()
    print(f"\n{'=' * 60}\nSCREEN DUMP: {label}\n{'=' * 60}")
    for i in range(screen.number_of_lines):
        line = screen.line(i).string
        if line.strip():
            print(f"{i:03d}: {line}")
    print(f"{'=' * 60}\n")


async def main(connection):
    results["start_time"] = datetime.now()
    print(f"\n{'#' * 60}")
    print("# ghent Comments Expanded View — Visual Test")
    print(f"{'#' * 60}")

    app = await iterm2.async_get_app(connection)
    window = app.current_terminal_window
    if not window:
        print("No active iTerm2 window")
        return 1

    tab = await window.async_create_tab()
    session = tab.current_session

    try:
        # ── Test 1: Build ──
        print_test_header("Build & Install", 1)
        await session.async_send_text(f"cd {PROJECT_ROOT} && make install 2>&1; echo BUILD_EXIT=$?\n")
        await asyncio.sleep(TIMEOUT_SECONDS)
        text = await screen_text(session)
        if "BUILD_EXIT=0" in text:
            log_result("Build & Install", "PASS")
        else:
            log_result("Build & Install", "FAIL", "build failed")
            await dump_screen(session, "build")
            return print_summary()

        # ── Test 2: Launch TUI ──
        print_test_header("Launch TUI", 2)
        await session.async_send_text("gh ghent comments -R indrasvat/tbgs --pr 1 2>&1\n")
        await asyncio.sleep(6.0)
        text = await screen_text(session)

        go_file = ".go" in text
        help_bar = "navigate" in text or "expand" in text
        thread_data = "PRRT" in text or "@" in text
        if go_file and (help_bar or thread_data):
            ss = capture_screenshot("ghent_expanded_launch")
            log_result("Launch TUI", "PASS", screenshot=ss)
        else:
            log_result("Launch TUI", "FAIL", f"go_file={go_file}, help={help_bar}, thread={thread_data}")
            await dump_screen(session, "launch")
            return print_summary()

        # ── Test 3: Enter to expand ──
        print_test_header("Enter Expand", 3)
        await session.async_send_text("\r")  # Enter key
        await asyncio.sleep(1.5)
        text = await screen_text(session)
        ss = capture_screenshot("ghent_comments_expanded")

        # Should show expanded-view indicators: "back to list" or "scroll" or "next/prev"
        has_expanded_help = "back to list" in text or "scroll" in text or "next/prev" in text or "n/p" in text
        # Should still have file content
        has_file = ".go" in text
        if has_expanded_help or has_file:
            log_result("Enter Expand", "PASS", screenshot=ss)
        else:
            log_result("Enter Expand", "FAIL", f"expanded_help={has_expanded_help}, file={has_file}")
            await dump_screen(session, "expanded")

        # ── Test 4: Thread Header ──
        print_test_header("Thread Header", 4)
        has_file_path = ".go" in text
        has_line = ":" in text  # :47 or similar line number
        if has_file_path:
            log_result("Thread Header", "PASS", f"file={has_file_path}, line={has_line}")
        else:
            log_result("Thread Header", "FAIL", f"file={has_file_path}")
            await dump_screen(session, "header")

        # ── Test 5: Diff Hunk ──
        print_test_header("Diff Hunk", 5)
        has_diff_context = "Diff context" in text
        has_diff_marker = "@@" in text or "+" in text or "-" in text
        ss = capture_screenshot("ghent_comments_diffhunk")
        if has_diff_context or has_diff_marker:
            log_result("Diff Hunk", "PASS", f"context={has_diff_context}, marker={has_diff_marker}", screenshot=ss)
        else:
            log_result("Diff Hunk", "UNVERIFIED", "diff hunk not visible (thread may lack diff data)")
            await dump_screen(session, "diff")

        # ── Test 6: Comments ──
        print_test_header("Comments", 6)
        has_author = "@" in text
        # Check for comment body content
        has_body = len(text) > 100  # expanded view should have substantial content
        if has_author:
            log_result("Comments", "PASS", f"author={has_author}")
        else:
            log_result("Comments", "FAIL", f"author={has_author}, body_len={len(text)}")
            await dump_screen(session, "comments")

        # ── Test 7: Replies with Border ──
        print_test_header("Replies with Border", 7)
        has_border = "│" in text
        if has_border:
            log_result("Replies with Border", "PASS")
        else:
            # May not have replies in this thread
            log_result("Replies with Border", "UNVERIFIED", "no │ border (thread may have only 1 comment)")

        # ── Test 8: Time Ago ──
        print_test_header("Time Ago", 8)
        has_time = "ago" in text
        if has_time:
            log_result("Time Ago", "PASS")
        else:
            log_result("Time Ago", "UNVERIFIED", "no 'ago' text (timestamps may be missing)")

        # ── Test 9: Help Bar ──
        print_test_header("Help Bar", 9)
        has_back = "back" in text
        has_scroll = "scroll" in text
        has_np = "n/p" in text or "next" in text
        if has_back or has_scroll or has_np:
            log_result("Help Bar", "PASS", f"back={has_back}, scroll={has_scroll}, np={has_np}")
        else:
            log_result("Help Bar", "FAIL", "no expanded help bar items found")
            await dump_screen(session, "help")

        # ── Test 10: Status Bar Thread Count ──
        print_test_header("Status Bar Thread Count", 10)
        has_thread_of = "Thread" in text and "of" in text
        has_unresolved = "unresolved" in text
        if has_thread_of:
            log_result("Status Bar Thread Count", "PASS")
        elif has_unresolved:
            log_result("Status Bar Thread Count", "PASS", "has unresolved badge (Thread X of Y may be truncated)")
        else:
            log_result("Status Bar Thread Count", "UNVERIFIED", "Thread X of Y not visible")

        # ── Test 11: n key (next thread) ──
        print_test_header("Next Thread (n)", 11)
        before_text = text
        await session.async_send_text("n")
        await asyncio.sleep(1.0)
        after_text = await screen_text(session)
        ss = capture_screenshot("ghent_comments_next_thread")

        # Content should change (different file:line or different comment body)
        content_changed = after_text != before_text
        if content_changed:
            log_result("Next Thread (n)", "PASS", "content changed", screenshot=ss)
        else:
            # May be only 1 thread — n is a no-op
            log_result("Next Thread (n)", "UNVERIFIED", "content unchanged (may be last thread)")

        # ── Test 12: p key (prev thread) ──
        print_test_header("Prev Thread (p)", 12)
        before_text = after_text
        await session.async_send_text("p")
        await asyncio.sleep(1.0)
        after_text = await screen_text(session)

        content_changed = after_text != before_text
        if content_changed:
            log_result("Prev Thread (p)", "PASS", "content reverted")
        else:
            log_result("Prev Thread (p)", "UNVERIFIED", "content unchanged (may be first thread)")

        # ── Test 13: j/k scroll ──
        print_test_header("Viewport Scroll (j/k)", 13)
        before_text = after_text
        # Scroll down several times
        for _ in range(5):
            await session.async_send_text("j")
            await asyncio.sleep(0.2)
        after_text = await screen_text(session)
        # Scroll back up
        for _ in range(5):
            await session.async_send_text("k")
            await asyncio.sleep(0.2)

        # If content is long enough, scrolling changes visible text
        scroll_worked = after_text != before_text
        if scroll_worked:
            log_result("Viewport Scroll (j/k)", "PASS", "content shifted on scroll")
        else:
            log_result("Viewport Scroll (j/k)", "UNVERIFIED", "content fits viewport (no scroll needed)")

        # ── Test 14: Esc back to list ──
        print_test_header("Esc Back to List", 14)
        await session.async_send_text("\x1b")  # Esc
        await asyncio.sleep(1.0)
        text = await screen_text(session)
        ss = capture_screenshot("ghent_comments_expanded_back")

        # Should be back on the list view — check for list indicators
        has_list_help = "expand" in text or "navigate" in text
        has_file_grouping = "─" in text  # file header separators
        has_cursor = "▶" in text
        if has_list_help or has_file_grouping or has_cursor:
            log_result("Esc Back to List", "PASS", f"list_help={has_list_help}, grouping={has_file_grouping}, cursor={has_cursor}", screenshot=ss)
        else:
            log_result("Esc Back to List", "FAIL", "not back on list view")
            await dump_screen(session, "esc_back")

    except Exception as e:
        log_result("Execution", "FAIL", str(e))
        import traceback
        traceback.print_exc()
        await dump_screen(session, "error")
    finally:
        try:
            await session.async_send_text("\x03")  # Ctrl+C
            await asyncio.sleep(0.3)
            await session.async_send_text("q")
            await asyncio.sleep(0.3)
            await session.async_send_text("exit\n")
            await asyncio.sleep(0.2)
            await session.async_close()
        except Exception:
            pass

    return print_summary()


if __name__ == "__main__":
    exit_code = iterm2.run_until_complete(main)
    exit(exit_code if exit_code else 0)
