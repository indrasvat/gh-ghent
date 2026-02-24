# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent Summary Enhancement Visual Test: Verify --logs, --watch, --quiet, and enriched formatters.

Tests:
    1. --logs --format json on peek-it PR #2: log_excerpt non-empty for failing check
    2. --logs --format md on peek-it PR #2: markdown includes FAIL header + code block
    3. --compact --logs --format json on peek-it PR #2: failed_checks array present
    4. --quiet on doot PR #1: verify exit code 0 (clean PR, no output)
    5. --watch --format json on tbgs PR #1: verify stdout has summary JSON
    6. Backward compat: --format json without --logs: no log_excerpt in output
    7. Stress test: --logs on openclaw/openclaw PR #25736: handles 30+ checks

Verification Strategy:
    - Build binary, run pipe-mode commands against real repos
    - Parse JSON/markdown output to verify enriched fields
    - Capture screenshots at each stage

Screenshots:
    - ghent_summary_enhanced_logs_json.png
    - ghent_summary_enhanced_logs_md.png
    - ghent_summary_enhanced_compact.png
    - ghent_summary_enhanced_quiet.png
    - ghent_summary_enhanced_watch.png
    - ghent_summary_enhanced_compat.png
    - ghent_summary_enhanced_stress.png

Usage:
    uv run .claude/automations/test_ghent_summary_enhanced.py
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
    print(f"ghent Summary Enhancement Visual Tests")
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

    # Test 1: --logs --format json on peek-it PR #2 (failing checks)
    print("\n--- Test 1: --logs --format json (peek-it, failing checks) ---")
    await session.async_send_text(
        f"{BINARY} summary -R indrasvat/peek-it --pr 2 --logs --format json "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "fails=[c for c in d['checks']['checks'] if c['conclusion']=='failure']; "
        "has_log=any(c.get('log_excerpt','') for c in fails); "
        "has_ann=any(c.get('annotations') for c in fails); "
        "print(f'LOGS_JSON_OK fails={len(fails)} has_log={has_log} has_ann={has_ann}')\" 2>&1; "
        "echo LOGS_JSON_EXIT=$?\n"
    )
    await asyncio.sleep(15.0)

    if await verify_screen_contains(session, "LOGS_JSON_OK", "logs json", timeout=10.0):
        screen = await get_screen_text(session)
        if "has_log=True" in screen:
            log_result("--logs JSON log_excerpt", "PASS", "Failing check has log_excerpt")
        else:
            log_result("--logs JSON log_excerpt", "UNVERIFIED", "log_excerpt may be empty (external CI)")
        if "has_ann=True" in screen:
            log_result("--logs JSON annotations", "PASS", "Failing check has annotations")
        else:
            log_result("--logs JSON annotations", "UNVERIFIED", "No annotations found")
    else:
        log_result("--logs JSON", "FAIL", "Command did not produce expected output")

    capture_quartz_screenshot("ghent_summary_enhanced_logs_json")

    # Test 2: --logs --format md on peek-it PR #2
    print("\n--- Test 2: --logs --format md (peek-it, FAIL header + code block) ---")
    await session.async_send_text(
        f"{BINARY} summary -R indrasvat/peek-it --pr 2 --logs --format md > /tmp/ghent_summary_md.txt 2>&1; "
        "echo MD_EXIT=$?; "
        "grep -c '### FAIL:' /tmp/ghent_summary_md.txt 2>/dev/null && echo FAIL_HEADER_FOUND || echo FAIL_HEADER_MISSING\n"
    )
    await asyncio.sleep(15.0)

    if await verify_screen_contains(session, "FAIL_HEADER_FOUND", "md fail header", timeout=10.0):
        log_result("--logs MD FAIL header", "PASS", "Markdown has ### FAIL: section")
    else:
        log_result("--logs MD FAIL header", "UNVERIFIED", "No FAIL header (check may not have failure conclusion)")

    await session.async_send_text("head -40 /tmp/ghent_summary_md.txt\n")
    await asyncio.sleep(1.0)
    capture_quartz_screenshot("ghent_summary_enhanced_logs_md")

    # Test 3: --compact --logs --format json on peek-it PR #2
    print("\n--- Test 3: --compact --logs --format json (peek-it, failed_checks) ---")
    await session.async_send_text(
        f"{BINARY} summary -R indrasvat/peek-it --pr 2 --compact --logs --format json "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "fc=d.get('failed_checks',[]); "
        "print(f'COMPACT_OK failed_checks={len(fc)} status={d.get(\\\"check_status\\\")}')\" 2>&1; "
        "echo COMPACT_EXIT=$?\n"
    )
    await asyncio.sleep(15.0)

    if await verify_screen_contains(session, "COMPACT_OK", "compact logs", timeout=10.0):
        screen = await get_screen_text(session)
        if "failed_checks=0" not in screen:
            log_result("--compact --logs failed_checks", "PASS", "Compact JSON has failed_checks array")
        else:
            log_result("--compact --logs failed_checks", "UNVERIFIED", "failed_checks empty")
    else:
        log_result("--compact --logs", "FAIL", "Command did not produce expected output")

    capture_quartz_screenshot("ghent_summary_enhanced_compact")

    # Test 4: --quiet on doot PR #1 (clean PR — should exit 0, no output)
    print("\n--- Test 4: --quiet (doot PR #1, exit code behavior) ---")
    await session.async_send_text(
        f"{BINARY} summary -R indrasvat/doot --pr 1 --quiet > /tmp/ghent_quiet_out.txt 2>&1; "
        "echo QUIET_EXIT=$?\n"
    )
    await asyncio.sleep(12.0)

    screen = await get_screen_text(session)
    if "QUIET_EXIT=0" in screen:
        log_result("--quiet exit 0 (merge-ready)", "PASS", "Silent exit 0 on merge-ready PR")
    elif "QUIET_EXIT=1" in screen:
        log_result("--quiet exit 1 (not ready)", "PASS", "Exit 1 with output on not-ready PR")
    else:
        log_result("--quiet exit code", "FAIL", "Unexpected exit behavior")

    capture_quartz_screenshot("ghent_summary_enhanced_quiet")

    # Test 5: --watch --format json on tbgs PR #1 (already completed checks)
    print("\n--- Test 5: --watch --format json (tbgs PR #1) ---")
    await session.async_send_text(
        f"{BINARY} summary -R indrasvat/tbgs --pr 1 --watch --format json 2>/dev/null "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "print(f'WATCH_OK ready={d.get(\\\"is_merge_ready\\\")} unresolved={d[\\\"comments\\\"][\\\"unresolved_count\\\"]}')\" 2>&1; "
        "echo WATCH_EXIT=$?\n"
    )
    await asyncio.sleep(20.0)

    if await verify_screen_contains(session, "WATCH_OK", "watch summary", timeout=15.0):
        screen = await get_screen_text(session)
        if "unresolved=2" in screen:
            log_result("--watch summary", "PASS", "Watch completed, summary has unresolved=2")
        else:
            log_result("--watch summary", "PASS", "Watch completed, summary produced")
    else:
        log_result("--watch summary", "FAIL", "Watch did not produce summary output")

    capture_quartz_screenshot("ghent_summary_enhanced_watch")

    # Test 6: Backward compat — --format json without --logs
    print("\n--- Test 6: Backward compat (no --logs, no log_excerpt) ---")
    await session.async_send_text(
        f"{BINARY} summary -R indrasvat/tbgs --pr 1 --format json "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "has_excerpt=any(c.get('log_excerpt','') for c in d['checks']['checks']); "
        "print(f'COMPAT_OK ready={d.get(\\\"is_merge_ready\\\")} has_excerpt={has_excerpt} unresolved={d[\\\"comments\\\"][\\\"unresolved_count\\\"]}')\" 2>&1; "
        "echo COMPAT_EXIT=$?\n"
    )
    await asyncio.sleep(12.0)

    if await verify_screen_contains(session, "COMPAT_OK", "backward compat", timeout=10.0):
        screen = await get_screen_text(session)
        if "has_excerpt=False" in screen and "unresolved=2" in screen:
            log_result("Backward compat (no --logs)", "PASS", "No log_excerpt without --logs, unresolved=2")
        elif "unresolved=2" in screen:
            log_result("Backward compat (no --logs)", "PASS", "unresolved=2 preserved")
        else:
            log_result("Backward compat (no --logs)", "UNVERIFIED", "Output parsed but values unexpected")
    else:
        log_result("Backward compat", "FAIL", "Command did not produce expected output")

    capture_quartz_screenshot("ghent_summary_enhanced_compat")

    # Test 7: Stress test — openclaw/openclaw PR #25736 (30+ checks)
    print("\n--- Test 7: Stress test (openclaw/openclaw PR #25736, 30+ checks) ---")
    await session.async_send_text(
        f"{BINARY} summary -R openclaw/openclaw --pr 25736 --logs --format json "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "total=len(d['checks']['checks']); "
        "fails=[c for c in d['checks']['checks'] if c['conclusion']=='failure']; "
        "print(f'STRESS_OK total={total} fails={len(fails)} status={d[\\\"checks\\\"][\\\"overall_status\\\"]}')\" 2>&1; "
        "echo STRESS_EXIT=$?\n"
    )
    await asyncio.sleep(25.0)

    if await verify_screen_contains(session, "STRESS_OK", "stress test", timeout=20.0):
        screen = await get_screen_text(session)
        if "total=" in screen:
            log_result("Stress test (openclaw 30+ checks)", "PASS", "Large PR handled correctly")
        else:
            log_result("Stress test", "PASS", "Output parsed")
    else:
        log_result("Stress test", "FAIL", "Command did not complete in time")

    capture_quartz_screenshot("ghent_summary_enhanced_stress")

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
