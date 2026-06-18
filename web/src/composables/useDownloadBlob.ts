export type DownloadBlobOptions = {
	blob: Blob;
	filename: string;
};

export function downloadBlob(options: DownloadBlobOptions): void {
	if (typeof document === "undefined" || typeof URL === "undefined") {
		return;
	}
	const url = URL.createObjectURL(options.blob);
	try {
		const anchor = document.createElement("a");
		anchor.href = url;
		anchor.download = normalizeFilename(options.filename);
		document.body.appendChild(anchor);
		anchor.click();
		anchor.remove();
	} finally {
		URL.revokeObjectURL(url);
	}
}

function normalizeFilename(filename: string): string {
	const normalized = filename.trim();
	return normalized === "" ? "download" : normalized;
}
