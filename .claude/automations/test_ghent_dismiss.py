# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Stale Review Dismissal Visual Test.

Verifies:
  1. Build/install succeeds
  2. Status TUI surfaces stale blockers in the approvals section
  3. Markdown status output recommends the dismiss command
  4. Dry-run dismiss markdown output renders the review table correctly
  5. Dry-run dismiss JSON output exposes a would_dismiss action

Default target:
  GHENT_STALE_REPO=clayliddell/AgentVM
  GHENT_STALE_PR=10

Override with environment variables when a different stale-review PR is known.
"""

import asyncio
import os
import subprocess
from datetime import datetime

import iterm2

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")
STALE_REPO = os.environ.get("GHENT_STALE_REPO", "clayliddell/AgentVM")
STALE_PR = os.environ.get("GHENT_STALE_PR", "10")

results = {
    "passed": 0,
    "failed": 0,
    "unverified": 0,
    "tests": [],
    "screenshots": [],
    "start_time": None,
}


def log_result(name: str, status: str, details: str = "", screenshot: str | None = None) -> None:
    results["tests"].append(
        {"name": name, "status": status, "details": details, "screenshot": screenshot}
    )
    if screenshot:
        results["screenshots"].append(screenshot)

    if status == "PASS":
        results["passed"] += 1
        print(f"  [+] PASS: {name}")
    elif status == "FAIL":
        results["failed"] += 1
        print(f"  [x] FAIL: {name} - {details}")
    else:
        results["unverified"] += 1
        print(f"  [?] UNVERIFIED: {name} - {details}")

    if screenshot:
        print(f"      Screenshot: {screenshot}")


def print_summary() -> int:
    end_time = datetime.now()
    total = results["passed"] + results["failed"] + results["unverified"]
    duration = (end_time - results["start_time"]).total_seconds()
    print(f"\n{'=' * 60}")
    print("TEST SUMMARY — Stale Review Dismissal")
    print(f"{'=' * 60}")
    print(f"Duration:   {duration:.1f}s")
    print(f"Total:      {total}")
    print(f"Passed:     {results['passed']}")
    print(f"Failed:     {results['failed']}")
    print(f"Unverified: {results['unverified']}")
    if results["screenshots"]:
        print("Screenshots:")
        for path in results["screenshots"]:
            print(f"  - {os.path.basename(path)}")
    print(f"{'=' * 60}")
    return 1 if results["failed"] > 0 else 0


def capture_screenshot(name: str) -> str:
    os.makedirs(SCREENSHOT_DIR, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    path = os.path.join(SCREENSHOT_DIR, f"{name}_{timestamp}.png")
    subprocess.run(["screencapture", "-x", "-o", path], check=True)
    return path


async def get_screen_text(session) -> str:
    screen = await session.async_get_screen_contents()
    return "\n".join(
        screen.line(i).string for i in range(screen.number_of_lines) if screen.line(i).string.strip()
    )


async def wait_for_text(session, text: str, timeout: float = 10.0) -> bool:
    deadline = asyncio.get_event_loop().time() + timeout
    while asyncio.get_event_loop().time() < deadline:
        if text in await get_screen_text(session):
            return True
        await asyncio.sleep(0.5)
    return False


async def cleanup(session) -> None:
    try:
        await session.async_send_text("\x03")
        await asyncio.sleep(0.3)
        await session.async_send_text("q")
        await asyncio.sleep(0.3)
        await session.async_send_text("exit\n")
        await asyncio.sleep(0.2)
    except Exception as exc:  # pragma: no cover - best effort cleanup
        print(f"  Cleanup warning: {exc}")


async def main(connection):
    results["start_time"] = datetime.now()

    print(f"\n{'#' * 60}")
    print("# ghent Stale Review Dismissal — Visual Test Suite")
    print(f"# target: {STALE_REPO}#{STALE_PR}")
    print(f"{'#' * 60}")

    app = await iterm2.async_get_app(connection)
    window = app.current_terminal_window
    if not window:
        log_result("Setup", "FAIL", "No active iTerm2 window")
        return print_summary()

    tab = await window.async_create_tab()
    session = tab.current_session

    try:
        await session.async_send_text(f"cd {PROJECT_ROOT} && make install 2>&1; echo BUILD_EXIT=$?\n")
        if await wait_for_text(session, "BUILD_EXIT=0", timeout=20.0):
            log_result("Build & Install", "PASS")
        else:
            screenshot = capture_screenshot("ghent_dismiss_build_failure")
            log_result("Build & Install", "FAIL", "make install failed", screenshot)
            return print_summary()

        await session.async_send_text(f"gh ghent status -R {STALE_REPO} --pr {STALE_PR} 2>&1\n")
        await asyncio.sleep(10.0)
        screen_text = await get_screen_text(session)
        screenshot = capture_screenshot("ghent_dismiss_status_tui")
        has_stale_header = "stale" in screen_text.lower()
        has_approvals = "Approvals" in screen_text
        has_stale_marker = "(stale)" in screen_text
        if has_stale_header and has_approvals and has_stale_marker:
            log_result(
                "Status TUI stale blocker surfacing",
                "PASS",
                "approvals section shows stale count and marker",
                screenshot,
            )
        else:
            log_result(
                "Status TUI stale blocker surfacing",
                "FAIL",
                f"approvals={has_approvals} stale_header={has_stale_header} stale_marker={has_stale_marker}",
                screenshot,
            )

        await session.async_send_text("q")
        await asyncio.sleep(1.0)

        await session.async_send_text(
            f"gh ghent status -R {STALE_REPO} --pr {STALE_PR} --format md --no-tui > /tmp/ghent_dismiss_status_md.txt 2>&1; "
            "grep -q 'Stale blocking reviews detected:' /tmp/ghent_dismiss_status_md.txt && echo STATUS_MD_STALE=1 || echo STATUS_MD_STALE=0; "
            "grep -q 'gh ghent dismiss --pr' /tmp/ghent_dismiss_status_md.txt && echo STATUS_MD_GUIDE=1 || echo STATUS_MD_GUIDE=0\n"
        )
        await asyncio.sleep(4.0)
        screen_text = await get_screen_text(session)
        screenshot = capture_screenshot("ghent_dismiss_status_md")
        if "STATUS_MD_STALE=1" in screen_text and "STATUS_MD_GUIDE=1" in screen_text:
            log_result("Status markdown guidance", "PASS", "markdown recommends dismiss", screenshot)
        else:
            log_result("Status markdown guidance", "FAIL", "missing stale review guidance", screenshot)

        await session.async_send_text(
            f"gh ghent dismiss -R {STALE_REPO} --pr {STALE_PR} --dry-run --format md --no-tui > /tmp/ghent_dismiss_md.txt 2>&1; "
            "grep -q '# Dismiss Results' /tmp/ghent_dismiss_md.txt && echo DISMISS_MD_HEADER=1 || echo DISMISS_MD_HEADER=0; "
            "grep -q 'would_dismiss' /tmp/ghent_dismiss_md.txt && echo DISMISS_MD_ACTION=1 || echo DISMISS_MD_ACTION=0; "
            "tail -20 /tmp/ghent_dismiss_md.txt\n"
        )
        await asyncio.sleep(4.0)
        screen_text = await get_screen_text(session)
        screenshot = capture_screenshot("ghent_dismiss_dry_run_md")
        if "DISMISS_MD_HEADER=1" in screen_text and "DISMISS_MD_ACTION=1" in screen_text:
            log_result("Dismiss dry-run markdown", "PASS", "markdown table rendered", screenshot)
        else:
            log_result("Dismiss dry-run markdown", "FAIL", "markdown output missing expected markers", screenshot)

        await session.async_send_text(
            f"gh ghent dismiss -R {STALE_REPO} --pr {STALE_PR} --dry-run --format json --no-tui "
            "| python3 -c \"import json,sys; d=json.load(sys.stdin); "
            "print('DISMISS_JSON_OK' if d.get('results') and d['results'][0]['action']=='would_dismiss' else 'DISMISS_JSON_BAD')\"\n"
        )
        await asyncio.sleep(4.0)
        screen_text = await get_screen_text(session)
        screenshot = capture_screenshot("ghent_dismiss_dry_run_json")
        if "DISMISS_JSON_OK" in screen_text:
            log_result("Dismiss dry-run JSON", "PASS", "JSON action verified", screenshot)
        else:
            log_result("Dismiss dry-run JSON", "FAIL", "JSON output missing would_dismiss", screenshot)

    finally:
        await cleanup(session)

    return print_summary()


if __name__ == "__main__":
    raise SystemExit(iterm2.run_until_complete(main))
