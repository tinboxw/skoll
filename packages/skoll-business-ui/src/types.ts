import type { Component } from "vue";

export type BusinessLocale = "zh-CN" | "en-US";
export type BusinessState = "ready" | "loading" | "empty" | "error" | "forbidden" | "conflict" | "destructive" | "success";
export type BusinessTone = "primary" | "success" | "warning" | "danger" | "info";

export type BusinessCommand = {
	id: string;
	label: string;
	icon?: Component;
	tone?: BusinessTone;
	disabled?: boolean;
	loading?: boolean;
	destructive?: boolean;
	confirmation?: string;
};

export type BusinessFilterOption = {
	label: string;
	value: string | number | boolean;
};

export type BusinessFilterField = {
	key: string;
	label: string;
	type: "search" | "text" | "select" | "date";
	placeholder?: string;
	clearable?: boolean;
	options?: readonly BusinessFilterOption[];
	defaultValue?: string | number | boolean;
};

export type BusinessFilterValue = string | number | boolean | undefined;
export type BusinessFilterModel = Record<string, BusinessFilterValue>;

export type BusinessListColumn = {
	key: string;
	label: string;
	width?: string | number;
	minWidth?: string | number;
	fixed?: "left" | "right";
	overflowTooltip?: boolean;
};

export type BusinessRecord = Record<string, unknown>;
