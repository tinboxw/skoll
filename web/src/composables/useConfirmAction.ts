import { ElMessageBox } from "element-plus";
import type { Action } from "element-plus";

export type ConfirmActionOptions = {
	title: string;
	message: string;
	confirmText: string;
	cancelText: string;
	type?: "success" | "warning" | "info" | "error";
	danger?: boolean;
};

export async function confirmAction(options: ConfirmActionOptions): Promise<boolean> {
	try {
		const action = await ElMessageBox.confirm(options.message, options.title, {
			type: options.type ?? (options.danger ? "warning" : "info"),
			confirmButtonText: options.confirmText,
			cancelButtonText: options.cancelText,
			confirmButtonClass: options.danger ? "el-button--danger" : undefined,
			distinguishCancelAndClose: true
		});
		return action === "confirm";
	} catch (action) {
		return (action as Action) === "confirm";
	}
}
