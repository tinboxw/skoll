import type { ThemeBridgePayload } from "../stores/theme";

export type HostSDKRequestOptions = {
	method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
	body?: unknown;
	headers?: Record<string, string>;
};

export type PluginHostSDKInit = {
	pluginId: string;
	apiBasePrefix: string;
	locale: string;
	locales: string[];
	token: string;
	theme: ThemeBridgePayload;
};

export type PluginHostSDK = {
	pluginId: string;
	locale: string;
	locales: string[];
	theme: ThemeBridgePayload;
	getToken: () => string;
	request: <T = unknown>(path: string, options?: HostSDKRequestOptions) => Promise<T>;
	auth: { me: () => Promise<unknown> };
	user: { me: () => Promise<unknown> };
	organization: {
		departments: () => Promise<unknown>;
		positions: () => Promise<unknown>;
	};
	dictionary: {
		list: () => Promise<unknown>;
		items: (type: string) => Promise<unknown>;
	};
	file: { list: (query?: Record<string, string | number | boolean>) => Promise<unknown> };
	audit: { list: (query?: Record<string, string | number | boolean>) => Promise<unknown> };
	config: {
		get: () => Promise<unknown>;
		update: (config: Record<string, unknown>) => Promise<unknown>;
	};
	permission: { list: () => Promise<unknown> };
};

declare global {
	interface Window {
		__SKOLL_HOST__?: PluginHostSDK;
	}
}

function normalizeAPIPrefix(raw: string): string {
	const value = raw.trim();
	if (value === "") {
		return "/skoll";
	}
	const withSlash = value.startsWith("/") ? value : `/${value}`;
	return withSlash.replace(/\/+$/, "") || "/skoll";
}

