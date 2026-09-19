/* Schedules module — Scheduled Tasks & Ambient Watches management */

import { req } from '../api.js';
import { showToast } from '../toast.js';
import { registerTab, currentTab } from '../router.js';

let tasks = [];
let pollInterval = null;
let countdownTimer = null;
let currentFilter = 'all';

const CONNECTOR_ICONS = {
    whatsapp: 'fab fa-whatsapp',
    discord: 'fab fa-discord',
    telegram: 'fab fa-telegram',
    web: 'fas fa-globe',
};

/**
 * Format relative countdown string from target ISO date.
 */
function formatCountdown(isoDate, isActive) {
    if (!isActive) {
        return { text: 'Paused', isDue: false };
    }
    if (!isoDate) {
        return { text: 'Not scheduled', isDue: false };
    }

    const target = new Date(isoDate).getTime();
    const now = Date.now();
    const diffMs = target - now;

    if (diffMs <= 0) {
        return { text: 'Due now', isDue: true };
    }

    const diffSec = Math.floor(diffMs / 1000);
    const days = Math.floor(diffSec / 86400);
    const hours = Math.floor((diffSec % 86400) / 3600);
    const minutes = Math.floor((diffSec % 3600) / 60);
    const seconds = diffSec % 60;

    if (days > 0) {
        return { text: `in ${days}d ${hours}h`, isDue: false };
    }
    if (hours > 0) {
        return { text: `in ${hours}h ${minutes}m`, isDue: false };
    }
    if (minutes > 0) {
        return { text: `in ${minutes}m ${seconds}s`, isDue: false };
    }
    return { text: `in ${seconds}s`, isDue: false };
}

/**
 * Fetch proactive tasks from API.
 */
async function loadTasks() {
    try {
        tasks = await req('GET', '/api/v1/proactive-tasks');
        renderTasks();
    } catch (err) {
        showToast(`Failed to load schedules: ${err.message}`, 'error');
    }
}

/**
 * Filter tasks according to selected dropdown.
 */
function getFilteredTasks() {
    switch (currentFilter) {
        case 'active':
            return tasks.filter(t => t.is_active);
        case 'paused':
            return tasks.filter(t => !t.is_active);
        case 'cron':
            return tasks.filter(t => t.task_type === 'cron');
        case 'watch':
            return tasks.filter(t => t.task_type === 'watch');
        default:
            return tasks;
    }
}

/**
 * Render all tasks to DOM.
 */
