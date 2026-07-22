from __future__ import annotations

import argparse
import json
import os
import shutil
import sys
from dataclasses import asdict, dataclass
from datetime import datetime
from pathlib import Path
from urllib.request import urlopen

from PIL import Image, ImageDraw
from selenium import webdriver
from selenium.common.exceptions import TimeoutException
from selenium.webdriver.common.by import By
from selenium.webdriver.common.action_chains import ActionChains
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support.ui import WebDriverWait


ROOT = Path(__file__).resolve().parents[1]
DEFAULT_OUTPUT = ROOT / "docs" / "refactor" / "current" / "evidence" / "pr4-04"
CUSTOMER_PATH = "/skoll/pharma-oa/customers"
DASHBOARD_PATH = "/skoll/pharma-oa/dashboard"
QUALIFICATION_PATH = "/skoll/pharma-oa/qualifications"

LOCALES = {
    "zh-CN": {
        "login": "登录",
        "scan": "到期扫描",
        "scan_confirm": "执行资质到期扫描",
    },
    "en-US": {
        "login": "Sign In",
        "scan": "Expiry scan",
        "scan_confirm": "Run qualification expiry scan",
    },
}

VIEWPORTS = {
    "desktop": (1440, 1000),
    "mobile": (390, 844),
}

STATE_ORDER = ["responsive", "loading", "empty", "error", "destructive", "saving", "no_permission"]
CURRENT_STAGE = "startup"


@dataclass
class MatrixResult:
    locale: str
    viewport: str
    state: str
    result: str
    route: str
    screenshot: str
    page_overflow: int
    text_overflows: int
    contrast_ratio: float
    unnamed_controls: int
    reduced_motion: bool
    evidence: str


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run the H4-04 bilingual responsive browser matrix.")
    parser.add_argument("--base-url", default=os.getenv("SKOLL_E2E_BASE_URL", "http://127.0.0.1:5174"))
    parser.add_argument("--admin-user", default=os.getenv("SKOLL_E2E_ADMIN_USER", "admin"))
    parser.add_argument("--admin-password", default=os.getenv("SKOLL_E2E_ADMIN_PASSWORD", "Admin@123456"))
    parser.add_argument("--restricted-user", default=os.getenv("SKOLL_E2E_RESTRICTED_USER", "dept_admin"))
    parser.add_argument("--restricted-password", default=os.getenv("SKOLL_E2E_RESTRICTED_PASSWORD", "Dept@123456"))
    parser.add_argument("--output", type=Path, default=DEFAULT_OUTPUT)
    parser.add_argument("--headed", action="store_true")
    return parser.parse_args()


def wait_for_health(base_url: str) -> None:
    with urlopen(f"{base_url.rstrip('/')}/skoll/health", timeout=10) as response:
        if response.status != 200:
            raise RuntimeError(f"health endpoint returned {response.status}")


def new_driver(headed: bool) -> webdriver.Chrome:
    options = webdriver.ChromeOptions()
    if not headed:
        options.add_argument("--headless=new")
    options.add_argument("--disable-gpu")
    options.add_argument("--no-sandbox")
    options.add_argument("--hide-scrollbars")
    options.add_argument("--force-device-scale-factor=1")
    options.set_capability("goog:loggingPrefs", {"browser": "ALL"})
    driver = webdriver.Chrome(options=options)
    driver.execute_cdp_cmd("Network.enable", {})
    driver.execute_cdp_cmd(
        "Emulation.setEmulatedMedia",
        {"features": [{"name": "prefers-reduced-motion", "value": "reduce"}]},
    )
    driver.execute_cdp_cmd(
        "Page.addScriptToEvaluateOnNewDocument",
        {
            "source": """
            (() => {
              const nativeFetch = window.fetch.bind(window);
              window.fetch = (...args) => {
                const input = args[0];
                const url = typeof input === 'string' ? input : String(input?.url || '');
                const failurePattern = localStorage.getItem('skoll.h4.failurePattern') || '';
                if (failurePattern && url.includes(failurePattern)) {
                  return Promise.reject(new TypeError('H4 matrix simulated network failure'));
                }
                const delayPattern = localStorage.getItem('skoll.h4.delayPattern') || '';
                const delayMs = Number(localStorage.getItem('skoll.h4.delayMs') || '0');
                if (delayPattern && delayMs > 0 && url.includes(delayPattern)) {
                  return new Promise((resolve, reject) => {
                    setTimeout(() => nativeFetch(...args).then(resolve, reject), delayMs);
                  });
                }
                return nativeFetch(...args);
              };
            })();
            """
        },
    )
    return driver


