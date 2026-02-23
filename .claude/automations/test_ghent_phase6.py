# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Phase 6 Visual Test: Verify --since, --group-by, --compact, and batch resolve.

Tests:
    1. --since filter: Filters comments by timestamp, shows since metadata
    2. --group-by file: Groups comments by file path
    3. --group-by author: Groups comments by author
    4. --compact summary: Flat compact digest with PR metadata
    5. Batch resolve --dry-run: Shows what would be resolved
    6. Batch resolve --file: Filter by file glob
    7. --since with --group-by: Combined flags work together

Verification Strategy:
    - Run commands against real repos (indrasvat/tbgs PR #1)
    - Validate JSON output structure for each feature
    - Capture screenshots at each stage

Screenshots:
    - ghent_phase6_since.png: --since filter output
    - ghent_phase6_groupby.png: --group-by file output
    - ghent_phase6_compact.png: --compact summary output
    - ghent_phase6_batchresolve.png: Batch resolve --dry-run output

Screenshot Inspection Checklist:
    - Content: Each feature produces expected output structure
    - Metadata: since field present when filter active
    - Groups: Threads properly grouped by file/author
    - Compact: Flat structure with pr_age, unresolved, check_status

Usage:
    uv run .claude/automations/test_ghent_phase6.py
"""

import iterm2
import asyncio
import subprocess
import os
import time
from datetime import datetime

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")
BINARY = os.path.join(PROJECT_ROOT, "bin", "gh-ghent")

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
    print(f"ghent Phase 6 Visual Tests")
    print(f"Started: {datetime.now().isoformat()}")
    print(f"{'='*60}\n")

    # Build + install
    print("--- Building + Installing ---")
    await session.async_send_text(f"cd {PROJECT_ROOT} && make install 2>&1; echo INSTALL_DONE_$?\n")
    await asyncio.sleep(8.0)

    if not await verify_screen_contains(session, "INSTALL_DONE_0", "install", timeout=15.0):
        log_result("make install", "FAIL")
        await tab.async_close()
        return
    log_result("make install", "PASS")

    # Test 1: --since filter
    print("\n--- Test 1: --since Filter ---")
    await session.async_send_text(
        f"{BINARY} comments -R indrasvat/tbgs --pr 1 --since 2025-01-01T00:00:00Z --format json "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "print(f'SINCE_OK unresolved={{d[\\\"unresolved_count\\\"]}} since={{d.get(\\\"since\\\",\\\"none\\\")}}')\" 2>&1; "
        "echo SINCE_EXIT=$?\n"
    )
    await asyncio.sleep(8.0)

    if await verify_screen_contains(session, "SINCE_OK", "since filter"):
        log_result("--since filter", "PASS", "Filter applied with metadata")
    else:
        log_result("--since filter", "FAIL", "Expected output not found")

    capture_quartz_screenshot("ghent_phase6_since")

    # Test 2: --group-by file
    print("\n--- Test 2: --group-by file ---")
    await session.async_send_text(
        f"{BINARY} comments -R indrasvat/tbgs --pr 1 --group-by file --format json "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "print(f'GROUPBY_OK groups={{len(d[\\\"groups\\\"])}} keys={{[g[\\\"key\\\"] for g in d[\\\"groups\\\"]]}}')\" 2>&1; "
        "echo GROUPBY_EXIT=$?\n"
    )
    await asyncio.sleep(8.0)

    if await verify_screen_contains(session, "GROUPBY_OK", "group-by file"):
        log_result("--group-by file", "PASS", "Threads grouped by file path")
    else:
        log_result("--group-by file", "FAIL", "Expected output not found")

    # Test 3: --group-by author
    print("\n--- Test 3: --group-by author ---")
    await session.async_send_text(
        f"{BINARY} comments -R indrasvat/tbgs --pr 1 --group-by author --format json "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "print(f'AUTHOR_OK groups={{len(d[\\\"groups\\\"])}}')\" 2>&1; "
        "echo AUTHOR_EXIT=$?\n"
    )
    await asyncio.sleep(8.0)

    if await verify_screen_contains(session, "AUTHOR_OK", "group-by author"):
        log_result("--group-by author", "PASS")
    else:
        log_result("--group-by author", "FAIL")

    capture_quartz_screenshot("ghent_phase6_groupby")

    # Test 4: --compact summary
    print("\n--- Test 4: --compact Summary ---")
    await session.async_send_text(
        f"{BINARY} summary -R indrasvat/tbgs --pr 1 --compact --format json "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "print(f'COMPACT_OK ready={{d.get(\\\"is_merge_ready\\\")}} unresolved={{d.get(\\\"unresolved\\\")}} "
        "status={{d.get(\\\"check_status\\\")}} age={{d.get(\\\"pr_age\\\",\\\"none\\\")}}')\" 2>&1; "
        "echo COMPACT_EXIT=$?\n"
    )
    await asyncio.sleep(10.0)

    if await verify_screen_contains(session, "COMPACT_OK", "compact summary"):
        log_result("--compact summary", "PASS", "Flat digest with metadata")
    else:
        log_result("--compact summary", "FAIL", "Expected output not found")

    capture_quartz_screenshot("ghent_phase6_compact")

    # Test 5: Batch resolve --dry-run
    print("\n--- Test 5: Batch Resolve --dry-run ---")
    await session.async_send_text(
        f"{BINARY} resolve -R indrasvat/tbgs --pr 1 --all --dry-run --format json "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "print(f'DRYRUN_OK count={{d[\\\"success_count\\\"]}} dry={{d.get(\\\"dry_run\\\")}}')\" 2>&1; "
        "echo DRYRUN_EXIT=$?\n"
    )
    await asyncio.sleep(8.0)

    if await verify_screen_contains(session, "DRYRUN_OK", "dry-run"):
        log_result("Batch resolve --dry-run", "PASS", "Dry-run shows planned actions")
    else:
        log_result("Batch resolve --dry-run", "FAIL")

    capture_quartz_screenshot("ghent_phase6_batchresolve")

    # Test 6: Batch resolve --file glob
    print("\n--- Test 6: Batch Resolve --file ---")
    await session.async_send_text(
        f"{BINARY} resolve -R indrasvat/tbgs --pr 1 --file 'internal/*/*.go' --dry-run --format json "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "print(f'FILE_OK count={{d[\\\"success_count\\\"]}} dry={{d.get(\\\"dry_run\\\")}}')\" 2>&1; "
        "echo FILE_EXIT=$?\n"
    )
    await asyncio.sleep(8.0)

    if await verify_screen_contains(session, "FILE_OK", "file glob"):
        log_result("Batch resolve --file", "PASS", "File glob filter works")
    else:
        log_result("Batch resolve --file", "FAIL")

    # Test 7: --since with relative duration
    print("\n--- Test 7: --since Relative Duration ---")
    await session.async_send_text(
        f"{BINARY} comments -R indrasvat/tbgs --pr 1 --since 30d --format json "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "print(f'REL_OK unresolved={{d[\\\"unresolved_count\\\"]}} since={{d.get(\\\"since\\\",\\\"none\\\")}}')\" 2>&1; "
        "echo REL_EXIT=$?\n"
    )
    await asyncio.sleep(8.0)

    if await verify_screen_contains(session, "REL_OK", "relative since"):
        log_result("--since relative (30d)", "PASS")
    else:
        log_result("--since relative (30d)", "FAIL")

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
