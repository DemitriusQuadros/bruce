/* Logs module */

import { req } from '../api.js';
import { showToast } from '../toast.js';
import { getState, subscribe } from '../store.js';
import { registerTab } from '../router.js';

function populateLogsDropdown() {
    const select = document.getElementById('logs-session-select');
    if (!select) return;  // Element might not exist yet

    const sessionsData = getState('sessions') || [];
    select.innerHTML = '<option value="">Select a session</option>';

    if (sessionsData.length === 0) {
        return;  // No sessions yet, wait for subscription update
    }

    sessionsData.forEach(session => {
        const option = document.createElement('option');
        option.value = session.id;
        // Show session ID and connector type (message_count not in list endpoint)
        option.textContent = `${session.id} (${session.connector_type})`;
        select.appendChild(option);
    });
}

async function loadLogs(sessionId) {
    try {
        const messages = await req('GET', `/api/v1/sessions/${sessionId}/messages?limit=50`);
        renderLogs(messages || []);
    } catch (err) {
        showToast(`Failed to load logs: ${err.message}`, 'error');
    }
}

function renderLogs(messages) {
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
        // API returns 'timestamp', not 'created_at'
        const timeStr = msg.timestamp || msg.created_at;
        timestamp.textContent = new Date(timeStr).toLocaleString();

        div.appendChild(content);
        div.appendChild(timestamp);
        container.appendChild(div);
    });

    // Scroll to bottom
    container.scrollTop = container.scrollHeight;
}

function setupEventHandlers() {
    const refreshBtn = document.getElementById('logs-refresh');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', async () => {
            const sessionId = document.getElementById('logs-session-select').value;
            if (sessionId) {
                await loadLogs(sessionId);
            }
        });
    }

    const select = document.getElementById('logs-session-select');
    if (select) {
        select.addEventListener('change', async (e) => {
            if (e.target.value) {
                await loadLogs(e.target.value);
            }
        });
    }
}

export function init() {
    // Setup event handlers first
    setupEventHandlers();

    // Initial dropdown population from store
    populateLogsDropdown();

    // Register tab activation callback to refresh dropdown
    registerTab('logs', () => {
        populateLogsDropdown();
    });

    // Subscribe to sessions changes — populate dropdown whenever sessions update
    // This ensures dropdown updates when sessions load, even if logs tab was visited first
    subscribe('sessions', (sessionsData) => {
        if (sessionsData && sessionsData.length > 0) {
            populateLogsDropdown();
        }
    });
}
