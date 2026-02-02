package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func serveDashboard(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, dashboardHTML)
}

const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>CR Agent Dashboard</title>
  <style>
    :root {
      --bg: #f7f4ef;
      --ink: #1b1b1b;
      --muted: #6b6b6b;
      --card: #ffffff;
      --accent: #0c3b2e;
      --accent-2: #c8a26b;
      --border: #e1d9ce;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Segoe UI", "Helvetica Neue", Arial, sans-serif;
      color: var(--ink);
      background: linear-gradient(180deg, #f7f4ef 0%, #f1ebe3 60%, #f4f0ea 100%);
    }
    header {
      padding: 24px 32px;
      border-bottom: 1px solid var(--border);
      background: #fffaf3;
      position: sticky;
      top: 0;
      z-index: 10;
    }
    h1 { margin: 0; font-size: 22px; letter-spacing: 0.5px; }
    .filters {
      margin-top: 12px;
      display: flex;
      flex-wrap: wrap;
      gap: 12px;
      align-items: center;
      font-size: 13px;
    }
    .filters input {
      padding: 6px 8px;
      border: 1px solid var(--border);
      border-radius: 6px;
      background: #fff;
    }
    .filters select {
      padding: 6px 8px;
      border: 1px solid var(--border);
      border-radius: 6px;
      background: #fff;
    }
    .filters button {
      padding: 7px 12px;
      border: none;
      border-radius: 6px;
      background: var(--accent);
      color: #fff;
      cursor: pointer;
    }
    .tabs {
      margin-top: 12px;
      display: flex;
      gap: 8px;
      flex-wrap: wrap;
    }
    .tab-btn {
      padding: 6px 12px;
      border-radius: 999px;
      border: 1px solid var(--border);
      background: #fff;
      cursor: pointer;
      font-size: 12px;
    }
    .tab-btn.active {
      background: var(--accent);
      border-color: var(--accent);
      color: #fff;
    }
    .tab-content { display: none; }
    .tab-content.active { display: block; }
    .layout {
      padding: 24px 32px 40px;
      display: grid;
      gap: 16px;
    }
    .cards {
      display: grid;
      gap: 12px;
      grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    }
    .card {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 14px 16px;
      box-shadow: 0 4px 12px rgba(0,0,0,0.04);
    }
    .card .label { color: var(--muted); font-size: 12px; }
    .card .value { font-size: 22px; margin-top: 6px; }
    .grid {
      display: grid;
      gap: 16px;
      grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    }
    .panel {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 12px 14px;
    }
    .panel h3 { margin: 0 0 8px 0; font-size: 14px; color: var(--muted); }
    .panel h4 { margin: 8px 0; font-size: 13px; color: var(--muted); }
    #runsChart, #hitsChart, #topRulesChart { height: 260px; }
    #changeTrendChart, #ruleQualityTrendChart { height: 220px; }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 12px;
    }
    thead th {
      text-align: left;
      padding: 8px 6px;
      border-bottom: 1px solid var(--border);
      color: var(--muted);
    }
    tbody td {
      padding: 8px 6px;
      border-bottom: 1px dashed var(--border);
    }
    .muted { color: var(--muted); }
    .mini-cards {
      display: grid;
      gap: 10px;
      grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
      margin-bottom: 12px;
    }
    .mini-card {
      padding: 10px 12px;
      border: 1px solid var(--border);
      border-radius: 8px;
      background: #fffdfa;
    }
    .mini-card .label { color: var(--muted); font-size: 11px; }
    .mini-card .value { font-size: 18px; margin-top: 4px; }
    .link-btn {
      padding: 4px 8px;
      border: 1px solid var(--border);
      border-radius: 6px;
      background: #fff;
      cursor: pointer;
      font-size: 11px;
    }
    .section-note {
      font-size: 12px;
      color: var(--muted);
      margin-top: 6px;
    }
    @media (max-width: 720px) {
      header { padding: 18px; }
      .layout { padding: 18px; }
    }
  </style>
