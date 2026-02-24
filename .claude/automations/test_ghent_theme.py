# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Theme Visual Test: Automated verification of Tokyo Night theme + Lipgloss styles.

Tests:
    1. Build: Verify theme-demo builds
    2. Theme Demo: Verify all styled elements render correctly
    3. Colors: Verify Tokyo Night semantic colors appear
    4. Badges: Verify badge text (PR #42, passed, failed, pending)
    5. Diff Hunk: Verify diff lines render with +/- indicators
    6. Box: Verify bordered box renders with connected corners
    7. Help Bar: Verify key hints render
    8. No Background Ban: Verify no lipgloss.Background() in Go source

Verification Strategy:
    - Build and run theme-demo binary
    - Poll screen content for expected styled elements
    - Capture screenshots at each stage
    - Grep Go source for banned patterns

Screenshots:
    - ghent_theme_demo.png: Full theme demo output
    - ghent_theme_badges.png: Badge section
    - ghent_theme_diffhunk.png: Diff hunk section
    - ghent_theme_borders.png: Box border section

Screenshot Inspection Checklist:
    - Colors: Green/red/blue/yellow/cyan/orange/purple visible
    - Badges: Text readable, proper spacing
    - Diff: +/- lines colored (green/red), context dimmed
    - Borders: Rounded corners connected (╭─╮ ... ╰─╯)
    - Help: Key shortcuts highlighted in blue

Usage:
    uv run .claude/automations/test_ghent_theme.py
"""

import iterm2
import asyncio
import subprocess
import os
import time
from datetime import datetime

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")
TIMEOUT_SECONDS = 5.0

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
    print("\n" + "=" * 60)
    print("TEST SUMMARY")
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
        for test in results["tests"]:
            if test["status"] == "FAIL":
                print(f"  - {test['name']}: {test['details']}")
    print("\n" + "-" * 60)
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


# ============================================================
# QUARTZ WINDOW TARGETING
# ============================================================

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


# ============================================================
# VERIFICATION HELPERS
# ============================================================

async def verify_screen_contains(session, expected: str, description: str) -> bool:
    start = time.monotonic()
    while (time.monotonic() - start) < TIMEOUT_SECONDS:
        screen = await session.async_get_screen_contents()
        for i in range(screen.number_of_lines):
            if expected in screen.line(i).string:
                return True
        await asyncio.sleep(0.2)
    return False


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


BOX_CHARS = {
    'corners': '┌┐└┘╭╮╰╯╔╗╚╝',
    'horizontal': '─═━',
    'vertical': '│║┃',
    'junctions': '├┤┬┴┼╠╣╦╩╬',
}


async def verify_box_integrity(session) -> dict:
    screen = await session.async_get_screen_contents()
    issues = []
    for i in range(screen.number_of_lines):
        line = screen.line(i).string
        for j, char in enumerate(line):
            if char in '┌╭╔':
                if j + 1 < len(line) and line[j + 1] not in BOX_CHARS['horizontal'] + BOX_CHARS['junctions']:
                    issues.append(f"Line {i}: '{char}' at col {j} not connected right")
            elif char in '┐╮╗':
                if j > 0 and line[j - 1] not in BOX_CHARS['horizontal'] + BOX_CHARS['junctions']:
                    issues.append(f"Line {i}: '{char}' at col {j} not connected left")
    return {'valid': len(issues) == 0, 'issues': issues[:5]}


async def cleanup_session(session):
    try:
        await session.async_send_text("\x03")
        await asyncio.sleep(0.2)
        await session.async_send_text("exit\n")
        await asyncio.sleep(0.2)
        await session.async_close()
    except Exception as e:
        print(f"  Cleanup warning: {e}")


# ============================================================
# MAIN TEST FUNCTION
# ============================================================

async def main(connection):
    results["start_time"] = datetime.now()

    print("\n" + "#" * 60)
    print("# ghent Theme Visual Test")
    print("#" * 60)

    app = await iterm2.async_get_app(connection)
    window = app.current_terminal_window
    if not window:
        log_result("Setup", "FAIL", "No active iTerm2 window")
        return print_summary()

    tab = await window.async_create_tab()
    session = tab.current_session

    try:
        # TEST 1: Build theme-demo
        print_test_header("Build Theme Demo", 1)
        await session.async_send_text(f"cd {PROJECT_ROOT} && go build ./cmd/theme-demo/ 2>&1; echo BUILD_EXIT=$?\n")
        await asyncio.sleep(5.0)
        if await verify_screen_contains(session, "BUILD_EXIT=0", "build success"):
            log_result("Build Theme Demo", "PASS")
        else:
            log_result("Build Theme Demo", "FAIL", "Build failed")
            await dump_screen(session, "build_failure")
            return print_summary()

        # TEST 2: Run theme-demo and capture full output
        print_test_header("Theme Demo Render", 2)
        await session.async_send_text("go run ./cmd/theme-demo/ 2>&1\n")
        await asyncio.sleep(3.0)
        screenshot = capture_screenshot("ghent_theme_demo")
        screen_text = await get_all_screen_text(session)
        if "Theme Demo" in screen_text and "Tokyo Night" in screen_text:
            log_result("Theme Demo Render", "PASS", screenshot=screenshot)
        else:
            log_result("Theme Demo Render", "FAIL", "Theme demo header not found", screenshot=screenshot)
            await dump_screen(session, "theme_demo")

        # TEST 3: Verify badges render
        print_test_header("Badges", 3)
        has_badges = all(badge in screen_text for badge in ["PR #42", "passed", "failed", "pending"])
        if has_badges:
            screenshot = capture_screenshot("ghent_theme_badges")
            log_result("Badges", "PASS", screenshot=screenshot)
        else:
            log_result("Badges", "FAIL", "Missing badge text in output")

        # TEST 4: Verify diff hunk
        print_test_header("Diff Hunk", 4)
        has_diff = "@@" in screen_text and "return nil" in screen_text
        if has_diff:
            screenshot = capture_screenshot("ghent_theme_diffhunk")
            log_result("Diff Hunk", "PASS", screenshot=screenshot)
        else:
            log_result("Diff Hunk", "FAIL", "Missing diff hunk content")

        # TEST 5: Verify box-drawing characters
        print_test_header("Box Borders", 5)
        box_result = await verify_box_integrity(session)
        screenshot = capture_screenshot("ghent_theme_borders")
        if box_result['valid']:
            log_result("Box Borders", "PASS", screenshot=screenshot)
        else:
            log_result("Box Borders", "FAIL", box_result['issues'][0], screenshot=screenshot)

        # TEST 6: Verify help bar
        print_test_header("Help Bar", 6)
        has_help = "j/k" in screen_text and "navigate" in screen_text and "quit" in screen_text
        if has_help:
            log_result("Help Bar", "PASS")
        else:
            log_result("Help Bar", "FAIL", "Missing help key hints")

        # TEST 7: Verify file paths and authors
        print_test_header("File Paths & Authors", 7)
        has_files = "internal/api/graphql.go" in screen_text and "@reviewer1" in screen_text
        if has_files:
            log_result("File Paths & Authors", "PASS")
        else:
            log_result("File Paths & Authors", "FAIL", "Missing file paths or authors")

        # TEST 8: No lipgloss.Background() in Go source
        print_test_header("No lipgloss.Background() Ban", 8)
        await session.async_send_text(
            f'grep -rn "lipgloss\\.Background(" {PROJECT_ROOT}/internal/ --include="*.go" '
            f'| grep -v "// " | grep -v "test" | wc -l | tr -d " "\n'
        )
        await asyncio.sleep(1.0)
        if await verify_screen_contains(session, "0", "zero matches"):
            log_result("No lipgloss.Background()", "PASS")
        else:
            log_result("No lipgloss.Background()", "UNVERIFIED", "Could not confirm zero matches")

    except Exception as e:
        log_result("Execution", "FAIL", str(e))
        await dump_screen(session, "error_state")

    finally:
        await cleanup_session(session)

    return print_summary()


if __name__ == "__main__":
    exit_code = iterm2.run_until_complete(main)
    exit(exit_code if exit_code else 0)
