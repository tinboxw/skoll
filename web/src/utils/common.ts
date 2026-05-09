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
	if (error instanceof Error) {
		return error.message;
	}
	return String(error);
}

