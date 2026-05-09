import type { RouteRecordRaw, Router } from "vue-router";

export type FrontendPluginManifest = {
  id: string;
  name: string;
  version: string;
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
  version: string;
};
