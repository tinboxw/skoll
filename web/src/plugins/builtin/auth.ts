import { computed, defineComponent, h, ref } from "vue";

import { useI18n } from "../../i18n";
import { useUserStore } from "../../stores/user";
import { apiPost, type ApiResponse } from "../../utils/api";
import { toErrorMessage } from "../../utils/common";
import type { FrontendPlugin } from "../types";

const AUTH_SESSION_META_KEY = "skoll.auth.sessionMeta";

type SessionMeta = {
  tokenType: string;
  expiresIn: number;
  permissions: string[];
  account: string;
};

const AuthPage = defineComponent({
  name: "AuthBuiltinPage",
  setup() {
    const { t } = useI18n();
    const userStore = useUserStore();
    const account = ref("admin");
    const password = ref("Admin@123456");
    const loading = ref(false);
    const error = ref("");
    const sessionMeta = ref<SessionMeta | null>(loadSessionMeta());
    const copyState = ref<"idle" | "ok" | "fail">("idle");

    const isAuthenticated = computed(() => userStore.isAuthenticated);
    const profileName = computed(() => sessionMeta.value?.account || userStore.profile?.name || "-");
    const profileRole = computed(() => userStore.profile?.role ?? "-");
    const maskedToken = computed(() => maskToken(userStore.token));
    const tokenPrefix = computed(() => extractTokenPrefix(maskedToken.value));

    type LoginPayload = {
      token: string;
      tokenType?: string;
      expiresIn?: number;
      permissions?: string[];
      user?: {
        id?: string;
        account?: string;
        name?: string;
        email?: string;
        role?: string;
      };
    };

    if (!userStore.isAuthenticated) {
      sessionMeta.value = null;
      saveSessionMeta(null);
    }

    async function login(): Promise<void> {
      error.value = "";
      loading.value = true;
      try {
        const payload = await apiPost<ApiResponse<LoginPayload>>("/v1/auth/login", {
          account: account.value,
          password: password.value
        });
        const token = payload.data?.token?.trim() ?? "";
        if (token === "") {
          throw new Error("missing token in login response");
        }
        const profileName = payload.data?.user?.name?.trim() || payload.data?.user?.account?.trim() || account.value.trim() || "admin";
        userStore.setSession(token, {
          id: payload.data?.user?.id?.trim() || profileName,
          name: profileName,
          role: payload.data?.user?.role?.trim() || "user",
          email: payload.data?.user?.email?.trim() || ""
        }, Array.isArray(payload.data?.permissions) ? payload.data.permissions : []);
        await userStore.hydrateProfile();
        sessionMeta.value = {
          tokenType: payload.data?.tokenType?.trim() || "Bearer",
          expiresIn: typeof payload.data?.expiresIn === "number" ? payload.data.expiresIn : 0,
          permissions: Array.isArray(payload.data?.permissions) ? payload.data.permissions : [],
          account: profileName
        };
        saveSessionMeta(sessionMeta.value);
        copyState.value = "idle";
      } catch (e) {
        error.value = toErrorMessage(e);
      } finally {
        loading.value = false;
      }
    }

    function logout(): void {
      userStore.logout();
      sessionMeta.value = null;
      saveSessionMeta(null);
      copyState.value = "idle";
    }

    async function copyTokenPrefix(): Promise<void> {
      if (!isAuthenticated.value || tokenPrefix.value === "") {
        copyState.value = "fail";
        return;
      }
      try {
        if (navigator.clipboard && navigator.clipboard.writeText) {
          await navigator.clipboard.writeText(tokenPrefix.value);
        } else {
          copyByTextarea(tokenPrefix.value);
        }
        copyState.value = "ok";
      } catch {
        copyState.value = "fail";
      }
      setTimeout(() => {
        copyState.value = "idle";
      }, 1400);
    }

    return () =>
      h("section", { class: "auth-plugin" }, [
        h("h2", t("plugin.auth.title")),
        h("p", { class: "desc" }, t("plugin.auth.desc")),
        error.value ? h("p", { class: "error" }, error.value) : null,
        h("div", { class: "panel" }, [
          h("label", { class: "field" }, [
            h("span", t("plugin.auth.account")),
            h("input", {
              value: account.value,
              onInput: (e: Event) => {
                account.value = (e.target as HTMLInputElement).value;
              },
              disabled: loading.value,
              placeholder: t("plugin.auth.accountPlaceholder")
            })
          ]),
          h("label", { class: "field" }, [
            h("span", t("plugin.auth.password")),
            h("input", {
              type: "password",
              value: password.value,
              onInput: (e: Event) => {
                password.value = (e.target as HTMLInputElement).value;
              },
              disabled: loading.value,
              placeholder: t("plugin.auth.passwordPlaceholder")
            })
          ]),
          h("div", { class: "actions" }, [
            h(
              "button",
              {
                type: "button",
                disabled: loading.value,
                onClick: () => void login()
              },
              loading.value ? t("plugin.auth.loggingIn") : t("plugin.auth.login")
            ),
            h(
              "button",
              {
                type: "button",
                class: "secondary",
                disabled: loading.value || !isAuthenticated.value,
                onClick: logout
              },
              t("plugin.auth.logout")
            )
          ]),
          h(
            "p",
            { class: "status" },
            isAuthenticated.value ? t("plugin.auth.statusOnline") : t("plugin.auth.statusOffline")
          ),
          h("div", { class: "session-grid" }, [
            h("p", [h("strong", `${t("plugin.auth.currentUser")}: `), profileName.value]),
            h("p", [h("strong", `${t("plugin.auth.currentRole")}: `), profileRole.value]),
            h("p", [
              h("strong", `${t("plugin.auth.tokenMask")}: `),
              maskedToken.value,
              h(
                "button",
                {
                  type: "button",
                  class: "copy-token",
                  disabled: !isAuthenticated.value || tokenPrefix.value === "",
                  onClick: () => void copyTokenPrefix()
                },
                copyState.value === "ok"
                  ? t("plugin.auth.copyDone")
                  : copyState.value === "fail"
                    ? t("plugin.auth.copyFailed")
                    : t("plugin.auth.copyPrefix")
              )
            ]),
            h("p", [h("strong", `${t("plugin.auth.tokenType")}: `), sessionMeta.value?.tokenType ?? "-" ]),
            h("p", [
              h("strong", `${t("plugin.auth.expiresIn")}: `),
              sessionMeta.value && sessionMeta.value.expiresIn > 0 ? `${sessionMeta.value.expiresIn}s` : "-"
            ])
          ]),
          h("div", { class: "permissions" }, [
            h("p", { class: "perm-title" }, t("plugin.auth.permissions")),
            sessionMeta.value && sessionMeta.value.permissions.length > 0
              ? h(
                  "div",
                  { class: "perm-list" },
                  sessionMeta.value.permissions.map((item) => h("span", { class: "perm-item" }, item))
                )
              : h("p", { class: "perm-empty" }, t("plugin.auth.noPermissions"))
          ])
        ])
      ]);
  }
});

