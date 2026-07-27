import type { BusinessLocale, BusinessState } from "./types";

type BusinessMessages = {
	search: string;
	reset: string;
	retry: string;
	confirm: string;
	cancel: string;
	previous: string;
	next: string;
	open: string;
	states: Record<Exclude<BusinessState, "ready">, { title: string; description: string }>;
};

const messages: Record<BusinessLocale, BusinessMessages> = {
	"zh-CN": {
		search: "查询",
		reset: "重置",
		retry: "重试",
		confirm: "确认",
		cancel: "取消",
		previous: "上一页",
		next: "下一页",
		open: "打开",
		states: {
			loading: { title: "正在加载", description: "正在获取最新业务数据" },
			empty: { title: "暂无数据", description: "当前条件下没有可显示的记录" },
			error: { title: "加载失败", description: "业务数据加载失败，请重试" },
			forbidden: { title: "无权访问", description: "当前账号没有此业务区域的访问权限" },
			conflict: { title: "数据已更新", description: "其他操作已修改当前数据，请刷新后继续" },
			destructive: { title: "确认高风险操作", description: "此操作会改变业务数据且可能无法撤销" },
			success: { title: "操作成功", description: "业务操作已完成" }
		}
	},
	"en-US": {
		search: "Search",
		reset: "Reset",
		retry: "Retry",
		confirm: "Confirm",
		cancel: "Cancel",
		previous: "Previous",
		next: "Next",
		open: "Open",
		states: {
			loading: { title: "Loading", description: "Retrieving current business data" },
			empty: { title: "No records", description: "No records match the current criteria" },
			error: { title: "Load failed", description: "Business data could not be loaded" },
			forbidden: { title: "Access denied", description: "Your account cannot access this business area" },
			conflict: { title: "Data changed", description: "Refresh before continuing because another operation changed this data" },
			destructive: { title: "Confirm high-risk action", description: "This action changes business data and may not be reversible" },
			success: { title: "Completed", description: "The business operation completed successfully" }
		}
	}
};

export function businessMessages(locale: BusinessLocale): BusinessMessages {
	return messages[locale];
}
