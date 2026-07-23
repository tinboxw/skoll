(function () {
	"use strict";

	var apiBase = "/v1/plugins/pharma_oa/api";
	var partyCollections = { customer: "/customers", supplier: "/suppliers" };
	var catalogCollections = { product: "/products", category: "/categories", unit: "/units", manufacturer: "/manufacturers" };
	var state = { items: [], reminders: [], lookups: {}, editing: null, target: null, loading: false, locale: "zh-CN", module: "employee", statusAction: "disable" };
	var app = document.querySelector("#app");
	var employeeDialog = document.querySelector("#employee-dialog");
	var attachmentDialog = document.querySelector("#attachment-dialog");
	var leaveDialog = document.querySelector("#leave-dialog");
	var employeeForm = document.querySelector("#employee-form");
	var attachmentForm = document.querySelector("#attachment-form");
	var leaveForm = document.querySelector("#leave-form");
	var partyDialog = document.querySelector("#party-dialog");
	var partyForm = document.querySelector("#party-form");
	var partyStatusDialog = document.querySelector("#party-status-dialog");
	var partyStatusForm = document.querySelector("#party-status-form");
	var catalogDialog = document.querySelector("#catalog-dialog");
	var catalogForm = document.querySelector("#catalog-form");
	var productDialog = document.querySelector("#product-dialog");
	var productForm = document.querySelector("#product-form");
	var translations = {
		"zh-CN": {
			module: "医药 OA · 人力资源", title: "员工管理", subtitle: "员工档案、任职信息与资质到期统一管理", newEmployee: "新建员工", tenant: "租户", organization: "组织", applyScope: "应用范围", totalEmployees: "员工总数", activeEmployees: "在职员工", departedEmployees: "已离职", expiringQualifications: "资质提醒", search: "搜索", searchPlaceholder: "搜索姓名、工号、部门或邮箱", employmentStatus: "任职状态", allStatuses: "全部状态", active: "在职", onLeave: "休假", left: "已离职", loading: "正在加载员工档案", loadFailed: "员工档案加载失败", retry: "重新加载", permissionRequired: "需要员工档案访问权限", contactAdmin: "请联系管理员配置医药 OA 员工权限。", noEmployees: "暂无员工档案", emptyHint: "创建首位员工，开始维护组织任职与资质信息。", employee: "员工", assignment: "任职信息", contact: "联系方式", qualifications: "资质", actions: "操作", employeeRecord: "员工档案", employeeCode: "员工工号", employeeName: "姓名", department: "部门", position: "职位", phone: "手机号", email: "邮箱", hireDate: "入职日期", qualification: "执业资质", qualificationName: "资质名称", qualificationNumber: "证书编号", expiryDate: "有效期至", cancel: "取消", save: "保存", employeeAttachment: "员工附件", selectFile: "选择文件", upload: "上传", employmentChange: "任职变更", leaveReason: "离职原因", confirmLeave: "确认离职", edit: "编辑", attachment: "附件", processLeave: "离职", records: "条", certificates: "项资质", noCertificate: "无资质", noContact: "未填写联系方式", hiredOn: "入职", createTitle: "新建员工", editTitle: "编辑员工", uploadTitle: "上传附件", leaveTitle: "办理离职", saved: "员工档案已保存", uploaded: "附件已上传", leftDone: "离职手续已完成", scopeApplied: "业务范围已更新", fileTooLarge: "文件不能超过 1 MB", hostUnavailable: "未连接 SKOLL 插件宿主", unknownError: "操作未完成，请稍后重试", saving: "保存中", uploading: "上传中", processing: "处理中", refresh: "刷新"
		},
		"en-US": {
			module: "Pharma OA · Workforce", title: "Employees", subtitle: "Manage employee records, assignments, and qualification expiry", newEmployee: "New employee", tenant: "Tenant", organization: "Organization", applyScope: "Apply scope", totalEmployees: "Employees", activeEmployees: "Active", departedEmployees: "Departed", expiringQualifications: "Qualification alerts", search: "Search", searchPlaceholder: "Search name, code, department, or email", employmentStatus: "Employment status", allStatuses: "All statuses", active: "Active", onLeave: "On leave", left: "Departed", loading: "Loading employee records", loadFailed: "Unable to load employee records", retry: "Retry", permissionRequired: "Employee access required", contactAdmin: "Ask an administrator to grant Pharma OA employee permissions.", noEmployees: "No employee records", emptyHint: "Create the first employee to manage assignments and qualifications.", employee: "Employee", assignment: "Assignment", contact: "Contact", qualifications: "Qualifications", actions: "Actions", employeeRecord: "Employee record", employeeCode: "Employee code", employeeName: "Name", department: "Department", position: "Position", phone: "Phone", email: "Email", hireDate: "Hire date", qualification: "Professional qualification", qualificationName: "Qualification", qualificationNumber: "Certificate number", expiryDate: "Expires on", cancel: "Cancel", save: "Save", employeeAttachment: "Employee attachment", selectFile: "Select file", upload: "Upload", employmentChange: "Employment change", leaveReason: "Departure reason", confirmLeave: "Confirm departure", edit: "Edit", attachment: "Attachment", processLeave: "Depart", records: "records", certificates: "qualifications", noCertificate: "No qualification", noContact: "No contact details", hiredOn: "Hired", createTitle: "New employee", editTitle: "Edit employee", uploadTitle: "Upload attachment", leaveTitle: "Process departure", saved: "Employee saved", uploaded: "Attachment uploaded", leftDone: "Departure completed", scopeApplied: "Business scope updated", fileTooLarge: "File must not exceed 1 MB", hostUnavailable: "SKOLL plugin host is unavailable", unknownError: "The operation could not be completed", saving: "Saving", uploading: "Uploading", processing: "Processing", refresh: "Refresh"
		}
	};

	function text(key) {
		return translations[state.locale][key] || key;
	}

	function applyLocale(locale) {
		state.locale = locale === "en-US" ? "en-US" : "zh-CN";
		document.documentElement.lang = state.locale;
		document.querySelectorAll("[data-i18n]").forEach(function (node) { node.textContent = text(node.dataset.i18n); });
		document.querySelectorAll("[data-i18n-placeholder]").forEach(function (node) { node.placeholder = text(node.dataset.i18nPlaceholder); });
		document.querySelector("#refresh-button").title = text("refresh");
		document.querySelector("#refresh-button").setAttribute("aria-label", text("refresh"));
		applyModule();
	}

	function moduleCopy() {
		var english = state.locale === "en-US";
		if (state.module === "employee") return { eyebrow: text("module"), search: text("searchPlaceholder"), title: text("title"), subtitle: text("subtitle"), create: text("newEmployee"), total: text("totalEmployees"), active: text("activeEmployees"), inactive: text("departedEmployees"), alert: text("expiringQualifications"), heads: [text("employee"), text("assignment"), text("contact"), text("qualifications"), text("employmentStatus"), text("actions")] };
		if (partyCollections[state.module]) {
			var customer = state.module === "customer";
			var singular = english ? (customer ? "Customer" : "Supplier") : (customer ? "客户" : "供应商");
			return { eyebrow: english ? "Pharma OA · Parties" : "医药 OA · 往来单位", search: english ? "Search name, code, credit code, or region" : "搜索单位名称、编码、信用代码或地区", title: english ? singular + " master data" : singular + "管理", subtitle: english ? "Identity, contacts, addresses, settlement, and lifecycle" : "统一维护单位身份、联系人、地址、结算条款与合作状态", create: english ? "New " + singular.toLowerCase() : "新建" + singular, total: english ? "Total" : singular + "总数", active: english ? "Active" : "合作中", inactive: english ? "Disabled" : "已停用", alert: english ? "Average rating" : "平均评级", heads: [singular, english ? "Identity" : "单位身份", english ? "Primary contact" : "主联系人", english ? "Settlement" : "结算条款", english ? "Status" : "合作状态", text("actions")] };
		}
		if (state.module === "product") return { eyebrow: english ? "Pharma OA · Catalog" : "医药 OA · 药品目录", search: english ? "Search product, SKU, approval number, or barcode" : "搜索药品名称、编码、SKU、批准文号或条码", title: english ? "Products" : "药品管理", subtitle: english ? "Govern product identity, pharmaceutical attributes, catalogs, and storage" : "统一维护药品身份、剂型规格、基础目录引用与储存要求", create: english ? "New product" : "新建药品", total: english ? "Products" : "药品总数", active: english ? "Active" : "启用药品", inactive: english ? "Disabled" : "已停用", alert: english ? "Cold-chain" : "低温药品", heads: [english ? "Product" : "药品", english ? "Pharmaceutical" : "剂型规格", english ? "Catalogs" : "目录归属", english ? "Storage" : "储存要求", english ? "Status" : "状态", text("actions")] };
		var labels = { category: english ? "Category" : "分类", unit: english ? "Unit" : "单位", manufacturer: english ? "Manufacturer" : "生产企业" };
		var singularCatalog = labels[state.module];
		return { eyebrow: english ? "Pharma OA · Catalog" : "医药 OA · 基础目录", search: english ? "Search code, name, identity, or license" : "搜索编码、名称、单位符号、信用代码或许可证", title: english ? singularCatalog + " catalog" : singularCatalog + "管理", subtitle: english ? "Govern reusable pharmaceutical catalog references and lifecycle" : "统一维护可复用目录、业务约束、引用关系与启停状态", create: english ? "New " + singularCatalog.toLowerCase() : "新建" + singularCatalog, total: english ? "Total" : singularCatalog + "总数", active: english ? "Active" : "启用中", inactive: english ? "Disabled" : "已停用", alert: state.module === "category" ? (english ? "Root categories" : "一级分类") : state.module === "unit" ? (english ? "Precise units" : "精度单位") : (english ? "Licensed" : "证照完整"), heads: [singularCatalog, english ? "Business identity" : "业务标识", english ? "Catalog detail" : "目录属性", english ? "Description" : "说明", english ? "Status" : "状态", text("actions")] };
	}

	function applyModule() {
		var copy = moduleCopy();
		document.querySelector(".page-header .eyebrow").textContent = copy.eyebrow;
		document.querySelector("h1").textContent = copy.title;
		document.querySelector(".subtitle").textContent = copy.subtitle;
		document.querySelector("#create-button span:last-child").textContent = copy.create;
		document.querySelector("#empty-create-button").textContent = copy.create;
		document.querySelector("#keyword-input").placeholder = copy.search;
		document.querySelectorAll(".metrics article span").forEach(function (node, index) { node.textContent = [copy.total, copy.active, copy.inactive, copy.alert][index]; });
		document.querySelectorAll("thead th").forEach(function (node, index) { node.textContent = copy.heads[index]; });
		document.querySelectorAll("[data-module]").forEach(function (button) { button.classList.toggle("active", button.dataset.module === state.module); });
		var status = document.querySelector("#status-filter");
		status.innerHTML = state.module === "employee" ? '<option value="">'+text("allStatuses")+'</option><option value="active">'+text("active")+'</option><option value="on_leave">'+text("onLeave")+'</option><option value="left">'+text("left")+'</option>' : '<option value="">'+text("allStatuses")+'</option><option value="active">'+(state.locale === "en-US" ? "Active" : "合作中")+'</option><option value="disabled">'+(state.locale === "en-US" ? "Disabled" : "已停用")+'</option>';
		var english = state.locale === "en-US";
		document.querySelector(".loading-state p").textContent = english ? "Loading " + copy.title.toLowerCase() : "正在加载" + copy.title;
		document.querySelector(".error-state strong").textContent = english ? "Unable to load " + copy.title.toLowerCase() : copy.title + "加载失败";
		document.querySelector(".denied-state strong").textContent = english ? "Access required" : "需要" + copy.title + "访问权限";
		document.querySelector(".denied-state p").textContent = english ? "Ask an administrator to grant the required Pharma OA permissions." : "请联系管理员配置对应的医药 OA 权限。";
		document.querySelector(".empty-state strong").textContent = english ? "No records" : "暂无" + copy.title + "数据";
		document.querySelector(".empty-state p").textContent = english ? "Create the first record for this module." : "新建首条记录，开始维护该模块的业务主数据。";
		render();
	}

	function collectionForModule() {
		if (state.module === "employee") return "/employees";
		return partyCollections[state.module] || catalogCollections[state.module];
	}

	function host() {
		var value = window.__SKOLL_HOST__;
		if (!value || value.pluginId !== "pharma_oa") throw new Error(text("hostUnavailable"));
		return value;
	}

	function query(values) {
		var params = new URLSearchParams();
		Object.keys(values).forEach(function (key) {
			if (values[key] !== "" && values[key] !== undefined) params.set(key, String(values[key]));
		});
		var encoded = params.toString();
		return encoded ? "?" + encoded : "";
	}

	function request(path, options) {
		return host().request(apiBase + path, options);
	}

	function idempotencyKey(prefix) {
		var suffix = window.crypto && window.crypto.randomUUID ? window.crypto.randomUUID() : Date.now().toString(36) + Math.random().toString(36).slice(2);
		return prefix + "-" + suffix;
	}

	function scope() {
		return { tenantId: document.querySelector("#tenant-input").value.trim(), organizationId: document.querySelector("#organization-input").value.trim() };
	}

	function loadScope() {
		var stored = {};
		try { stored = JSON.parse(localStorage.getItem("pharma_oa.employee.scope") || "{}"); } catch (_error) { stored = {}; }
		document.querySelector("#tenant-input").value = stored.tenantId || "";
		document.querySelector("#organization-input").value = stored.organizationId || "";
	}

	function saveScope() {
		localStorage.setItem("pharma_oa.employee.scope", JSON.stringify(scope()));
		showToast(text("scopeApplied"));
	}

	async function load() {
		if (state.loading) return;
		state.loading = true;
		app.dataset.state = "loading";
		try {
			var keyword = document.querySelector("#keyword-input").value.trim();
			var status = document.querySelector("#status-filter").value;
			var collection = collectionForModule();
			var results;
			if (state.module === "employee") results = await Promise.all([request(collection + query({ keyword: keyword, status: status, limit: 200 })), request("/employees/qualification-reminders?days=30")]);
			else if (state.module === "product") results = await Promise.all([request(collection + query({ keyword: keyword, status: status, limit: 200 })), request("/categories?status=active&limit=200"), request("/units?status=active&limit=200"), request("/manufacturers?status=active&limit=200")]);
			else results = [await request(collection + query({ keyword: keyword, status: status, limit: 200 })), { items: [] }];
			state.items = Array.isArray(results[0].items) ? results[0].items : [];
			state.reminders = Array.isArray(results[1].items) ? results[1].items : [];
			state.lookups = state.module === "product" ? { category: indexById(results[1].items), unit: indexById(results[2].items), manufacturer: indexById(results[3].items) } : {};
			app.dataset.state = state.items.length ? "ready" : "empty";
			render();
		} catch (error) {
			var message = error && error.message ? error.message : text("unknownError");
			document.querySelector("#error-message").textContent = message;
			app.dataset.state = /forbidden|permission|denied|权限|拒绝/i.test(message) ? "denied" : "error";
		} finally {
			state.loading = false;
		}
	}

	function render() {
		if (partyCollections[state.module]) { renderParties(); return; }
		if (state.module === "product") { renderProducts(); return; }
		if (catalogCollections[state.module]) { renderCatalogs(); return; }
		var active = state.items.filter(function (item) { return item.employmentStatus === "active"; }).length;
		var left = state.items.filter(function (item) { return item.employmentStatus === "left"; }).length;
		document.querySelector("#metric-total").textContent = String(state.items.length);
		document.querySelector("#metric-active").textContent = String(active);
		document.querySelector("#metric-left").textContent = String(left);
		document.querySelector("#metric-reminders").textContent = String(state.reminders.length);
		document.querySelector("#result-count").textContent = state.items.length + " " + text("records");
		document.querySelector("#employee-table").innerHTML = state.items.map(renderRow).join("");
		document.querySelector("#employee-cards").innerHTML = state.items.map(renderCard).join("");
	}

	function renderParties() {
		var active = state.items.filter(function (item) { return item.status === "active"; }).length;
		var rating = state.items.length ? state.items.reduce(function (sum, item) { return sum + Number(item.rating || 0); }, 0) / state.items.length : 0;
		document.querySelector("#metric-total").textContent = String(state.items.length); document.querySelector("#metric-active").textContent = String(active); document.querySelector("#metric-left").textContent = String(state.items.length-active); document.querySelector("#metric-reminders").textContent = rating.toFixed(1);
		document.querySelector("#result-count").textContent = state.items.length + " " + text("records"); document.querySelector("#employee-table").innerHTML = state.items.map(renderPartyRow).join(""); document.querySelector("#employee-cards").innerHTML = state.items.map(renderPartyCard).join("");
	}

	function renderCatalogs() {
		var active = state.items.filter(function (item) { return item.status === "active"; }).length;
		var highlighted = state.items.filter(function (item) {
			if (state.module === "category") return !item.parentId;
			if (state.module === "unit") return Number(item.decimalPlaces || 0) > 0;
			return Boolean(item.unifiedSocialCreditCode && item.licenseNumber);
		}).length;
		setMetrics(state.items.length, active, state.items.length - active, highlighted);
		document.querySelector("#result-count").textContent = state.items.length + " " + text("records");
		document.querySelector("#employee-table").innerHTML = state.items.map(renderCatalogRow).join("");
		document.querySelector("#employee-cards").innerHTML = state.items.map(renderCatalogCard).join("");
	}

	function renderCatalogRow(item) {
		return '<tr><td><div class="employee-cell"><span class="avatar">'+escapeHTML(item.name.slice(0,1))+'</span><div><strong>'+escapeHTML(item.name)+'</strong><small>'+escapeHTML(item.code)+'</small></div></div></td><td>'+catalogIdentity(item)+'</td><td>'+catalogDetail(item)+'</td><td><span>'+escapeHTML(item.description||'-')+'</span></td><td>'+statusHTML(item)+'</td><td>'+catalogActionHTML(item)+'</td></tr>';
	}

	function renderCatalogCard(item) {
		return '<article class="employee-card"><header><div class="employee-cell"><span class="avatar">'+escapeHTML(item.name.slice(0,1))+'</span><div><strong>'+escapeHTML(item.name)+'</strong><small>'+escapeHTML(item.code)+'</small></div></div>'+statusHTML(item)+'</header><dl><div><dt>业务标识</dt><dd>'+catalogIdentity(item)+'</dd></div><div><dt>目录属性</dt><dd>'+catalogDetail(item)+'</dd></div><div><dt>说明</dt><dd>'+escapeHTML(item.description||'-')+'</dd></div></dl><footer>'+catalogActionHTML(item)+'</footer></article>';
	}

	function catalogIdentity(item) {
		if (state.module === "manufacturer") return '<strong>'+escapeHTML(item.unifiedSocialCreditCode)+'</strong><small>'+escapeHTML(item.licenseNumber)+'</small>';
		if (state.module === "unit") return '<strong>'+escapeHTML(item.symbol)+'</strong><small>'+Number(item.decimalPlaces||0)+' 位小数</small>';
		return '<strong>'+escapeHTML(item.code)+'</strong><small>'+(item.parentId?'子分类':'一级分类')+'</small>';
	}

	function catalogDetail(item) {
		if (state.module === "category") {
			var parent = state.items.find(function (candidate) { return candidate.id === item.parentId; });
			return '<strong>'+(parent?escapeHTML(parent.name):'无上级分类')+'</strong><small>'+escapeHTML(item.parentId||'根目录')+'</small>';
		}
		if (state.module === "unit") return '<strong>'+escapeHTML(item.name)+'（'+escapeHTML(item.symbol)+'）</strong><small>数量精度 '+Number(item.decimalPlaces||0)+'</small>';
		return '<strong>生产许可</strong><small>'+escapeHTML(item.licenseNumber)+'</small>';
	}

	function renderProducts() {
		var active = state.items.filter(function (item) { return item.status === "active"; }).length;
		var cold = state.items.filter(function (item) { return Number(item.temperatureMax) <= 8; }).length;
		setMetrics(state.items.length, active, state.items.length - active, cold);
		document.querySelector("#result-count").textContent = state.items.length + " " + text("records");
		document.querySelector("#employee-table").innerHTML = state.items.map(renderProductRow).join("");
		document.querySelector("#employee-cards").innerHTML = state.items.map(renderProductCard).join("");
	}

	function renderProductRow(item) {
		return '<tr><td><div class="employee-cell"><span class="avatar">'+escapeHTML(item.name.slice(0,1))+'</span><div><strong>'+escapeHTML(item.name)+'</strong><small>'+escapeHTML(item.code)+' · '+escapeHTML(item.sku)+'</small></div></div></td><td><strong>'+escapeHTML(item.dosageForm)+' · '+escapeHTML(item.specification)+'</strong><small>'+escapeHTML(item.approvalNumber)+'</small></td><td>'+productCatalogHTML(item)+'</td><td><strong>'+escapeHTML(item.storageCondition)+'</strong><small>'+Number(item.temperatureMin)+'℃ 至 '+Number(item.temperatureMax)+'℃</small></td><td>'+statusHTML(item)+'</td><td>'+productActionHTML(item)+'</td></tr>';
	}

	function renderProductCard(item) {
		return '<article class="employee-card"><header><div class="employee-cell"><span class="avatar">'+escapeHTML(item.name.slice(0,1))+'</span><div><strong>'+escapeHTML(item.name)+'</strong><small>'+escapeHTML(item.code)+' · '+escapeHTML(item.sku)+'</small></div></div>'+statusHTML(item)+'</header><dl><div><dt>剂型规格</dt><dd>'+escapeHTML(item.dosageForm)+' · '+escapeHTML(item.specification)+'<small>'+escapeHTML(item.approvalNumber)+'</small></dd></div><div><dt>目录归属</dt><dd>'+productCatalogHTML(item)+'</dd></div><div><dt>储存要求</dt><dd>'+escapeHTML(item.storageCondition)+'<small>'+Number(item.temperatureMin)+'℃ 至 '+Number(item.temperatureMax)+'℃</small></dd></div></dl><footer>'+productActionHTML(item)+'</footer></article>';
	}

	function productCatalogHTML(item) {
		return '<strong>'+escapeHTML(lookupName("category",item.categoryId))+' · '+escapeHTML(lookupName("unit",item.unitId))+'</strong><small>'+escapeHTML(lookupName("manufacturer",item.manufacturerId))+'</small>';
	}

	function lookupName(kind, id) {
		return state.lookups[kind] && state.lookups[kind][id] ? state.lookups[kind][id].name : id;
	}

	function indexById(items) {
		return (Array.isArray(items)?items:[]).reduce(function (result, item) { result[item.id]=item; return result; }, {});
	}

	function setMetrics(total, active, inactive, highlighted) {
		document.querySelector("#metric-total").textContent=String(total);
		document.querySelector("#metric-active").textContent=String(active);
		document.querySelector("#metric-left").textContent=String(inactive);
		document.querySelector("#metric-reminders").textContent=String(highlighted);
	}

	function statusHTML(item) {
		var label=item.status==='active'?(state.locale==='en-US'?'Active':'启用中'):(state.locale==='en-US'?'Disabled':'已停用');
		return '<span class="status status-'+escapeHTML(item.status)+'">'+label+'</span><small>'+escapeHTML(item.disableReason||'')+'</small>';
	}

	function catalogActionHTML(item) {
		var id=escapeHTML(item.id);
		return '<div class="row-actions"><button class="text-button" type="button" data-action="catalog-edit" data-id="'+id+'">'+text("edit")+'</button><button class="text-button '+(item.status==='active'?'danger-text':'')+'" type="button" data-action="catalog-status" data-id="'+id+'">'+(item.status==='active'?'停用':'启用')+'</button></div>';
	}

	function productActionHTML(item) {
		var id=escapeHTML(item.id);
		return '<div class="row-actions"><button class="text-button" type="button" data-action="product-edit" data-id="'+id+'">'+text("edit")+'</button><button class="text-button '+(item.status==='active'?'danger-text':'')+'" type="button" data-action="product-status" data-id="'+id+'">'+(item.status==='active'?'停用':'启用')+'</button></div>';
	}

	function renderPartyRow(item) {
		var contact = item.contacts && item.contacts[0] || {}; var settlement = item.settlementTerms || {};
		return '<tr><td><div class="employee-cell"><span class="avatar">'+escapeHTML(item.name.slice(0,1))+'</span><div><strong>'+escapeHTML(item.name)+'</strong><small>'+escapeHTML(item.code)+'</small></div></div></td><td><strong>'+escapeHTML(item.unifiedSocialCreditCode)+'</strong><small>'+escapeHTML(item.region)+' · '+Number(item.rating)+' 星</small></td><td><strong>'+escapeHTML(contact.name||'-')+'</strong><small>'+escapeHTML(contact.phone||contact.email||'')+'</small></td><td><strong>'+escapeHTML(settlement.currency||'CNY')+' · '+Number(settlement.paymentDays||0)+' 天</strong><small>授信 '+Number(settlement.creditLimit||0).toLocaleString(state.locale)+'</small></td><td><span class="status status-'+escapeHTML(item.status)+'">'+(item.status==='active'?(state.locale==='en-US'?'Active':'合作中'):(state.locale==='en-US'?'Disabled':'已停用'))+'</span><small>'+escapeHTML(item.disableReason||'')+'</small></td><td>'+partyActionHTML(item)+'</td></tr>';
	}
	function renderPartyCard(item) { var contact=item.contacts&&item.contacts[0]||{}, settlement=item.settlementTerms||{}; return '<article class="employee-card"><header><div class="employee-cell"><span class="avatar">'+escapeHTML(item.name.slice(0,1))+'</span><div><strong>'+escapeHTML(item.name)+'</strong><small>'+escapeHTML(item.code)+'</small></div></div><span class="status status-'+escapeHTML(item.status)+'">'+(item.status==='active'?'合作中':'已停用')+'</span></header><dl><div><dt>单位身份</dt><dd>'+escapeHTML(item.unifiedSocialCreditCode)+'<small>'+escapeHTML(item.region)+'</small></dd></div><div><dt>主联系人</dt><dd>'+escapeHTML(contact.name||'-')+'<small>'+escapeHTML(contact.phone||contact.email||'')+'</small></dd></div><div><dt>结算条款</dt><dd>'+escapeHTML(settlement.currency||'CNY')+' · '+Number(settlement.paymentDays||0)+' 天</dd></div></dl><footer>'+partyActionHTML(item)+'</footer></article>'; }
	function partyActionHTML(item) { var id=escapeHTML(item.id); return '<div class="row-actions"><button class="text-button" type="button" data-action="party-edit" data-id="'+id+'">'+text("edit")+'</button><button class="text-button '+(item.status==='active'?'danger-text':'')+'" type="button" data-action="party-status" data-id="'+id+'">'+(item.status==='active'?'停用':'启用')+'</button></div>'; }

	function renderRow(item) {
		return "<tr>" +
			"<td><div class=\"employee-cell\"><span class=\"avatar\">" + escapeHTML(item.name.slice(0, 1)) + "</span><div><strong>" + escapeHTML(item.name) + "</strong><small>" + escapeHTML(item.code) + "</small></div></div></td>" +
			"<td><strong>" + escapeHTML(item.departmentId) + "</strong><small>" + escapeHTML(item.positionId) + "</small></td>" +
			"<td>" + contactHTML(item) + "</td>" +
			"<td>" + qualificationHTML(item) + "</td>" +
			"<td><span class=\"status status-" + escapeHTML(item.employmentStatus) + "\">" + text(item.employmentStatus === "on_leave" ? "onLeave" : item.employmentStatus) + "</span><small>" + text("hiredOn") + " " + formatDate(item.hireDate) + "</small></td>" +
			"<td>" + actionHTML(item) + "</td></tr>";
	}

	function renderCard(item) {
		return "<article class=\"employee-card\"><header><div class=\"employee-cell\"><span class=\"avatar\">" + escapeHTML(item.name.slice(0, 1)) + "</span><div><strong>" + escapeHTML(item.name) + "</strong><small>" + escapeHTML(item.code) + "</small></div></div><span class=\"status status-" + escapeHTML(item.employmentStatus) + "\">" + text(item.employmentStatus === "on_leave" ? "onLeave" : item.employmentStatus) + "</span></header><dl><div><dt>" + text("assignment") + "</dt><dd>" + escapeHTML(item.departmentId) + " · " + escapeHTML(item.positionId) + "</dd></div><div><dt>" + text("contact") + "</dt><dd>" + contactHTML(item) + "</dd></div><div><dt>" + text("qualifications") + "</dt><dd>" + qualificationHTML(item) + "</dd></div></dl><footer>" + actionHTML(item) + "</footer></article>";
	}

	function actionHTML(item) {
		var id = escapeHTML(item.id);
		var actions = "<button class=\"text-button\" type=\"button\" data-action=\"edit\" data-id=\"" + id + "\">" + text("edit") + "</button>" +
			"<button class=\"text-button\" type=\"button\" data-action=\"attach\" data-id=\"" + id + "\">" + text("attachment") + "</button>";
		if (item.employmentStatus !== "left") actions += "<button class=\"text-button danger-text\" type=\"button\" data-action=\"leave\" data-id=\"" + id + "\">" + text("processLeave") + "</button>";
		return "<div class=\"row-actions\">" + actions + "</div>";
	}

	function contactHTML(item) {
		if (!item.phone && !item.email) return "<span class=\"muted\">" + text("noContact") + "</span>";
		return (item.phone ? "<span>" + escapeHTML(item.phone) + "</span>" : "") + (item.email ? "<small>" + escapeHTML(item.email) + "</small>" : "");
	}

	function qualificationHTML(item) {
		var certificates = Array.isArray(item.certificates) ? item.certificates : [];
		if (!certificates.length) return "<span class=\"muted\">" + text("noCertificate") + "</span>";
		var first = certificates[0];
		return "<span>" + escapeHTML(first.name) + "</span><small>" + formatDate(first.expiresAt) + (certificates.length > 1 ? " · " + certificates.length + " " + text("certificates") : "") + "</small>";
	}

	function openEmployee(item) {
		state.editing = item || null;
		employeeForm.reset();
		document.querySelector("#employee-form-error").textContent = "";
		document.querySelector("#employee-dialog-title").textContent = text(item ? "editTitle" : "createTitle");
		if (item) {
			["code", "name", "departmentId", "positionId", "phone", "email"].forEach(function (field) { employeeForm.elements[field].value = item[field] || ""; });
			employeeForm.elements.hireDate.value = dateInput(item.hireDate);
			var certificate = Array.isArray(item.certificates) && item.certificates.length ? item.certificates[0] : {};
			employeeForm.elements.certificateName.value = certificate.name || "";
			employeeForm.elements.certificateNumber.value = certificate.number || "";
			employeeForm.elements.certificateExpiry.value = dateInput(certificate.expiresAt);
		}
		employeeDialog.showModal();
	}

	function openParty(item) {
		state.editing=item||null; partyForm.reset(); document.querySelector("#party-form-error").textContent=""; var label=state.module==="customer"?"客户":"供应商"; document.querySelector("#party-dialog-title").textContent=(item?"编辑":"新建")+label;
		if(item){ var contact=item.contacts&&item.contacts[0]||{}, address=item.addresses&&item.addresses[0]||{}, settlement=item.settlementTerms||{}; var values={code:item.code,name:item.name,creditCode:item.unifiedSocialCreditCode,region:item.region,rating:item.rating,contactName:contact.name,contactTitle:contact.title,contactPhone:contact.phone,contactEmail:contact.email,addressLabel:address.label,province:address.province,city:address.city,district:address.district,addressDetail:address.detail,currency:settlement.currency,paymentDays:settlement.paymentDays,creditLimit:settlement.creditLimit}; Object.keys(values).forEach(function(key){partyForm.elements[key].value=values[key]??"";}); }
		partyDialog.showModal();
	}

	async function saveParty(event){ event.preventDefault(); if(!partyForm.reportValidity())return; var data=new FormData(partyForm), current=state.editing, body=Object.assign({},scope(),{code:String(data.get("code")||"").trim(),name:String(data.get("name")||"").trim(),unifiedSocialCreditCode:String(data.get("creditCode")||"").trim(),region:String(data.get("region")||"").trim(),rating:Number(data.get("rating")),contacts:[{id:current&&current.contacts[0]?current.contacts[0].id:"contact-1",name:String(data.get("contactName")||"").trim(),title:String(data.get("contactTitle")||"").trim(),phone:String(data.get("contactPhone")||"").trim(),email:String(data.get("contactEmail")||"").trim(),primary:true}],addresses:[{id:current&&current.addresses[0]?current.addresses[0].id:"address-1",label:String(data.get("addressLabel")||"").trim(),province:String(data.get("province")||"").trim(),city:String(data.get("city")||"").trim(),district:String(data.get("district")||"").trim(),detail:String(data.get("addressDetail")||"").trim(),default:true}],settlementTerms:{currency:String(data.get("currency")||"").trim(),paymentDays:Number(data.get("paymentDays")),creditLimit:Number(data.get("creditLimit"))}}); if(current)body.version=current.version; setButtonBusy("#party-save-button",true,"saving"); try{var path=partyCollections[state.module]+(current?"/"+encodeURIComponent(current.id):""); await request(path,{method:current?"PUT":"POST",body:body,headers:{"Idempotency-Key":idempotencyKey(state.module+(current?"-update":"-create"))}}); partyDialog.close(); showToast(state.module==="customer"?"客户资料已保存":"供应商资料已保存"); await load();}catch(error){document.querySelector("#party-form-error").textContent=error.message||text("unknownError");}finally{setButtonBusy("#party-save-button",false,"save");}}

	async function openCatalog(item) {
		state.editing=item||null;
		catalogForm.reset();
		document.querySelector("#catalog-form-error").textContent="";
		document.querySelector("#catalog-dialog-title").textContent=(item?"编辑":"新建")+moduleLabel();
		document.querySelectorAll("[data-catalog-field]").forEach(function(field){var visible=field.dataset.catalogField===state.module; field.hidden=!visible; field.querySelectorAll("input,select").forEach(function(control){control.disabled=!visible;});});
		catalogForm.elements.symbol.required=state.module==="unit";
		catalogForm.elements.unifiedSocialCreditCode.required=state.module==="manufacturer";
		catalogForm.elements.licenseNumber.required=state.module==="manufacturer";
		if(state.module==="category"){
			try{var page=await request("/categories?status=active&limit=200"); populateSelect(catalogForm.elements.parentId,page.items,"无上级分类",item&&item.parentId,item&&item.id);}catch(error){document.querySelector("#catalog-form-error").textContent=error.message||text("unknownError");}
		}
		if(item){["code","name","description","parentId","symbol","decimalPlaces","unifiedSocialCreditCode","licenseNumber"].forEach(function(field){if(catalogForm.elements[field])catalogForm.elements[field].value=item[field]??"";});}
		catalogDialog.showModal();
	}

	async function saveCatalog(event) {
		event.preventDefault();
		if(!catalogForm.reportValidity())return;
		var data=new FormData(catalogForm),current=state.editing;
		var body=Object.assign({},scope(),{code:String(data.get("code")||"").trim(),name:String(data.get("name")||"").trim(),description:String(data.get("description")||"").trim(),parentId:String(data.get("parentId")||"").trim(),symbol:String(data.get("symbol")||"").trim(),decimalPlaces:Number(data.get("decimalPlaces")||0),unifiedSocialCreditCode:String(data.get("unifiedSocialCreditCode")||"").trim(),licenseNumber:String(data.get("licenseNumber")||"").trim()});
		if(current)body.version=current.version;
		setButtonBusy("#catalog-save-button",true,"saving");
		try{var path=catalogCollections[state.module]+(current?"/"+encodeURIComponent(current.id):""); await request(path,{method:current?"PUT":"POST",body:body,headers:{"Idempotency-Key":idempotencyKey(state.module+(current?"-update":"-create"))}}); catalogDialog.close(); showToast(moduleLabel()+"资料已保存"); await load();}
		catch(error){document.querySelector("#catalog-form-error").textContent=error.message||text("unknownError");}
		finally{setButtonBusy("#catalog-save-button",false,"save");}
	}

	async function openProduct(item) {
		state.editing=item||null;
		productForm.reset();
		document.querySelector("#product-form-error").textContent="";
		document.querySelector("#product-dialog-title").textContent=item?"编辑药品":"新建药品";
		try{
			var pages=await Promise.all([request("/categories?status=active&limit=200"),request("/units?status=active&limit=200"),request("/manufacturers?status=active&limit=200")]);
			populateSelect(productForm.elements.categoryId,pages[0].items,"请选择药品分类",item&&item.categoryId);
			populateSelect(productForm.elements.unitId,pages[1].items,"请选择计量单位",item&&item.unitId);
			populateSelect(productForm.elements.manufacturerId,pages[2].items,"请选择生产企业",item&&item.manufacturerId);
			if(pages.some(function(page){return !Array.isArray(page.items)||page.items.length===0;}))document.querySelector("#product-form-error").textContent="请先建立启用的分类、单位和生产企业。";
		}catch(error){document.querySelector("#product-form-error").textContent=error.message||text("unknownError");}
		if(item){["code","sku","name","genericName","categoryId","unitId","manufacturerId","dosageForm","specification","approvalNumber","barcode","storageCondition","temperatureMin","temperatureMax"].forEach(function(field){productForm.elements[field].value=item[field]??"";});}
		productDialog.showModal();
	}

	async function saveProduct(event) {
		event.preventDefault();
		if(!productForm.reportValidity())return;
		var data=new FormData(productForm),current=state.editing;
		var body=Object.assign({},scope(),{code:String(data.get("code")||"").trim(),sku:String(data.get("sku")||"").trim(),name:String(data.get("name")||"").trim(),genericName:String(data.get("genericName")||"").trim(),categoryId:String(data.get("categoryId")||"").trim(),unitId:String(data.get("unitId")||"").trim(),manufacturerId:String(data.get("manufacturerId")||"").trim(),dosageForm:String(data.get("dosageForm")||"").trim(),specification:String(data.get("specification")||"").trim(),approvalNumber:String(data.get("approvalNumber")||"").trim(),barcode:String(data.get("barcode")||"").trim(),storageCondition:String(data.get("storageCondition")||"").trim(),temperatureMin:Number(data.get("temperatureMin")),temperatureMax:Number(data.get("temperatureMax"))});
		if(current)body.version=current.version;
		setButtonBusy("#product-save-button",true,"saving");
		try{var path="/products"+(current?"/"+encodeURIComponent(current.id):""); await request(path,{method:current?"PUT":"POST",body:body,headers:{"Idempotency-Key":idempotencyKey("product-"+(current?"update":"create"))}}); productDialog.close(); showToast("药品资料已保存"); await load();}
		catch(error){document.querySelector("#product-form-error").textContent=error.message||text("unknownError");}
		finally{setButtonBusy("#product-save-button",false,"save");}
	}

	function populateSelect(select,items,placeholder,selected,excludeID){select.innerHTML='<option value="">'+escapeHTML(placeholder)+'</option>'+(Array.isArray(items)?items:[]).filter(function(item){return item.id!==excludeID;}).map(function(item){return '<option value="'+escapeHTML(item.id)+'"'+(item.id===selected?' selected':'')+'>'+escapeHTML(item.name)+'（'+escapeHTML(item.code)+'）</option>';}).join("");}
	function moduleLabel(){return {customer:"客户",supplier:"供应商",product:"药品",category:"分类",unit:"单位",manufacturer:"生产企业"}[state.module]||"记录";}
	function openPartyStatus(item){state.target=item; state.statusAction=item.status==="active"?"disable":"enable"; partyStatusForm.reset(); var disabling=state.statusAction==="disable"; document.querySelector("#party-status-title").textContent=(disabling?"停用":"启用")+moduleLabel()+" · "+item.name; document.querySelector("#party-status-reason-field").hidden=!disabling; partyStatusForm.elements.reason.required=disabling; document.querySelector("#party-status-save-button").textContent=disabling?"确认停用":"确认启用"; document.querySelector("#party-status-save-button").className="button "+(disabling?"danger":"primary"); document.querySelector("#party-status-error").textContent=""; partyStatusDialog.showModal();}
	async function savePartyStatus(event){event.preventDefault(); if(!partyStatusForm.reportValidity())return; setButtonBusy("#party-status-save-button",true,"processing"); try{await request(collectionForModule()+"/"+encodeURIComponent(state.target.id)+"/"+state.statusAction,{method:"POST",body:{reason:partyStatusForm.elements.reason.value.trim(),version:state.target.version},headers:{"Idempotency-Key":idempotencyKey(state.module+"-"+state.statusAction)}}); partyStatusDialog.close(); showToast(state.statusAction==="disable"?"已停用":"已启用"); await load();}catch(error){document.querySelector("#party-status-error").textContent=error.message||text("unknownError");}finally{document.querySelector("#party-status-save-button").disabled=false;}}

	async function saveEmployee(event) {
		event.preventDefault();
		if (!employeeForm.reportValidity()) return;
		var values = new FormData(employeeForm);
		var certificateName = String(values.get("certificateName") || "").trim();
		var certificateNumber = String(values.get("certificateNumber") || "").trim();
		var certificateExpiry = String(values.get("certificateExpiry") || "").trim();
		var body = Object.assign({}, scope(), {
			code: String(values.get("code") || "").trim(), name: String(values.get("name") || "").trim(), departmentId: String(values.get("departmentId") || "").trim(), positionId: String(values.get("positionId") || "").trim(), phone: String(values.get("phone") || "").trim(), email: String(values.get("email") || "").trim(), hireDate: String(values.get("hireDate") || "").trim(), certificates: []
		});
		if (certificateName || certificateNumber || certificateExpiry) {
			var existingCertificate = state.editing && Array.isArray(state.editing.certificates) ? state.editing.certificates[0] : null;
			body.certificates.push({ id: existingCertificate ? existingCertificate.id : idempotencyKey("cert"), name: certificateName, number: certificateNumber, expiresAt: certificateExpiry });
		}
		if (state.editing) body.version = state.editing.version;
		setButtonBusy("#employee-save-button", true, "saving");
		try {
			var path = state.editing ? "/employees/" + encodeURIComponent(state.editing.id) : "/employees";
			await request(path, { method: state.editing ? "PUT" : "POST", body: body, headers: { "Idempotency-Key": idempotencyKey(state.editing ? "employee-update" : "employee-create") } });
			employeeDialog.close();
			showToast(text("saved"));
			await load();
		} catch (error) {
			document.querySelector("#employee-form-error").textContent = error.message || text("unknownError");
		} finally { setButtonBusy("#employee-save-button", false, "save"); }
	}

	function openAttachment(item) {
		state.target = item;
		attachmentForm.reset();
		document.querySelector("#attachment-form-error").textContent = "";
		document.querySelector("#attachment-title").textContent = text("uploadTitle") + " · " + item.name;
		attachmentDialog.showModal();
	}

	async function uploadAttachment(event) {
		event.preventDefault();
		var file = attachmentForm.elements.file.files[0];
		if (!file) return;
		if (file.size > 1024 * 1024) { document.querySelector("#attachment-form-error").textContent = text("fileTooLarge"); return; }
		setButtonBusy("#attachment-save-button", true, "uploading");
		try {
			var content = await fileToBase64(file);
			await request("/employees/" + encodeURIComponent(state.target.id) + "/attachments", { method: "POST", body: { name: file.name, contentBase64: content, version: state.target.version }, headers: { "Idempotency-Key": idempotencyKey("employee-attach") } });
			attachmentDialog.close();
			showToast(text("uploaded"));
			await load();
		} catch (error) { document.querySelector("#attachment-form-error").textContent = error.message || text("unknownError"); }
		finally { setButtonBusy("#attachment-save-button", false, "upload"); }
	}

	function openLeave(item) {
		state.target = item;
		leaveForm.reset();
		document.querySelector("#leave-form-error").textContent = "";
		document.querySelector("#leave-title").textContent = text("leaveTitle") + " · " + item.name;
		leaveDialog.showModal();
	}

	async function saveLeave(event) {
		event.preventDefault();
		if (!leaveForm.reportValidity()) return;
		setButtonBusy("#leave-save-button", true, "processing");
		try {
			await request("/employees/" + encodeURIComponent(state.target.id) + "/leave", { method: "POST", body: { reason: leaveForm.elements.reason.value.trim(), version: state.target.version }, headers: { "Idempotency-Key": idempotencyKey("employee-leave") } });
			leaveDialog.close();
			showToast(text("leftDone"));
			await load();
		} catch (error) { document.querySelector("#leave-form-error").textContent = error.message || text("unknownError"); }
		finally { setButtonBusy("#leave-save-button", false, "confirmLeave"); }
	}

	function fileToBase64(file) {
		return new Promise(function (resolve, reject) {
			var reader = new FileReader();
			reader.onload = function () { resolve(String(reader.result).split(",")[1] || ""); };
			reader.onerror = function () { reject(reader.error); };
			reader.readAsDataURL(file);
		});
	}

	function setButtonBusy(selector, busy, labelKey) {
		var button = document.querySelector(selector);
		button.disabled = busy;
		button.textContent = text(labelKey);
	}

	function showToast(message) {
		var toast = document.querySelector("#toast");
		toast.textContent = message;
		toast.classList.add("visible");
		window.clearTimeout(showToast.timer);
		showToast.timer = window.setTimeout(function () { toast.classList.remove("visible"); }, 2400);
	}

	function formatDate(value) {
		if (!value) return "-";
		var date = new Date(value);
		return Number.isNaN(date.getTime()) ? "-" : new Intl.DateTimeFormat(state.locale).format(date);
	}

	function dateInput(value) {
		if (!value) return "";
		var date = new Date(value);
		return Number.isNaN(date.getTime()) ? "" : date.toISOString().slice(0, 10);
	}

	function escapeHTML(value) {
		return String(value == null ? "" : value).replace(/[&<>"']/g, function (character) { return { "&": "&amp;", "<": "&lt;", ">": "&gt;", "\"": "&quot;", "'": "&#39;" }[character]; });
	}

	function findEmployee(id) {
		return state.items.find(function (item) { return item.id === id; });
	}

	function openCurrent(item) {
		if (state.module === "employee") return openEmployee(item);
		if (partyCollections[state.module]) return openParty(item);
		if (state.module === "product") return openProduct(item);
		return openCatalog(item);
	}

	function debounce(fn, delay) {
		var timer;
		return function () { window.clearTimeout(timer); timer = window.setTimeout(fn, delay); };
	}

	document.querySelector("#create-button").addEventListener("click", function () { openCurrent(null); });
	document.querySelector("#empty-create-button").addEventListener("click", function () { openCurrent(null); });
	document.querySelector("#refresh-button").addEventListener("click", load);
	document.querySelector("#retry-button").addEventListener("click", load);
	document.querySelector("#scope-button").addEventListener("click", function () { saveScope(); load(); });
	document.querySelector("#keyword-input").addEventListener("input", debounce(load, 280));
	document.querySelector("#status-filter").addEventListener("change", load);
	document.querySelectorAll("[data-module]").forEach(function(button){button.addEventListener("click",function(){if(state.module===button.dataset.module)return; state.module=button.dataset.module; state.items=[]; state.reminders=[]; state.lookups={}; document.querySelector("#keyword-input").value=""; applyModule(); load();});});
	document.addEventListener("click", function (event) {
		var close = event.target.closest("[data-close]");
		if (close) document.querySelector("#" + close.dataset.close).close();
		var button = event.target.closest("[data-action]");
		if (!button) return;
		var item = findEmployee(button.dataset.id);
		if (!item) return;
		if (button.dataset.action === "edit") openEmployee(item);
		if (button.dataset.action === "attach") openAttachment(item);
		if (button.dataset.action === "leave") openLeave(item);
		if (button.dataset.action === "party-edit") openParty(item);
		if (button.dataset.action === "party-status") openPartyStatus(item);
		if (button.dataset.action === "catalog-edit") openCatalog(item);
		if (button.dataset.action === "catalog-status") openPartyStatus(item);
		if (button.dataset.action === "product-edit") openProduct(item);
		if (button.dataset.action === "product-status") openPartyStatus(item);
	});
	employeeForm.addEventListener("submit", saveEmployee);
	attachmentForm.addEventListener("submit", uploadAttachment);
	leaveForm.addEventListener("submit", saveLeave);
	partyForm.addEventListener("submit", saveParty);
	partyStatusForm.addEventListener("submit", savePartyStatus);
	catalogForm.addEventListener("submit", saveCatalog);
	productForm.addEventListener("submit", saveProduct);
	window.addEventListener("skoll:locale", function (event) { applyLocale(event.detail && event.detail.locale); });
	window.addEventListener("skoll:host-ready", load);

	loadScope();
	applyLocale(window.__SKOLL_HOST__ && window.__SKOLL_HOST__.locale);
	load();
})();
