import { describe, expect, it } from "vitest";

import {
	BusinessCommandBar,
	BusinessComments,
	BusinessDocumentDetail,
	BusinessDocumentForm,
	BusinessDocumentList,
	BusinessFieldInput,
	BusinessFilePanel,
	BusinessFilterBar,
	BusinessList,
	BusinessState,
	BusinessTimeline,
	BusinessWorkflowPanel,
	BusinessWorkspace,
	BUSINESS_THEME_TOKENS
} from "../src";
import {
	BusinessCommandBar as CoreCommandBar,
	BusinessFilterBar as CoreFilterBar,
	BusinessList as CoreList,
	BusinessWorkspace as CoreWorkspace
} from "../src/core";

describe("business UI public contract", () => {
	it("exports every plugin workspace composition primitive", () => {
		expect([
			BusinessCommandBar,
			BusinessComments,
			BusinessDocumentDetail,
			BusinessDocumentForm,
			BusinessDocumentList,
			BusinessFieldInput,
			BusinessFilePanel,
			BusinessFilterBar,
			BusinessList,
			BusinessState,
			BusinessTimeline,
			BusinessWorkflowPanel,
			BusinessWorkspace
		]).not.toContain(undefined);
		expect(BUSINESS_THEME_TOKENS).toContain("--color-primary");
		expect(BUSINESS_THEME_TOKENS).toContain("--control-height");
	});

	it("exports the lightweight core contract without document primitives", () => {
		expect([CoreCommandBar, CoreFilterBar, CoreList, CoreWorkspace]).not.toContain(undefined);
	});
});
