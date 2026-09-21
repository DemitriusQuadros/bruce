/* Logs module — Messages subtab + Tools subtab with real-time filtering & smart scroll */

import { req } from '../api.js';
import { showToast } from '../toast.js';
import { getState, setState, subscribe } from '../store.js';
import { registerTab } from '../router.js';

// ─── state ────────────────────────────────────────────────────────────────────
let toolsPollingTimer = null;
let toolsSort = { col: 'created_at', dir: 'desc' };
let allExecutions = [];   // cached for client-side filter/sort
let allMessages = [];     // cached for client-side filter
let userScrolledUp = false;
let searchQuery = '';
let messageRoleFilter = '';
let expandedExecutionIds = new Set();
let selectedSessionId = null;
let isPopulatingDropdown = false;

function isLogsTabVisible() {
    const section = document.getElementById('tab-logs');
    return section && !section.hidden;
}

async function populateLogsDropdown(forceSelectId = null) {
    if (isPopulatingDropdown) return;
    isPopulatingDropdown = true;
    try {
        const select = document.getElementById('logs-session-select');
        if (!select) return;

        let sessionsData = getState('sessions') || [];
        try {
            const fresh = await req('GET', '/api/v1/sessions');
            if (Array.isArray(fresh)) {
                sessionsData = fresh;
            }
        } catch (e) {
            // use cached sessionsData
        }

        const desiredVal = forceSelectId || selectedSessionId || (select.value && select.value !== 'all' ? select.value : null) || getState('activeChatId') || (sessionsData.length > 0 ? sessionsData[0].id : 'all');

        // Check if existing options match new sessions data
        const currentOptions = Array.from(select.options).map(o => o.value);
        const expectedOptions = ['all', ...sessionsData.map(s => s.id)];

        const needsRebuild = currentOptions.length !== expectedOptions.length ||
            !currentOptions.every((v, i) => v === expectedOptions[i]);

        if (needsRebuild) {
            select.innerHTML = '<option value="all">Select a session / All Sessions (Global Tools)</option>';
            sessionsData.forEach(session => {
                const option = document.createElement('option');
                option.value = session.id;
                const title = session.title ? session.title : (session.id ? session.id.slice(0, 8) : 'Session');
                option.textContent = `${title} (${session.connector_type || 'chat'})`;
                select.appendChild(option);
            });
        }

        if (desiredVal && Array.from(select.options).some(o => o.value === desiredVal)) {
            select.value = desiredVal;
            selectedSessionId = desiredVal;
        } else if (sessionsData.length > 0) {
            select.value = sessionsData[0].id;
            selectedSessionId = sessionsData[0].id;
        } else {
            select.value = 'all';
            selectedSessionId = 'all';
        }

        if (isLogsTabVisible()) {
            await loadCurrentLogs();
        }
    } finally {
        isPopulatingDropdown = false;
    }
}

async function loadCurrentLogs() {
    const select = document.getElementById('logs-session-select');
    if (!select) return;
    const id = selectedSessionId || select.value;
    if (id && select.value !== id && Array.from(select.options).some(o => o.value === id)) {
        select.value = id;
    }
    const activeSubtab = document.querySelector('.logs-subtab-btn.active')?.dataset.subtab || 'messages';

    stopToolsPolling();
    if (activeSubtab === 'messages') {
        await loadMessages(id);
    } else {
        await loadToolExecutions(id);
        startToolsPolling();
    }
}

async function loadMessages(sessionId) {
    const container = document.getElementById('logs-messages');
    if (!container) return;

    if (!sessionId || sessionId === 'all') {
        container.innerHTML = '<div style="color: var(--text-muted); text-align: center; padding: 2rem;">Select a specific session from the dropdown above to view chat messages.</div>';
        allMessages = [];
        updateMessagesCount(0, 0);
        return;
    }

    try {
        const messages = await req('GET', `/api/v1/sessions/${sessionId}/messages?limit=100`);
        // Chronological order: oldest -> newest
        const chronological = Array.isArray(messages) ? messages.slice().reverse() : [];
        allMessages = chronological;
        renderMessages();
    } catch (err) {
        showToast(`Failed to load messages: ${err.message}`, 'error');
    }
}