function renderTasks() {
    const listEl = document.getElementById('schedules-list');
    const emptyEl = document.getElementById('schedules-empty');
    if (!listEl || !emptyEl) return;

    const filtered = getFilteredTasks();

    if (filtered.length === 0) {
        listEl.innerHTML = '';
        emptyEl.hidden = false;
        return;
    }

    emptyEl.hidden = true;
    listEl.innerHTML = '';

    filtered.forEach(task => {
        const card = document.createElement('div');
        card.className = `schedule-card ${task.is_active ? '' : 'is-paused'}`;
        card.dataset.id = task.id;

        const isCron = task.task_type === 'cron';
        const typeBadge = isCron
            ? '<span class="pill-badge pill-badge--cron"><i class="fas fa-clock"></i> Cron</span>'
            : '<span class="pill-badge pill-badge--watch"><i class="fas fa-eye"></i> Watch</span>';

        const statusBadge = task.is_active
            ? '<span class="pill-badge pill-badge--active">Active</span>'
            : '<span class="pill-badge pill-badge--paused">Paused</span>';

        const targetConn = task.target_connector || task.connector_type || 'web';
        const connIcon = CONNECTOR_ICONS[targetConn] || 'fas fa-arrow-right';
        const countdownInfo = formatCountdown(task.next_run_at, task.is_active);

        const scheduleHuman = isCron
            ? task.schedule_expr
            : `Every ${task.schedule_expr}m`;

        card.innerHTML = `
            <div>
                <div class="schedule-card__header">
                    <div class="schedule-card__title-wrap">
                        <h4 class="schedule-card__title" title="${escapeHtml(task.title)}">${escapeHtml(task.title)}</h4>
                        <div class="schedule-card__badges">
                            ${typeBadge}
                            ${statusBadge}
                        </div>
                    </div>
                </div>

                <div class="schedule-card__body">
                    <div class="schedule-card__prompt" title="Instruction Condition">
                        ${escapeHtml(task.prompt_condition)}
                    </div>

                    <div class="schedule-card__meta-grid">
                        <div class="schedule-card__meta-item">
                            <span class="schedule-card__meta-label">Schedule</span>
                            <span class="schedule-card__meta-value" title="${escapeHtml(scheduleHuman)}">
                                <i class="fas fa-calendar-alt"></i> ${escapeHtml(scheduleHuman)}
                            </span>
                        </div>
                        <div class="schedule-card__meta-item">
                            <span class="schedule-card__meta-label">Timezone</span>
                            <span class="schedule-card__meta-value" title="${escapeHtml(task.timezone || 'UTC')}">
                                <i class="fas fa-globe-americas"></i> ${escapeHtml(task.timezone || 'UTC')}
                            </span>
                        </div>
                        <div class="schedule-card__meta-item">
                            <span class="schedule-card__meta-label">Target Route</span>
                            <span class="schedule-card__meta-value" title="${escapeHtml(targetConn)}: ${escapeHtml(task.target_channel_id || '')}">
                                <i class="${connIcon}"></i> ${escapeHtml(task.target_channel_id || targetConn)}
                            </span>
                        </div>
                        <div class="schedule-card__meta-item">
                            <span class="schedule-card__meta-label">Last Run</span>
                            <span class="schedule-card__meta-value" title="${task.last_run_at ? new Date(task.last_run_at).toLocaleString() : 'Never'}">
                                <i class="fas fa-history"></i> ${task.last_run_at ? formatRelativeTime(task.last_run_at) : 'Never'}
                            </span>
                        </div>
                    </div>

                    <div class="schedule-card__countdown ${countdownInfo.isDue ? 'is-due' : ''}" data-next-run="${task.next_run_at || ''}" data-is-active="${task.is_active}">
                        <span class="schedule-card__countdown-label">
                            <i class="fas fa-hourglass-half"></i> Next Run:
                        </span>
                        <span class="schedule-card__countdown-time">${countdownInfo.text}</span>
                    </div>
                </div>
            </div>

            <div class="schedule-card__actions">
                <div class="schedule-card__actions-left">
                    <button type="button" class="btn-card-action btn-card-action--run" title="Run immediately now" data-action="run">
                        <i class="fas fa-play"></i> Run Now
                    </button>
                    <button type="button" class="btn-card-action" title="${task.is_active ? 'Pause task' : 'Resume task'}" data-action="toggle">
                        <i class="fas ${task.is_active ? 'fa-pause' : 'fa-play'}"></i> ${task.is_active ? 'Pause' : 'Resume'}
                    </button>
                </div>
                <button type="button" class="btn-card-action btn-card-action--delete" title="Delete task" data-action="delete">
                    <i class="fas fa-trash"></i>
                </button>
            </div>
        `;

        // Wire action button events
        const runBtn = card.querySelector('[data-action="run"]');
        const toggleBtn = card.querySelector('[data-action="toggle"]');
        const deleteBtn = card.querySelector('[data-action="delete"]');

        runBtn.addEventListener('click', () => handleRunNow(task));
        toggleBtn.addEventListener('click', () => handleToggleStatus(task));
        deleteBtn.addEventListener('click', () => handleDeleteTask(task));

        listEl.appendChild(card);
    });
}

/**
 * Handle immediate manual execution.
 */
async function handleRunNow(task) {
    try {
        await req('POST', `/api/v1/proactive-tasks/${task.id}/run`);
        showToast(`Test run enqueued for "${task.title}"`, 'success');
    } catch (err) {
        showToast(`Run failed: ${err.message}`, 'error');
    }
}

/**
 * Handle pausing / resuming a task.
 */
async function handleToggleStatus(task) {
    const newStatus = !task.is_active;
    try {
        await req('PATCH', `/api/v1/proactive-tasks/${task.id}`, { is_active: newStatus });
        task.is_active = newStatus;
        showToast(`Task ${newStatus ? 'resumed' : 'paused'}`, 'success');
        await loadTasks();
    } catch (err) {
        showToast(`Failed to update status: ${err.message}`, 'error');
    }
}

/**
 * Handle task deletion with confirmation.
 */
async function handleDeleteTask(task) {
    if (!window.confirm(`Are you sure you want to delete "${task.title}"?`)) {
        return;
    }

    try {
        await req('DELETE', `/api/v1/proactive-tasks/${task.id}`);
        showToast(`Task "${task.title}" deleted`, 'success');
        tasks = tasks.filter(t => t.id !== task.id);
        renderTasks();
    } catch (err) {
        showToast(`Failed to delete task: ${err.message}`, 'error');
    }
}

/**
 * Periodically update all countdowns in DOM without re-rendering whole cards.
 */
function tickCountdowns() {
    const countdownEls = document.querySelectorAll('.schedule-card__countdown');
    countdownEls.forEach(el => {
        const nextRun = el.dataset.nextRun;
        const isActive = el.dataset.isActive === 'true';
        const timeEl = el.querySelector('.schedule-card__countdown-time');
        if (!timeEl) return;

        const info = formatCountdown(nextRun, isActive);
        timeEl.textContent = info.text;
        if (info.isDue) {
            el.classList.add('is-due');
        } else {
            el.classList.remove('is-due');
        }
    });
}

/**
 * Setup modal dialog open/close and creation form.
 */
