/* Logs module — Messages subtab + Tools subtab */

import { req } from '../api.js';
import { showToast } from '../toast.js';
import { getState, subscribe } from '../store.js';
import { registerTab } from '../router.js';

// ─── state ────────────────────────────────────────────────────────────────────
let toolsPollingTimer = null;
let toolsSort = { col: 'created_at', dir: 'desc' };
let allExecutions = [];   // cached for client-side filter/sort

// ─── Messages subtab ──────────────────────────────────────────────────────────

function populateLogsDropdown() {
    const select = document.getElementById('logs-session-select');
    if (!select) return;

    const sessionsData = getState('sessions') || [];
    select.innerHTML = '<option value="">Select a session</option>';

    if (sessionsData.length === 0) return;

    sessionsData.forEach(session => {
        const option = document.createElement('option');
        option.value = session.id;
        option.textContent = `${session.id} (${session.connector_type})`;
        select.appendChild(option);
    });
}

async function loadMessages(sessionId) {
    try {
        const messages = await req('GET', `/api/v1/sessions/${sessionId}/messages?limit=50`);
        renderMessages(messages || []);
    } catch (err) {
        showToast(`Failed to load messages: ${err.message}`, 'error');
    }
}

function renderMessages(messages) {
    const container = document.getElementById('logs-messages');
    container.innerHTML = '';

    if (messages.length === 0) {
        container.innerHTML = '<div style="color: var(--text-muted); text-align: center; padding: 2rem;">No messages</div>';
        return;
    }

    messages.forEach(msg => {
        const div = document.createElement('div');
        const role = msg.role || 'system';
        div.className = `message message-${role}`;

        const content = document.createElement('div');
        content.textContent = msg.content || '';

        const timestamp = document.createElement('div');
        timestamp.className = 'message-timestamp';
        const timeStr = msg.timestamp || msg.created_at;
        timestamp.textContent = new Date(timeStr).toLocaleString();

        div.appendChild(content);
        div.appendChild(timestamp);
        container.appendChild(div);
    });

    container.scrollTop = container.scrollHeight;
}

// ─── Tools subtab ─────────────────────────────────────────────────────────────

async function loadToolExecutions(sessionId) {
    try {
        const data = await req('GET', `/api/v1/sessions/${sessionId}/tool-executions?limit=100`);
        allExecutions = data.executions || [];
        populateToolNameFilter(allExecutions);
        renderToolsTable();
    } catch (err) {
        showToast(`Failed to load tool executions: ${err.message}`, 'error');
    }
}

function populateToolNameFilter(executions) {
    const select = document.getElementById('tool-name-filter');
    if (!select) return;
    const names = [...new Set(executions.map(e => e.tool_name))].sort();
    const current = select.value;
    select.innerHTML = '<option value="">All Tools</option>';
    names.forEach(name => {
        const opt = document.createElement('option');
        opt.value = name;
        opt.textContent = name;
        if (name === current) opt.selected = true;
        select.appendChild(opt);
    });
}

function getFilteredSorted() {
    const nameFilter  = document.getElementById('tool-name-filter')?.value  || '';
    const statusFilter = document.getElementById('tool-status-filter')?.value || '';

    let items = allExecutions.filter(e => {
        if (nameFilter && e.tool_name !== nameFilter) return false;
        if (statusFilter === 'success' && !e.success) return false;
        if (statusFilter === 'failed'  &&  e.success) return false;
        return true;
    });

    items.sort((a, b) => {
        let va, vb;
        switch (toolsSort.col) {
            case 'tool_name': va = a.tool_name; vb = b.tool_name; break;
            case 'status':    va = a.success ? 1 : 0; vb = b.success ? 1 : 0; break;
            case 'latency_ms': va = a.latency_ms; vb = b.latency_ms; break;
            default:          va = a.created_at; vb = b.created_at; break;
        }
        if (va < vb) return toolsSort.dir === 'asc' ? -1 : 1;
        if (va > vb) return toolsSort.dir === 'asc' ?  1 : -1;
        return 0;
    });

    return items;
}

function renderToolsTable() {
    const tbody = document.getElementById('tools-log-tbody');
    const empty = document.getElementById('tools-empty');
    if (!tbody) return;

    const items = getFilteredSorted();

    // Update sort icons
    document.querySelectorAll('#tools-log-table th.sortable').forEach(th => {
        const icon = th.querySelector('.sort-icon');
        if (!icon) return;
        if (th.dataset.col === toolsSort.col) {
            icon.textContent = toolsSort.dir === 'asc' ? ' ▲' : ' ▼';
        } else {
            icon.textContent = '';
        }
    });

    if (items.length === 0) {
        tbody.innerHTML = '';
        if (empty) empty.hidden = false;
        return;
    }
    if (empty) empty.hidden = true;

    tbody.innerHTML = '';
    items.forEach(e => {
        const tr = document.createElement('tr');

        // Tool name
        const tdName = document.createElement('td');
        tdName.className = 'tools-td-name';
        tdName.textContent = e.tool_name;
        tr.appendChild(tdName);

        // Parameters (truncated preview)
        const tdParams = document.createElement('td');
        tdParams.className = 'tools-td-params';
        const preview = e.input.length > 100 ? e.input.slice(0, 100) + '…' : e.input;
        tdParams.textContent = preview;
        tr.appendChild(tdParams);

        // Status
        const tdStatus = document.createElement('td');
        if (e.success) {
            tdStatus.innerHTML = '<span class="tools-badge tools-badge--success">Success</span>';
        } else {
            const msg = e.error_msg ? `: ${e.error_msg.slice(0, 60)}` : '';
            tdStatus.innerHTML = `<span class="tools-badge tools-badge--error">Error${msg}</span>`;
        }
        tr.appendChild(tdStatus);

        // Latency
        const tdLatency = document.createElement('td');
        tdLatency.className = 'tools-td-latency';
        tdLatency.textContent = formatLatency(e.latency_ms);
        tr.appendChild(tdLatency);

        // Timestamp
        const tdTime = document.createElement('td');
        tdTime.className = 'tools-td-time';
        const exact = new Date(e.created_at);
        tdTime.textContent = relativeTime(exact);
        tdTime.title = exact.toLocaleString();
        tr.appendChild(tdTime);

        // Expand button
        const tdActions = document.createElement('td');
        const btn = document.createElement('button');
        btn.className = 'btn-link tools-expand-btn';
        btn.textContent = 'Expand';
        btn.addEventListener('click', () => toggleExpand(tr, e));
        tdActions.appendChild(btn);
        tr.appendChild(tdActions);

        tbody.appendChild(tr);
    });
}