</head>
<body>
  <header>
    <h1>CR Agent Dashboard</h1>
    <div class="filters">
      <label>From <input type="datetime-local" id="fromInput"></label>
      <label>To <input type="datetime-local" id="toInput"></label>
      <label>Repo <input type="text" id="repoInput" placeholder="org/name"></label>
      <button id="refreshBtn">Refresh</button>
      <span class="muted" id="rangeHint"></span>
    </div>
    <div class="tabs">
      <button class="tab-btn active" data-tab="overview">Overview</button>
      <button class="tab-btn" data-tab="effectiveness">Change Effectiveness</button>
      <button class="tab-btn" data-tab="rule-quality">Rule Quality</button>
    </div>
  </header>

  <section id="tab-overview" class="tab-content active">
    <div class="layout">
    <div class="cards">
      <div class="card"><div class="label">Total Runs</div><div class="value" id="totalRuns">-</div></div>
      <div class="card"><div class="label">Total Hits</div><div class="value" id="totalHits">-</div></div>
      <div class="card"><div class="label">Avg Hit Density</div><div class="value" id="avgDensity">-</div></div>
      <div class="card"><div class="label">Active Repos</div><div class="value" id="activeRepos">-</div></div>
    </div>

    <div class="grid">
      <div class="panel">
        <h3>Runs Over Time</h3>
        <div id="runsChart"></div>
      </div>
      <div class="panel">
        <h3>Hits Over Time</h3>
        <div id="hitsChart"></div>
      </div>
      <div class="panel">
        <h3>Top Rules</h3>
        <div id="topRulesChart"></div>
      </div>
    </div>

    <div class="panel">
      <h3>Change Effectiveness Snapshot</h3>
      <div class="mini-cards">
        <div class="mini-card"><div class="label">Effective Changes (>=2 runs)</div><div class="value" id="ceTotal">-</div></div>
        <div class="mini-card"><div class="label">Improving Changes</div><div class="value" id="ceImproving">-</div></div>
        <div class="mini-card"><div class="label">Stable Changes</div><div class="value" id="ceStable">-</div></div>
        <div class="mini-card"><div class="label">Avg Improvement Rate</div><div class="value" id="ceAvgRate">-</div></div>
      </div>
      <div class="grid">
        <div>
          <h4>Top Improvements</h4>
          <table>
            <thead>
              <tr>
                <th>Repo</th>
                <th>Change ID</th>
                <th>Delta</th>
                <th>Rate</th>
              </tr>
            </thead>
            <tbody id="topImprovingTable"></tbody>
          </table>
        </div>
        <div>
          <h4>Low Improvements</h4>
          <table>
            <thead>
              <tr>
                <th>Repo</th>
                <th>Change ID</th>
                <th>Delta</th>
                <th>Rate</th>
              </tr>
            </thead>
            <tbody id="lowImprovingTable"></tbody>
          </table>
        </div>
      </div>
    </div>

    <div class="panel">
      <h3>Rule Quality Snapshot</h3>
      <div class="mini-cards">
        <div class="mini-card"><div class="label">Active Rules</div><div class="value" id="rqTotal">-</div></div>
        <div class="mini-card"><div class="label">Avg Fix Rate</div><div class="value" id="rqAvgFix">-</div></div>
        <div class="mini-card"><div class="label">Avg Disappear Rate</div><div class="value" id="rqAvgDisappear">-</div></div>
        <div class="mini-card"><div class="label">Avg Hit Rate</div><div class="value" id="rqAvgHit">-</div></div>
      </div>
      <div class="grid">
        <div>
          <h4>Top Fix Rate</h4>
          <table>
            <thead>
              <tr>
                <th>Rule ID</th>
                <th>Hits</th>
                <th>Fix Rate</th>
              </tr>
            </thead>
            <tbody id="rqTopTable"></tbody>
          </table>
        </div>
        <div>
          <h4>Low Fix Rate</h4>
          <table>
            <thead>
              <tr>
                <th>Rule ID</th>
                <th>Hits</th>
                <th>Fix Rate</th>
              </tr>
            </thead>
            <tbody id="rqLowTable"></tbody>
          </table>
        </div>
      </div>
    </div>

    <div class="panel">
      <h3>Recent Runs</h3>
      <table>
        <thead>
          <tr>
            <th>Reported At</th>
            <th>Repo</th>
            <th>Change ID</th>
            <th>Run ID</th>
            <th>Hits</th>
            <th>Diff Lines</th>
            <th>Density</th>
            <th>Agent</th>
            <th>Ruleset</th>
          </tr>
        </thead>
        <tbody id="runsTable"></tbody>
      </table>
    </div>
    </div>
  </section>

  <section id="tab-effectiveness" class="tab-content">
    <div class="layout">
      <div class="panel">
        <h3>Change Effectiveness</h3>
        <div class="filters">
          <label>Min Runs <input type="number" id="ceMinRuns" min="1" value="2" style="width:80px"></label>
          <label>Change ID <input type="text" id="ceChangeId" placeholder="code_change_id"></label>
          <label>Sort
            <select id="ceSort">
              <option value="improvement_rate">Improvement Rate</option>
              <option value="delta">Delta</option>
              <option value="last_reported_at">Last Reported</option>
              <option value="run_count">Run Count</option>
            </select>
          </label>
          <label>Order
            <select id="ceOrder">
              <option value="desc">Desc</option>
              <option value="asc">Asc</option>
            </select>
          </label>
          <label>Limit <input type="number" id="ceLimit" min="1" max="500" value="50" style="width:80px"></label>
          <button id="ceRefreshBtn">Load</button>
        </div>
        <div class="section-note">Uses max/min hits across runs to measure improvement. Click "Trend" to load per-change history on demand.</div>
      </div>

      <div class="panel">
        <h3>Change List</h3>
        <table>
          <thead>
            <tr>
              <th>Repo</th>
              <th>Change ID</th>
              <th>Runs</th>
              <th>Max</th>
              <th>Min</th>
              <th>Delta</th>
              <th>Rate</th>
              <th>Last Reported</th>
              <th>Ruleset</th>
              <th>Trend</th>
            </tr>
          </thead>
          <tbody id="changeTable"></tbody>
        </table>
      </div>

      <div class="panel" id="trendPanel" style="display:none">
        <h3 id="trendTitle">Change Trend</h3>
        <div id="changeTrendChart"></div>
      </div>
    </div>
  </section>

  <section id="tab-rule-quality" class="tab-content">
    <div class="layout">
      <div class="panel">
        <h3>Rule Quality</h3>
        <div class="filters">
          <label>Rule ID <input type="text" id="rqRuleId" placeholder="rule_id"></label>
          <label>Ruleset <input type="text" id="rqRuleset" placeholder="ruleset_version"></label>
          <label>Min Runs <input type="number" id="rqMinRuns" min="1" value="1" style="width:80px"></label>
          <label>Min Changes <input type="number" id="rqMinChanges" min="1" value="2" style="width:90px"></label>
          <label>Sort
            <select id="rqSort">
              <option value="fix_rate">Fix Rate</option>
              <option value="disappear_rate">Disappear Rate</option>
              <option value="total_hits">Total Hits</option>
              <option value="run_count">Run Count</option>
              <option value="last_seen_at">Last Seen</option>
              <option value="avg_drop">Avg Drop</option>
            </select>
          </label>
          <label>Order
            <select id="rqOrder">
              <option value="desc">Desc</option>
              <option value="asc">Asc</option>
            </select>
          </label>
          <label>Limit <input type="number" id="rqLimit" min="1" max="500" value="50" style="width:80px"></label>
          <button id="rqRefreshBtn">Load</button>
        </div>
        <div class="section-note">Fix rate measures how often a rule’s hits decrease within the same change. Trend loads on demand.</div>
      </div>

      <div class="panel">
        <h3>Rule List</h3>
        <table>
          <thead>
            <tr>
              <th>Rule ID</th>
              <th>Total Hits</th>
              <th>Run Count</th>
              <th>Hit Rate</th>
              <th>Fix Rate</th>
              <th>Disappear Rate</th>
              <th>Avg Drop</th>
              <th>Last Seen</th>
              <th>Trend</th>
            </tr>
          </thead>
          <tbody id="rqTable"></tbody>
        </table>
      </div>

      <div class="panel" id="rqTrendPanel" style="display:none">
        <h3 id="rqTrendTitle">Rule Trend</h3>
        <div id="ruleQualityTrendChart"></div>
      </div>
    </div>
  </section>

  <script src="https://cdn.jsdelivr.net/npm/echarts@5/dist/echarts.min.js"></script>
  <script>
    const els = {
      from: document.getElementById('fromInput'),
      to: document.getElementById('toInput'),
      repo: document.getElementById('repoInput'),
      refresh: document.getElementById('refreshBtn'),
      hint: document.getElementById('rangeHint'),
      totalRuns: document.getElementById('totalRuns'),
      totalHits: document.getElementById('totalHits'),
      avgDensity: document.getElementById('avgDensity'),
      activeRepos: document.getElementById('activeRepos'),
      runsTable: document.getElementById('runsTable'),
      ceTotal: document.getElementById('ceTotal'),
      ceImproving: document.getElementById('ceImproving'),
      ceStable: document.getElementById('ceStable'),
      ceAvgRate: document.getElementById('ceAvgRate'),
      topImprovingTable: document.getElementById('topImprovingTable'),
      lowImprovingTable: document.getElementById('lowImprovingTable'),
      ceMinRuns: document.getElementById('ceMinRuns'),
      ceChangeId: document.getElementById('ceChangeId'),
      ceSort: document.getElementById('ceSort'),
      ceOrder: document.getElementById('ceOrder'),
      ceLimit: document.getElementById('ceLimit'),
      ceRefreshBtn: document.getElementById('ceRefreshBtn'),
      changeTable: document.getElementById('changeTable'),
      trendPanel: document.getElementById('trendPanel'),
      trendTitle: document.getElementById('trendTitle'),
      rqTotal: document.getElementById('rqTotal'),
      rqAvgFix: document.getElementById('rqAvgFix'),
      rqAvgDisappear: document.getElementById('rqAvgDisappear'),
      rqAvgHit: document.getElementById('rqAvgHit'),
      rqTopTable: document.getElementById('rqTopTable'),
      rqLowTable: document.getElementById('rqLowTable'),
      rqRuleId: document.getElementById('rqRuleId'),
      rqRuleset: document.getElementById('rqRuleset'),
      rqMinRuns: document.getElementById('rqMinRuns'),
      rqMinChanges: document.getElementById('rqMinChanges'),
      rqSort: document.getElementById('rqSort'),
      rqOrder: document.getElementById('rqOrder'),
      rqLimit: document.getElementById('rqLimit'),
      rqRefreshBtn: document.getElementById('rqRefreshBtn'),
      rqTable: document.getElementById('rqTable'),
      rqTrendPanel: document.getElementById('rqTrendPanel'),
      rqTrendTitle: document.getElementById('rqTrendTitle'),
      tabButtons: document.querySelectorAll('.tab-btn'),
      overviewTab: document.getElementById('tab-overview'),
      effectivenessTab: document.getElementById('tab-effectiveness'),
      ruleQualityTab: document.getElementById('tab-rule-quality')
    };

    const charts = {
      runs: echarts.init(document.getElementById('runsChart')),
      hits: echarts.init(document.getElementById('hitsChart')),
      rules: echarts.init(document.getElementById('topRulesChart')),
      changeTrend: null,
      ruleTrend: null
    };

    function toLocalInputValue(date) {
      const pad = n => String(n).padStart(2, '0');
      return date.getFullYear() + '-' + pad(date.getMonth() + 1) + '-' + pad(date.getDate()) +
        'T' + pad(date.getHours()) + ':' + pad(date.getMinutes());
    }

    function fromLocalInput(value) {
      if (!value) return null;
      const d = new Date(value);
      return isNaN(d.getTime()) ? null : d.toISOString();
    }

    function buildQuery() {
      const params = new URLSearchParams();
      const from = fromLocalInput(els.from.value);
      const to = fromLocalInput(els.to.value);
      const repo = els.repo.value.trim();
      if (from) params.set('from', from);
      if (to) params.set('to', to);
      if (repo) params.set('repo', repo);
      return params.toString();
    }

    async function fetchJSON(path) {
      const res = await fetch(path);
      if (!res.ok) throw new Error('Request failed: ' + res.status);
      return res.json();
    }

    function renderSummary(data) {
      els.totalRuns.textContent = data.total_runs;
      els.totalHits.textContent = data.total_hits;
      els.avgDensity.textContent = data.avg_hit_density.toFixed(4);
      els.activeRepos.textContent = data.active_repos;
      els.hint.textContent = new Date(data.from).toLocaleString() + ' ~ ' + new Date(data.to).toLocaleString();
    }

    function renderTimeseries(chart, title, data, color) {
      const labels = data.map(p => p.bucket);
      const values = data.map(p => p.value || 0);
      chart.setOption({
        tooltip: { trigger: 'axis' },
        xAxis: { type: 'category', data: labels, axisLabel: { color: '#6b6b6b' } },
        yAxis: { type: 'value', axisLabel: { color: '#6b6b6b' } },
        series: [{ name: title, type: 'line', data: values, smooth: true, areaStyle: { opacity: 0.12 }, lineStyle: { color } }]
      });
    }

    function renderTopRules(chart, data) {
      const labels = data.map(r => r.rule_id);
      const values = data.map(r => r.total_hits);
      chart.setOption({
        tooltip: { trigger: 'axis' },
        xAxis: { type: 'value', axisLabel: { color: '#6b6b6b' } },
        yAxis: { type: 'category', data: labels, axisLabel: { color: '#6b6b6b' } },
        series: [{ type: 'bar', data: values, itemStyle: { color: '#0c3b2e' } }]
      });
    }

    function renderRunsTable(rows) {
      els.runsTable.innerHTML = rows.map(r =>
        '<tr>' +
          '<td>' + new Date(r.reported_at).toLocaleString() + '</td>' +
          '<td>' + r.repo + '</td>' +
          '<td>' + r.code_change_id + '</td>' +
          '<td>' + r.agent_run_id + '</td>' +
          '<td>' + r.triggered_total_hits + '</td>' +
          '<td>' + r.diff_lines + '</td>' +
          '<td>' + r.hit_density.toFixed(4) + '</td>' +
          '<td>' + r.agent_version + '</td>' +
          '<td>' + r.ruleset_version + '</td>' +
        '</tr>'
      ).join('');
    }

    function formatPercent(value) {
      if (value === null || value === undefined) return 'N/A';
      return (value * 100).toFixed(2) + '%';
    }

    function formatRate(value) {
      if (value === null || value === undefined) return 'N/A';
      return (value * 100).toFixed(2) + '%';
    }

    function buildRuleQualityQuery(includeRuleId) {
      const params = new URLSearchParams(buildQuery());
      const ruleId = els.rqRuleId.value.trim();
      const ruleset = els.rqRuleset.value.trim();
      const minRuns = parseInt(els.rqMinRuns.value, 10);
      const minChanges = parseInt(els.rqMinChanges.value, 10);
      if (includeRuleId && ruleId) params.set('rule_id', ruleId);
      if (ruleset) params.set('ruleset_version', ruleset);
      if (!isNaN(minRuns)) params.set('min_runs', minRuns);
      if (!isNaN(minChanges)) params.set('min_changes', minChanges);
      return params.toString();
    }

    function renderRuleQualitySummary(data) {
      els.rqTotal.textContent = data.total_rules;
      els.rqAvgFix.textContent = formatRate(data.avg_fix_rate);
      els.rqAvgDisappear.textContent = formatRate(data.avg_disappear_rate);
      els.rqAvgHit.textContent = formatRate(data.avg_hit_rate);
    }

    function renderRuleQualityTop(tableEl, rows) {
      tableEl.innerHTML = rows.map(r =>
        '<tr>' +
          '<td>' + r.rule_id + '</td>' +
          '<td>' + r.total_hits + '</td>' +
          '<td>' + formatRate(r.fix_rate) + '</td>' +
        '</tr>'
      ).join('');
    }

    function renderRuleQualityList(rows) {
      els.rqTable.innerHTML = rows.map(r =>
        '<tr>' +
          '<td>' + r.rule_id + '</td>' +
          '<td>' + r.total_hits + '</td>' +
          '<td>' + r.run_count + '</td>' +
          '<td>' + formatRate(r.hit_rate) + '</td>' +
          '<td>' + formatRate(r.fix_rate) + '</td>' +
          '<td>' + formatRate(r.disappear_rate) + '</td>' +
          '<td>' + (r.avg_drop === null || r.avg_drop === undefined ? 'N/A' : r.avg_drop.toFixed(2)) + '</td>' +
          '<td>' + new Date(r.last_seen_at).toLocaleString() + '</td>' +
          '<td><button class="link-btn" data-rule="' + r.rule_id + '">Trend</button></td>' +
        '</tr>'
      ).join('');
    }

    function renderRuleTrend(rows, ruleId) {
      els.rqTrendPanel.style.display = 'block';
      els.rqTrendTitle.textContent = 'Rule Trend: ' + ruleId;
      if (!charts.ruleTrend) {
        charts.ruleTrend = echarts.init(document.getElementById('ruleQualityTrendChart'));
      }
      const labels = rows.map(r => r.bucket);
      const values = rows.map(r => r.value || 0);
      charts.ruleTrend.setOption({
        tooltip: { trigger: 'axis' },
        xAxis: { type: 'category', data: labels, axisLabel: { color: '#6b6b6b' } },
        yAxis: { type: 'value', axisLabel: { color: '#6b6b6b' } },
        series: [{ name: 'Hits', type: 'line', data: values, smooth: true, lineStyle: { color: '#0c3b2e' } }]
      });
    }

    function buildChangeQuery(includeChangeId) {
      const params = new URLSearchParams(buildQuery());
      const minRuns = parseInt(els.ceMinRuns.value, 10);
      if (!isNaN(minRuns)) params.set('min_runs', minRuns);
      if (includeChangeId) {
        const changeId = els.ceChangeId.value.trim();
        if (changeId) params.set('code_change_id', changeId);
      }
      return params.toString();
    }

    function renderChangeSummary(data) {
      els.ceTotal.textContent = data.total_changes;
      els.ceImproving.textContent = data.improving_changes;
      els.ceStable.textContent = data.stable_changes;
      els.ceAvgRate.textContent = formatRate(data.avg_improvement_rate);
    }

    function renderChangeTop(tableEl, rows) {
      tableEl.innerHTML = rows.map(r =>
        '<tr>' +
          '<td>' + r.repo + '</td>' +
          '<td>' + r.code_change_id + '</td>' +
          '<td>' + r.delta + '</td>' +
          '<td>' + formatRate(r.improvement_rate) + '</td>' +
        '</tr>'
      ).join('');
    }

    function renderChangeList(rows) {
      els.changeTable.innerHTML = rows.map(r =>
        '<tr>' +
          '<td>' + r.repo + '</td>' +
          '<td>' + r.code_change_id + '</td>' +
          '<td>' + r.run_count + '</td>' +
          '<td>' + r.max_total_hits + '</td>' +
          '<td>' + r.min_total_hits + '</td>' +
          '<td>' + r.delta + '</td>' +
          '<td>' + formatRate(r.improvement_rate) + '</td>' +
          '<td>' + new Date(r.last_reported_at).toLocaleString() + '</td>' +
          '<td>' + r.last_ruleset_version + '</td>' +
          '<td><button class="link-btn" data-change="' + r.code_change_id + '" data-repo="' + r.repo + '">Trend</button></td>' +
        '</tr>'
      ).join('');
    }

    function renderChangeTrend(rows, changeId, repo) {
      els.trendPanel.style.display = 'block';
      els.trendTitle.textContent = 'Change Trend: ' + changeId + (repo ? ' (' + repo + ')' : '');
      if (!charts.changeTrend) {
        charts.changeTrend = echarts.init(document.getElementById('changeTrendChart'));
      }
      const labels = rows.map(r => new Date(r.reported_at).toLocaleString());
      const values = rows.map(r => r.triggered_total_hits);
      charts.changeTrend.setOption({
        tooltip: { trigger: 'axis' },
        xAxis: { type: 'category', data: labels, axisLabel: { color: '#6b6b6b' } },
        yAxis: { type: 'value', axisLabel: { color: '#6b6b6b' } },
        series: [{ name: 'Hits', type: 'line', data: values, smooth: true, lineStyle: { color: '#0c3b2e' } }]
      });
    }

    async function loadAll() {
      const qs = buildQuery();
      const summary = await fetchJSON('/api/summary' + (qs ? '?' + qs : ''));
      renderSummary(summary);

      const runsSeries = await fetchJSON('/api/timeseries?metric=runs&bucket=hour' + (qs ? '&' + qs : ''));
      renderTimeseries(charts.runs, 'Runs', runsSeries.data, '#0c3b2e');

      const hitsSeries = await fetchJSON('/api/timeseries?metric=hits&bucket=hour' + (qs ? '&' + qs : ''));
      renderTimeseries(charts.hits, 'Hits', hitsSeries.data, '#c8a26b');

      const topRules = await fetchJSON('/api/rules/top?limit=10' + (qs ? '&' + qs : ''));
      renderTopRules(charts.rules, topRules.data);

      const recent = await fetchJSON('/api/runs/recent?limit=50' + (qs ? '&' + qs : ''));
      renderRunsTable(recent.data);

      await loadChangeSnapshot();
      await loadRuleQualitySnapshot();
      if (els.effectivenessTab.classList.contains('active')) {
        await loadChangeList();
      }
      if (els.ruleQualityTab.classList.contains('active')) {
        await loadRuleQualityList();
      }
    }

    async function loadChangeSnapshot() {
      const qs = buildChangeQuery(false);
      const summary = await fetchJSON('/api/change-effectiveness/summary' + (qs ? '?' + qs : ''));
      renderChangeSummary(summary);

      const top = await fetchJSON('/api/change-effectiveness/top?direction=high&limit=5' + (qs ? '&' + qs : ''));
      renderChangeTop(els.topImprovingTable, top.data);

      const low = await fetchJSON('/api/change-effectiveness/top?direction=low&limit=5' + (qs ? '&' + qs : ''));
      renderChangeTop(els.lowImprovingTable, low.data);
    }

    async function loadChangeList() {
      const params = new URLSearchParams(buildChangeQuery(true));
      params.set('sort', els.ceSort.value);
      params.set('order', els.ceOrder.value);
      params.set('limit', els.ceLimit.value);
      const list = await fetchJSON('/api/change-effectiveness/list?' + params.toString());
      renderChangeList(list.data || []);
    }

    async function loadChangeTrend(changeId, repo) {
      const params = new URLSearchParams();
      params.set('code_change_id', changeId);
      if (repo) params.set('repo', repo);
      params.set('limit', '50');
      const data = await fetchJSON('/api/change-effectiveness/runs?' + params.toString());
      renderChangeTrend(data.data || [], changeId, repo);
    }

    async function loadRuleQualitySnapshot() {
      const qs = buildRuleQualityQuery(false);
      const summary = await fetchJSON('/api/rule-quality/summary' + (qs ? '?' + qs : ''));
      renderRuleQualitySummary(summary);

      const top = await fetchJSON('/api/rule-quality/top?direction=high&limit=5' + (qs ? '&' + qs : ''));
      renderRuleQualityTop(els.rqTopTable, top.data);

      const low = await fetchJSON('/api/rule-quality/top?direction=low&limit=5' + (qs ? '&' + qs : ''));
      renderRuleQualityTop(els.rqLowTable, low.data);
    }

    async function loadRuleQualityList() {
      const params = new URLSearchParams(buildRuleQualityQuery(true));
      params.set('sort', els.rqSort.value);
      params.set('order', els.rqOrder.value);
      params.set('limit', els.rqLimit.value);
      const list = await fetchJSON('/api/rule-quality/list?' + params.toString());
      renderRuleQualityList(list.data || []);
    }

    async function loadRuleQualityTrend(ruleId) {
      const params = new URLSearchParams(buildRuleQualityQuery(true));
      params.set('rule_id', ruleId);
      params.set('bucket', 'day');
      const data = await fetchJSON('/api/rule-quality/trend?' + params.toString());
      renderRuleTrend(data.data || [], ruleId);
    }

    function setActiveTab(name) {
      els.tabButtons.forEach(btn => {
        btn.classList.toggle('active', btn.dataset.tab === name);
      });
      els.overviewTab.classList.toggle('active', name === 'overview');
      els.effectivenessTab.classList.toggle('active', name === 'effectiveness');
      els.ruleQualityTab.classList.toggle('active', name === 'rule-quality');
      if (name === 'effectiveness') {
        loadChangeList().catch(console.error);
      }
      if (name === 'rule-quality') {
        loadRuleQualityList().catch(console.error);
      }
    }

    function initRange() {
      const now = new Date();
      const from = new Date(now.getTime() - 7 * 24 * 3600 * 1000);
      els.from.value = toLocalInputValue(from);
      els.to.value = toLocalInputValue(now);
    }

    els.refresh.addEventListener('click', () => loadAll().catch(console.error));
    els.ceRefreshBtn.addEventListener('click', () => loadChangeList().catch(console.error));
    els.rqRefreshBtn.addEventListener('click', () => loadRuleQualityList().catch(console.error));
    els.tabButtons.forEach(btn => {
      btn.addEventListener('click', () => {
        setActiveTab(btn.dataset.tab);
      });
    });
    els.changeTable.addEventListener('click', (event) => {
      const btn = event.target.closest('button[data-change]');
      if (!btn) return;
      loadChangeTrend(btn.getAttribute('data-change'), btn.getAttribute('data-repo')).catch(console.error);
    });
    els.rqTable.addEventListener('click', (event) => {
      const btn = event.target.closest('button[data-rule]');
      if (!btn) return;
      loadRuleQualityTrend(btn.getAttribute('data-rule')).catch(console.error);
    });
    window.addEventListener('resize', () => {
      charts.runs.resize();
      charts.hits.resize();
      charts.rules.resize();
      if (charts.changeTrend) charts.changeTrend.resize();
      if (charts.ruleTrend) charts.ruleTrend.resize();
    });

    initRange();
    loadAll().catch(console.error);
    loadChangeList().catch(console.error);
    loadRuleQualityList().catch(console.error);
  </script>
</body>
</html>`