function setupModal() {
    const backdrop = document.getElementById('schedules-modal-backdrop');
    const newBtn = document.getElementById('schedules-new-btn');
    const emptyNewBtn = document.getElementById('schedules-empty-new-btn');
    const closeBtn = document.getElementById('schedules-modal-close-btn');
    const cancelBtn = document.getElementById('schedules-modal-cancel-btn');
    const form = document.getElementById('schedules-create-form');
    const typeSelect = document.getElementById('task-type');
    const scheduleLabel = document.getElementById('task-schedule-label');
    const scheduleInput = document.getElementById('task-schedule-expr');
    const scheduleHint = document.getElementById('task-schedule-hint');
    const timezoneInput = document.getElementById('task-timezone');

    if (!backdrop || !form) return;

    // Prefill local timezone if empty
    try {
        const localTz = Intl.DateTimeFormat().resolvedOptions().timeZone;
        if (timezoneInput && !timezoneInput.value) {
            timezoneInput.value = localTz || 'America/Sao_Paulo';
        }
    } catch (_) {}

    const openModal = () => {
        backdrop.hidden = false;
        document.getElementById('task-title').focus();
    };

    const closeModal = () => {
        backdrop.hidden = true;
        form.reset();
        // Reset type behavior
        updateTypeForm('cron');
        try {
            const localTz = Intl.DateTimeFormat().resolvedOptions().timeZone;
            if (timezoneInput) timezoneInput.value = localTz || 'America/Sao_Paulo';
        } catch (_) {}
    };

    if (newBtn) newBtn.addEventListener('click', openModal);
    if (emptyNewBtn) emptyNewBtn.addEventListener('click', openModal);
    if (closeBtn) closeBtn.addEventListener('click', closeModal);
    if (cancelBtn) cancelBtn.addEventListener('click', closeModal);

    backdrop.addEventListener('click', e => {
        if (e.target === backdrop) closeModal();
    });

    const updateTypeForm = type => {
        if (type === 'watch') {
            scheduleLabel.textContent = 'Poll Interval (Minutes)';
            scheduleInput.placeholder = '30';
            scheduleHint.textContent = 'Interval in minutes (minimum 5). e.g. 15 or 30';
        } else {
            scheduleLabel.textContent = 'Schedule Expression';
            scheduleInput.placeholder = '0 9 * * 1-5';
            scheduleHint.textContent = 'Cron expression (min hour dom mon dow), e.g. 0 9 * * 1-5';
        }
    };

    if (typeSelect) {
        typeSelect.addEventListener('change', e => {
            updateTypeForm(e.target.value);
        });
    }

    form.addEventListener('submit', async e => {
        e.preventDefault();

        const title = document.getElementById('task-title').value.trim();
        const taskType = document.getElementById('task-type').value;
        const scheduleExpr = document.getElementById('task-schedule-expr').value.trim();
        const targetConnector = document.getElementById('task-target-connector').value;
        const targetChannel = document.getElementById('task-target-channel').value.trim();
        const timezone = document.getElementById('task-timezone').value.trim();
        const prompt = document.getElementById('task-prompt').value.trim();
        const toolsRaw = document.getElementById('task-tools').value.trim();

        if (!title || !scheduleExpr || !prompt) {
            showToast('Please fill all required fields', 'error');
            return;
        }

        let targetTools = [];
        if (toolsRaw) {
            targetTools = toolsRaw.split(',').map(s => s.trim()).filter(Boolean);
        }

        const payload = {
            session_id: 'default',
            connector_type: targetConnector,
            channel_id: targetChannel,
            target_connector: targetConnector,
            target_channel_id: targetChannel,
            title,
            task_type: taskType,
            schedule_expr: scheduleExpr,
            timezone,
            prompt_condition: prompt,
            target_tools: targetTools,
        };

        try {
            await req('POST', '/api/v1/proactive-tasks', payload);
            showToast('Proactive task created', 'success');
            closeModal();
            await loadTasks();
        } catch (err) {
            showToast(`Failed to create task: ${err.message}`, 'error');
        }
    });
}

function setupFilters() {
    const filterSelect = document.getElementById('schedules-filter-status');
    if (!filterSelect) return;

    filterSelect.addEventListener('change', e => {
        currentFilter = e.target.value;
        renderTasks();
    });
}

function startTimers() {
    if (countdownTimer) clearInterval(countdownTimer);
    countdownTimer = setInterval(tickCountdowns, 1000);

    if (pollInterval) clearInterval(pollInterval);
    pollInterval = setInterval(() => {
        if (currentTab() === 'schedules') {
            loadTasks();
        }
    }, 15000);
}

function stopTimers() {
    if (countdownTimer) {
        clearInterval(countdownTimer);
        countdownTimer = null;
    }
    if (pollInterval) {
        clearInterval(pollInterval);
        pollInterval = null;
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

function formatRelativeTime(isoDate) {
    const date = new Date(isoDate);
    const now = new Date();
    const diffSec = Math.floor((now - date) / 1000);

    if (diffSec < 60) return 'Just now';
    if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
    if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
    return date.toLocaleDateString();
}

/**
 * Initialize schedules module.
 */
export function init() {
    setupModal();
    setupFilters();

    registerTab('schedules', () => {
        loadTasks();
        startTimers();
    }, () => {
        stopTimers();
    });
}
