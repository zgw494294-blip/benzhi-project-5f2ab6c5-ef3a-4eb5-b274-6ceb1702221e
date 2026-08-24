const state = { dashboard: null, batch: null, selectedTree: null, filter: "all", action: null };

const statusLabels = {
  open: "登记中", in_progress: "巡检进行中", completed: "批次已完成",
  registered: "待采集证据", evidence_submitted: "待风险评估", awaiting_remediation: "待修复",
  awaiting_recheck: "待复验", closed: "已关闭",
  assigned: "已派发", in_progress: "处理中", awaiting_recheck: "待复验", rework_required: "需返工", closed: "已关闭"
};
const riskLabels = { low: "低风险", medium: "中风险", high: "高风险", critical: "紧急风险" };
const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => [...document.querySelectorAll(selector)];
const uid = () => `${Date.now()}-${crypto.randomUUID()}`;
const nowLocal = () => { const d = new Date(); d.setMinutes(d.getMinutes() - d.getTimezoneOffset()); return d.toISOString().slice(0, 16); };
const dayLater = (days) => { const d = new Date(Date.now() + days * 86400000); d.setMinutes(d.getMinutes() - d.getTimezoneOffset()); return d.toISOString().slice(0, 16); };

async function api(path, options = {}) {
  const response = await fetch(path, { ...options, headers: { "Content-Type": "application/json", ...(options.headers || {}) } });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(body.error?.message || `请求失败 (${response.status})`);
  return body.data;
}

function toast(message, error = false) {
  const node = document.createElement("div"); node.className = `toast${error ? " error" : ""}`; node.textContent = message;
  $("#toast-region").append(node); setTimeout(() => node.remove(), 3600);
}

async function loadDashboard(selectFirst = false) {
  state.dashboard = await api("/api/batches"); renderDashboard();
  if (selectFirst && state.dashboard.batches.length) await selectBatch(state.dashboard.batches[0].batch.id);
}

function renderDashboard() {
  const d = state.dashboard;
  $("#metric-batches").textContent = d.batches.length; $("#metric-trees").textContent = d.totalTrees;
  $("#metric-tasks").textContent = d.openTasks; $("#metric-certs").textContent = d.certificates;
  $("#today-label").textContent = new Intl.DateTimeFormat("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit", weekday: "short" }).format(new Date());
  $("#batch-list").innerHTML = d.batches.length ? d.batches.map(({ batch, treeCount, completion, riskCount }) => `
    <button class="batch-card ${state.batch?.batch.id === batch.id ? "active" : ""}" data-batch="${batch.id}">
      <strong>${escapeHTML(batch.name)}</strong><div class="batch-meta"><span>${escapeHTML(batch.area)}</span><span>${treeCount} 棵 · ${riskCount} 风险</span></div>
      <div class="mini-progress"><i style="width:${completion}%"></i></div>
    </button>`).join("") : `<p class="detail-placeholder">暂无批次。新建批次后开始登记树木。</p>`;
  $$('[data-batch]').forEach((button) => button.addEventListener("click", () => selectBatch(button.dataset.batch)));
}

async function selectBatch(id) {
  state.batch = await api(`/api/batches/${id}`); state.selectedTree = null; renderDashboard(); renderBatch(); closeDetail();
}

function renderBatch() {
  const b = state.batch; if (!b) return;
  $("#batch-title").textContent = b.batch.name; $("#batch-subtitle").textContent = `${b.batch.area} · ${statusLabels[b.batch.status] || b.batch.status}`;
  $("#add-tree-button").disabled = b.batch.status === "completed"; $("#tree-count").textContent = `${b.total} 棵树`;
  $("#progress-fill").style.width = `${b.total ? b.closed * 100 / b.total : 0}%`;
  let trees = b.trees;
  if (state.filter === "risk") trees = trees.filter((item) => item.task && item.task.status !== "closed");
  if (state.filter === "closed") trees = trees.filter((item) => item.tree.currentStatus === "closed");
  $("#tree-list").innerHTML = trees.length ? trees.map(treeRow).join("") : `<div class="empty-state"><h3>当前筛选无树木</h3><p>切换筛选条件或登记新的树木档案。</p></div>`;
  $$('[data-tree]').forEach((row) => {
    row.addEventListener("click", () => showTree(row.dataset.tree));
    row.addEventListener("keydown", (event) => { if (event.key === "Enter" || event.key === " ") { event.preventDefault(); showTree(row.dataset.tree); } });
  });
}

function treeRow(item) {
  const t = item.tree, risky = item.task && item.task.status !== "closed";
  return `<article class="tree-row ${state.selectedTree?.tree.id === t.id ? "active" : ""}" data-tree="${t.id}" tabindex="0" role="button">
    <div class="tree-identity"><strong>${escapeHTML(t.name)}</strong><span>${escapeHTML(t.species)} · 胸径 ${t.diameterCM} cm</span></div>
    <div class="tree-cell"><b>${escapeHTML(t.roadLocation)}</b><small>${escapeHTML(t.responsibilityArea)}</small></div>
    <div class="tree-cell">${item.task ? `<span class="status ${risky ? "status-risk" : "status-closed"}">${riskLabels[item.task.riskLevel]}</span>` : `<span class="status">未评估</span>`}</div>
    <div class="tree-cell"><b>${statusLabels[t.currentStatus] || t.currentStatus}</b><small>v${t.version}</small></div>
  </article>`;
}

