/* Sessions module */

import { req } from '../api.js';
import { showToast } from '../toast.js';
import { setState, getState, subscribe } from '../store.js';
import { registerTab } from '../router.js';

async function loadSessions() {
    try {
        const sessionsData = await req('GET', '/api/v1/sessions');
        setState({ sessions: sessionsData });
        renderSessions();
    } catch (err) {
        showToast(`Failed to load sessions: ${err.message}`, 'error');
    }
}

function renderSessions() {
    const list = document.getElementById('sessions-list');
    const sessionsData = getState('sessions') || [];
    const selectedSessionId = getState('selectedSessionId');
    list.innerHTML = '';

    sessionsData.forEach(session => {
        const row = document.createElement('div');
        row.className = 'session-row';
        // API returns 'is_active', not 'active'
        if (!session.is_active) {
            row.classList.add('paused');
        }
        if (session.id === selectedSessionId) {
            row.classList.add('is-selected');
        }

        const timestamp = new Date(session.created_at).toLocaleDateString();
        row.innerHTML = `
            <h4>${session.id}</h4>
            <div class="session-meta">Created: ${timestamp}</div>
            <div class="session-meta">Connector: ${session.connector_type}</div>
        `;

        row.addEventListener('click', () => loadSessionDetail(session.id));
        list.appendChild(row);
    });
}

async function loadSessionDetail(sessionId) {
    try {
        const session = await req('GET', `/api/v1/sessions/${sessionId}`);

        setState({ selectedSessionId: sessionId });

        document.getElementById('session-prompt').value = session.system_prompt || '';
        // API returns 'is_active', not 'active'
        document.getElementById('session-active').checked = session.is_active || false;
        document.getElementById('session-detail').hidden = false;

        // Render to update selected styling
        renderSessions();
    } catch (err) {
        showToast(`Failed to load session: ${err.message}`, 'error');
    }
}

function setupSaveHandler() {
    const saveBtn = document.getElementById('session-save');
    if (saveBtn) {
        saveBtn.addEventListener('click', async () => {
            const selectedSessionId = getState('selectedSessionId');
            if (!selectedSessionId) return;

            try {
                const prompt = document.getElementById('session-prompt').value;
                const active = document.getElementById('session-active').checked;

                await req('PATCH', `/api/v1/sessions/${selectedSessionId}`, {
                    system_prompt: prompt,
                    active: active,
                });

                showToast('Session saved', 'success');
                await loadSessions();
            } catch (err) {
                showToast(`Failed to save session: ${err.message}`, 'error');
            }
        });
    }
}

export function init() {
    // Load sessions on init
    loadSessions();

    // Register tab activation callback
    registerTab('sessions', () => {
        loadSessions();
    });

    // Subscribe to sessions changes to re-render
    subscribe('sessions', () => {
        renderSessions();
    });

    // Setup save handler
    setupSaveHandler();
}
