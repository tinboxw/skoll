import { describe, expect, it } from "vitest";
import { language, locale, t } from "./i18n";

describe("equipment maintenance locale bridge", () => {
  it("updates bilingual copy from the current host event", () => {
    locale.value = "zh-CN";
    expect(t("assets")).toBe("设备台账");
    window.dispatchEvent(new CustomEvent("skoll:locale", { detail: { locale: "en-US" } }));
    expect(language.value).toBe("en-US");
    expect(t("assets")).toBe("Assets");
  });

  it("uses Chinese for unsupported locales in the current contract", () => {
    window.dispatchEvent(new CustomEvent("skoll:locale", { detail: { locale: "fr-FR" } }));
    expect(language.value).toBe("zh-CN");
  });
});