def reset_network(driver: webdriver.Chrome) -> None:
    driver.execute_cdp_cmd("Network.setBlockedURLs", {"urls": []})
    driver.execute_cdp_cmd(
        "Network.emulateNetworkConditions",
        {"offline": False, "latency": 0, "downloadThroughput": -1, "uploadThroughput": -1},
    )


def configure_fetch(driver: webdriver.Chrome, delay_pattern: str = "", failure_pattern: str = "", delay_ms: int = 10000) -> None:
    driver.execute_script(
        """
        localStorage.setItem('skoll.h4.delayPattern', arguments[0]);
        localStorage.setItem('skoll.h4.failurePattern', arguments[1]);
        localStorage.setItem('skoll.h4.delayMs', String(arguments[2]));
        """,
        delay_pattern,
        failure_pattern,
        delay_ms,
    )


def clear_fetch_configuration(driver: webdriver.Chrome) -> None:
    driver.execute_script(
        "localStorage.removeItem('skoll.h4.delayPattern'); localStorage.removeItem('skoll.h4.failurePattern'); localStorage.removeItem('skoll.h4.delayMs');"
    )


def announce(locale: str, viewport: str, state: str) -> None:
    global CURRENT_STAGE
    CURRENT_STAGE = f"{locale}/{viewport}/{state}"
    print(f"[RUN] {locale}/{viewport}/{state}", flush=True)


def clear_session(driver: webdriver.Chrome, base_url: str, locale: str) -> None:
    driver.get(f"{base_url}/skoll/login")
    driver.execute_script("localStorage.clear(); localStorage.setItem('skoll.ui.locale', arguments[0]);", locale)
    driver.refresh()


def login(driver: webdriver.Chrome, wait: WebDriverWait, base_url: str, locale: str, account: str, password: str) -> None:
    clear_session(driver, base_url, locale)
    inputs = wait.until(
        lambda current: [
            element
            for element in current.find_elements(By.CSS_SELECTOR, "main input[autocomplete='username'], main input[autocomplete='current-password']")
            if element.is_displayed()
        ]
    )
    if len(inputs) != 2:
        raise RuntimeError(f"expected two login inputs, got {len(inputs)}")
    for element, value in zip(inputs, (account, password), strict=True):
        element.send_keys(Keys.CONTROL, "a")
        element.send_keys(value)
    buttons = driver.find_elements(By.CSS_SELECTOR, "main button")
    if len(buttons) != 1:
        raise RuntimeError(f"expected one login button, got {len(buttons)}")
    buttons[0].click()
    wait.until(lambda current: "/login" not in current.current_url)


def open_page(driver: webdriver.Chrome, wait: WebDriverWait, base_url: str, path: str) -> None:
    driver.get(f"{base_url}{path}")
    wait.until(lambda current: current.find_elements(By.CSS_SELECTOR, ".page-shell"))


def wait_idle(driver: webdriver.Chrome, wait: WebDriverWait) -> None:
    wait.until(
        lambda current: current.find_elements(By.CSS_SELECTOR, ".page-shell")
        and current.find_element(By.CSS_SELECTOR, ".page-shell").get_attribute("aria-busy") != "true"
    )


def wait_visible(driver: webdriver.Chrome, wait: WebDriverWait, selector: str) -> None:
    wait.until(lambda current: any(element.is_displayed() for element in current.find_elements(By.CSS_SELECTOR, selector)))


