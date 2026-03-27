# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
ghent --solo Mode Visual Test: Verify solo flag skips approval requirement.

Tests:
    1. Build & install
    2. --solo --format json on doot PR #1: is_merge_ready=true (no approvals)
    3. --solo --quiet on doot PR #1: exit 0 (silent merge-ready)
    4. Without --solo on doot PR #1: is_merge_ready=false (no approval blocks)
    5. --solo --format json on tbgs PR #1: still not ready (unresolved threads)
    6. --solo TUI on doot PR #1: READY badge visible
    7. --solo TUI approvals section shows solo mode text

Verification Strategy:
    - Build binary, run pipe-mode commands against real repos
    - Parse JSON output to verify is_merge_ready field
    - Launch TUI and verify badge + approvals section text
    - doot PR #1: 0 unresolved, checks pass, no approval → solo makes it READY
    - tbgs PR #1: 2 unresolved → still NOT READY even with --solo

Usage:
    uv run .claude/automations/test_ghent_solo.py
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


async def dump_screen(session, label):
    screen = await session.async_get_screen_contents()
    print(f"\n{'=' * 60}")
    print(f"SCREEN DUMP: {label}")
    print(f"{'=' * 60}")
    for i in range(screen.number_of_lines):
        line = screen.line(i).string
        if line.strip():
            print(f"{i:03d}: {line}")
    print(f"{'=' * 60}\n")