function toggleExpand(tr, e) {
    const existing = tr.nextElementSibling;
    if (existing && existing.classList.contains('tools-expand-row')) {
        existing.remove();
        tr.querySelector('.tools-expand-btn').textContent = 'Expand';
        return;
    }

    tr.querySelector('.tools-expand-btn').textContent = 'Collapse';

    const expandRow = document.createElement('tr');
    expandRow.className = 'tools-expand-row';
    const expandCell = document.createElement('td');
    expandCell.colSpan = 6;
    expandCell.className = 'tools-expand-cell';

    expandCell.innerHTML = `
        <div class="tools-expand-content">
            <div class="tools-expand-section">
                <strong>Input</strong>
                <pre class="tools-expand-pre">${escapeHtml(prettyJSON(e.input))}</pre>
            </div>
            <div class="tools-expand-section">
                <strong>Output</strong>
                <pre class="tools-expand-pre">${escapeHtml(prettyJSON(e.output))}</pre>
            </div>
        </div>`;

    expandRow.appendChild(expandCell);
    tr.after(expandRow);
}

function formatLatency(ms) {
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
}

function relativeTime(date) {
    const diff = Math.floor((Date.now() - date.getTime()) / 1000);
    if (diff < 60)   return `${diff}s ago`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    return `${Math.floor(diff / 86400)}d ago`;
}

function prettyJSON(str) {
    try { return JSON.stringify(JSON.parse(str), null, 2); }
    catch { return str; }
}

function escapeHtml(s) {
    return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

// ─── Polling ──────────────────────────────────────────────────────────────────

function startToolsPolling(sessionId) {
    stopToolsPolling();
    toolsPollingTimer = setInterval(() => {
        const activeSubtab = document.querySelector('.logs-subtab-btn.active')?.dataset.subtab;
        if (activeSubtab === 'tools') {
            loadToolExecutions(sessionId);
        }
    }, 3000);
}

function stopToolsPolling() {
    if (toolsPollingTimer) {
        clearInterval(toolsPollingTimer);
        toolsPollingTimer = null;
    }
}

// ─── Subtab switching ─────────────────────────────────────────────────────────

function activateSubtab(name) {
    document.querySelectorAll('.logs-subtab-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.subtab === name);
    });
    document.querySelectorAll('.logs-subtab-panel').forEach(panel => {
        panel.hidden = panel.id !== `logs-subtab-${name}`;
    });
}

// ─── Init ─────────────────────────────────────────────────────────────────────

export function init() {
    setupEventHandlers();
    populateLogsDropdown();

    registerTab('logs', () => {
        populateLogsDropdown();
    });

    subscribe('sessions', (sessionsData) => {
        if (sessionsData && sessionsData.length > 0) {
            populateLogsDropdown();
        }
    });
}

function setupEventHandlers() {
    // Session selector
    const select = document.getElementById('logs-session-select');
    if (select) {
        select.addEventListener('change', async (e) => {
            const id = e.target.value;
            stopToolsPolling();
            if (!id) return;

            const activeSubtab = document.querySelector('.logs-subtab-btn.active')?.dataset.subtab || 'messages';
            if (activeSubtab === 'messages') {
                await loadMessages(id);
            } else {
                await loadToolExecutions(id);
                startToolsPolling(id);
            }
        });
    }

    // Refresh button
    const refreshBtn = document.getElementById('logs-refresh');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', async () => {
            const id = document.getElementById('logs-session-select')?.value;
            if (!id) return;
            const activeSubtab = document.querySelector('.logs-subtab-btn.active')?.dataset.subtab || 'messages';
            if (activeSubtab === 'messages') {
                await loadMessages(id);
            } else {
                await loadToolExecutions(id);
            }
        });
    }

    // Subtab buttons
    document.querySelectorAll('.logs-subtab-btn').forEach(btn => {
        btn.addEventListener('click', async () => {
            const name = btn.dataset.subtab;
            activateSubtab(name);
            const id = document.getElementById('logs-session-select')?.value;
            if (!id) return;

            stopToolsPolling();
            if (name === 'messages') {
                await loadMessages(id);
            } else {
                await loadToolExecutions(id);
                startToolsPolling(id);
            }
        });
    });

    // Tools filter controls
    document.getElementById('tool-name-filter')?.addEventListener('change', renderToolsTable);
    document.getElementById('tool-status-filter')?.addEventListener('change', renderToolsTable);

    // Sort headers
    document.querySelectorAll('#tools-log-table th.sortable').forEach(th => {
        th.addEventListener('click', () => {
            const col = th.dataset.col;
            if (toolsSort.col === col) {
                toolsSort.dir = toolsSort.dir === 'asc' ? 'desc' : 'asc';
            } else {
                toolsSort.col = col;
                toolsSort.dir = 'desc';
            }
            renderToolsTable();
        });
    });
}
