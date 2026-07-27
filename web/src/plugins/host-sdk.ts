import {
	PLUGIN_HOST_CAPABILITIES,
	PLUGIN_HOST_CONTRACT,
	PLUGIN_HOST_VERSION,
	type PluginHostIdentity,
	type PluginHostLifecycle,
	type PluginHostRequestOptions,
	type PluginHostSDK as PublicPluginHostSDK,
	type PluginHostTheme
} from "@skoll/plugin-sdk";

import type { ThemeBridgePayload } from "../stores/theme";

export type HostSDKRequestOptions = PluginHostRequestOptions;
export type PluginHostSDK = PublicPluginHostSDK;

export type PluginHostSDKInit = {
	pluginId: string;
	pluginVersion: string;
	apiBasePrefix: string;
	locale: string;
	locales: string[];
	token: string;
	theme: ThemeBridgePayload;
	identity: PluginHostIdentity;
	lifecycle: PluginHostLifecycle;
};

function normalizeAPIPrefix(raw: string): string {
	const value = raw.trim();
	if (value === "") {
		return "/skoll";
	}
	const withSlash = value.startsWith("/") ? value : `/${value}`;
	return withSlash.replace(/\/+$/, "") || "/skoll";
}

function compact(values: readonly string[]): string[] {
	return Array.from(new Set(values.map((value) => String(value || "").trim()).filter(Boolean)));
}