function maskToken(token: string): string {
  const raw = token.trim();
  if (raw.length < 10) {
    return raw === "" ? "-" : "***";
  }
  return `${raw.slice(0, 6)}...${raw.slice(-4)}`;
}

function extractTokenPrefix(masked: string): string {
  const value = masked.trim();
  if (value === "" || value === "-" || value === "***") {
    return "";
  }
  const idx = value.indexOf("...");
  if (idx <= 0) {
    return "";
  }
  return value.slice(0, idx);
}

function loadSessionMeta(): SessionMeta | null {
  const raw = localStorage.getItem(AUTH_SESSION_META_KEY);
  if (!raw) {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as Partial<SessionMeta>;
    return {
      tokenType: typeof parsed.tokenType === "string" && parsed.tokenType.trim() ? parsed.tokenType.trim() : "Bearer",
      expiresIn: typeof parsed.expiresIn === "number" ? parsed.expiresIn : 0,
      permissions: Array.isArray(parsed.permissions) ? parsed.permissions.filter((v): v is string => typeof v === "string") : [],
      account: typeof parsed.account === "string" && parsed.account.trim() ? parsed.account.trim() : "Admin"
    };
  } catch {
    return null;
  }
}

function saveSessionMeta(meta: SessionMeta | null): void {
  if (!meta) {
    localStorage.removeItem(AUTH_SESSION_META_KEY);
    return;
  }
  localStorage.setItem(AUTH_SESSION_META_KEY, JSON.stringify(meta));
}

function copyByTextarea(text: string): void {
  const area = document.createElement("textarea");
  area.value = text;
  area.setAttribute("readonly", "readonly");
  area.style.position = "absolute";
  area.style.left = "-9999px";
  document.body.appendChild(area);
  area.select();
  document.execCommand("copy");
  document.body.removeChild(area);
}

export const builtinAuthPlugin: FrontendPlugin = {
  manifest: {
    id: "builtin-auth",
    name: "Builtin Auth",
    version: "1.0.0",
    enabled: true,
    uiMode: "frontend_only",
    systemBuiltin: true,
    backendEndpoint: "/api/v1/auth/login",
    route: {
      path: "/plugins/auth",
      name: "plugin-auth",
      component: AuthPage
    }
  },
  setup(ctx) {
    if (builtinAuthPlugin.manifest.route) {
      ctx.registerRoute(builtinAuthPlugin.manifest.route);
    }
  }
};