def layout_metrics(driver: webdriver.Chrome) -> dict[str, object]:
    return driver.execute_script(
        r"""
        const root = document.documentElement;
        const visible = (element) => {
          const style = getComputedStyle(element);
          const rect = element.getBoundingClientRect();
          return style.display !== 'none' && style.visibility !== 'hidden' && rect.width > 0 && rect.height > 0;
        };
        const selectors = 'button,.el-tag,h1,h2,h3,.state-block p,.page-shell__description';
        const textOverflows = [...document.querySelectorAll(selectors)]
          .filter(visible)
          .filter((element) => element.scrollWidth > element.clientWidth + 2 || element.scrollHeight > element.clientHeight + 2)
          .slice(0, 20)
          .map((element) => ({ tag: element.tagName, text: (element.textContent || '').trim().slice(0, 80) }));
        const parseColor = (value) => {
          const match = String(value || '').match(/[\d.]+/g);
          return match && match.length >= 3 ? match.slice(0, 3).map(Number) : [0, 0, 0];
        };
        const luminance = (rgb) => {
          const channels = rgb.map((value) => {
            const normalized = value / 255;
            return normalized <= 0.03928 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4;
          });
          return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
        };
        const bodyStyle = getComputedStyle(document.body);
        const foreground = luminance(parseColor(bodyStyle.color));
        const background = luminance(parseColor(bodyStyle.backgroundColor));
        const contrastRatio = (Math.max(foreground, background) + 0.05) / (Math.min(foreground, background) + 0.05);
        const controls = [...document.querySelectorAll('button,a[href],input,textarea,select,[role="button"]')].filter(visible);
        const accessibleName = (element) => {
          const id = element.getAttribute('id');
          const label = id ? document.querySelector(`label[for="${CSS.escape(id)}"]`) : null;
          const wrappingLabel = element.closest('label');
          return [element.getAttribute('aria-label'), element.getAttribute('title'), label?.textContent, wrappingLabel?.textContent, element.textContent, element.getAttribute('placeholder')]
            .some((value) => String(value || '').trim().length > 0);
        };
        const unnamedControls = controls.filter((element) => !accessibleName(element)).map((element) => element.outerHTML.slice(0, 180));
        const pageShell = document.querySelector('.page-shell');
        const labelledBy = pageShell?.getAttribute('aria-labelledby') || '';
        const labelledHeading = labelledBy ? document.getElementById(labelledBy) : null;
        const motionProbe = document.querySelector('.el-button') || document.body;
        const motionStyle = getComputedStyle(motionProbe);
        return {
          pageOverflow: Math.max(0, root.scrollWidth - root.clientWidth),
          textOverflows,
          lang: root.lang,
          labelledPage: Boolean(labelledHeading && String(labelledHeading.textContent || '').trim()),
          unnamedControls,
          contrastRatio,
          reducedMotion: matchMedia('(prefers-reduced-motion: reduce)').matches,
          transitionDuration: motionStyle.transitionDuration,
          animationDuration: motionStyle.animationDuration,
          viewport: { width: root.clientWidth, height: root.clientHeight }
        };
        """
    )


def verify_keyboard_focus(driver: webdriver.Chrome) -> None:
    driver.execute_script("if (document.activeElement instanceof HTMLElement) document.activeElement.blur();")
    ActionChains(driver).send_keys(Keys.TAB).perform()
    focus = driver.execute_script(
        """
        const element = document.activeElement;
        const style = element ? getComputedStyle(element) : null;
        return {
          tag: element?.tagName || '',
          name: String(element?.getAttribute('aria-label') || element?.getAttribute('title') || element?.getAttribute('placeholder') || element?.closest('label')?.textContent || element?.textContent || '').trim(),
          outlineWidth: parseFloat(style?.outlineWidth || '0'),
          outlineStyle: style?.outlineStyle || 'none',
          element: element?.outerHTML?.slice(0, 320) || ''
        };
        """
    )
    if not focus["tag"] or not focus["name"]:
        raise AssertionError(f"keyboard: first tab stop has no accessible name: {focus}")
    if float(focus["outlineWidth"]) < 2 or focus["outlineStyle"] in {"none", "hidden"}:
        raise AssertionError(f"keyboard: first tab stop has no visible focus outline: {focus}")


