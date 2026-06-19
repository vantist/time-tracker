package report

import (
	"database/sql"
	"net/http"
	"time"
)

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>tt dashboard</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #0f1117; color: #e2e8f0; padding: 24px; }
h1 { font-size: 1.5rem; margin-bottom: 24px; color: #f8fafc; }
h2 { font-size: 1rem; margin-bottom: 12px; color: #94a3b8; text-transform: uppercase; letter-spacing: .05em; }
.cards { display: flex; flex-wrap: wrap; gap: 12px; margin-bottom: 32px; }
.card { background: #1e2433; border-radius: 8px; padding: 16px 20px; min-width: 160px; }
.card-label { font-size: .75rem; color: #64748b; margin-bottom: 4px; }
.card-value { font-size: 1.4rem; font-weight: 600; color: #f1f5f9; }
.section { margin-bottom: 32px; }
.chart { display: flex; align-items: flex-end; gap: 6px; height: 80px; margin-bottom: 8px; }
.bar-wrap { display: flex; flex-direction: column; align-items: center; flex: 1; height: 100%; justify-content: flex-end; }
.bar { width: 100%; background: #3b82f6; border-radius: 3px 3px 0 0; min-height: 1px; transition: height .3s; }
.bar-label { font-size: .65rem; color: #64748b; margin-top: 4px; white-space: nowrap; }
table { width: 100%; border-collapse: collapse; font-size: .875rem; }
th { text-align: left; padding: 8px 12px; color: #64748b; border-bottom: 1px solid #2d3748; font-weight: 500; }
td { padding: 8px 12px; border-bottom: 1px solid #1a2030; }
tr:hover td { background: #1a2234; }
.status { font-size: .75rem; color: #64748b; margin-top: 8px; }
.ratio-bar-item { height: 100%; display: flex; align-items: center; justify-content: center; font-size: .7rem; color: #fff; font-weight: 500; text-overflow: ellipsis; overflow: hidden; white-space: nowrap; transition: width .3s; padding: 0 4px; }
</style>
</head>
<body>
<h1>tt dashboard</h1>

<div class="cards">
  <div class="card"><div class="card-label">Sessions</div><div class="card-value" id="v-sessions">—</div></div>
  <div class="card"><div class="card-label">Agent time</div><div class="card-value" id="v-agent">—</div></div>
  <div class="card"><div class="card-label">User active</div><div class="card-value" id="v-user">—</div></div>
  <div class="card"><div class="card-label">Input tokens</div><div class="card-value" id="v-input">—</div></div>
  <div class="card"><div class="card-label">Output tokens</div><div class="card-value" id="v-output">—</div></div>
  <div class="card"><div class="card-label">Cache read</div><div class="card-value" id="v-cache-read">—</div></div>
  <div class="card"><div class="card-label">Cache create</div><div class="card-value" id="v-cache-create">—</div></div>
  <div class="card"><div class="card-label">Est. cost</div><div class="card-value" id="v-cost">—</div></div>
</div>

<div class="section">
  <h2>Daily (7 days)</h2>
  <div class="chart" id="chart"></div>
</div>

<div class="section" id="section-model-usages">
  <h2>By Model & Role</h2>
  <div id="ratio-bar" style="height: 24px; display: flex; border-radius: 4px; overflow: hidden; margin-bottom: 16px; background: #1a2030;"></div>
  <table>
    <thead><tr><th>Model</th><th>Role</th><th>Input Tokens</th><th>Output Tokens</th><th>Cost</th></tr></thead>
    <tbody id="tbl-model-usages"></tbody>
  </table>
</div>

<div class="section">
  <h2>By Project</h2>
  <table>
    <thead><tr><th>Project</th><th>Sessions</th><th>Agent time</th><th>User time</th><th>Tokens</th><th>Cost</th></tr></thead>
    <tbody id="tbl-project"></tbody>
  </table>
</div>

<div class="section">
  <h2>By Agent</h2>
  <table>
    <thead><tr><th>Agent</th><th>Sessions</th><th>Agent time</th><th>User time</th><th>Tokens</th><th>Cost</th></tr></thead>
    <tbody id="tbl-agent"></tbody>
  </table>
</div>

<div class="section" id="section-workitem">
  <h2>By Work Item</h2>
  <table>
    <thead><tr><th>Label</th><th>Project</th><th>Sessions</th><th>Agent time</th><th>User time</th><th>Cost</th></tr></thead>
    <tbody id="tbl-workitem"></tbody>
  </table>
</div>

<div class="section">
  <h2>Sessions</h2>
  <table>
    <thead><tr><th>Time</th><th>Project</th><th>Branch</th><th>Agent</th><th>Model</th><th>Turns</th><th>Agent time</th><th>User time</th><th>Work item</th><th>Cost</th></tr></thead>
    <tbody id="tbl-sessions"></tbody>
  </table>
</div>

<div class="status" id="status"></div>

<script>
const fmt = n => (n || 0).toLocaleString();
const fmtTime = sec => {
  const h = Math.floor(sec / 3600), m = Math.floor((sec % 3600) / 60);
  return h > 0 ? h + 'h ' + m + 'm' : m + 'm';
};
const fmtCost = c => c == null ? 'N/A' : '$' + c.toFixed(4);
const esc = s => String(s || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

function render(d) {
  document.getElementById('v-sessions').textContent = d.sessions_count || 0;
  document.getElementById('v-agent').textContent = fmtTime(d.agent_time_sec || 0);
  document.getElementById('v-user').textContent = fmtTime(d.user_active_time_sec || 0);
  document.getElementById('v-input').textContent = fmt(d.input_tokens);
  document.getElementById('v-output').textContent = fmt(d.output_tokens);
  document.getElementById('v-cache-read').textContent = fmt(d.cache_read_tokens);
  document.getElementById('v-cache-create').textContent = fmt(d.cache_creation_tokens);
  document.getElementById('v-cost').textContent = fmtCost(d.estimated_cost_usd);

  const daily = d.daily || [];
  const maxSess = Math.max(1, ...daily.map(x => x.sessions));
  const chart = document.getElementById('chart');
  chart.innerHTML = '';
  daily.forEach(day => {
    const pct = Math.round((day.sessions / maxSess) * 100);
    const wrap = document.createElement('div'); wrap.className = 'bar-wrap';
    const bar = document.createElement('div'); bar.className = 'bar';
    bar.style.height = pct + '%';
    const lbl = document.createElement('div'); lbl.className = 'bar-label';
    lbl.textContent = day.date.slice(5); // MM-DD
    wrap.appendChild(bar); wrap.appendChild(lbl);
    chart.appendChild(wrap);
  });

  const colors = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#06b6d4'];
  const usages = d.model_usages || [];
  const totalCost = usages.reduce((acc, x) => acc + (x.estimated_cost_usd || 0), 0);
  const totalTokens = usages.reduce((acc, x) => acc + (x.input_tokens + x.output_tokens), 0);
  const mBar = document.getElementById('ratio-bar');
  mBar.innerHTML = '';
  const mBody = document.getElementById('tbl-model-usages');
  mBody.innerHTML = '';
  if (usages.length === 0) {
    document.getElementById('section-model-usages').style.display = 'none';
  } else {
    document.getElementById('section-model-usages').style.display = '';
    usages.forEach((mu, idx) => {
      let pct = 0;
      if (totalCost > 0) {
        pct = Math.round(((mu.estimated_cost_usd || 0) / totalCost) * 100);
      } else if (totalTokens > 0) {
        pct = Math.round(((mu.input_tokens + mu.output_tokens) / totalTokens) * 100);
      }
      if (pct > 0) {
        const item = document.createElement('div');
        item.className = 'ratio-bar-item';
        item.style.width = pct + '%';
        item.style.backgroundColor = colors[idx % colors.length];
        item.textContent = mu.model + ' (' + pct + '%)';
        item.title = mu.model + ' (' + (mu.is_subagent ? 'Subagent' : 'Main') + '): ' + pct + '%';
        mBar.appendChild(item);
      }
      const tr = document.createElement('tr');
      const role = mu.is_subagent ? 'Subagent' : 'Main';
      tr.innerHTML = '<td><span style="display:inline-block; width:8px; height:8px; border-radius:50%; background-color:' + colors[idx % colors.length] + '; margin-right:8px;"></span>' + esc(mu.model) + '</td><td>' + role + '</td><td>' + fmt(mu.input_tokens) + '</td><td>' + fmt(mu.output_tokens) + '</td><td>' + fmtCost(mu.estimated_cost_usd) + '</td>';
      mBody.appendChild(tr);
    });
  }

  const projBody = document.getElementById('tbl-project');
  projBody.innerHTML = '';
  (d.by_project || []).forEach(p => {
    const tr = document.createElement('tr');
    tr.innerHTML = '<td>' + esc(p.project) + '</td><td>' + p.sessions + '</td><td>' + fmtTime(p.agent_time_seconds || 0) + '</td><td>' + fmtTime(p.user_active_time_sec || 0) + '</td><td>' + fmt(p.input_tokens) + ' / ' + fmt(p.output_tokens) + '</td><td>' + fmtCost(p.cost_usd) + '</td>';
    projBody.appendChild(tr);
  });

  const agentBody = document.getElementById('tbl-agent');
  agentBody.innerHTML = '';
  (d.by_agent || []).forEach(a => {
    const tr = document.createElement('tr');
    tr.innerHTML = '<td>' + esc(a.agent) + '</td><td>' + a.sessions + '</td><td>' + esc(a.agent_time) + '</td><td>' + esc(a.user_time) + '</td><td>' + esc(a.tokens) + '</td><td>' + fmtCost(a.cost) + '</td>';
    agentBody.appendChild(tr);
  });

  const groups = d.groups || [];
  const wiSection = document.getElementById('section-workitem');
  wiSection.style.display = groups.length <= 1 ? 'none' : '';
  const wiBody = document.getElementById('tbl-workitem');
  wiBody.innerHTML = '';
  groups.forEach(g => {
    const tr = document.createElement('tr');
    tr.innerHTML = '<td>' + esc(g.label) + '</td><td>' + esc(g.project) + '</td><td>' + (g.sessions_count || 0) + '</td><td>' + fmtTime(g.agent_time_sec || 0) + '</td><td>' + fmtTime(g.user_active_time_sec || 0) + '</td><td>' + fmtCost(g.estimated_cost_usd) + '</td>';
    wiBody.appendChild(tr);
  });

  const sessBody = document.getElementById('tbl-sessions');
  sessBody.innerHTML = '';
  (d.sessions || []).forEach(s => {
    const tr = document.createElement('tr');
    const t = new Date(s.started_at || 0);
    tr.innerHTML = '<td>' + t.toLocaleString() + '</td><td>' + esc(s.project) + '</td><td>' + esc(s.branch) + '</td><td>' + esc(s.tool) + '</td><td>' + esc(s.model) + '</td><td>' + (s.turns || 0) + '</td><td>' + fmtTime(s.agent_time_sec || 0) + '</td><td>' + fmtTime(s.user_time_sec || 0) + '</td><td>' + esc(s.work_item) + '</td><td>' + fmtCost(s.cost_usd) + '</td>';
    sessBody.appendChild(tr);
  });

  document.getElementById('status').textContent = 'Updated: ' + new Date().toLocaleTimeString();
}

function load() {
  fetch('/api/report')
    .then(r => r.json())
    .then(render)
    .catch(e => console.error('fetch error', e));
}

load();
setInterval(load, 60000);
</script>
</body>
</html>`

// HandleDashboard serves the HTML dashboard.
func HandleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(dashboardHTML)) //nolint:errcheck
}

// HandleAPIReport returns JSON report data (same structure as tt report --json).
func HandleAPIReport(conn *sql.DB, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opts.Since.IsZero() {
			opts.Since = time.Now().UTC().Add(-7 * 24 * time.Hour)
		}
		result, err := Query(conn, opts)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(FormatJSON(result))) //nolint:errcheck
	}
}
