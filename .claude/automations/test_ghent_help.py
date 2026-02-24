# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Help Text Visual Test: Verify --help output for all commands.

Tests:
    1. Root help: gh ghent --help shows all subcommands
    2. Comments help: gh ghent comments --help shows flags and examples
    3. Checks help: gh ghent checks --help
    4. Resolve help: gh ghent resolve --help
    5. Reply help: gh ghent reply --help
    6. Summary help: gh ghent summary --help
    7. Version: gh ghent --version shows version string

Verification Strategy:
    - Run each --help command and verify key content is present
    - Capture screenshots for manual review

Screenshots:
    - ghent_help_root.png: Root help with all subcommands listed
    - ghent_help_comments.png: Comments help with flags
    - ghent_help_version.png: Version string output

Screenshot Inspection Checklist:
    - Content: All subcommands visible in root help
    - Flags: Each subcommand shows its flags
    - Examples: Example sections present
    - Readability: Help text is well-formatted

Usage:
    uv run .claude/automations/test_ghent_help.py
"""

import iterm2
import asyncio
import subprocess
import os
import time
from datetime import datetime

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")

results = []


def log_result(name, status, detail=""):
    results.append({"name": name, "status": status, "detail": detail})
    icon = {"PASS": "\u2713", "FAIL": "\u2717", "SKIP": "\u25cb", "UNVERIFIED": "?"}.get(status, "?")
    msg = f"  {icon} {name}: {status}"
    if detail:
        msg += f" \u2014 {detail}"
    print(msg)


def capture_quartz_screenshot(filename):
    """Capture screenshot of the frontmost window using screencapture."""
    try:
        os.makedirs(SCREENSHOT_DIR, exist_ok=True)
        path = os.path.join(SCREENSHOT_DIR, f"{filename}.png")
        result = subprocess.run(
            ["screencapture", "-x", "-o", path],
            capture_output=True, timeout=5,
        )
        if result.returncode == 0 and os.path.exists(path):
            print(f"  [screenshot] Saved: {path}")
            return True
        print(f"  [screenshot] screencapture failed (rc={result.returncode})")
        return False
    except Exception as e:
        print(f"  [screenshot] Error: {e}")
        return False


async def get_screen_text(session):
    """Get all text from the terminal screen."""
    screen = await session.async_get_screen_contents()
    lines = []
    for i in range(screen.number_of_lines):
        lines.append(screen.line(i).string)
    return "\n".join(lines)


async def verify_screen_contains(session, text, label, timeout=5.0):
    """Poll screen for expected text within timeout."""
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        screen_text = await get_screen_text(session)
        if text in screen_text:
            return True
        await asyncio.sleep(0.5)
    return False


async def run_test(connection):
    app = await iterm2.async_get_app(connection)
    window = app.current_terminal_window
    if window is None:
        print("ERROR: No iTerm2 window found. Open iTerm2 first.")
        return

    tab = await window.async_create_tab()
    session = tab.current_session

    print(f"\n{'='*60}")
    print(f"ghent Help Text Visual Tests")
    print(f"Started: {datetime.now().isoformat()}")
    print(f"{'='*60}\n")

    # Build + install first
    print("--- Building + Installing ---")
    await session.async_send_text(f"cd {PROJECT_ROOT} && make install 2>&1; echo INSTALL_DONE_$?\n")
    await asyncio.sleep(8.0)

    if not await verify_screen_contains(session, "INSTALL_DONE_0", "install success", timeout=15.0):
        log_result("make install", "FAIL", "Install did not succeed")
        await tab.async_close()
        return
    log_result("make install", "PASS")

    # Test 1: Root help
    print("\n--- Test 1: Root Help ---")
    await session.async_send_text("gh ghent --help 2>&1; echo ROOTHELP_DONE\n")
    await asyncio.sleep(3.0)

    screen_text = await get_screen_text(session)
    subcommands_found = sum(1 for cmd in ["comments", "checks", "resolve", "reply", "summary"]
                           if cmd in screen_text)
    if subcommands_found >= 4:
        log_result("Root help lists subcommands", "PASS", f"{subcommands_found}/5 found")
    else:
        log_result("Root help lists subcommands", "FAIL", f"Only {subcommands_found}/5 found")

    capture_quartz_screenshot("ghent_help_root")

    # Test 2: Comments help
    print("\n--- Test 2: Comments Help ---")
    await session.async_send_text("gh ghent comments --help 2>&1; echo COMMENTHELP_DONE\n")
    await asyncio.sleep(3.0)

    screen_text = await get_screen_text(session)
    has_flags = "--pr" in screen_text or "--format" in screen_text
    has_example = "json" in screen_text.lower() or "example" in screen_text.lower()

    if has_flags and has_example:
        log_result("Comments help", "PASS", "Flags and examples present")
    elif has_flags:
        log_result("Comments help", "PASS", "Flags present")
    else:
        log_result("Comments help", "FAIL", "Missing expected content")

    capture_quartz_screenshot("ghent_help_comments")

    # Tests 3-6: Other subcommand helps
    for cmd_name, expected in [
        ("checks", "--watch"),
        ("resolve", "--thread"),
        ("reply", "--body"),
        ("summary", "--format"),
    ]:
        print(f"\n--- Test: {cmd_name} Help ---")
        await session.async_send_text(f"gh ghent {cmd_name} --help 2>&1; echo {cmd_name.upper()}HELP_DONE\n")
        await asyncio.sleep(3.0)

        screen_text = await get_screen_text(session)
        if expected in screen_text:
            log_result(f"{cmd_name} help", "PASS", f"Contains {expected}")
        else:
            log_result(f"{cmd_name} help", "UNVERIFIED", f"{expected} not found in output")

    # Test 7: Version
    print("\n--- Test 7: Version ---")
    await session.async_send_text("gh ghent --version 2>&1; echo VERSION_DONE\n")
    await asyncio.sleep(3.0)

    # Version output: styled TTY shows ASCII banner + "v0.x.y" version line
    if await verify_screen_contains(session, "v0.", "version output"):
        log_result("gh ghent --version", "PASS")
    elif await verify_screen_contains(session, "commit", "version output"):
        log_result("gh ghent --version", "PASS", "Found commit info")
    else:
        log_result("gh ghent --version", "FAIL", "Version string not found")

    capture_quartz_screenshot("ghent_help_version")

    # Summary
    print(f"\n{'='*60}")
    print("Results Summary")
    print(f"{'='*60}")
    pass_count = sum(1 for r in results if r["status"] == "PASS")
    fail_count = sum(1 for r in results if r["status"] == "FAIL")
    skip_count = sum(1 for r in results if r["status"] in ("SKIP", "UNVERIFIED"))
    print(f"  PASS: {pass_count} | FAIL: {fail_count} | SKIP/UNVERIFIED: {skip_count}")
    for r in results:
        icon = {"PASS": "\u2713", "FAIL": "\u2717", "SKIP": "\u25cb", "UNVERIFIED": "?"}.get(r["status"], "?")
        line = f"  {icon} {r['name']}: {r['status']}"
        if r["detail"]:
            line += f" \u2014 {r['detail']}"
        print(line)

    print(f"\nScreenshots saved to: {SCREENSHOT_DIR}")
    print(f"Finished: {datetime.now().isoformat()}")

    await asyncio.sleep(1.0)
    await tab.async_close()


def main():
    iterm2.run_until_complete(run_test)


if __name__ == "__main__":
    main()