function showTree(id) {
  state.selectedTree = state.batch.trees.find((item) => item.tree.id === id); renderBatch(); renderDetail(); $("#detail-panel").classList.add("open");
}

function renderDetail() {
  const item = state.selectedTree, t = item.tree;
  let html = `<section class="detail-block"><h3>${escapeHTML(t.name)} · 档案</h3><div class="detail-grid">
    <div><small>树种</small><b>${escapeHTML(t.species)}</b></div><div><small>胸径</small><b>${t.diameterCM} cm</b></div>
    <div><small>道路位置</small><b>${escapeHTML(t.roadLocation)}</b></div><div><small>责任区域</small><b>${escapeHTML(t.responsibilityArea)}</b></div>
    <div><small>流程状态</small><b>${statusLabels[t.currentStatus]}</b></div><div><small>实体版本</small><b>v${t.version}</b></div></div></section>`;
  if (item.evidence) html += `<section class="detail-block"><h3>巡检证据</h3><div class="detail-grid"><div><small>叶片 / 树干 / 病虫</small><b>${item.evidence.leafCondition} / ${item.evidence.trunkDefect} / ${item.evidence.pestSigns}</b></div><div><small>提交人</small><b>${escapeHTML(item.evidence.submittedBy)}</b></div></div><p>${escapeHTML(item.evidence.notes)}</p><div class="digest">EVIDENCE ${item.evidence.digest}</div></section>`;
  if (item.task) html += `<section class="detail-block"><h3>修复任务</h3><div class="detail-grid"><div><small>风险等级</small><b>${riskLabels[item.task.riskLevel]}</b></div><div><small>负责人</small><b>${escapeHTML(item.task.assignee)}</b></div><div><small>任务状态</small><b>${statusLabels[item.task.status]}</b></div><div><small>任务版本</small><b>v${item.task.version}</b></div></div><p>${escapeHTML(item.task.recommendedAction)}</p></section>`;
  if (item.certificate) html += `<section class="detail-block"><h3>养护验收凭据</h3><div class="detail-grid"><div><small>复验人</small><b>${escapeHTML(item.certificate.inspector)}</b></div><div><small>结论</small><b>${escapeHTML(item.certificate.result)}</b></div></div><div class="digest">CERTIFICATE ${item.certificate.digest}</div></section>`;
  const action = nextAction(item); if (action) html += `<button class="primary next-action" type="button" data-tree-action="${action.name}">${action.label}</button>`;
  $("#detail-content").innerHTML = html;
  const button = $('[data-tree-action]'); if (button) button.addEventListener("click", () => openAction(button.dataset.treeAction));
}

function nextAction(item) {
  const status = item.tree.currentStatus;
  if (status === "registered") return { name: "evidence", label: "提交巡检证据" };
  if (status === "evidence_submitted") return { name: "assess", label: "评估风险并派发" };
  if (status === "awaiting_remediation") return { name: "remediation", label: item.task?.status === "rework_required" ? "登记返工结果" : "确认修复完成" };
  if (status === "awaiting_recheck") return { name: "recheck", label: "提交复验结论" };
  return null;
}

const field = (name, label, type = "text", value = "", extra = "", full = false) => `<div class="field ${full ? "full" : ""}"><label for="f-${name}">${label}<span> *</span></label><input id="f-${name}" name="${name}" type="${type}" value="${value}" ${extra} required></div>`;
const select = (name, label) => `<div class="field"><label for="f-${name}">${label}<span> *</span></label><select id="f-${name}" name="${name}" required><option value="0">0 · 健康</option><option value="1">1 · 轻微</option><option value="2">2 · 中度</option><option value="3">3 · 严重</option></select></div>`;

