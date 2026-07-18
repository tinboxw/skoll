import { defineStore } from "pinia";

import { hasPermissionValue } from "../permissions/implied";
import { apiGet, type ApiResponse } from "../utils/api";
import { clearToken, getToken, hasToken, setToken } from "../utils/auth";

const USER_SESSION_KEY = "skoll.auth.userSession";

export type UserProfile = {
	id: string;
	name: string;
	role: string;
	roles: string[];
	organizationId: string;
	organizationPath: string[];
	email?: string;
	avatarUrl?: string;
};

type UserState = {
	token: string;
	profile: UserProfile | null;
	permissions: string[];
};

type UserSessionSnapshot = {
	profile: UserProfile;
	permissions: string[];
};

function loadPersistedSession(): UserSessionSnapshot | null {
	const raw = localStorage.getItem(USER_SESSION_KEY);
	if (!raw) {
		return null;
	}
	try {
		const parsed = JSON.parse(raw) as Partial<UserSessionSnapshot>;
		if (!parsed.profile || typeof parsed.profile !== "object") {
			return null;
		}
		const profile = parsed.profile as Partial<UserProfile>;
		if (
			typeof profile.id !== "string" ||
			typeof profile.name !== "string" ||
			typeof profile.role !== "string" ||
			!Array.isArray(profile.roles) ||
			typeof profile.organizationId !== "string" ||
			!Array.isArray(profile.organizationPath)
		) {
			return null;
		}
		return {
			profile: {
				id: profile.id,
				name: profile.name,
				role: profile.role,
				roles: cleanStringArray(profile.roles),
				organizationId: profile.organizationId,
				organizationPath: cleanStringArray(profile.organizationPath),
				email: typeof profile.email === "string" ? profile.email : "",
				avatarUrl: typeof profile.avatarUrl === "string" ? profile.avatarUrl : ""
			},
			permissions: cleanStringArray(parsed.permissions)
		};
	} catch {
		return null;
	}
}

function persistSession(profile: UserProfile, permissions: string[]): void {
	localStorage.setItem(
		USER_SESSION_KEY,
		JSON.stringify({
			profile,
			permissions
		})
	);
}

function clearPersistedSession(): void {
	localStorage.removeItem(USER_SESSION_KEY);
}

export function getStoredUserRole(): string {
	const session = loadPersistedSession();
	return session?.profile.role?.trim() ?? "";
}

export function getStoredPermissions(): string[] {
	const session = loadPersistedSession();
	return Array.isArray(session?.permissions) ? session!.permissions : [];
}

export function hasStoredPermission(permission: string): boolean {
	const required = permission.trim();
	if (required === "") {
		return true;
	}
	if (getStoredUserRole() === "super_admin") {
		return true;
	}
	return hasPermissionValue(getStoredPermissions(), required);
}

function cleanStringArray(value: unknown): string[] {
	return Array.isArray(value)
		? value.filter((item): item is string => typeof item === "string" && item.trim() !== "").map((item) => item.trim())
		: [];
}

const persistedSession = loadPersistedSession();

export const useUserStore = defineStore("user", {
	state: (): UserState => ({
		token: getToken(),
		profile: hasToken() ? persistedSession?.profile ?? null : null,
		permissions: hasToken() ? persistedSession?.permissions ?? [] : []
	}),
	getters: {
		isAuthenticated: (state): boolean => state.token.trim() !== ""
	},
	actions: {
		setProfile(profile: UserProfile): void {
			this.profile = profile;
			if (this.profile) {
				persistSession(this.profile, this.permissions);
			}
		},
		setSession(token: string, profile: UserProfile, permissions: string[] = []): void {
			this.token = token;
			this.profile = profile;
			this.permissions = permissions;
			setToken(token);
			persistSession(profile, permissions);
		},
		async hydrateProfile(): Promise<void> {
			if (!this.isAuthenticated) {
				return;
			}
			type MePayload = {
				id: string;
				account?: string;
				name?: string;
				email?: string;
				role?: string;
				roles?: string[];
				organizationId?: string;
				organizationPath?: string[];
				permissions?: string[];
			};
			try {
				const payload = await apiGet<ApiResponse<MePayload>>("/v1/auth/me");
				const data = payload.data;
				const profile: UserProfile = {
					id: String(data?.id ?? this.profile?.id ?? ""),
					name: String(data?.name ?? data?.account ?? this.profile?.name ?? "").trim() || "User",
					role: String(data?.role ?? this.profile?.role ?? "user").trim() || "user",
					roles: cleanStringArray(data?.roles).length > 0 ? cleanStringArray(data?.roles) : this.profile?.roles ?? ["user"],
					organizationId: String(data?.organizationId ?? this.profile?.organizationId ?? "").trim(),
					organizationPath: Array.isArray(data?.organizationPath)
						? cleanStringArray(data.organizationPath)
						: this.profile?.organizationPath ?? [],
					email: String(data?.email ?? this.profile?.email ?? ""),
					avatarUrl: this.profile?.avatarUrl ?? ""
				};
				this.profile = profile;
				this.permissions = Array.isArray(data?.permissions)
					? cleanStringArray(data.permissions)
					: this.permissions;
				persistSession(profile, this.permissions);
			} catch {
				// Keep existing local session if /auth/me is temporarily unavailable.
			}
		},
		logout(): void {
			this.token = "";
			this.profile = null;
			this.permissions = [];
			clearToken();
			clearPersistedSession();
		}
	}
});

