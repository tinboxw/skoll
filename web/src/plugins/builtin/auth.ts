import { h, defineComponent } from "vue";

import type { FrontendPlugin } from "../types";

const AuthPage = defineComponent({
  name: "AuthBuiltinPage",
  setup() {
    return () => h("div", "Builtin Auth Plugin Page");
  }
});

export const builtinAuthPlugin: FrontendPlugin = {
  manifest: {
    id: "builtin-auth",
    name: "Builtin Auth",
    version: "1.0.0",
    backendEndpoint: "/v1/auth/login",
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
