/* Settings module */

import { req } from '../api.js';
import { showToast } from '../toast.js';
import { registerTab } from '../router.js';

async function loadSettings() {
    try {
        const config = await req('GET', '/api/v1/config');
        populateSettings(config);
    } catch (err) {
        showToast(`Failed to load settings: ${err.message}`, 'error');
    }
}

function populateSettings(config) {
    const form = document.getElementById('settings-form');

    // Flatten nested config for form inputs
    const flatConfig = flattenConfig(config);
    for (const [key, value] of Object.entries(flatConfig)) {
        const input = form.querySelector(`[name="${key}"]`);
        if (input) {
            input.value = value || '';
        }
    }
}

function flattenConfig(obj, prefix = '') {
    const result = {};
    for (const [key, value] of Object.entries(obj)) {
        const fullKey = prefix ? `${prefix}.${key}` : key;
        if (value !== null && typeof value === 'object' && !Array.isArray(value)) {
            Object.assign(result, flattenConfig(value, fullKey));
        } else {
            result[fullKey] = value;
        }
    }
    return result;
}

function unflattenConfig(flat) {
    const result = {};
    for (const [key, value] of Object.entries(flat)) {
        const parts = key.split('.');
        let current = result;
        for (let i = 0; i < parts.length - 1; i++) {
            if (!current[parts[i]]) {
                current[parts[i]] = {};
            }
            current = current[parts[i]];
        }
        current[parts[parts.length - 1]] = value;
    }
    return result;
}

function setupSubmitHandler() {
    const form = document.getElementById('settings-form');
    if (form) {
        form.addEventListener('submit', async (e) => {
            e.preventDefault();

            try {
                const formData = new FormData(e.target);
                const flat = Object.fromEntries(formData);

                // API expects individual {key, value} objects sent separately
                let savedCount = 0;
                for (const [key, value] of Object.entries(flat)) {
                    // Skip empty strings
                    if (value === '') continue;

                    let configValue = value;

                    // Convert and validate number fields
                    if (key === 'claude.max_tokens' || key === 'claude.context_window' || key === 'gemini.max_tokens' || key === 'openai.max_tokens') {
                        const num = parseInt(value, 10);
                        if (isNaN(num) || num <= 0) {
                            continue;  // Skip invalid numbers
                        }
                        configValue = String(num);
                    }

                    // Send individual PUT request for each key
                    // API expects: {"key": "...", "value": "..."}
                    await req('PUT', '/api/v1/config', {
                        key: key,
                        value: configValue
                    });
                    savedCount++;
                }

                if (savedCount > 0) {
                    showToast('Settings saved', 'success');
                    await loadSettings();  // Reload to verify
                } else {
                    showToast('No changes to save', 'info');
                }
            } catch (err) {
                showToast(`Failed to save settings: ${err.message}`, 'error');
            }
        });
    }
}

export function init() {
    // Load settings on init
    loadSettings();

    // Register tab activation callback
    registerTab('settings', () => {
        loadSettings();
    });

    // Setup submit handler
    setupSubmitHandler();
}
