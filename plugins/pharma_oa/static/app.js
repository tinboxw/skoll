(function () {
	"use strict";

	var apiBase = "/v1/plugins/pharma_oa/api";
	var state = { items: [], reminders: [], editing: null, target: null, loading: false, locale: "zh-CN" };
	var app = document.querySelector("#app");
	var employeeDialog = document.querySelector("#employee-dialog");
	var attachmentDialog = document.querySelector("#attachment-dialog");
	var leaveDialog = document.querySelector("#leave-dialog");
	var employeeForm = document.querySelector("#employee-form");
	var attachmentForm = document.querySelector("#attachment-form");
	var leaveForm = document.querySelector("#leave-form");
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
		render();
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
			var results = await Promise.all([
				request("/employees" + query({ keyword: keyword, status: status, limit: 200 })),
				request("/employees/qualification-reminders?days=30")
			]);
			state.items = Array.isArray(results[0].items) ? results[0].items : [];
			state.reminders = Array.isArray(results[1].items) ? results[1].items : [];
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

	function debounce(fn, delay) {
		var timer;
		return function () { window.clearTimeout(timer); timer = window.setTimeout(fn, delay); };
	}

	document.querySelector("#create-button").addEventListener("click", function () { openEmployee(null); });
	document.querySelector("#empty-create-button").addEventListener("click", function () { openEmployee(null); });
	document.querySelector("#refresh-button").addEventListener("click", load);
	document.querySelector("#retry-button").addEventListener("click", load);
	document.querySelector("#scope-button").addEventListener("click", function () { saveScope(); load(); });
	document.querySelector("#keyword-input").addEventListener("input", debounce(load, 280));
	document.querySelector("#status-filter").addEventListener("change", load);
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
	});
	employeeForm.addEventListener("submit", saveEmployee);
	attachmentForm.addEventListener("submit", uploadAttachment);
	leaveForm.addEventListener("submit", saveLeave);
	window.addEventListener("skoll:locale", function (event) { applyLocale(event.detail && event.detail.locale); });
	window.addEventListener("skoll:host-ready", load);

	loadScope();
	applyLocale(window.__SKOLL_HOST__ && window.__SKOLL_HOST__.locale);
	load();
})();
