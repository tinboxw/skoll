import type { RouteRecordRaw, Router } from "vue-router";

export type FrontendPluginManifest = {
  id: string;
  name: string;
  nameZhCN?: string;
  nameEnUS?: string;
  version: string;
  enabled?: boolean;
  description?: string;
  entryPath?: string;
  uiMode?: "backend_only" | "frontend_only" | "monolith" | "separated";
  level?: "system" | "app";
  appId?: string;
  mountPolicy?: "admin" | "user" | "mixed";
  uiNavPosition?: "none" | "sidebar" | "top_tab";
  uiOpenMode?: "integrated" | "standalone";
  uiTabMode?: "optional" | "fixed" | "disabled";
  i18nLocales?: string[];
  systemBuiltin?: boolean;
  route?: RouteRecordRaw;
  backendEndpoint?: string;
};

export type PluginRuntimeContext = {
  router: Router;
  registerRoute: (route: RouteRecordRaw) => void;
};

export type FrontendPlugin = {
  manifest: FrontendPluginManifest;
  setup: (ctx: PluginRuntimeContext) => void;
};

export type BackendPluginRecord = {
  id: string;
  name: string;
  nameZhCN?: string;
  nameEnUS?: string;
  version: string;
  enabled?: boolean;
  uiMode?: "backend_only" | "frontend_only" | "monolith" | "separated";
  level?: "system" | "app";
  appId?: string;
  mountPolicy?: "admin" | "user" | "mixed";
  uiNavPosition?: "none" | "sidebar" | "top_tab";
  uiOpenMode?: "integrated" | "standalone";
  uiTabMode?: "optional" | "fixed" | "disabled";
  i18nLocales?: string[];
  frontendEntry?: string;
  systemBuiltin?: boolean;
};