def capture(
    driver: webdriver.Chrome,
    output: Path,
    locale: str,
    viewport: str,
    state: str,
    route: str,
    expected_selector: str,
    evidence: str,
) -> MatrixResult:
    matching = [element for element in driver.find_elements(By.CSS_SELECTOR, expected_selector) if element.is_displayed()]
    if not matching:
        raise AssertionError(f"{state}: expected visible selector {expected_selector}")
    metrics = layout_metrics(driver)
    page_overflow = int(metrics["pageOverflow"])
    text_overflows = list(metrics["textOverflows"])
    if page_overflow > 1:
        raise AssertionError(f"{state}: document horizontal overflow is {page_overflow}px")
    if text_overflows:
        raise AssertionError(f"{state}: text overflow detected: {text_overflows}")
    if metrics["lang"] != locale:
        raise AssertionError(f"{state}: document language is {metrics['lang']}, expected {locale}")
    if not metrics["labelledPage"]:
        raise AssertionError(f"{state}: page shell is not linked to a visible heading")
    unnamed_controls = list(metrics["unnamedControls"])
    if unnamed_controls:
        raise AssertionError(f"{state}: unnamed interactive controls: {unnamed_controls}")
    if float(metrics["contrastRatio"]) < 4.5:
        raise AssertionError(f"{state}: body contrast is {metrics['contrastRatio']:.2f}:1")
    if not metrics["reducedMotion"]:
        raise AssertionError(f"{state}: reduced-motion media query is not active")
    durations = [str(metrics["transitionDuration"]), str(metrics["animationDuration"])]
    duration_seconds = [float(value.removesuffix("s") or "0") for value in durations]
    if any(value > 0.00001 for value in duration_seconds):
        raise AssertionError(f"{state}: motion duration is not reduced: {durations}")
    filename = f"{locale}-{viewport}-{state}.png"
    path = output / ".raw" / filename
    driver.save_screenshot(str(path))
    return MatrixResult(
        locale=locale,
        viewport=viewport,
        state=state,
        result="Pass",
        route=route,
        screenshot=filename,
        page_overflow=page_overflow,
        text_overflows=0,
        contrast_ratio=round(float(metrics["contrastRatio"]), 2),
        unnamed_controls=0,
        reduced_motion=bool(metrics["reducedMotion"]),
        evidence=evidence,
    )


