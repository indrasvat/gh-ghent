# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Watch Mode LIVE test: Launch TUI --watch against a repo with
in-progress CI checks. Captures the polling/spinner state.

Usage:
    uv run .claude/automations/test_ghent_watch_live.py
"""

import asyncio
import os
import subprocess
from datetime import datetime

import iterm2

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")


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


async def main(connection):
    print("\n# ghent Watch Mode — LIVE CI Test")
    print("# Watching indrasvat/gh-ghent PR #1 with in-progress CI\n")

    app = await iterm2.async_get_app(connection)
    window = app.current_terminal_window
    if not window:
        print("ERROR: No active iTerm2 window")
        return 1

    tab = await window.async_create_tab()
    session = tab.current_session

    try:
        # Launch watch TUI
        await session.async_send_text(f"cd {PROJECT_ROOT} && gh ghent checks --watch -R indrasvat/gh-ghent --pr 1 2>&1\n")

        # Wait for initial render (first poll)
        print("Waiting 12s for first poll...")
        await asyncio.sleep(12.0)

        text1 = await get_all_screen_text(session)
        await dump_screen(session, "after_first_poll")
        ss1 = capture_screenshot("ghent_watch_live_poll1")

        # Check if we see the polling state
        if "watching" in text1:
            print("\n  [+] POLLING STATE: Spinner + 'watching' visible!")
        elif "all checks passed" in text1:
            print("\n  [~] DONE STATE: CI already completed before we caught it")
        elif "failure detected" in text1:
            print("\n  [~] FAILED STATE: CI failed before we caught it")
        else:
            print("\n  [?] UNKNOWN STATE")

        # Wait for another poll cycle (10s interval)
        print("\nWaiting 12s for second poll...")
        await asyncio.sleep(12.0)

        text2 = await get_all_screen_text(session)
        await dump_screen(session, "after_second_poll")
        ss2 = capture_screenshot("ghent_watch_live_poll2")

        if "watching" in text2:
            print("\n  [+] STILL POLLING: CI still in progress")
        elif "all checks passed" in text2:
            print("\n  [+] DONE: CI completed during our watch!")
        elif "failure detected" in text2:
            print("\n  [+] FAILED: CI failed during our watch")

        # One more wait if still polling
        if "watching" in text2:
            print("\nWaiting 12s for third poll...")
            await asyncio.sleep(12.0)
            text3 = await get_all_screen_text(session)
            await dump_screen(session, "after_third_poll")
            capture_screenshot("ghent_watch_live_poll3")

        print("\n  LIVE TEST COMPLETE")
        print(f"  Screenshots: {ss1}, {ss2}")

    except Exception as e:
        print(f"ERROR: {e}")
        import traceback
        traceback.print_exc()
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

    return 0


if __name__ == "__main__":
    exit_code = iterm2.run_until_complete(main)
    exit(exit_code if exit_code else 0)
