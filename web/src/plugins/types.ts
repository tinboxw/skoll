import type { RouteRecordRaw, Router } from "vue-router";

export type PluginMenuManifest = {
  label?: string;
  labelZhCN?: string;
  labelEnUS?: string;
  path?: string;
  icon?: string;
  order?: number;
  requiredRoles?: string[];
  requiredPermissions?: string[];
};

export type PluginConfigOption = {
  label?: string;
  labelZhCN?: string;
  labelEnUS?: string;
  value: string;
};

export type PluginConfigField = {
  key: string;
  label?: string;
  labelZhCN?: string;
  labelEnUS?: string;
  type?: "string" | "textarea" | "number" | "boolean" | "select";
  required?: boolean;
  default?: string;
  placeholder?: string;
  help?: string;
  min?: number;
  max?: number;
  minLength?: number;
  maxLength?: number;
  pattern?: string;
  options?: PluginConfigOption[];
};

export type PluginConfigSchema = {
  title?: string;
  titleZhCN?: string;
  titleEnUS?: string;
  description?: string;
  fields?: PluginConfigField[];
};

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
  uiMenu?: PluginMenuManifest;
  configSchema?: PluginConfigSchema;
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
  uiMenu?: PluginMenuManifest;
  configSchema?: PluginConfigSchema;
  frontendEntry?: string;
  systemBuiltin?: boolean;
};