def capture_admin_states(
    driver: webdriver.Chrome,
    wait: WebDriverWait,
    args: argparse.Namespace,
    locale: str,
    viewport: str,
) -> list[MatrixResult]:
    labels = LOCALES[locale]
    results: list[MatrixResult] = []
    login(driver, wait, args.base_url, locale, args.admin_user, args.admin_password)

    announce(locale, viewport, "responsive")
    open_page(driver, wait, args.base_url, DASHBOARD_PATH)
    wait_idle(driver, wait)
    verify_keyboard_focus(driver)
    results.append(capture(driver, args.output, locale, viewport, "responsive", DASHBOARD_PATH, ".page-shell__body", "Business dashboard renders at the target viewport."))

    announce(locale, viewport, "loading")
    open_page(driver, wait, args.base_url, CUSTOMER_PATH)
    wait_idle(driver, wait)
    configure_fetch(driver, delay_pattern="/skoll/v1/plugins/pharma_oa/api/customers")
    driver.refresh()
    wait_visible(driver, wait, ".page-shell[aria-busy='true'] .page-shell__loading")
    results.append(capture(driver, args.output, locale, viewport, "loading", CUSTOMER_PATH, ".page-shell__loading", "Page skeleton remains visible while API requests are delayed."))
    clear_fetch_configuration(driver)
    driver.get("about:blank")

    announce(locale, viewport, "empty")
    open_page(driver, wait, args.base_url, CUSTOMER_PATH)
    wait_idle(driver, wait)
    filters = [element for element in driver.find_elements(By.CSS_SELECTOR, ".filter-bar input") if element.is_displayed()]
    if not filters:
        raise AssertionError("empty: customer keyword filter is not visible")
    filters[0].clear()
    filters[0].send_keys("__h4_visual_empty__")
    wait_visible(driver, wait, ".data-table .state-block--empty")
    results.append(capture(driver, args.output, locale, viewport, "empty", CUSTOMER_PATH, ".data-table .state-block--empty", "A no-match customer filter renders the localized empty state."))

    announce(locale, viewport, "error")
    configure_fetch(driver, failure_pattern="/skoll/v1/plugins/pharma_oa/api/customers")
    driver.refresh()
    wait_visible(driver, wait, ".page-shell .state-block--error[role='alert']")
    results.append(capture(driver, args.output, locale, viewport, "error", CUSTOMER_PATH, ".page-shell .state-block--error[role='alert']", "A blocked customer API request renders the localized error state."))
    clear_fetch_configuration(driver)
    driver.get("about:blank")

    announce(locale, viewport, "destructive")
    open_page(driver, wait, args.base_url, QUALIFICATION_PATH)
    wait_idle(driver, wait)
    scan_buttons = [button for button in driver.find_elements(By.CSS_SELECTOR, ".page-shell__actions button") if button.is_displayed() and button.text.strip() == labels["scan"]]
    if len(scan_buttons) != 1:
        raise AssertionError(f"destructive: expected one scan button, got {len(scan_buttons)}")
    scan_buttons[0].click()
    wait_visible(driver, wait, ".el-message-box")
    dialog_focus = driver.execute_script("return Boolean(document.activeElement && document.activeElement.closest('.el-message-box'));")
    if not dialog_focus:
        raise AssertionError("destructive: focus did not move into the confirmation dialog")
    results.append(capture(driver, args.output, locale, viewport, "destructive", QUALIFICATION_PATH, ".el-message-box", "The guarded scan dialog exposes localized confirm and cancel actions."))

    announce(locale, viewport, "saving")
    confirm_buttons = [
        button
        for button in driver.find_elements(By.CSS_SELECTOR, ".el-message-box button")
        if button.is_displayed() and button.text.strip() == labels["scan_confirm"]
    ]
    if len(confirm_buttons) != 1:
        raise AssertionError(f"saving: expected one confirmation button, got {len(confirm_buttons)}")
    configure_fetch(driver, delay_pattern="/skoll/v1/plugins/pharma_oa/api/qualifications/expiry-scan")
    confirm_buttons[0].click()
    wait_visible(driver, wait, ".page-shell__actions button.is-loading")
    results.append(capture(driver, args.output, locale, viewport, "saving", QUALIFICATION_PATH, ".page-shell__actions button.is-loading", "The scan action exposes a stable localized saving state while its request is delayed."))
    clear_fetch_configuration(driver)
    driver.get("about:blank")
    return results


def capture_forbidden_state(
    driver: webdriver.Chrome,
    wait: WebDriverWait,
    args: argparse.Namespace,
    locale: str,
    viewport: str,
) -> MatrixResult:
    announce(locale, viewport, "no_permission")
    login(driver, wait, args.base_url, locale, args.restricted_user, args.restricted_password)
    open_page(driver, wait, args.base_url, CUSTOMER_PATH)
    wait_visible(driver, wait, ".state-block--forbidden[role='status']")
    if "/skoll/forbidden" not in driver.current_url:
        raise AssertionError(f"no_permission: expected forbidden route, got {driver.current_url}")
    return capture(
        driver,
        args.output,
        locale,
        viewport,
        "no_permission",
        CUSTOMER_PATH,
        ".state-block--forbidden[role='status']",
        "A restricted user sees the localized denial page instead of a silent dashboard redirect.",
    )


