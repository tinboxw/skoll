import { defineStore } from "pinia";

import { clearToken, getToken, hasToken, setToken } from "../utils/auth";

const USER_SESSION_KEY = "skoll.auth.userSession";

type UserProfile = {
	id: string;
	name: string;
	role: string;
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
		if (typeof profile.id !== "string" || typeof profile.name !== "string" || typeof profile.role !== "string") {
			return null;
		}
		return {
			profile: {
				id: profile.id,
				name: profile.name,
				role: profile.role
			},
			permissions: Array.isArray(parsed.permissions)
				? parsed.permissions.filter((item): item is string => typeof item === "string" && item.trim() !== "")
				: []
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

const persistedSession = loadPersistedSession();

export const useUserStore = defineStore("user", {
	state: (): UserState => ({
		token: getToken(),
		profile: hasToken()
			? persistedSession?.profile ?? {
				id: "local-admin",
				name: "Admin",
				role: "super_admin"
			}
			: null,
		permissions: hasToken() ? persistedSession?.permissions ?? [] : []
	}),
	getters: {
		isAuthenticated: (state): boolean => state.token.trim() !== ""
	},
	actions: {
		setSession(token: string, profile: UserProfile, permissions: string[] = []): void {
			this.token = token;
			this.profile = profile;
			this.permissions = permissions;
			setToken(token);
			persistSession(profile, permissions);
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

