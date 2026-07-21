from __future__ import annotations

import argparse
import json
import os
import time
from pathlib import Path

from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.support.ui import WebDriverWait
from selenium.common.exceptions import StaleElementReferenceException, TimeoutException


DEFAULT_OUTPUT = Path("docs/refactor/current/evidence/h5-02")
VIEWPORTS = {"desktop": (1440, 1000), "mobile": (390, 844)}
LOCALES = ("zh-CN", "en-US")
PAGES = (
    ("employees", "/skoll/pharma-oa/employees", "H5E", 125),
    ("customers", "/skoll/pharma-oa/customers", "H5C", 125),
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run the H5-02 bilingual large-list browser smoke.")
    parser.add_argument("--base-url", default=os.getenv("SKOLL_E2E_BASE_URL", "http://127.0.0.1:5175"))
    parser.add_argument("--admin-user", default=os.getenv("SKOLL_E2E_ADMIN_USER", "admin"))
    parser.add_argument("--admin-password", default=os.getenv("SKOLL_E2E_ADMIN_PASSWORD", "Admin@123456"))
    parser.add_argument("--output", type=Path, default=DEFAULT_OUTPUT)
    parser.add_argument("--headed", action="store_true")
    parser.add_argument("--locale", choices=LOCALES)
    parser.add_argument("--viewport", choices=VIEWPORTS.keys())
    return parser.parse_args()


def new_driver(headed: bool) -> webdriver.Chrome:
    options = webdriver.ChromeOptions()
    if not headed:
        options.add_argument("--headless=new")
    options.add_argument("--disable-gpu")
    options.add_argument("--no-sandbox")
    options.add_argument("--force-device-scale-factor=1")
    options.set_capability("goog:loggingPrefs", {"browser": "ALL"})
    return webdriver.Chrome(options=options)


def login(driver: webdriver.Chrome, wait: WebDriverWait, args: argparse.Namespace, locale: str) -> None:
    driver.get(f"{args.base_url}/skoll/login")
    driver.execute_script("localStorage.clear(); localStorage.setItem('skoll.ui.locale', arguments[0]);", locale)
    driver.refresh()
    inputs = wait.until(lambda current: current.find_elements(By.CSS_SELECTOR, "main input"))
    if len(inputs) != 2:
        raise RuntimeError(f"expected two login inputs, got {len(inputs)}")
    for element, value in zip(inputs, (args.admin_user, args.admin_password), strict=True):
        element.clear()
        element.send_keys(value)
    driver.find_element(By.CSS_SELECTOR, "main button").click()
    wait.until(lambda current: "/login" not in current.current_url)


def set_keyword(driver: webdriver.Chrome, value: str) -> None:
    element = driver.find_element(By.CSS_SELECTOR, ".page-toolbar input")
    element.send_keys(Keys.CONTROL, "a")
    element.send_keys(value)


def total_value(driver: webdriver.Chrome) -> int:
    text = driver.find_element(By.CSS_SELECTOR, ".summary-grid .summary-tile strong").text.strip()
    return int(text.replace(",", "")) if text else -1


def active_page(driver: webdriver.Chrome) -> str:
    return driver.find_element(By.CSS_SELECTOR, ".el-pagination .el-pager li.is-active").text.strip()


def request_count(driver: webdriver.Chrome, resource: str, offset: int) -> int:
    marker = f"/v1/pharma-oa/{resource}?"
    expected = f"offset={offset}"
    return int(driver.execute_script(
        "return performance.getEntriesByType('resource').filter((entry) => entry.name.includes(arguments[0]) && entry.name.includes(arguments[1])).length;",
        marker,
        expected,
    ))


def keyword_request_count(driver: webdriver.Chrome, resource: str, keyword: str) -> int:
    marker = f"/v1/pharma-oa/{resource}?"
    expected = f"keyword={keyword}"
    return int(driver.execute_script(
        "return performance.getEntriesByType('resource').filter((entry) => entry.name.includes(arguments[0]) && entry.name.includes(arguments[1])).length;",
        marker,
        expected,
    ))


def page_metrics(driver: webdriver.Chrome) -> dict[str, object]:
    return driver.execute_script(
        """
        const root = document.documentElement;
        const pagination = document.querySelector('.el-pagination');
        const table = document.querySelector('.data-table__virtual');
        return {
          pageOverflow: Math.max(0, root.scrollWidth - root.clientWidth),
          virtualRows: document.querySelectorAll('.el-table-v2__row').length,
          paginationOverflow: pagination ? Math.max(0, pagination.scrollWidth - pagination.clientWidth) : -1,
          tableHeight: table ? Math.round(table.getBoundingClientRect().height) : 0,
          totalText: document.querySelector('.el-pagination__total')?.textContent?.trim() || '',
          tableText: table?.textContent || ''
        };
        """
    )


def click_pagination(driver: webdriver.Chrome, wait: WebDriverWait, selector: str) -> None:
    wait.until(lambda current: current.find_element(By.CSS_SELECTOR, ".page-shell").get_attribute("aria-busy") != "true")
    for _ in range(5):
        try:
            button = wait.until(lambda current: current.find_element(By.CSS_SELECTOR, selector))
            driver.execute_script("arguments[0].click();", button)
            return
        except StaleElementReferenceException:
            time.sleep(0.1)
    raise RuntimeError(f"pagination control remained stale: {selector}")


def verify_page(driver: webdriver.Chrome, wait: WebDriverWait, base_url: str, resource: str, path: str, keyword: str, expected_total: int, locale: str) -> dict[str, object]:
    driver.get(f"{base_url}{path}")
    wait.until(lambda current: current.find_elements(By.CSS_SELECTOR, ".page-shell"))
    try:
        wait.until(lambda current: current.find_element(By.CSS_SELECTOR, ".page-shell").get_attribute("aria-busy") != "true")
    except TimeoutException:
        diagnostics = {
            "url": driver.current_url,
            "ariaBusy": driver.find_element(By.CSS_SELECTOR, ".page-shell").get_attribute("aria-busy"),
            "bodyText": driver.find_element(By.TAG_NAME, "body").text[:1000],
            "console": driver.get_log("browser"),
        }
        raise RuntimeError(f"{resource} did not become idle: {json.dumps(diagnostics, ensure_ascii=False)}")
    prior_keyword_requests = keyword_request_count(driver, resource, keyword)
    set_keyword(driver, keyword)
    wait.until(lambda current: keyword_request_count(current, resource, keyword) > prior_keyword_requests)
    wait.until(lambda current: total_value(current) == expected_total)
    wait.until(lambda current: current.find_element(By.CSS_SELECTOR, ".page-shell").get_attribute("aria-busy") != "true")
    wait.until(lambda current: current.find_elements(By.CSS_SELECTOR, ".el-table-v2"))

    first_metrics = page_metrics(driver)
    if not 0 < int(first_metrics["virtualRows"]) < 50:
        raise RuntimeError(f"{resource} did not virtualize its 50-row page: {first_metrics}")
    if int(first_metrics["tableHeight"]) != 520:
        raise RuntimeError(f"{resource} virtual table height drifted: {first_metrics}")
    expected_total_word = "共" if locale == "zh-CN" else "Total"
    if expected_total_word not in str(first_metrics["totalText"]):
        raise RuntimeError(f"{resource} pagination locale mismatch: {first_metrics['totalText']}")

    page_one_requests = request_count(driver, resource, 0)
    click_pagination(driver, wait, ".el-pagination .btn-next")
    wait.until(lambda current: active_page(current) == "2")
    wait.until(lambda current: request_count(current, resource, 50) >= 1)
    second_metrics = page_metrics(driver)
    expected_second_code = f"{keyword}0051"
    if expected_second_code not in str(second_metrics["tableText"]):
        raise RuntimeError(f"{resource} second page is not stably sorted: expected {expected_second_code}")

    click_pagination(driver, wait, ".el-pagination .btn-prev")
    wait.until(lambda current: active_page(current) == "1")
    time.sleep(0.4)
    if request_count(driver, resource, 0) != page_one_requests:
        raise RuntimeError(f"{resource} page-one request was not served from the short page cache")

    final_metrics = page_metrics(driver)
    if int(final_metrics["pageOverflow"]) != 0 or int(final_metrics["paginationOverflow"]) != 0:
        raise RuntimeError(f"{resource} responsive overflow: {final_metrics}")
    return {
        "resource": resource,
        "total": expected_total,
        "pageSize": 50,
        "virtualRows": first_metrics["virtualRows"],
        "tableHeight": first_metrics["tableHeight"],
        "totalText": first_metrics["totalText"],
        "cacheHit": True,
        "pageOverflow": final_metrics["pageOverflow"],
        "paginationOverflow": final_metrics["paginationOverflow"],
    }


def verify_cancellation(driver: webdriver.Chrome, wait: WebDriverWait) -> dict[str, object]:
    driver.execute_cdp_cmd("Network.enable", {})
    driver.execute_cdp_cmd("Network.emulateNetworkConditions", {"offline": False, "latency": 900, "downloadThroughput": 2_000_000, "uploadThroughput": 2_000_000})
    try:
        set_keyword(driver, "H5E0")
        time.sleep(0.35)
        set_keyword(driver, "H5E01")
        wait.until(lambda current: total_value(current) == 26)
        if driver.find_elements(By.CSS_SELECTOR, ".state-block--error"):
            raise RuntimeError("aborted employee request surfaced as an error state")
        return {"latestKeyword": "H5E01", "latestTotal": 26, "errorState": False}
    finally:
        driver.execute_cdp_cmd("Network.emulateNetworkConditions", {"offline": False, "latency": 0, "downloadThroughput": -1, "uploadThroughput": -1})


def run_matrix(args: argparse.Namespace) -> list[dict[str, object]]:
    args.output.mkdir(parents=True, exist_ok=True)
    results: list[dict[str, object]] = []
    locales = (args.locale,) if args.locale else LOCALES
    viewports = {args.viewport: VIEWPORTS[args.viewport]} if args.viewport else VIEWPORTS
    for locale in locales:
        for viewport, size in viewports.items():
            driver = new_driver(args.headed)
            wait = WebDriverWait(driver, 20)
            try:
                driver.set_window_size(*size)
                login(driver, wait, args, locale)
                scenario: dict[str, object] = {"locale": locale, "viewport": viewport, "width": size[0], "height": size[1], "pages": []}
                for resource, path, keyword, expected_total in PAGES:
                    page_result = verify_page(driver, wait, args.base_url, resource, path, keyword, expected_total, locale)
                    scenario["pages"].append(page_result)
                    driver.save_screenshot(str(args.output / f"{locale}-{viewport}-{resource}.png"))
                if locale == "zh-CN" and viewport == "desktop":
                    driver.get(f"{args.base_url}/skoll/pharma-oa/employees")
                    wait.until(lambda current: current.find_elements(By.CSS_SELECTOR, ".page-toolbar input"))
                    scenario["cancellation"] = verify_cancellation(driver, wait)
                severe = [
                    entry for entry in driver.get_log("browser")
                    if entry.get("level") == "SEVERE" and "/favicon.ico" not in str(entry.get("message", ""))
                ]
                if severe:
                    raise RuntimeError(f"browser console errors: {severe}")
                results.append(scenario)
                print(f"[PASS] {locale}/{viewport}: employees + customers", flush=True)
            finally:
                driver.quit()
    return results


def main() -> None:
    args = parse_args()
    results = run_matrix(args)
    payload = {"workItem": "H5-02", "baseUrl": args.base_url, "scenarios": results, "passed": True}
    (args.output / "browser-matrix.json").write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"H5-02 browser matrix passed: {len(results)} locale/viewport scenarios, 8 page checks.")


if __name__ == "__main__":
    main()
