# /// script
# requires-python = ">=3.14"
# dependencies = [
#   "iterm2",
#   "pyobjc",
#   "pyobjc-framework-Quartz",
# ]
# ///

"""
Exhaustive visual and live-PR verification for ghent --await-review.

Real PR targets:
    - indrasvat/yathaavat#1 : existing multi-bot review activity, should settle high
    - indrasvat/doot#1      : quiet PR, short timeout should remain provisional/low

Scenarios covered:
    1. Build + install ghent
    2. Pipe mode timeout path returns review_monitor timeout/low
    3. Pipe mode settled path returns review_monitor settled/high
    4. Compatibility alias review_settled still present
    5. TUI initial watch screen renders "watching"
    6. TUI review-await screen renders "awaiting reviews"
    7. TUI tail confirmation renders "confirming review quiet"
    8. TUI settled summary renders "Review activity stabilized"
    9. TUI timeout summary renders "Review monitor provisional"

Screenshots:
    - ghent_await_review_ci_phase.png
    - ghent_await_review_awaiting.png
    - ghent_await_review_tail_settled.png
    - ghent_await_review_summary.png
    - ghent_await_review_timeout_warning.png
"""

from __future__ import annotations

import asyncio
import json
import os
import subprocess
import sys
import time
from dataclasses import dataclass
from datetime import datetime
from typing import Any

import iterm2

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
SCREENSHOT_DIR = os.path.join(PROJECT_ROOT, ".claude", "screenshots")
SETTLED_REPO = "indrasvat/yathaavat"
SETTLED_PR = 1
TIMEOUT_REPO = "indrasvat/doot"
TIMEOUT_PR = 1
PIPE_TIMEOUT_SECONDS = 240
SCREEN_TIMEOUT_SECONDS = 150
WINDOW_WIDTH = 1180
WINDOW_HEIGHT = 760
SESSION_PREFIX = "ghent-await-review-"


@dataclass
class TestResult:
    name: str
    status: str
    detail: str = ""
    screenshot: str | None = None


RESULTS: list[TestResult] = []


def record(name: str, status: str, detail: str = "", screenshot: str | None = None) -> None:
    RESULTS.append(TestResult(name=name, status=status, detail=detail, screenshot=screenshot))
    symbol = {"PASS": "+", "FAIL": "x"}.get(status, "?")
    print(f"[{symbol}] {status}: {name}")
    if detail:
        print(f"    {detail}")
    if screenshot:
        print(f"    screenshot: {screenshot}")


def fail_and_return(name: str, detail: str) -> int:
    record(name, "FAIL", detail)
    return 1


def run_json(command: list[str], *, timeout: int = PIPE_TIMEOUT_SECONDS) -> tuple[int, str, str]:
    proc = subprocess.run(
        command,
        cwd=PROJECT_ROOT,
        capture_output=True,
        text=True,
        timeout=timeout,
    )
    return proc.returncode, proc.stdout, proc.stderr


def parse_status_json(stdout: str) -> dict:
    payload = stdout.strip()
    if not payload:
        raise ValueError("no stdout payload to parse")
    return json.loads(payload)


try:
    import Quartz

    def find_quartz_window_id(target_x: float, target_w: float, target_h: float, tolerance: int = 30):
        window_list = Quartz.CGWindowListCopyWindowInfo(
            Quartz.kCGWindowListOptionOnScreenOnly
            | Quartz.kCGWindowListExcludeDesktopElements,
            Quartz.kCGNullWindowID,
        )
        best_id, best_score = None, float("inf")
        for window in window_list:
            if "iTerm" not in window.get("kCGWindowOwnerName", ""):
                continue
            bounds = window.get("kCGWindowBounds", {})
            score = (
                abs(float(bounds.get("X", 0)) - target_x) * 2
                + abs(float(bounds.get("Width", 0)) - target_w)
                + abs(float(bounds.get("Height", 0)) - target_h)
            )
            if score < best_score:
                best_score, best_id = score, window.get("kCGWindowNumber")
        return best_id if best_score < tolerance else None

except ImportError:
    Quartz = None

    def find_quartz_window_id(target_x: float, target_w: float, target_h: float, tolerance: int = 30):
        return None


