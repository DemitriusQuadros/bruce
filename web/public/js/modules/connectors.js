/* Connectors module */

import { req } from '../api.js';
import { showToast } from '../toast.js';
import { setState, getState } from '../store.js';
import { registerTab, currentTab } from '../router.js';

const CONNECTOR_INFO = {
    whatsapp: {
        label: 'WhatsApp',
        description: 'Connect via WhatsApp messenger',
        icon: 'fa-brands fa-whatsapp',
    },
    discord: {
        label: 'Discord',
        description: 'Connect via Discord bot',
        icon: 'fa-brands fa-discord',
    },
};

let pollInterval = null;

async function loadConnectors() {
    try {
        const connectorsData = await req('GET', '/api/v1/connectors');
        setState({ connectors: connectorsData });
        renderConnectors();
    } catch (err) {
        showToast(`Failed to load connectors: ${err.message}`, 'error');
    }
}

function renderConnectors() {
    const list = document.getElementById('connectors-list');
    const connectorsData = getState('connectors');
    list.innerHTML = '';

    connectorsData.forEach(connector => {
        const card = document.createElement('div');
        card.className = 'card connector-card';

        const type = connector.type || 'unknown';
        const info = CONNECTOR_INFO[type] || { label: type, description: '', icon: '⚙️' };

        let status = connector.status || 'disabled';
        let statusClass = `badge--${status}`;

        const infoDiv = document.createElement('div');
        infoDiv.className = 'connector-info';
        const iconHtml = info.icon.startsWith('fa-')
            ? `<i class="fas ${info.icon}"></i>`
            : `<span class="connector-icon">${info.icon}</span>`;
        infoDiv.innerHTML = `
            <div class="connector-header">
                ${iconHtml}
                <div>
                    <h3>${info.label}</h3>
                    <p class="connector-desc">${info.description}</p>
                </div>
            </div>
            <span class="badge ${statusClass}">${status}</span>
        `;

        const controls = document.createElement('div');
        controls.className = 'connector-controls';

        if (type === 'discord') {
            const input = document.createElement('input');
            input.type = 'password';
            input.placeholder = 'Bot token';
            input.value = connector.config?.bot_token || '';

            const button = document.createElement('button');
            button.className = 'btn-primary';
            button.textContent = 'Save';
            button.addEventListener('click', async () => {
                await saveDiscordToken(input.value);
            });

            controls.appendChild(input);
            controls.appendChild(button);
        } else if (type === 'whatsapp') {
            const status = document.createElement('div');
            status.className = 'connector-status-msg';
            status.textContent = connector.status === 'needs_qr'
                ? 'Scan QR code in logs to connect'
                : 'WhatsApp configuration in settings';
            controls.appendChild(status);
        }

        card.appendChild(infoDiv);
        card.appendChild(controls);
        list.appendChild(card);
    });
}

async function saveDiscordToken(token) {
    try {
        await req('PUT', '/api/v1/config', {
            connectors: {
                discord: {
                    bot_token: token,
                },
            },
        });
        showToast('Discord token saved', 'success');
        await loadConnectors();
    } catch (err) {
        showToast(`Failed to save token: ${err.message}`, 'error');
    }
}

function startPolling() {
    if (pollInterval) clearInterval(pollInterval);

    pollInterval = setInterval(() => {
        if (currentTab() === 'connectors') {
            loadConnectors();
        }
    }, 10000);
}

function stopPolling() {
    if (pollInterval) {
        clearInterval(pollInterval);
        pollInterval = null;
    }
}

export function init() {
    // Load connectors on init
    loadConnectors();

    // Register tab activation callback
    registerTab('connectors', () => {
        loadConnectors();
        startPolling();
    });

    // Stop polling when leaving tab
    window.addEventListener('store:change', (e) => {
        if (e.detail.key === 'currentTab' && e.detail.value !== 'connectors') {
            stopPolling();
        }
    });
}
