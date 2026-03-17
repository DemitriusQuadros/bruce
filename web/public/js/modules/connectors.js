/* Connectors module — runtime status + configuration in one place */

import { req } from '../api.js';
import { showToast } from '../toast.js';
import { setState, getState } from '../store.js';
import { registerTab, currentTab } from '../router.js';

const CONNECTOR_META = {
    whatsapp: {
        label: 'WhatsApp',
        description: 'Connect via WhatsApp messenger',
        icon: 'fab fa-whatsapp',
    },
    discord: {
        label: 'Discord',
        description: 'Connect via Discord bot',
        icon: 'fab fa-discord',
    },
};

let pollInterval = null;

async function loadAll() {
    try {
        const [connectors, configEntries] = await Promise.all([
            req('GET', '/api/v1/connectors'),
            req('GET', '/api/v1/config'),
        ]);

        const configMap = {};
        for (const { key, value } of configEntries) configMap[key] = value;

        setState({ connectors, connectorConfig: configMap });
        renderConnectors();
    } catch (err) {
        showToast(`Failed to load connectors: ${err.message}`, 'error');
    }
}

function renderConnectors() {
    const list = document.getElementById('connectors-list');
    const connectors = getState('connectors') || [];
    const cfg = getState('connectorConfig') || {};
    list.innerHTML = '';

    connectors.forEach(connector => {
        const type = connector.type || 'unknown';
        const meta = CONNECTOR_META[type] || { label: type, description: '', icon: 'fas fa-plug' };
        const status = connector.status || 'disabled';

        const card = document.createElement('div');
        card.className = 'card connector-card';
        card.dataset.connectorType = type;

        // ── Header ──────────────────────────────────────────────
        const header = document.createElement('div');
        header.className = 'connector-header';
        header.innerHTML = `
            <div style="display:flex;align-items:center;gap:0.75rem;">
                <i class="${meta.icon}" style="font-size:1.4rem;color:var(--accent)"></i>
                <div>
                    <h3 style="margin:0">${meta.label}</h3>
                    <p class="connector-desc">${meta.description}</p>
                </div>
            </div>
            <span class="badge badge--${status}">${status}</span>
        `;

        // ── Enable toggle ────────────────────────────────────────
        const enableKey = `connectors.${type}.enabled`;
        const isEnabled = cfg[enableKey] === 'true';

        const toggleRow = document.createElement('div');
        toggleRow.className = 'connector-toggle-row';
        const toggleId = `connector-${type}-enabled`;
        toggleRow.innerHTML = `
            <label class="connector-toggle-label" for="${toggleId}">
                <input type="checkbox" id="${toggleId}" ${isEnabled ? 'checked' : ''} />
                Enable connector
            </label>
        `;
        toggleRow.querySelector('input').addEventListener('change', async e => {
            await saveConfigKey(enableKey, e.target.checked ? 'true' : 'false');
        });

        card.appendChild(header);
        card.appendChild(toggleRow);

        // ── Discord-specific: bot token ──────────────────────────
        if (type === 'discord') {
            const tokenKey = 'connectors.discord.bot_token';
            const hasToken = cfg[tokenKey] === '****';

            const tokenGroup = document.createElement('div');
            tokenGroup.className = 'form-group connector-token-group';
            tokenGroup.innerHTML = `
                <label>Bot Token</label>
                <div class="field-password-wrap">
                    <input type="password" placeholder="${hasToken ? '••••••••  (already set)' : 'Bot token...'}" />
                    <button type="button" class="btn-eye" tabindex="-1"><i class="fas fa-eye"></i></button>
                </div>
            `;

            const tokenInput = tokenGroup.querySelector('input');
            const eyeBtn = tokenGroup.querySelector('.btn-eye');

            eyeBtn.addEventListener('click', () => {
                const isPass = tokenInput.type === 'password';
                tokenInput.type = isPass ? 'text' : 'password';
                eyeBtn.querySelector('i').className = isPass ? 'fas fa-eye-slash' : 'fas fa-eye';
            });

            const tokenFooter = document.createElement('div');
            tokenFooter.className = 'connector-token-footer';
            tokenFooter.innerHTML = `<button type="button" class="btn-primary btn-sm">Save Token</button>`;
            const saveBtn = tokenFooter.querySelector('button');

            saveBtn.addEventListener('click', async () => {
                const val = tokenInput.value.trim();
                if (!val) { showToast('Token cannot be empty', 'error'); return; }
                saveBtn.disabled = true;
                await saveConfigKey(tokenKey, val);
                tokenInput.value = '';
                saveBtn.disabled = false;
                await loadAll();
            });

            card.appendChild(tokenGroup);
            card.appendChild(tokenFooter);
        }

        // ── WhatsApp-specific: QR hint ───────────────────────────
        if (type === 'whatsapp' && status === 'needs_qr') {
            const hint = document.createElement('p');
            hint.className = 'connector-desc';
            hint.style.marginTop = '0.5rem';
            hint.innerHTML = '<i class="fas fa-qrcode"></i> Scan the QR code shown in the logs to connect.';
            card.appendChild(hint);
        }

        list.appendChild(card);
    });
}

async function saveConfigKey(key, value) {
    try {
        await req('PUT', '/api/v1/config', { key, value });
        showToast('Saved', 'success');
    } catch (err) {
        showToast(`Failed to save: ${err.message}`, 'error');
    }
}

function startPolling() {
    if (pollInterval) clearInterval(pollInterval);
    pollInterval = setInterval(() => {
        if (currentTab() === 'connectors') loadAll();
    }, 10000);
}

function stopPolling() {
    if (pollInterval) { clearInterval(pollInterval); pollInterval = null; }
}

export function init() {
    loadAll();

    registerTab('connectors', () => {
        loadAll();
        startPolling();
    });

    window.addEventListener('store:change', e => {
        if (e.detail.key === 'currentTab' && e.detail.value !== 'connectors') {
            stopPolling();
        }
    });
}
