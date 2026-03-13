/* Monitoring Dashboard Module */

import { setState, getState } from '../store.js';
import { registerTab } from '../router.js';

let autoRefreshInterval = null;
let logCurrentPage = 1;
let logTotalPages = 1;

export function init() {
    // Register tab with activation and cleanup callbacks
    registerTab('monitoring', renderMonitoring, cleanup);

    // Set up event listeners
    const levelFilter = document.getElementById('monitoring-level-filter');
    if (levelFilter) {
        levelFilter.addEventListener('change', () => {
            logCurrentPage = 1; // Reset to first page on filter change
            loadLogs();
        });
    }

    const refreshBtn = document.getElementById('monitoring-refresh-btn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', loadLogs);
    }

    const prevBtn = document.getElementById('logs-prev-btn');
    if (prevBtn) {
        prevBtn.addEventListener('click', prevLogPage);
    }

    const nextBtn = document.getElementById('logs-next-btn');
    if (nextBtn) {
        nextBtn.addEventListener('click', nextLogPage);
    }

    const loggingToggle = document.getElementById('monitoring-logging-toggle');
    if (loggingToggle) {
        loggingToggle.addEventListener('change', toggleLoggingConfig);
    }

    console.log('Monitoring module initialized');
}

async function renderMonitoring() {
    // Load initial data
    await loadMetrics();
    await loadLogs();
    await loadConfig();

    // Start auto-refresh (every 30 seconds)
    if (autoRefreshInterval) clearInterval(autoRefreshInterval);
    autoRefreshInterval = setInterval(async () => {
        await loadMetrics();
        await loadLogs();
    }, 30000);

    console.log('Monitoring dashboard rendered; auto-refresh started');
}

async function loadMetrics() {
    try {
        const response = await fetch('/api/v1/metrics?aggregation_window=300');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);

        const data = await response.json();
        setState({ metrics: data.metrics || [] });

        // Render metric cards
        const container = document.getElementById('metrics-cards');
        if (!container) return;

        container.innerHTML = '';
        (data.metrics || []).forEach(metric => {
            const card = createMetricCard(metric);
            container.appendChild(card);
        });
    } catch (err) {
        console.error('Failed to load metrics:', err);
    }
}

function createMetricCard(metric) {
    const div = document.createElement('div');
    div.className = 'metric-card';

    const latestDatapoint = metric.datapoints?.[metric.datapoints.length - 1];
    const value = latestDatapoint?.value ?? 'N/A';
    const timestamp = latestDatapoint?.timestamp ?? 'unknown';

    div.innerHTML = `
        <div class="metric-card-label">${metric.name}</div>
        <div class="metric-card-value">${formatValue(value)}</div>
        <div class="metric-card-unit">${metric.type || ''}</div>
        <div class="metric-footer">Updated ${formatTimestamp(timestamp)}</div>
    `;

    return div;
}

function formatValue(value) {
    if (typeof value !== 'number') return value;
    if (value >= 1000000) return (value / 1000000).toFixed(2) + 'M';
    if (value >= 1000) return (value / 1000).toFixed(2) + 'K';
    return value.toFixed(2);
}

function formatTimestamp(ts) {
    const now = new Date();
    const then = new Date(ts);
    const diff = Math.floor((now - then) / 1000);

    if (diff < 60) return 'just now';
    if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
    if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
    return Math.floor(diff / 86400) + 'd ago';
}

async function loadLogs() {
    try {
        const level = document.getElementById('monitoring-level-filter')?.value || '';

        const params = new URLSearchParams({
            limit: 50,
            offset: (logCurrentPage - 1) * 50,
        });
        if (level) params.append('level', level);

        const response = await fetch(`/api/v1/logs?${params}`);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);

        const data = await response.json();
        setState({ logs: data.logs || [] });

        // Render log table
        const tbody = document.getElementById('logs-tbody');
        if (!tbody) return;

        tbody.innerHTML = '';
        if (data.logs && data.logs.length > 0) {
            data.logs.forEach(log => {
                const row = document.createElement('tr');
                row.className = `log-level-${log.level.toLowerCase()}`;
                row.innerHTML = `
                    <td class="log-timestamp">${new Date(log.timestamp).toLocaleString()}</td>
                    <td><span class="log-level log-level-${log.level.toLowerCase()}">${log.level}</span></td>
                    <td class="log-message" title="${log.message}">${truncate(log.message, 80)}</td>
                `;
                tbody.appendChild(row);
            });
        } else {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="3" style="text-align: center; color: var(--text-muted);">No logs found</td>';
            tbody.appendChild(row);
        }

        // Update pagination info
        // Backend doesn't return total, so estimate from data length
        logTotalPages = Math.max(1, Math.ceil((data.total || data.logs?.length || 50) / 50));
        const pageInfo = document.getElementById('logs-page-info');
        if (pageInfo) {
            pageInfo.textContent = `Page ${logCurrentPage} of ${logTotalPages}`;
        }

        // Update button states
        const prevBtn = document.getElementById('logs-prev-btn');
        const nextBtn = document.getElementById('logs-next-btn');
        if (prevBtn) prevBtn.disabled = logCurrentPage <= 1;
        if (nextBtn) nextBtn.disabled = logCurrentPage >= logTotalPages;

    } catch (err) {
        console.error('Failed to load logs:', err);
    }
}

function truncate(str, len) {
    return str.length > len ? str.substring(0, len) + '…' : str;
}

function prevLogPage() {
    if (logCurrentPage > 1) {
        logCurrentPage--;
        loadLogs();
    }
}

function nextLogPage() {
    if (logCurrentPage < logTotalPages) {
        logCurrentPage++;
        loadLogs();
    }
}

async function loadConfig() {
    try {
        const response = await fetch('/api/v1/monitoring/config');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);

        const config = await response.json();
        setState({ monitoringConfig: config });

        // Update UI
        const loggingToggle = document.getElementById('monitoring-logging-toggle');
        if (loggingToggle) loggingToggle.checked = config.log_enabled;

    } catch (err) {
        console.error('Failed to load config:', err);
    }
}

async function toggleLoggingConfig() {
    const toggle = document.getElementById('monitoring-logging-toggle');
    const currentConfig = getState('monitoringConfig') || {};

    try {
        const response = await fetch('/api/v1/monitoring/config', {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                log_enabled: toggle.checked,
                metrics_enabled: currentConfig.metrics_enabled !== undefined ? currentConfig.metrics_enabled : true,
                retention_days: currentConfig.retention_days || 7,
            }),
        });

        if (!response.ok) {
            const errText = await response.text();
            throw new Error(`HTTP ${response.status}: ${errText}`);
        }

        const updated = await response.json();
        setState({ monitoringConfig: updated });
        console.log('Config updated:', updated);

    } catch (err) {
        console.error('Failed to save config:', err);
        // Revert toggle on error
        toggle.checked = !toggle.checked;
    }
}

// Called when leaving the monitoring tab
export function cleanup() {
    if (autoRefreshInterval) {
        clearInterval(autoRefreshInterval);
        autoRefreshInterval = null;
    }
    console.log('Monitoring module cleaned up');
}
