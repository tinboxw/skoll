const TOKEN_KEY = "skoll.auth.token";

export function getToken(): string {
	return localStorage.getItem(TOKEN_KEY) ?? "";
}

export function setToken(token: string): void {
	if (token.trim() === "") {
		localStorage.removeItem(TOKEN_KEY);
		return;
	}
	localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken(): void {
	localStorage.removeItem(TOKEN_KEY);
}

export function hasToken(): boolean {
	return getToken().trim() !== "";
}