export function buildPluginHostBridgeScript(input: PluginHostSDKInit): string {
	const pluginId = input.pluginId.trim();
	const pluginVersion = input.pluginVersion.trim();
	if (!/^[a-z][a-z0-9_]{0,63}$/.test(pluginId) || pluginVersion === "") {
		throw new Error("plugin host identity is invalid");
	}
	const payload = {
		contract: PLUGIN_HOST_CONTRACT,
		version: PLUGIN_HOST_VERSION,
		pluginId,
		pluginVersion,
		apiBasePrefix: normalizeAPIPrefix(input.apiBasePrefix),
		capabilities: [...PLUGIN_HOST_CAPABILITIES],
		locale: input.locale.trim() || "zh-CN",
		locales: compact(input.locales).length > 0 ? compact(input.locales) : ["zh-CN", "en-US"],
		token: input.token,
		theme: input.theme satisfies PluginHostTheme,
		identity: {
			subject: input.identity.subject.trim(),
			roles: compact(input.identity.roles),
			permissions: compact(input.identity.permissions)
		},
		lifecycle: input.lifecycle
	};

	return `<script>
	(function () {
		'use strict';
		var ctx = ${JSON.stringify(payload)};
		var allowedCapabilities = Object.freeze(ctx.capabilities.slice());

		function freezeRecord(value) {
			return Object.freeze(Object.assign({}, value || {}));
		}

		function freezeList(value) {
			return Object.freeze(Array.isArray(value) ? value.slice() : []);
		}

		function normalizeTheme(value) {
			var source = value || {};
			return Object.freeze({
				colorScheme: source.colorScheme === 'dark' ? 'dark' : 'light',
				density: source.density === 'compact' ? 'compact' : 'comfortable',
				tokens: freezeRecord(source.tokens)
			});
		}

		function normalizeIdentity(value) {
			var source = value || {};
			return Object.freeze({
				subject: String(source.subject || '').trim(),
				roles: freezeList(source.roles),
				permissions: freezeList(source.permissions)
			});
		}

		function normalizeLifecycle(value) {
			var source = value || {};
			var state = source.state;
			if (state !== 'enabled' && state !== 'disabled' && state !== 'degraded') state = 'degraded';
			return Object.freeze({ state: state, health: String(source.health || '').trim() });
		}

		ctx.theme = normalizeTheme(ctx.theme);
		ctx.identity = normalizeIdentity(ctx.identity);
		ctx.lifecycle = normalizeLifecycle(ctx.lifecycle);
		ctx.locales = freezeList(ctx.locales);

		function bridgeError(code, message, detail, report) {
			var error = new Error(message);
			error.name = 'PluginHostError';
			error.code = code;
			error.detail = freezeRecord(detail);
			if (report !== false && window.parent && window.parent !== window) {
				window.parent.postMessage({
					type: 'skoll:plugin-error',
					contract: ctx.contract,
					version: ctx.version,
					pluginId: ctx.pluginId,
					error: { code: code, message: message, detail: error.detail }
				}, window.location.origin);
			}
			return error;
		}

		function hasCapability(capability) {
			return allowedCapabilities.indexOf(String(capability || '').trim()) >= 0;
		}

		function requireCapability(capability) {
			if (!hasCapability(capability)) {
				throw bridgeError('CAPABILITY_MISSING', "Skoll plugin host capability '" + capability + "' is unavailable", { capability: String(capability || '') });
			}
		}

		function withPrefix(path) {
			var value = String(path || '').trim();
			if (!value) return ctx.apiBasePrefix;
			if (/^https?:\\/\\//i.test(value)) throw bridgeError('CONTRACT_MISMATCH', 'Absolute plugin host request URLs are forbidden', { path: value });
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
			ctx.theme = normalizeTheme(nextTheme || ctx.theme);
			if (document && document.documentElement) {
				var root = document.documentElement;
				root.setAttribute('data-theme', ctx.theme.colorScheme);
				root.setAttribute('data-density', ctx.theme.density);
				root.style.colorScheme = ctx.theme.colorScheme;
				Object.keys(ctx.theme.tokens).forEach(function (key) { root.style.setProperty(key, ctx.theme.tokens[key]); });
			}
			window.dispatchEvent(new CustomEvent('skoll:theme', { detail: ctx.theme }));
		}

		function request(path, options) {
			requireCapability('request');
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
						throw bridgeError('REQUEST_FAILED', message, { status: String(response.status) }, false);
					}
					return payload && Object.prototype.hasOwnProperty.call(payload, 'data') ? payload.data : payload;
				});
			});
		}

		function postHostMessage(type, detail) {
			if (!window.parent || window.parent === window) {
				throw bridgeError('HOST_UNAVAILABLE', 'Skoll plugin parent host is unavailable');
			}
			window.parent.postMessage(Object.assign({
				type: type,
				contract: ctx.contract,
				version: ctx.version,
				pluginId: ctx.pluginId
			}, detail || {}), window.location.origin);
		}

		function navigate(mode, path) {
			requireCapability('navigation');
			var target = String(path || '').trim();
			if (!target || /^([a-z]+:)?\\/\\//i.test(target) || target.indexOf('..') >= 0) {
				throw bridgeError('CONTRACT_MISMATCH', 'Plugin navigation target is invalid', { path: target });
			}
			postHostMessage('skoll:navigation', { mode: mode, path: target });
		}

		function executeCommand(command) {
			requireCapability('commands');
			var value = String(command || '').trim();
			if (value !== 'reload' && value !== 'home') {
				throw bridgeError('CONTRACT_MISMATCH', 'Plugin host command is invalid', { command: value });
			}
			postHostMessage('skoll:command', { command: value });
		}

		function hasPermission(permission) {
			requireCapability('permissions');
			var wanted = String(permission || '').trim();
			if (!wanted) return false;
			return ctx.identity.permissions.some(function (granted) {
				return granted === '*' || granted === wanted || (granted.slice(-2) === '.*' && wanted.indexOf(granted.slice(0, -1)) === 0);
			});
		}

		function requirePermission(permission) {
			if (!hasPermission(permission)) {
				throw bridgeError('PERMISSION_DENIED', "Plugin permission '" + permission + "' is denied", { permission: String(permission || '') });
			}
		}

		function service(capability, value) {
			return Object.freeze(Object.keys(value).reduce(function (result, key) {
				result[key] = function () {
					requireCapability(capability);
					return value[key].apply(null, arguments);
				};
				return result;
			}, {}));
		}

		var host = {
			hasCapability: hasCapability,
			requireCapability: requireCapability,
			request: request,
			navigation: Object.freeze({
				push: function (path) { navigate('push', path); },
				replace: function (path) { navigate('replace', path); },
				back: function () { requireCapability('navigation'); postHostMessage('skoll:navigation', { mode: 'back' }); }
			}),
			commands: Object.freeze({ execute: executeCommand }),
			permissions: Object.freeze({ has: hasPermission, require: requirePermission }),
			auth: service('auth', { me: function () { return request('/v1/auth/me'); } }),
			user: service('user', { me: function () { return request('/v1/auth/me'); } }),
			organization: service('organization', {
				departments: function () { return request('/v1/system/settings/skoll.organization.departments'); },
				positions: function () { return request('/v1/system/settings/skoll.organization.positions'); }
			}),
			dictionary: service('dictionary', {
				list: function () { return request('/v1/system/dictionaries'); },
				items: function (type) { return request('/v1/system/dictionaries/' + encodeURIComponent(type) + '/items'); }
			}),
			file: service('file', { list: function (params) { return request('/v1/files' + query(params)); } }),
			audit: service('audit', { list: function (params) { return request('/v1/audit' + query(params)); } }),
			config: service('config', {
				get: function () { return request('/v1/plugins/' + encodeURIComponent(ctx.pluginId) + '/config'); },
				update: function (config) { return request('/v1/plugins/' + encodeURIComponent(ctx.pluginId) + '/config', { method: 'PUT', body: { config: config || {} } }); }
			})
		};

		Object.defineProperties(host, {
			contract: { enumerable: true, get: function () { return ctx.contract; } },
			version: { enumerable: true, get: function () { return ctx.version; } },
			pluginId: { enumerable: true, get: function () { return ctx.pluginId; } },
			pluginVersion: { enumerable: true, get: function () { return ctx.pluginVersion; } },
			capabilities: { enumerable: true, get: function () { return allowedCapabilities; } },
			identity: { enumerable: true, get: function () { return ctx.identity; } },
			locale: { enumerable: true, get: function () { return ctx.locale; } },
			locales: { enumerable: true, get: function () { return ctx.locales; } },
			theme: { enumerable: true, get: function () { return ctx.theme; } },
			lifecycle: { enumerable: true, get: function () { return ctx.lifecycle; } }
		});
		Object.freeze(host);
		Object.defineProperty(window, '__SKOLL_HOST__', { value: host, enumerable: false, configurable: false, writable: false });
		if (document && document.documentElement) document.documentElement.setAttribute('lang', ctx.locale);
		applyTheme(ctx.theme);
		window.dispatchEvent(new CustomEvent('skoll:host-ready', { detail: host }));
		window.dispatchEvent(new CustomEvent('skoll:locale', { detail: { locale: ctx.locale, locales: ctx.locales } }));

		window.addEventListener('message', function (event) {
			if (event.source !== window.parent || event.origin !== window.location.origin) return;
			var data = event && event.data;
			if (!data || data.contract !== ctx.contract || data.version !== ctx.version || data.pluginId !== ctx.pluginId) return;
			if (data.type === 'skoll:locale') {
				var locale = String(data.locale || '').trim();
				var locales = freezeList(data.locales);
				if (!locale || locales.indexOf(locale) < 0) return;
				ctx.locale = locale;
				ctx.locales = locales;
				if (document && document.documentElement) document.documentElement.setAttribute('lang', ctx.locale);
				window.dispatchEvent(new CustomEvent('skoll:locale', { detail: { locale: ctx.locale, locales: ctx.locales } }));
				return;
			}
			if (data.type === 'skoll:theme') {
				applyTheme(data.theme);
				return;
			}
			if (data.type === 'skoll:lifecycle') {
				ctx.lifecycle = normalizeLifecycle(data.lifecycle);
				window.dispatchEvent(new CustomEvent('skoll:lifecycle', { detail: ctx.lifecycle }));
			}
		});
	})();
	</script>`;
}