function getFilteredMessages() {
    const q = searchQuery.toLowerCase();
    return allMessages.filter(msg => {
        if (messageRoleFilter && msg.role !== messageRoleFilter) return false;
        if (q) {
            const content = (msg.content || '').toLowerCase();
            const role = (msg.role || '').toLowerCase();
            const time = (msg.timestamp || msg.created_at || '').toLowerCase();
            if (!content.includes(q) && !role.includes(q) && !time.includes(q)) {
                return false;
            }
        }
        return true;
    });
}

function updateMessagesCount(filteredCount, totalCount) {
    const badge = document.getElementById('logs-messages-count');
    if (!badge) return;
    if (totalCount === 0) {
        badge.textContent = '';
    } else if (filteredCount === totalCount) {
        badge.textContent = `(${totalCount})`;
    } else {
        badge.textContent = `(${filteredCount}/${totalCount})`;
    }
}

function updateToolsCount(filteredCount, totalCount) {
    const badge = document.getElementById('logs-tools-count');
    if (!badge) return;
    if (totalCount === 0) {
        badge.textContent = '';
    } else if (filteredCount === totalCount) {
        badge.textContent = `(${totalCount})`;
    } else {
        badge.textContent = `(${filteredCount}/${totalCount})`;
    }
}

function renderMessages() {
    const container = document.getElementById('logs-messages');
    if (!container) return;

    const filtered = getFilteredMessages();
    updateMessagesCount(filtered.length, allMessages.length);

    // Toggle clear filter button
    const clearBtn = document.getElementById('messages-clear-filter-btn');
    if (clearBtn) {
        clearBtn.style.display = (searchQuery || messageRoleFilter) ? 'inline-block' : 'none';
    }

    if (allMessages.length === 0) {
        container.innerHTML = '<div style="color: var(--text-muted); text-align: center; padding: 2rem;">No messages in this session.</div>';
        return;
    }

    if (filtered.length === 0) {
        container.innerHTML = '<div style="color: var(--text-muted); text-align: center; padding: 2rem;">No messages match the current filter.</div>';
        return;
    }

    container.innerHTML = '';

    filtered.forEach(msg => {
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

    // Smart auto-scroll:
    // Only scroll to the bottom if the user has NOT scrolled up AND auto-scroll toggle is enabled
    const autoScrollToggle = document.getElementById('logs-autoscroll');
    const isAutoScrollEnabled = autoScrollToggle ? autoScrollToggle.checked : true;

    if (isAutoScrollEnabled && !userScrolledUp) {
        container.scrollTop = container.scrollHeight;
    }
}

// ─── Tools subtab ─────────────────────────────────────────────────────────────

async function loadToolExecutions(sessionId) {
    try {
        const url = (sessionId && sessionId !== 'all')
            ? `/api/v1/sessions/${sessionId}/tool-executions?limit=100`
            : `/api/v1/tool-executions?limit=100`;
        const data = await req('GET', url);
        const incoming = data.executions || [];

        // Check if data is identical to avoid disrupting DOM and scroll position
        if (isSameExecutions(incoming, allExecutions)) {
            return;
        }

        allExecutions = incoming;
        populateToolNameFilter(allExecutions);
        renderToolsTable();
    } catch (err) {
        showToast(`Failed to load tool executions: ${err.message}`, 'error');
    }
}

function isSameExecutions(a, b) {
    if (a.length !== b.length) return false;
    if (a.length === 0) return true;
    return a[0].id === b[0].id &&
           a[a.length - 1].id === b[b.length - 1].id &&
           a[0].latency_ms === b[0].latency_ms &&
           a[0].success === b[0].success;
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
    const q = searchQuery.toLowerCase();

    let items = allExecutions.filter(e => {
        if (nameFilter && e.tool_name !== nameFilter) return false;
        if (statusFilter === 'success' && !e.success) return false;
        if (statusFilter === 'failed'  &&  e.success) return false;
        if (q) {
            const tool = (e.tool_name || '').toLowerCase();
            const input = (e.input || '').toLowerCase();
            const output = (e.output || '').toLowerCase();
            const errMsg = (e.error_msg || '').toLowerCase();
            if (!tool.includes(q) && !input.includes(q) && !output.includes(q) && !errMsg.includes(q)) {
                return false;
            }
        }
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
    const container = document.querySelector('.tools-table-container');
    if (!tbody) return;

    const items = getFilteredSorted();
    updateToolsCount(items.length, allExecutions.length);

    // Toggle clear filter button
    const clearBtn = document.getElementById('tools-clear-filter-btn');
    if (clearBtn) {
        const nameFilter = document.getElementById('tool-name-filter')?.value || '';
        const statusFilter = document.getElementById('tool-status-filter')?.value || '';
        clearBtn.style.display = (searchQuery || nameFilter || statusFilter) ? 'inline-block' : 'none';
    }

    // Preserve scroll position
    const prevScrollTop = container ? container.scrollTop : 0;

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
        if (empty) {
            empty.hidden = false;
            empty.textContent = (searchQuery || document.getElementById('tool-name-filter')?.value || document.getElementById('tool-status-filter')?.value)
                ? 'No tool executions match the current filter.'
                : 'No tool executions for this session.';
        }
        return;
    }
    if (empty) empty.hidden = true;

    tbody.innerHTML = '';
    items.forEach(e => {
        const tr = document.createElement('tr');
        tr.dataset.id = e.id;

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
        const isExpanded = expandedExecutionIds.has(e.id);
        btn.textContent = isExpanded ? 'Collapse' : 'Expand';
        btn.addEventListener('click', () => toggleExpand(tr, e));
        tdActions.appendChild(btn);
        tr.appendChild(tdActions);

        tbody.appendChild(tr);

        // Re-expand if active
        if (isExpanded) {
            insertExpandRow(tr, e);
        }
    });

    // Restore table container scroll position
    if (container) {
        container.scrollTop = prevScrollTop;
    }
}

function toggleExpand(tr, e) {
    const existing = tr.nextElementSibling;
    if (existing && existing.classList.contains('tools-expand-row')) {
        existing.remove();
        expandedExecutionIds.delete(e.id);
        tr.querySelector('.tools-expand-btn').textContent = 'Expand';
        return;
    }

    expandedExecutionIds.add(e.id);
    tr.querySelector('.tools-expand-btn').textContent = 'Collapse';
    insertExpandRow(tr, e);
}

function insertExpandRow(tr, e) {
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

function startToolsPolling() {
    stopToolsPolling();
    toolsPollingTimer = setInterval(() => {
        const activeSubtab = document.querySelector('.logs-subtab-btn.active')?.dataset.subtab;
        if (activeSubtab === 'tools') {
            const id = document.getElementById('logs-session-select')?.value;
            loadToolExecutions(id);
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

    const searchInput = document.getElementById('logs-search-filter');
    if (searchInput) {
        searchInput.placeholder = (name === 'messages') ? 'Search messages...' : 'Search tools, inputs, outputs...';
    }
}

// ─── Init ─────────────────────────────────────────────────────────────────────

export function init() {
    setupEventHandlers();
    setupScrollTracking();
    populateLogsDropdown();

    registerTab('logs', () => {
        if (!selectedSessionId) {
            populateLogsDropdown();
        } else {
            loadCurrentLogs();
        }
    }, () => {
        stopToolsPolling();
    });

    subscribe('sessions', (sessionsData) => {
        if (sessionsData && sessionsData.length > 0 && isLogsTabVisible()) {
            populateLogsDropdown(selectedSessionId);
        }
    });
}

function setupScrollTracking() {
    const container = document.getElementById('logs-messages');
    if (container) {
        container.addEventListener('scroll', () => {
            const distFromBottom = container.scrollHeight - container.scrollTop - container.clientHeight;
            if (distFromBottom > 60) {
                userScrolledUp = true;
            } else if (distFromBottom <= 20) {
                userScrolledUp = false;
            }
        });
    }

    const autoScrollToggle = document.getElementById('logs-autoscroll');
    if (autoScrollToggle) {
        autoScrollToggle.addEventListener('change', () => {
            if (autoScrollToggle.checked) {
                userScrolledUp = false;
                const activeSubtab = document.querySelector('.logs-subtab-btn.active')?.dataset.subtab || 'messages';
                if (activeSubtab === 'messages') {
                    const c = document.getElementById('logs-messages');
                    if (c) c.scrollTop = c.scrollHeight;
                }
            }
        });
    }
}

function setupEventHandlers() {
    // Session selector
    const select = document.getElementById('logs-session-select');
    if (select) {
        select.addEventListener('change', async (e) => {
            selectedSessionId = e.target.value;
            const id = selectedSessionId;
            userScrolledUp = false; // Reset on session switch
            stopToolsPolling();
            const activeSubtab = document.querySelector('.logs-subtab-btn.active')?.dataset.subtab || 'messages';
            if (activeSubtab === 'messages') {
                await loadMessages(id);
            } else {
                await loadToolExecutions(id);
                startToolsPolling();
            }
        });
    }

    // Refresh button
    const refreshBtn = document.getElementById('logs-refresh');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', async () => {
            const icon = refreshBtn.querySelector('i');
            if (icon) icon.classList.add('fa-spin');
            await populateLogsDropdown();
            if (icon) icon.classList.remove('fa-spin');
            showToast('Logs refreshed', 'info');
        });
    }

    // Universal search filter
    const searchInput = document.getElementById('logs-search-filter');
    if (searchInput) {
        searchInput.addEventListener('input', (e) => {
            searchQuery = e.target.value.trim();
            const activeSubtab = document.querySelector('.logs-subtab-btn.active')?.dataset.subtab || 'messages';
            if (activeSubtab === 'messages') {
                renderMessages();
            } else {
                renderToolsTable();
            }
        });
    }

    // Messages role filter
    const roleSelect = document.getElementById('message-role-filter');
    if (roleSelect) {
        roleSelect.addEventListener('change', (e) => {
            messageRoleFilter = e.target.value;
            renderMessages();
        });
    }

    // Clear filters buttons
    document.getElementById('messages-clear-filter-btn')?.addEventListener('click', () => {
        searchQuery = '';
        messageRoleFilter = '';
        if (searchInput) searchInput.value = '';
        if (roleSelect) roleSelect.value = '';
        renderMessages();
    });

    document.getElementById('tools-clear-filter-btn')?.addEventListener('click', () => {
        searchQuery = '';
        if (searchInput) searchInput.value = '';
        const nameSelect = document.getElementById('tool-name-filter');
        if (nameSelect) nameSelect.value = '';
        const statusSelect = document.getElementById('tool-status-filter');
        if (statusSelect) statusSelect.value = '';
        renderToolsTable();
    });

    // Subtab buttons
    document.querySelectorAll('.logs-subtab-btn').forEach(btn => {
        btn.addEventListener('click', async () => {
            const name = btn.dataset.subtab;
            activateSubtab(name);
            const id = document.getElementById('logs-session-select')?.value;

            stopToolsPolling();
            if (name === 'messages') {
                await loadMessages(id);
            } else {
                await loadToolExecutions(id);
                startToolsPolling();
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
