/* Settings module — card-based load/save with per-card Save buttons */

import { req } from '../api.js';
import { showToast } from '../toast.js';
import { registerTab } from '../router.js';

async function loadConfig() {
    const entries = await req('GET', '/api/v1/config');
    const map = {};
    for (const { key, value } of entries) map[key] = value;
    return map;
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
    updateBadge(cardEl, configMap);
}

function updateBadge(cardEl, configMap) {
    const badge = cardEl.querySelector('[data-status]');
    if (!badge) return;
    const keys = [...cardEl.querySelectorAll('[name]')].map(el => el.name);
    const ok = keys.some(k => {
        const v = configMap[k];
        // '****' = sensitive value stored in DB → counts as configured.
        // 'false'/'0'/'' = not set or disabled → not configured.
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
            const map = await loadConfig();
            document.querySelectorAll('[data-connector]').forEach(c => updateBadge(c, map));
        } finally {
            btn.disabled = false;
        }
    });
}

function setupProviderSelector(configMap) {
    const current = configMap['llm.provider'] || 'claude';
    document.querySelectorAll('input[name="llm.provider"]').forEach(radio => {
        // Clone to clear any stale listeners from previous tab activations
        const fresh = radio.cloneNode(true);
        fresh.checked = (fresh.value === current);
        radio.replaceWith(fresh);
        fresh.addEventListener('change', async () => {
            if (!fresh.checked) return;
            try {
                await req('PUT', '/api/v1/config', { key: 'llm.provider', value: fresh.value });
                showToast(`Active provider: ${fresh.value}`, 'success');
                const map = await loadConfig();
                document.querySelectorAll('[data-connector]').forEach(c => updateBadge(c, map));
            } catch (err) {
                showToast(err.message, 'error');
            }
        });
    });
}

export function init() {
    setupEyeToggles();
    setupSaveButtons();

    registerTab('settings', async () => {
        try {
            const map = await loadConfig();
            document.querySelectorAll('[data-connector]').forEach(c => populateCard(c, map));
            setupProviderSelector(map);
        } catch (err) {
            showToast(`Failed to load settings: ${err.message}`, 'error');
        }
    });
}