export function buildPluginHostBridgeScript(input: PluginHostSDKInit): string {
	const payload = {
		pluginId: input.pluginId.trim(),
		apiBasePrefix: normalizeAPIPrefix(input.apiBasePrefix),
		locale: input.locale.trim() || "zh-CN",
		locales: input.locales.length > 0 ? input.locales : ["zh-CN", "en-US"],
		token: input.token,
		theme: input.theme
	};

	return `<script>
(function () {
	var ctx = ${JSON.stringify(payload)};

	function withPrefix(path) {
		var value = String(path || '').trim();
		if (!value) return ctx.apiBasePrefix;
		if (/^https?:\\/\\//i.test(value)) return value;
		if (value.indexOf(ctx.apiBasePrefix + '/') === 0 || value === ctx.apiBasePrefix) return value;
		return value.charAt(0) === '/' ? ctx.apiBasePrefix + value : ctx.apiBasePrefix + '/' + value;
	}

	function query(params) {
		if (!params) return '';
		var result = new URLSearchParams();
		Object.keys(params).forEach(function (key) {
			var value = params[key];
			if (value !== undefined && value !== null && String(value) !== '') result.set(key, String(value));
		});
		var text = result.toString();
		return text ? '?' + text : '';
	}

	function applyTheme(nextTheme) {
		ctx.theme = nextTheme || ctx.theme;
		if (window.__SKOLL_PLUGIN_CONTEXT) window.__SKOLL_PLUGIN_CONTEXT.theme = ctx.theme;
		window.__SKOLL_THEME = ctx.theme;
		if (document && document.documentElement) {
			var root = document.documentElement;
			root.setAttribute('data-theme', ctx.theme.colorScheme || 'light');
			root.setAttribute('data-density', ctx.theme.density || 'comfortable');
			root.style.colorScheme = ctx.theme.colorScheme || 'light';
			var tokens = ctx.theme.tokens || {};
			Object.keys(tokens).forEach(function (key) { root.style.setProperty(key, tokens[key]); });
		}
		window.dispatchEvent(new CustomEvent('skoll:theme', { detail: ctx.theme }));
	}

	function request(path, options) {
		var opts = options || {};
		var headers = Object.assign({ 'Content-Type': 'application/json' }, opts.headers || {});
		var token = String(ctx.token || '').trim();
		if (token) headers.Authorization = token.toLowerCase().indexOf('bearer ') === 0 ? token : 'Bearer ' + token;
		return fetch(withPrefix(path), {
			method: opts.method || 'GET',
			headers: headers,
			body: opts.body === undefined ? undefined : JSON.stringify(opts.body)
		}).then(function (response) {
			return response.text().then(function (text) {
				var payload = text ? JSON.parse(text) : null;
				if (!response.ok) {
					var message = payload && payload.message ? payload.message : 'request failed: ' + response.status;
					throw new Error(message);
				}
				return payload && Object.prototype.hasOwnProperty.call(payload, 'data') ? payload.data : payload;
			});
		});
	}

	var host = {
		pluginId: ctx.pluginId,
		locale: ctx.locale,
		locales: ctx.locales,
		theme: ctx.theme,
		getToken: function () { return ctx.token; },
		request: request,
		auth: { me: function () { return request('/v1/auth/me'); } },
		user: { me: function () { return request('/v1/auth/me'); } },
		organization: {
			departments: function () { return request('/v1/system/settings/skoll.organization.departments'); },
			positions: function () { return request('/v1/system/settings/skoll.organization.positions'); }
		},
		dictionary: {
			list: function () { return request('/v1/system/dictionaries'); },
			items: function (type) { return request('/v1/system/dictionaries/' + encodeURIComponent(type) + '/items'); }
		},
		file: { list: function (params) { return request('/v1/files' + query(params)); } },
		audit: { list: function (params) { return request('/v1/audit' + query(params)); } },
		config: {
			get: function () { return request('/v1/plugins/' + encodeURIComponent(ctx.pluginId) + '/config'); },
			update: function (config) { return request('/v1/plugins/' + encodeURIComponent(ctx.pluginId) + '/config', { method: 'PUT', body: { config: config || {} } }); }
		},
		permission: { list: function () { return request('/v1/permissions'); } }
	};

	window.__SKOLL_HOST__ = host;
	window.__SKOLL_LOCALE = ctx.locale;
	window.__SKOLL_LOCALES = ctx.locales;
	window.__SKOLL_TOKEN = ctx.token;
	window.__SKOLL_PLUGIN_CONTEXT = { locale: ctx.locale, locales: ctx.locales, token: ctx.token, theme: ctx.theme, host: host };
	if (document && document.documentElement) document.documentElement.setAttribute('lang', ctx.locale);
	applyTheme(ctx.theme);
	window.dispatchEvent(new CustomEvent('skoll:host-ready', { detail: { pluginId: ctx.pluginId, capabilities: ['auth', 'user', 'organization', 'dictionary', 'file', 'audit', 'config', 'permission'] } }));
	window.dispatchEvent(new CustomEvent('skoll:locale', { detail: { locale: ctx.locale, locales: ctx.locales } }));

	window.addEventListener('message', function (event) {
		var data = event && event.data;
		if (!data) return;
		if (data.type === 'skoll:locale') {
			ctx.locale = String(data.locale || '').trim() || ctx.locale;
			ctx.locales = Array.isArray(data.locales) ? data.locales : ctx.locales;
			host.locale = ctx.locale;
			host.locales = ctx.locales;
			window.__SKOLL_LOCALE = ctx.locale;
			window.__SKOLL_LOCALES = ctx.locales;
			window.__SKOLL_PLUGIN_CONTEXT = { locale: ctx.locale, locales: ctx.locales, token: ctx.token, theme: ctx.theme, host: host };
			if (document && document.documentElement) document.documentElement.setAttribute('lang', ctx.locale);
			window.dispatchEvent(new CustomEvent('skoll:locale', { detail: { locale: ctx.locale, locales: ctx.locales } }));
			return;
		}
		if (data.type === 'skoll:theme') {
			applyTheme(data.theme);
			host.theme = ctx.theme;
		}
	});
})();
</script>`;
}