def create_contact_sheet(raw_dir: Path, output: Path, locale: str, viewport: str) -> str:
    if viewport == "desktop":
        cell_width, cell_height, columns = 640, 440, 2
    else:
        cell_width, cell_height, columns = 300, 650, 4
    rows = (len(STATE_ORDER) + columns - 1) // columns
    sheet = Image.new("RGB", (cell_width * columns, cell_height * rows), "#f4f6f8")
    draw = ImageDraw.Draw(sheet)
    for index, state in enumerate(STATE_ORDER):
        source = Image.open(raw_dir / f"{locale}-{viewport}-{state}.png").convert("RGB")
        source.thumbnail((cell_width - 20, cell_height - 44))
        column = index % columns
        row = index // columns
        x = column * cell_width + (cell_width - source.width) // 2
        y = row * cell_height + 34
        draw.text((column * cell_width + 10, row * cell_height + 10), state, fill="#172033")
        sheet.paste(source, (x, y))
    filename = f"{locale}-{viewport}-matrix.png"
    sheet.save(output / filename, optimize=True)
    return filename


def write_evidence(output: Path, results: list[MatrixResult], sheets: list[str]) -> None:
    payload = {
        "workItem": "PR4-04",
        "generatedAt": datetime.now().astimezone().isoformat(timespec="seconds"),
        "summary": {"total": len(results), "passed": sum(item.result == "Pass" for item in results)},
        "contactSheets": sheets,
        "results": [asdict(item) for item in results],
    }
    (output / "matrix.json").write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    lines = [
        "# PR4-04 双语响应式浏览器矩阵",
        "",
        "> 默认中文；覆盖中文/英文、桌面/390x844，以及 loading、empty、error、no-permission、saving、destructive、responsive 状态。",
        "",
        "## Contact Sheets",
        "",
    ]
    for sheet in sheets:
        lines.append(f"- `{sheet}`")
    lines.extend(["", "## Results", "", "| Locale | Viewport | State | Result | Overflow | Evidence |", "| --- | --- | --- | --- | --- | --- |"])
    for item in results:
        lines.append(f"| {item.locale} | {item.viewport} | {item.state} | {item.result} | {item.page_overflow}px | {item.evidence} |")
    lines.extend(
        [
            "",
            "## Reproduce",
            "",
            "```powershell",
            "python -m pip install -r scripts/requirements-browser.txt",
            "python scripts/h4-browser-matrix.py --base-url http://127.0.0.1:5174",
            "```",
            "",
        ]
    )
    (output / "README.md").write_text("\n".join(lines), encoding="utf-8")


def main() -> int:
    args = parse_args()
    args.base_url = args.base_url.rstrip("/")
    args.output = args.output.resolve()
    raw_dir = args.output / ".raw"
    if args.output.exists():
        shutil.rmtree(args.output)
    raw_dir.mkdir(parents=True)
    wait_for_health(args.base_url)
    driver = new_driver(args.headed)
    wait = WebDriverWait(driver, 20)
    results: list[MatrixResult] = []
    try:
        for locale in LOCALES:
            for viewport, (width, height) in VIEWPORTS.items():
                driver.set_window_size(width, height)
                results.extend(capture_admin_states(driver, wait, args, locale, viewport))
                results.append(capture_forbidden_state(driver, wait, args, locale, viewport))
        sheets = [create_contact_sheet(raw_dir, args.output, locale, viewport) for locale in LOCALES for viewport in VIEWPORTS]
        write_evidence(args.output, results, sheets)
        shutil.rmtree(raw_dir)
    except (AssertionError, RuntimeError, TimeoutException) as error:
        failure_path = args.output / "failure.png"
        try:
            driver.save_screenshot(str(failure_path))
        except Exception:
            pass
        diagnostics: dict[str, object] = {"stage": CURRENT_STAGE, "errorType": type(error).__name__}
        try:
            diagnostics["url"] = driver.current_url
            diagnostics["body"] = driver.find_element(By.TAG_NAME, "body").text[:1200]
            diagnostics["console"] = driver.get_log("browser")[-10:]
        except Exception as diagnostic_error:
            diagnostics["diagnosticError"] = str(diagnostic_error)
        print(f"PR4 browser matrix failed: {error}; diagnostics={json.dumps(diagnostics, ensure_ascii=False)}", file=sys.stderr)
        return 1
    finally:
        reset_network(driver)
        driver.quit()
    print(f"PR4 browser matrix passed: {len(results)} states across 2 locales and 2 viewports.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