async def capture_screenshot(window, name: str) -> str:
    os.makedirs(SCREENSHOT_DIR, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    filename = f"{name}_{timestamp}.png"
    filepath = os.path.join(SCREENSHOT_DIR, filename)

    frame = await window.async_get_frame()
    window_id = find_quartz_window_id(frame.origin.x, frame.size.width, frame.size.height)
    if window_id is not None:
        subprocess.run(["screencapture", "-x", "-l", str(window_id), filepath], check=True)
    else:
        subprocess.run(["screencapture", "-x", filepath], check=True)

    return filepath


async def get_screen_text(session) -> str:
    screen = await session.async_get_screen_contents()
    return "\n".join(screen.line(i).string for i in range(screen.number_of_lines))


async def wait_for_text(
    session,
    needle: str,
    *,
    timeout: float = SCREEN_TIMEOUT_SECONDS,
    interval: float = 0.5,
) -> bool:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        if needle.lower() in (await get_screen_text(session)).lower():
            return True
        await asyncio.sleep(interval)
    return False


async def wait_for_any_text(
    session,
    needles: list[str],
    *,
    timeout: float = SCREEN_TIMEOUT_SECONDS,
    interval: float = 0.5,
) -> str | None:
    deadline = time.monotonic() + timeout
    lowered_needles = [needle.lower() for needle in needles]
    while time.monotonic() < deadline:
        text = (await get_screen_text(session)).lower()
        for needle in lowered_needles:
            if needle in text:
                return needle
        await asyncio.sleep(interval)
    return None


async def dump_screen(session, label: str) -> None:
    print(f"\n==== SCREEN DUMP: {label} ====")
    screen = await get_screen_text(session)
    print(screen)
    print("==== END SCREEN DUMP ====\n")


async def cleanup_session(session) -> None:
    try:
        await session.async_send_text("\u0003")
    except Exception:  # noqa: BLE001
        pass
    try:
        await session.async_send_text("q")
    except Exception:  # noqa: BLE001
        pass
    try:
        await session.async_send_text("exit\n")
    except Exception:  # noqa: BLE001
        pass
    await asyncio.sleep(0.2)
    try:
        await session.async_close()
    except Exception:  # noqa: BLE001
        pass


async def cleanup_stale_windows(connection, *, prefix: str = SESSION_PREFIX) -> None:
    app = await iterm2.async_get_app(connection)
    for window in app.terminal_windows:
        for tab in window.tabs:
            for session in tab.sessions:
                if session.name and session.name.startswith(prefix):
                    await cleanup_session(session)


async def create_test_window(connection, *, name: str, x_pos: int) -> tuple[Any, Any]:
    window = await iterm2.Window.async_create(connection)
    if window is None:
        raise RuntimeError("Window.async_create() returned None")

    await asyncio.sleep(0.5)
    app = await iterm2.async_get_app(connection)
    if window.current_tab is None:
        for refreshed in app.terminal_windows:
            if refreshed.window_id == window.window_id:
                window = refreshed
                break

    session = None
    for _ in range(30):
        if window.current_tab and window.current_tab.current_session:
            session = window.current_tab.current_session
            break
        await asyncio.sleep(0.2)
        app = await iterm2.async_get_app(connection)
        for refreshed in app.terminal_windows:
            if refreshed.window_id == window.window_id:
                window = refreshed
                break
    if session is None:
        raise RuntimeError("window session not ready after refresh + probe")

    frame = await window.async_get_frame()
    frame.origin.x = x_pos
    frame.origin.y = 80
    frame.size.width = WINDOW_WIDTH
    frame.size.height = WINDOW_HEIGHT
    await window.async_set_frame(frame)
    await asyncio.sleep(0.3)

    await session.async_set_name(name)
    screen = await session.async_get_screen_contents()
    if screen is None:
        raise RuntimeError("screen not readable after window creation")

    return window, session


async def run_tui_settled_scenario(connection) -> int:
    test_name = "TUI settled/high-confidence path"
    window, session = await create_test_window(
        connection,
        name=f"{SESSION_PREFIX}settled",
        x_pos=80,
    )
    try:
        cmd = (
            f"cd {PROJECT_ROOT} && "
            f"gh ghent status -R {SETTLED_REPO} --pr {SETTLED_PR} "
            "--await-review --solo --logs\n"
        )
        await session.async_send_text(cmd)

        first_render = await wait_for_any_text(
            session,
            ["watching", "awaiting reviews", "event log"],
            timeout=15,
            interval=0.2,
        )
        if first_render is None:
            await dump_screen(session, "initial watch render not found")
            return fail_and_return(test_name, "initial watch screen never rendered")

        screenshot = await capture_screenshot(window, "ghent_await_review_ci_phase")
        record(
            "TUI CI phase screenshot",
            "PASS",
            f"captured initial watch render ({first_render})",
            screenshot,
        )

        if not await wait_for_text(session, "awaiting reviews", timeout=25):
            await dump_screen(session, "awaiting reviews not found")
            return fail_and_return(test_name, "never reached awaiting reviews")

        screenshot = await capture_screenshot(window, "ghent_await_review_awaiting")
        record("TUI awaiting review screenshot", "PASS", "awaiting reviews detected", screenshot)

        if not await wait_for_text(session, "confirming review quiet", timeout=70):
            await dump_screen(session, "confirming review quiet not found")
            return fail_and_return(test_name, "never reached tail confirmation")

        screenshot = await capture_screenshot(window, "ghent_await_review_tail_settled")
        record("TUI tail confirmation screenshot", "PASS", "tail confirmation detected", screenshot)

        if not await wait_for_text(session, "Review activity stabilized", timeout=100):
            await dump_screen(session, "review activity stabilized not found")
            return fail_and_return(test_name, "never reached settled status summary")

        screenshot = await capture_screenshot(window, "ghent_await_review_summary")
        record(
            test_name,
            "PASS",
            "status summary rendered settled/high-confidence banner",
            screenshot,
        )
        return 0
    finally:
        await cleanup_session(session)


async def run_tui_timeout_scenario(connection) -> int:
    test_name = "TUI timeout/low-confidence path"
    window, session = await create_test_window(
        connection,
        name=f"{SESSION_PREFIX}timeout",
        x_pos=1400,
    )
    try:
        cmd = (
            f"cd {PROJECT_ROOT} && "
            f"gh ghent status -R {TIMEOUT_REPO} --pr {TIMEOUT_PR} "
            "--await-review --review-timeout 5s --solo --logs\n"
        )
        await session.async_send_text(cmd)

        if not await wait_for_text(session, "Review monitor provisional", timeout=30):
            await dump_screen(session, "review monitor provisional not found")
            return fail_and_return(test_name, "timeout warning banner never appeared")

        screenshot = await capture_screenshot(window, "ghent_await_review_timeout_warning")
        record(
            test_name,
            "PASS",
            "status summary rendered provisional timeout banner",
            screenshot,
        )
        return 0
    finally:
        await cleanup_session(session)


def run_pipe_tests() -> int:
    command = [
        "gh",
        "ghent",
        "status",
        "-R",
        TIMEOUT_REPO,
        "--pr",
        str(TIMEOUT_PR),
        "--await-review",
        "--review-timeout",
        "5s",
        "--solo",
        "--logs",
        "--format",
        "json",
        "--no-tui",
    ]
    code, stdout, stderr = run_json(command)
    if code != 0:
        return fail_and_return("Pipe timeout path", f"exit {code}: {stderr.strip()}")
    try:
        payload = parse_status_json(stdout)
    except Exception as exc:  # noqa: BLE001
        return fail_and_return("Pipe timeout path", f"failed to parse JSON: {exc}")

    monitor = payload.get("review_monitor", {})
    alias = payload.get("review_settled", {})
    if monitor.get("phase") != "timeout" or monitor.get("confidence") != "low":
        return fail_and_return("Pipe timeout path", f"unexpected review_monitor: {monitor}")
    if alias.get("phase") != "timeout" or alias.get("confidence") != "low":
        return fail_and_return("Pipe timeout alias", f"unexpected review_settled: {alias}")
    record("Pipe timeout path", "PASS", json.dumps(monitor, sort_keys=True))

    command = [
        "gh",
        "ghent",
        "status",
        "-R",
        SETTLED_REPO,
        "--pr",
        str(SETTLED_PR),
        "--await-review",
        "--solo",
        "--logs",
        "--format",
        "json",
        "--no-tui",
    ]
    code, stdout, stderr = run_json(command)
    if code != 0:
        return fail_and_return("Pipe settled path", f"exit {code}: {stderr.strip()}")
    try:
        payload = parse_status_json(stdout)
    except Exception as exc:  # noqa: BLE001
        return fail_and_return("Pipe settled path", f"failed to parse JSON: {exc}")

    monitor = payload.get("review_monitor", {})
    alias = payload.get("review_settled", {})
    if monitor.get("phase") != "settled" or monitor.get("confidence") != "high":
        return fail_and_return("Pipe settled path", f"unexpected review_monitor: {monitor}")
    if monitor.get("tail_probes", 0) < 2:
        return fail_and_return("Pipe settled path", f"expected tail_probes >= 2, got {monitor}")
    if alias.get("phase") != "settled" or alias.get("confidence") != "high":
        return fail_and_return("Pipe settled alias", f"unexpected review_settled: {alias}")
    record("Pipe settled path", "PASS", json.dumps(monitor, sort_keys=True))
    return 0


def install_binary() -> int:
    proc = subprocess.run(
        ["make", "install"],
        cwd=PROJECT_ROOT,
        capture_output=True,
        text=True,
    )
    if proc.returncode != 0:
        return fail_and_return("Build/install", proc.stderr or proc.stdout)
    record("Build/install", "PASS", "make install succeeded")
    return 0


def summarize_and_exit() -> int:
    passed = sum(1 for result in RESULTS if result.status == "PASS")
    failed = sum(1 for result in RESULTS if result.status == "FAIL")
    print("\n============================================================")
    print("ghent --await-review visual/live verification")
    print("============================================================")
    print(f"passed: {passed}")
    print(f"failed: {failed}")
    if failed:
        print("OVERALL: FAILED")
        return 1
    print("OVERALL: PASSED")
    return 0


async def main(connection) -> int:
    await cleanup_stale_windows(connection)

    if install_binary() != 0:
        return summarize_and_exit()

    if run_pipe_tests() != 0:
        return summarize_and_exit()

    if await run_tui_settled_scenario(connection) != 0:
        return summarize_and_exit()

    if await run_tui_timeout_scenario(connection) != 0:
        return summarize_and_exit()

    return summarize_and_exit()


if __name__ == "__main__":
    os.makedirs(SCREENSHOT_DIR, exist_ok=True)
    sys.exit(iterm2.run_until_complete(main))
