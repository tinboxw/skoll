const SYSTEM_LABELS_BY_PATH: ReadonlyArray<readonly [string, string]> = [
	["/skoll/form-builder", "menu.formBuilder"],
	["/skoll/workflow", "menu.workflow"],
	["/skoll/todo", "menu.todo"]
];

const SYSTEM_LABELS_BY_ID: Readonly<Record<string, string>> = {
	workflow: "menu.workflow",
	todo: "menu.todo",
	"todo-center": "menu.todo",
	formBuilder: "menu.formBuilder",
	"form-builder": "menu.formBuilder"
};

export function resolveSystemNavigationLabel(
	path: string,
	id: string,
	t: (key: string) => string,
	fallback: string
): string {
	const normalizedPath = String(path || "").trim().replace(/\/+$/, "") || "/";
	const pathMatch = SYSTEM_LABELS_BY_PATH.find(([base]) => normalizedPath === base || normalizedPath.startsWith(`${base}/`));
	const key = pathMatch?.[1] || SYSTEM_LABELS_BY_ID[String(id || "").trim()];
	if (!key) {
		return fallback;
	}
	const translated = t(key);
	return translated === key ? fallback : translated;
}
