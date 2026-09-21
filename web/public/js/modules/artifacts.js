/* Artifacts module — Static HTML & Document Artifacts management */

import { req } from '../api.js';
import { showToast } from '../toast.js';
import { registerTab } from '../router.js';

let artifacts = [];
let searchQuery = '';

/**
 * Format bytes into human readable string.
 */
function formatBytes(bytes) {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

/**
 * Format relative time or readable date.
 */
function formatDate(isoDate) {
    if (!isoDate) return '';
    const date = new Date(isoDate);
    const now = new Date();
    const diffSec = Math.floor((now - date) / 1000);

    if (diffSec < 60) return 'Just now';
    if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
    if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
    return date.toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' });
}

/**
 * Pick icon and CSS modifier for file type.
 */
function getFileIcon(filename) {
    const ext = filename.split('.').pop().toLowerCase();
    switch (ext) {
        case 'html':
        case 'htm':
            return { icon: 'fab fa-html5', modifier: 'artifact-card__icon--html' };
        case 'json':
            return { icon: 'fas fa-code', modifier: 'artifact-card__icon--json' };
        case 'css':
            return { icon: 'fab fa-css3-alt', modifier: 'artifact-card__icon--css' };
        case 'png':
        case 'jpg':
        case 'jpeg':
        case 'gif':
        case 'svg':
        case 'webp':
            return { icon: 'fas fa-file-image', modifier: 'artifact-card__icon--text' };
        default:
            return { icon: 'fas fa-file-alt', modifier: 'artifact-card__icon--text' };
    }
}

/**
 * Load artifacts list from the API.
 */
export async function loadArtifacts() {
    try {
        artifacts = await req('GET', '/api/v1/artifacts');
        renderArtifacts();
    } catch (err) {
        showToast(`Failed to load artifacts: ${err.message}`, 'error');
    }
}

/**
 * Filter artifacts by search query.
 */
function getFilteredArtifacts() {
    if (!searchQuery) return artifacts;
    const q = searchQuery.toLowerCase();
    return artifacts.filter(item => item.filename.toLowerCase().includes(q));
}

/**
 * Render the artifacts grid or empty state.
 */
function renderArtifacts() {
    const list = document.getElementById('artifacts-list');
    const emptyState = document.getElementById('artifacts-empty');
    if (!list) return;

    const filtered = getFilteredArtifacts();

    if (!filtered || filtered.length === 0) {
        list.innerHTML = '';
        if (emptyState) {
            emptyState.hidden = false;
            const emptyTitle = emptyState.querySelector('h3');
            const emptyDesc = emptyState.querySelector('p');
            if (searchQuery) {
                if (emptyTitle) emptyTitle.textContent = 'No matching artifacts found';
                if (emptyDesc) emptyDesc.textContent = `No artifacts matched "${searchQuery}".`;
            } else {
                if (emptyTitle) emptyTitle.textContent = 'No artifacts yet';
                if (emptyDesc) emptyDesc.textContent = 'Ask Bruce in chat to research a topic and generate an HTML report or document artifact.';
            }
        }
        return;
    }

    if (emptyState) emptyState.hidden = true;

    list.innerHTML = filtered.map(item => {
        const { icon, modifier } = getFileIcon(item.filename);
        const sizeStr = formatBytes(item.size_bytes ?? item.size);
        const dateStr = formatDate(item.updated_at || item.modified_at);
        const targetUrl = item.url || item.relative_url || `/artifacts/${item.filename}`;
        const escapedName = escapeHtml(item.filename);

        return `
            <div class="artifact-card" data-filename="${escapedName}">
                <div class="artifact-card__header">
                    <div class="artifact-card__icon ${modifier}">
                        <i class="${icon}"></i>
                    </div>
                    <div class="artifact-card__meta">
                        <span class="artifact-card__name" title="${escapedName}">${escapedName}</span>
                        <div class="artifact-card__details">
                            <span class="artifact-card__badge">${sizeStr}</span>
                            <span>${dateStr}</span>
                        </div>
                    </div>
                </div>

                <div class="artifact-card__url" title="${escapeHtml(targetUrl)}">
                    <i class="fas fa-link"></i>
                    <span>${escapeHtml(targetUrl)}</span>
                </div>

                <div class="artifact-card__actions">
                    <a href="${escapeHtml(targetUrl)}" target="_blank" rel="noopener noreferrer" class="btn-primary btn-sm btn-open">
                        <i class="fas fa-external-link-alt"></i> Open
                    </a>
                    <button type="button" class="btn-secondary btn-sm btn-copy" data-url="${escapeHtml(targetUrl)}">
                        <i class="fas fa-copy"></i> Copy Link
                    </button>
                    <button type="button" class="btn-danger-ghost btn-delete" title="Delete artifact" data-filename="${escapedName}">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            </div>
        `;
    }).join('');
}

/**
 * Wire event listeners for copy, open, delete, search, and refresh.
 */
function setupEventListeners() {
    const searchInput = document.getElementById('artifacts-search');
    if (searchInput) {
        searchInput.addEventListener('input', e => {
            searchQuery = e.target.value.trim();
            renderArtifacts();
        });
    }

    const refreshBtn = document.getElementById('artifacts-refresh-btn');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', async () => {
            const icon = refreshBtn.querySelector('i');
            if (icon) icon.classList.add('fa-spin');
            await loadArtifacts();
            if (icon) icon.classList.remove('fa-spin');
            showToast('Artifacts refreshed', 'info');
        });
    }

    const list = document.getElementById('artifacts-list');
    if (list) {
        list.addEventListener('click', async e => {
            // Copy link button
            const copyBtn = e.target.closest('.btn-copy');
            if (copyBtn) {
                const rawUrl = copyBtn.dataset.url;
                let fullUrl = rawUrl;
                if (rawUrl.startsWith('/')) {
                    fullUrl = `${window.location.origin}${rawUrl}`;
                }
                try {
                    await navigator.clipboard.writeText(fullUrl);
                    showToast('Artifact link copied to clipboard', 'success');
                } catch {
                    showToast('Failed to copy link', 'error');
                }
                return;
            }

            // Delete button
            const deleteBtn = e.target.closest('.btn-delete');
            if (deleteBtn) {
                const filename = deleteBtn.dataset.filename;
                if (!confirm(`Are you sure you want to delete "${filename}"?`)) {
                    return;
                }
                try {
                    await req('DELETE', `/api/v1/artifacts/${encodeURIComponent(filename)}`);
                    showToast(`Deleted ${filename}`, 'success');
                    await loadArtifacts();
                } catch (err) {
                    showToast(`Failed to delete artifact: ${err.message}`, 'error');
                }
            }
        });
    }
}

function escapeHtml(str) {
    if (!str) return '';
    return str
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}

/**
 * Initialize artifacts module.
 */
export function init() {
    setupEventListeners();

    registerTab('artifacts', () => {
        loadArtifacts();
    });
}
