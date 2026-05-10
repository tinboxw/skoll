import { translate } from "../i18n";
import { ApiError } from "./api";

export function formatDateTime(iso?: string): string {
	if (!iso) {
		return "-";
	}
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) {
		return "-";
	}
	return d.toLocaleString();
}

export function toErrorMessage(error: unknown): string {
	if (error instanceof ApiError) {
		const code = error.code.trim().toLowerCase();
		if (code === "network_error") {
			return translate("error.network");
		}
		if (code === "invalid_credentials") {
			return translate("error.invalidCredentials");
		}
		if (code === "unauthorized") {
			return translate("error.unauthorized");
		}
		if (code === "forbidden") {
			return translate("error.forbidden");
		}
		if (code === "not_found") {
			return translate("error.notFound");
		}

		if (error.status === 0) {
			return translate("error.network");
		}
		if (error.status === 401) {
			return translate("error.unauthorized");
		}
		if (error.status === 403) {
			return translate("error.forbidden");
		}
		if (error.status === 404) {
			return translate("error.notFound");
		}
		if (error.status >= 500) {
			return translate("error.server");
		}
		return error.message?.trim() ? error.message : translate("error.requestFailed");
	}

	if (error instanceof Error) {
		if (error.message.toLowerCase() === "failed to fetch") {
			return translate("error.network");
		}
		return error.message;
	}

	const message = String(error);
	if (message.toLowerCase() === "failed to fetch") {
		return translate("error.network");
	}
	return message;
}