function openAction(action) {
  state.action = action; const dialog = $("#action-dialog"), fields = $("#form-fields");
  const configs = {
    "create-batch": ["新建巡检批次", field("name", "批次名称") + field("area", "责任区域") + field("treeName", "首棵树木名称") + field("species", "树种") + field("roadLocation", "道路位置", "text", "", "", true) + field("diameterCM", "胸径（厘米）", "number", "", 'min="0.1" max="500" step="0.1"') + field("responsibilityArea", "树木责任区") + field("operator", "登记人", "text", "", "", true)],
    "add-tree": ["登记树木档案", field("name", "树木名称") + field("species", "树种") + field("roadLocation", "道路位置", "text", "", "", true) + field("diameterCM", "胸径（厘米）", "number", "", 'min="0.1" max="500" step="0.1"') + field("responsibilityArea", "责任区域") + field("operator", "登记人", "text", "", "", true)],
    evidence: ["提交巡检证据", field("inspectedAt", "巡检时间", "datetime-local", nowLocal()) + field("photoDigest", "照片摘要") + select("leafCondition", "叶片状况") + select("trunkDefect", "树干缺陷") + select("pestSigns", "病虫害迹象") + field("submittedBy", "巡检员") + `<div class="field full"><label for="f-notes">观察备注<span> *</span></label><textarea id="f-notes" name="notes" required></textarea></div>`],
    assess: ["评估风险并派发", field("assignee", "修复负责人") + field("dueDate", "截止时间", "datetime-local", dayLater(3)) + field("operator", "评估负责人", "text", "", "", true)],
    remediation: ["确认修复完成", field("completedAt", "完成时间", "datetime-local", nowLocal()) + field("operator", "修复人员") + `<div class="field full"><label for="f-completionNote">修复说明<span> *</span></label><textarea id="f-completionNote" name="completionNote" required></textarea></div>`],
    recheck: ["提交复验结论", field("recheckedAt", "复验时间", "datetime-local", nowLocal()) + field("inspector", "复验人") + field("crown", "树冠稳定") + field("trunk", "树干安全") + field("pest", "病虫受控") + `<div class="field full"><label for="f-result">复验结论<span> *</span></label><select id="f-result" name="result"><option>通过</option><option>不通过</option></select></div>`]
  };
  $("#dialog-title").textContent = configs[action][0]; fields.innerHTML = configs[action][1]; dialog.showModal();
}

async function submitAction(event) {
  event.preventDefault(); const data = Object.fromEntries(new FormData(event.currentTarget)); const item = state.selectedTree;
  let path, body; const headers = { "Idempotency-Key": uid() };
  if (state.action === "create-batch") { path = "/api/batches"; body = { name: data.name, area: data.area, operator: data.operator, tree: { name: data.treeName, species: data.species, roadLocation: data.roadLocation, diameterCM: Number(data.diameterCM), responsibilityArea: data.responsibilityArea } }; }
  if (state.action === "add-tree") { path = `/api/batches/${state.batch.batch.id}/trees`; body = { ...data, diameterCM: Number(data.diameterCM) }; }
  if (state.action === "evidence") { path = `/api/trees/${item.tree.id}/evidence`; body = { expectedVersion: item.tree.version, inspectedAt: new Date(data.inspectedAt).toISOString(), photoDigest: data.photoDigest, leafCondition: Number(data.leafCondition), trunkDefect: Number(data.trunkDefect), pestSigns: Number(data.pestSigns), notes: data.notes, submittedBy: data.submittedBy }; }
  if (state.action === "assess") { path = `/api/trees/${item.tree.id}/assess`; body = { expectedVersion: item.tree.version, assignee: data.assignee, dueDate: new Date(data.dueDate).toISOString(), operator: data.operator }; }
  if (state.action === "remediation") { path = `/api/trees/${item.tree.id}/remediation`; body = { expectedTreeVersion: item.tree.version, expectedTaskVersion: item.task.version, completionNote: data.completionNote, completedAt: new Date(data.completedAt).toISOString(), operator: data.operator }; }
  if (state.action === "recheck") { path = `/api/trees/${item.tree.id}/recheck`; body = { expectedTreeVersion: item.tree.version, expectedTaskVersion: item.task.version, recheckedAt: new Date(data.recheckedAt).toISOString(), inspector: data.inspector, result: data.result, metrics: { "树冠稳定": data.crown, "树干安全": data.trunk, "病虫受控": data.pest } }; }
  const submit = $("#submit-action"); submit.disabled = true;
  try { const result = await api(path, { method: "POST", headers, body: JSON.stringify(body) }); $("#action-dialog").close(); toast("操作已保存并写入审计记录"); await loadDashboard(); const batchID = state.action === "create-batch" ? result.batch.id : state.batch?.batch.id; if (batchID) { await selectBatch(batchID); const refreshed = state.batch.trees.find((x) => x.tree.id === item?.tree.id); if (refreshed) { state.selectedTree = refreshed; renderBatch(); renderDetail(); } } }
  catch (error) { toast(error.message, true); }
  finally { submit.disabled = false; }
}

function closeDetail() { $("#detail-panel").classList.remove("open"); }
function escapeHTML(value = "") { const div = document.createElement("div"); div.textContent = String(value); return div.innerHTML; }

$$('[data-open]').forEach((button) => button.addEventListener("click", () => openAction(button.dataset.open)));
$$('[data-filter]').forEach((button) => button.addEventListener("click", () => { $$('[data-filter]').forEach((x) => x.classList.remove("active")); button.classList.add("active"); state.filter = button.dataset.filter; renderBatch(); }));
$("#close-detail").addEventListener("click", closeDetail); $("[data-cancel]").addEventListener("click", () => $("#action-dialog").close());
$("#action-form").addEventListener("submit", submitAction);
loadDashboard(true).catch((error) => toast(error.message, true));
