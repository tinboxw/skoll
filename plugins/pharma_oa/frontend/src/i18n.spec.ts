import { describe, expect, it } from "vitest";
import { language, setLocale, t } from "./i18n";

describe("pharma OA locale", () => {
  it("is Chinese-first and switches without stale labels", () => {
    setLocale("zh-CN");
    expect(t("workspace")).toBe("主数据工作台");
    setLocale("en-US");
    expect(t("workspace")).toBe("Master Data Workspace");
    expect(document.documentElement.lang).toBe("en-US");
  });

  it("reacts to host locale events", () => {
    window.dispatchEvent(new CustomEvent("skoll:locale", { detail: { locale: "zh-CN" } }));
    expect(language.value).toBe("zh-CN");
    expect(t("qualification")).toBe("资质台账");
  });
});
