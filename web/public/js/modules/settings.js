/* Settings module — unified card-based configuration with category filtering & live connector status */

import { req } from '../api.js';
import { showToast } from '../toast.js';
import { registerTab } from '../router.js';

let pollInterval = null;
let currentCategory = 'all';

async function loadConfig() {
    const entries = await req('GET', '/api/v1/config');
    const map = {};
    for (const { key, value } of entries) map[key] = value;
    return map;
}

async function loadConnectors() {
    try {
        return await req('GET', '/api/v1/connectors');
    } catch {
        return [];
    }
}

function populateCard(cardEl, configMap) {
    cardEl.querySelectorAll('[name]').forEach(input => {
        const v = configMap[input.name];
        if (v === undefined) return;
        if (input.type === 'checkbox') {
            input.checked = (v === 'true');
        } else if (input.type === 'radio') {
            input.checked = (input.value === v);
        } else {
            // Leave blank for masked values — placeholder signals "already set"
            input.value = (v === '****') ? '' : (v || '');
        }
    });
}

function updateBadge(cardEl, configMap, connectorsList = []) {
    const badge = cardEl.querySelector('[data-status]');
    if (!badge) return;

    const connectorType = cardEl.dataset.connector;

    // Connectors have live runtime status
    if (['discord', 'whatsapp', 'telegram'].includes(connectorType)) {
        const connector = connectorsList.find(c => c.type === connectorType);
        const status = connector?.status || (cardEl.querySelector('input[type="checkbox"]')?.checked ? 'connected' : 'disabled');

        badge.textContent = status.charAt(0).toUpperCase() + status.slice(1).replace('_', ' ');
        badge.className = `badge badge--${status}`;

        // WhatsApp QR notice
        if (connectorType === 'whatsapp') {
            const qrNotice = cardEl.querySelector('[data-whatsapp-qr]');
            if (qrNotice) {
                qrNotice.hidden = (status !== 'needs_qr');
            }
        }
        return;
    }

    // Standard configuration badge
    const keys = [...cardEl.querySelectorAll('[name]')].map(el => el.name);
    const ok = keys.some(k => {
        const v = configMap[k];
        return v && v !== '' && v !== 'false' && v !== '0';
    });
    badge.textContent = ok ? 'Configured' : 'Not configured';
    badge.className = `badge ${ok ? 'badge--configured' : 'badge--empty'}`;
}

async function saveCard(cardEl) {
    const inputs = cardEl.querySelectorAll('[name]');
    let saved = 0;
    const errors = [];

    for (const input of inputs) {
        let value;
        if (input.type === 'checkbox') {
            value = input.checked ? 'true' : 'false';
        } else if (input.type === 'radio') {
            if (!input.checked) continue;
            value = input.value;
        } else {
            value = input.value.trim();
            if (!value) continue; // skip blanks
        }

        if (input.type === 'number') {
            const n = parseInt(value, 10);
            if (isNaN(n) || n <= 0) continue;
            value = String(n);
        }

        try {
            await req('PUT', '/api/v1/config', { key: input.name, value });
            saved++;
        } catch (err) {
            errors.push(`${input.name}: ${err.message}`);
        }
    }

    if (errors.length) {
        showToast(`Errors: ${errors.join(', ')}`, 'error');
    } else if (saved > 0) {
        showToast('Saved', 'success');
    } else {
        showToast('No changes', 'info');
    }
}

function setupEyeToggles() {
    document.getElementById('tab-settings').addEventListener('click', e => {
        const btn = e.target.closest('.btn-eye');
        if (!btn) return;
        const input = btn.closest('.field-password-wrap').querySelector('input');
        const isPassword = input.type === 'password';
        input.type = isPassword ? 'text' : 'password';
        btn.querySelector('i').className = isPassword ? 'fas fa-eye-slash' : 'fas fa-eye';
    });
}

function setupSaveButtons() {
    document.getElementById('tab-settings').addEventListener('click', async e => {
        const btn = e.target.closest('[data-save-btn]');
        if (!btn) return;
        const card = btn.closest('[data-connector]');
        if (!card) return;
        btn.disabled = true;
        try {
            await saveCard(card);
            await refreshView();
        } finally {
            btn.disabled = false;
        }
    });
}

function setupProviderSelector(configMap) {
    const current = configMap['llm.provider'] || 'claude';
    document.querySelectorAll('input[name="llm.provider"]').forEach(radio => {
        const fresh = radio.cloneNode(true);
        fresh.checked = (fresh.value === current);
        radio.replaceWith(fresh);
        fresh.addEventListener('change', async () => {
            if (!fresh.checked) return;
            try {
                await req('PUT', '/api/v1/config', { key: 'llm.provider', value: fresh.value });
                showToast(`Active provider: ${fresh.value}`, 'success');
                await refreshView();
            } catch (err) {
                showToast(err.message, 'error');
            }
        });
    });
}

function setCategory(category) {
    currentCategory = category || 'all';

    // Update active nav button
    document.querySelectorAll('.settings-nav-btn').forEach(btn => {
        const isActive = btn.dataset.settingsCat === currentCategory;
        btn.classList.toggle('settings-nav-btn--active', isActive);
        btn.setAttribute('aria-selected', isActive ? 'true' : 'false');
    });

    // Filter section visibility
    document.querySelectorAll('.settings-section').forEach(section => {
        const secCat = section.dataset.settingsSection;
        if (currentCategory === 'all' || secCat === currentCategory) {
            section.hidden = false;
        } else {
            section.hidden = true;
        }
    });
}

function setupCategoryFilter() {
    const nav = document.querySelector('.settings-nav');
    if (!nav) return;

    nav.addEventListener('click', e => {
        const btn = e.target.closest('.settings-nav-btn');
        if (!btn) return;
        const cat = btn.dataset.settingsCat;
        if (cat) setCategory(cat);
    });
}

async function refreshView() {
    const [map, connectors] = await Promise.all([
        loadConfig(),
        loadConnectors(),
    ]);

    document.querySelectorAll('[data-connector]').forEach(c => {
        populateCard(c, map);
        updateBadge(c, map, connectors);
    });
    setupProviderSelector(map);
}

function startPolling() {
    stopPolling();
    pollInterval = setInterval(async () => {
        const connectors = await loadConnectors();
        document.querySelectorAll('[data-connector]').forEach(c => {
            if (['discord', 'whatsapp', 'telegram'].includes(c.dataset.connector)) {
                updateBadge(c, {}, connectors);
            }
        });
    }, 10000);
}

function stopPolling() {
    if (pollInterval) {
        clearInterval(pollInterval);
        pollInterval = null;
    }
}

export function init() {
    setupEyeToggles();
    setupSaveButtons();
    setupCategoryFilter();

    registerTab('settings', async (subRoute) => {
        try {
            if (subRoute === 'connectors') {
                setCategory('connectors');
            } else if (!subRoute) {
                setCategory(currentCategory);
            }
            await refreshView();
            startPolling();
        } catch (err) {
            showToast(`Failed to load settings: ${err.message}`, 'error');
        }
    }, () => {
        stopPolling();
    });
}
