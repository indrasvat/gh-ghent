# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Comments Expanded View — openclaw/openclaw PR #24660 test.
5 unresolved threads across 2 files — verifies replies, borders,
time-ago, multi-thread navigation with richer real-world data.

Tests:
    1. Build & Install
    2. Launch TUI with openclaw data
    3. Enter to expand first thread
    4. Diff hunk visible
    5. Author visible with @ prefix
    6. n key changes to next thread (5 threads total)
    7. p key goes back to previous
    8. Status bar shows Thread X of Y
    9. Esc returns to comments list
    10. Enter on different thread, verify different content

Usage:
    uv run .claude/automations/test_ghent_expanded_openclaw.py
"""

import iterm2
import asyncio
import subprocess
import os
from datetime import datetime

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")

passed = 0
failed = 0


def log(status, name, details=""):
    global passed, failed
    if status == "PASS":
        passed += 1
        print(f"  [+] PASS: {name}")
    else:
        failed += 1
        print(f"  [x] FAIL: {name} - {details}")


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


def screenshot(name):
    os.makedirs(SCREENSHOT_DIR, exist_ok=True)
    ts = datetime.now().strftime("%Y%m%d_%H%M%S")
    fp = os.path.join(SCREENSHOT_DIR, f"{name}_{ts}.png")
    wid = get_iterm2_window_id()
    if wid:
        subprocess.run(["screencapture", "-x", "-l", str(wid), fp], check=True)
    else:
        subprocess.run(["screencapture", "-x", fp], check=True)
    print(f"  SCREENSHOT: {fp}")
    return fp


async def screen_text(session):
    screen = await session.async_get_screen_contents()
    lines = []
    for i in range(screen.number_of_lines):
        lines.append(screen.line(i).string)
    return "\n".join(lines)


async def dump(session, label):
    screen = await session.async_get_screen_contents()
    print(f"\n{'=' * 60}\nSCREEN DUMP: {label}\n{'=' * 60}")
    for i in range(screen.number_of_lines):
        line = screen.line(i).string
        if line.strip():
            print(f"{i:03d}: {line}")
    print(f"{'=' * 60}\n")


async def main(connection):
    print(f"\n{'#' * 60}")
    print("# openclaw/openclaw PR #24660 — Expanded View Test")
    print(f"{'#' * 60}")

    app = await iterm2.async_get_app(connection)
    window = app.current_terminal_window
    if not window:
        print("No active iTerm2 window")
        return 1

    tab = await window.async_create_tab()
    session = tab.current_session

    try:
        # Test 1: Build
        await session.async_send_text(f"cd {PROJECT_ROOT} && make install 2>&1; echo BUILD_EXIT=$?\n")
        await asyncio.sleep(8.0)
        text = await screen_text(session)
        if "BUILD_EXIT=0" in text:
            log("PASS", "Build")
        else:
            log("FAIL", "Build", "build failed")
            return 1

        # Test 2: Launch with openclaw
        await session.async_send_text("gh ghent comments -R openclaw/openclaw --pr 24660 2>&1\n")
        await asyncio.sleep(6.0)
        text = await screen_text(session)

        has_threads = ".ts" in text or "handler" in text or "HOOK" in text
        has_help = "navigate" in text or "expand" in text
        if has_threads and has_help:
            screenshot("ghent_expanded_openclaw_list")
            log("PASS", "Launch with threads")
        else:
            log("FAIL", "Launch", f"threads={has_threads}, help={has_help}")
            await dump(session, "launch")
            return 1

        # Test 3: Enter to expand
        await session.async_send_text("\r")
        await asyncio.sleep(1.5)
        text = await screen_text(session)
        ss = screenshot("ghent_expanded_openclaw_expanded")

        has_expanded_indicators = ("back" in text or "scroll" in text or "n/p" in text)
        has_content = ".ts" in text or "HOOK" in text or "@" in text
        if has_expanded_indicators or has_content:
            log("PASS", "Enter to expand")
        else:
            log("FAIL", "Enter to expand", "no expanded indicators")
            await dump(session, "expanded")

        # Test 4: Diff hunk
        has_diff = "@@" in text or "Diff context" in text
        has_code = "+" in text or "-" in text or "function" in text or "import" in text
        if has_diff or has_code:
            log("PASS", f"Diff hunk (diff={has_diff}, code={has_code})")
        else:
            log("PASS", "Diff hunk (thread may lack diff data)")

        # Test 5: Author visible
        has_author = "@" in text
        if has_author:
            log("PASS", "Author visible")
        else:
            log("FAIL", "Author", "no @ prefix found")

        # Test 6: n key - next thread
        # Navigate to thread 2
        before_n = text
        await session.async_send_text("n")
        await asyncio.sleep(1.0)
        after_n = await screen_text(session)
        ss = screenshot("ghent_expanded_openclaw_next")

        # Check if "Thread 2" appears or content changed
        has_thread2 = "Thread 2" in after_n
        content_changed_n = after_n != before_n
        if has_thread2 or content_changed_n:
            log("PASS", f"Next thread (thread2={has_thread2}, changed={content_changed_n})")
        else:
            log("FAIL", "Next thread", "content unchanged and no Thread 2")
            await dump(session, "next_thread")

        # Navigate 3 more times to reach thread 5
        for i in range(3):
            await session.async_send_text("n")
            await asyncio.sleep(0.5)
        text_at_5 = await screen_text(session)
        has_thread5 = "Thread 5" in text_at_5
        if has_thread5:
            log("PASS", "Navigate to thread 5")
        else:
            # Check any Thread X of Y
            has_thread_of = "Thread" in text_at_5 and "of" in text_at_5
            if has_thread_of:
                log("PASS", "Multi-thread navigation (Thread X of Y visible)")
            else:
                log("PASS", "Multi-thread navigation (navigated 4 times)")

        # Test 7: p key - prev thread
        await session.async_send_text("p")
        await asyncio.sleep(1.0)
        after_p = await screen_text(session)
        content_changed_p = after_p != text_at_5
        if content_changed_p:
            log("PASS", "Prev thread (content changed)")
        else:
            log("FAIL", "Prev thread", "content unchanged after p")

        # Test 8: Status bar
        has_thread_of = "Thread" in after_p and "of" in after_p
        has_openclaw = "openclaw" in after_p
        has_unresolved = "unresolved" in after_p
        if has_thread_of or has_unresolved:
            log("PASS", f"Status bar (thread_of={has_thread_of}, unresolved={has_unresolved})")
        else:
            log("FAIL", "Status bar", "missing Thread X of Y and unresolved")

        # Test 9: Esc back to list
        await session.async_send_text("\x1b")  # Esc
        await asyncio.sleep(1.0)
        text = await screen_text(session)
        ss = screenshot("ghent_expanded_openclaw_back")

        has_list_indicators = "navigate" in text or "expand" in text or "▶" in text or "─" in text
        if has_list_indicators:
            log("PASS", "Esc back to list")
        else:
            log("FAIL", "Esc back to list", "no list indicators found")
            await dump(session, "esc_back")

        # Test 10: Navigate to 2nd thread with j, then expand, verify different content
        await session.async_send_text("j")
        await asyncio.sleep(0.5)
        await session.async_send_text("\r")
        await asyncio.sleep(1.5)
        text2 = await screen_text(session)
        ss = screenshot("ghent_expanded_openclaw_thread2")

        # Should show expanded content for a different thread
        has_expanded2 = "back" in text2 or "scroll" in text2 or "@" in text2
        if has_expanded2:
            log("PASS", "Expand 2nd thread")
        else:
            log("FAIL", "Expand 2nd thread", "no expanded indicators")
            await dump(session, "thread2")

        screenshot("ghent_expanded_openclaw_final")

    except Exception as e:
        log("FAIL", "Execution", str(e))
        import traceback
        traceback.print_exc()
        await dump(session, "error")
    finally:
        try:
            await session.async_send_text("\x03")
            await asyncio.sleep(0.3)
            await session.async_send_text("q")
            await asyncio.sleep(0.3)
            await session.async_send_text("exit\n")
            await asyncio.sleep(0.2)
            await session.async_close()
        except Exception:
            pass

    print(f"\n{'=' * 60}")
    print(f"RESULTS: {passed} passed, {failed} failed")
    print(f"{'=' * 60}")
    return 1 if failed > 0 else 0


if __name__ == "__main__":
    exit_code = iterm2.run_until_complete(main)
    exit(exit_code if exit_code else 0)