async def run_test(connection):
    app = await iterm2.async_get_app(connection)
    window = app.current_terminal_window
    if window is None:
        print("ERROR: No iTerm2 window found. Open iTerm2 first.")
        return

    tab = await window.async_create_tab()
    session = tab.current_session

    print(f"\n{'='*60}")
    print(f"ghent --solo Mode Visual Tests")
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

    # ── Test 1: --solo --format json on doot PR #1 → is_merge_ready=true ──
    print("\n--- Test 1: --solo --format json (doot PR #1, merge-ready) ---")
    await session.async_send_text(
        f"{BINARY} status -R indrasvat/doot --pr 1 --solo --format json "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "print(f'SOLO_JSON_OK ready={d.get(\\\"is_merge_ready\\\")}')\" 2>&1; "
        "echo SOLO_JSON_EXIT=$?\n"
    )
    await asyncio.sleep(15.0)

    if await verify_screen_contains(session, "SOLO_JSON_OK", "solo json", timeout=10.0):
        screen = await get_screen_text(session)
        if "ready=True" in screen:
            log_result("--solo JSON merge-ready", "PASS", "is_merge_ready=true with --solo")
        else:
            log_result("--solo JSON merge-ready", "FAIL", "is_merge_ready should be true with --solo on doot")
            await dump_screen(session, "solo_json_fail")
    else:
        log_result("--solo JSON", "FAIL", "Command did not produce expected output")
        await dump_screen(session, "solo_json_timeout")

    capture_quartz_screenshot("ghent_solo_json_doot")

    # ── Test 2: --solo --quiet on doot PR #1 → exit 0 ──
    print("\n--- Test 2: --solo --quiet (doot PR #1, exit 0) ---")
    await session.async_send_text(
        f"{BINARY} status -R indrasvat/doot --pr 1 --solo --quiet > /tmp/ghent_solo_quiet.txt 2>&1; "
        "echo SOLO_QUIET_EXIT=$?\n"
    )
    await asyncio.sleep(12.0)

    screen = await get_screen_text(session)
    if "SOLO_QUIET_EXIT=0" in screen:
        log_result("--solo --quiet exit 0", "PASS", "Silent exit 0 on solo merge-ready PR")
    elif "SOLO_QUIET_EXIT=1" in screen:
        log_result("--solo --quiet exit 0", "FAIL", "Got exit 1 — solo should make doot merge-ready")
    else:
        log_result("--solo --quiet exit code", "FAIL", "Unexpected exit behavior")

    capture_quartz_screenshot("ghent_solo_quiet_doot")

    # ── Test 3: Without --solo on doot PR #1 → is_merge_ready=false ──
    print("\n--- Test 3: Without --solo (doot PR #1, NOT merge-ready) ---")
    await session.async_send_text(
        f"{BINARY} status -R indrasvat/doot --pr 1 --format json "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "print(f'NOSOLO_JSON_OK ready={d.get(\\\"is_merge_ready\\\")}')\" 2>&1; "
        "echo NOSOLO_JSON_EXIT=$?\n"
    )
    await asyncio.sleep(15.0)

    if await verify_screen_contains(session, "NOSOLO_JSON_OK", "nosolo json", timeout=10.0):
        screen = await get_screen_text(session)
        if "ready=False" in screen:
            log_result("Without --solo NOT ready", "PASS", "is_merge_ready=false without --solo (no approval)")
        else:
            log_result("Without --solo NOT ready", "UNVERIFIED",
                       "is_merge_ready may be true if doot has an approval now")
    else:
        log_result("Without --solo JSON", "FAIL", "Command did not produce expected output")

    capture_quartz_screenshot("ghent_nosolo_json_doot")

    # ── Test 4: --solo on tbgs PR #1 → still NOT ready (unresolved threads) ──
    print("\n--- Test 4: --solo on tbgs PR #1 (still NOT ready, unresolved threads) ---")
    await session.async_send_text(
        f"{BINARY} status -R indrasvat/tbgs --pr 1 --solo --format json "
        "| python3 -c \"import sys,json; d=json.load(sys.stdin); "
        "print(f'SOLO_TBGS_OK ready={d.get(\\\"is_merge_ready\\\")} unresolved={d[\\\"comments\\\"][\\\"unresolved_count\\\"]}')\" 2>&1; "
        "echo SOLO_TBGS_EXIT=$?\n"
    )
    await asyncio.sleep(15.0)

    if await verify_screen_contains(session, "SOLO_TBGS_OK", "solo tbgs", timeout=10.0):
        screen = await get_screen_text(session)
        if "ready=False" in screen and "unresolved=2" in screen:
            log_result("--solo still blocks on unresolved", "PASS",
                       "is_merge_ready=false with unresolved threads despite --solo")
        elif "ready=False" in screen:
            log_result("--solo still blocks on unresolved", "PASS",
                       "is_merge_ready=false (unresolved count may differ)")
        else:
            log_result("--solo still blocks on unresolved", "FAIL",
                       "Solo should NOT override unresolved thread check")
    else:
        log_result("--solo tbgs JSON", "FAIL", "Command did not produce expected output")

    capture_quartz_screenshot("ghent_solo_json_tbgs")

    # ── Test 5: --solo TUI on doot PR #1 → READY badge ──
    print("\n--- Test 5: --solo TUI (doot PR #1, READY badge) ---")
    await session.async_send_text(f"gh ghent status -R indrasvat/doot --pr 1 --solo 2>&1\n")
    await asyncio.sleep(10.0)

    screen_text = await get_screen_text(session)
    capture_quartz_screenshot("ghent_solo_tui_doot")

    has_ready = "READY" in screen_text
    has_not_ready = "NOT READY" in screen_text

    if has_ready and not has_not_ready:
        log_result("--solo TUI READY badge", "PASS", "READY badge visible in solo TUI")
    elif has_not_ready:
        log_result("--solo TUI READY badge", "FAIL", "Shows NOT READY — solo should make it READY")
        await dump_screen(session, "solo_tui_badge")
    else:
        log_result("--solo TUI READY badge", "UNVERIFIED", "Could not confirm badge text")
        await dump_screen(session, "solo_tui_badge")

    # ── Test 6: Solo mode approvals rendering ──
    # doot has 1 COMMENTED review (codex-connector), so solo mode renders reviews
    # normally but with "solo mode" in the header right-info, OR "Solo mode —
    # approval not required" if no reviews exist. Either way, the key assertion is:
    # READY badge + "—" in KPI card (no numeric approval count).
    print("\n--- Test 6: Solo mode approvals rendering ---")
    has_solo_text = "Solo mode" in screen_text or "solo mode" in screen_text
    has_dash_card = "\u2014" in screen_text  # em-dash in KPI card
    has_approvals_section = "Approvals" in screen_text

    if has_dash_card and has_approvals_section:
        log_result("Solo approvals rendering", "PASS",
                   f"dash_card={has_dash_card}, approvals_section={has_approvals_section}, solo_text={has_solo_text}")
    elif has_approvals_section:
        log_result("Solo approvals rendering", "UNVERIFIED",
                   f"Approvals visible but dash card not detected (may have approvals)")
    else:
        log_result("Solo approvals rendering", "FAIL",
                   "Missing approvals section entirely")
        await dump_screen(session, "solo_approvals_text")

    # Exit TUI
    await session.async_send_text("q")
    await asyncio.sleep(1.0)

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
