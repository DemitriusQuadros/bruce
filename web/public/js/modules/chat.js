/* Chat module — web chat interface */

import { req } from '../api.js';
import { registerTab } from '../router.js';
import { setState, getState } from '../store.js';
import { showToast } from '../toast.js';

let activeChatId = null;

export function init() {
    registerTab('chat', onActivate);

    document.getElementById('chat-new-btn').addEventListener('click', createSession);
    document.getElementById('chat-send-btn').addEventListener('click', sendMessage);
    document.getElementById('chat-delete-btn').addEventListener('click', deleteSession);

    document.getElementById('chat-memory-badge')?.addEventListener('click', () => {
        const banner = document.getElementById('chat-memory-banner');
        if (banner) banner.hidden = !banner.hidden;
    });

    document.getElementById('chat-memory-close-btn')?.addEventListener('click', () => {
        const banner = document.getElementById('chat-memory-banner');
        if (banner) banner.hidden = true;
    });

    const input = document.getElementById('chat-input');
    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage();
        }
    });

    // Auto-resize textarea
    input.addEventListener('input', () => {
        input.style.height = 'auto';
        input.style.height = Math.min(input.scrollHeight, 150) + 'px';
    });
}

function onActivate() {
    loadSessions();
}

async function loadSessions() {
    try {
        const sessions = await req('GET', '/api/v1/chat/sessions');
        setState({ chatSessions: sessions, sessions });
        renderSessionList(sessions);
    } catch (err) {
        showToast('Failed to load conversations', 'error');
    }
}

function renderSessionList(sessions) {
    const list = document.getElementById('chat-session-list');
    if (!sessions || sessions.length === 0) {
        list.innerHTML = '<div class="chat-empty-state"><p>No conversations yet</p></div>';
        return;
    }

    list.innerHTML = sessions.map(s => {
        const title = escapeHTML(s.title || 'New Conversation');
        const time = formatTime(s.updated_at);
        const activeClass = s.id === activeChatId ? ' active' : '';
        return `<div class="chat-session-item${activeClass}" data-id="${s.id}">
            <div class="chat-session-item-title">${title}</div>
            <div class="chat-session-item-time">${time}</div>
        </div>`;
    }).join('');

    list.querySelectorAll('.chat-session-item').forEach(item => {
        item.addEventListener('click', () => selectSession(item.dataset.id));
    });
}

async function createSession() {
    try {
        const session = await req('POST', '/api/v1/chat/sessions', {});
        activeChatId = session.id;
        setState({ activeChatId: session.id });
        await loadSessions();
        showChatArea(session, []);
    } catch (err) {
        showToast('Failed to create conversation', 'error');
    }
}

async function selectSession(id) {
    try {
        const data = await req('GET', `/api/v1/chat/sessions/${id}`);
        activeChatId = id;
        setState({ activeChatId: id });
        highlightSession(id);
        showChatArea(data.session, data.messages);
    } catch (err) {
        showToast('Failed to load conversation', 'error');
    }
}

async function deleteSession() {
    if (!activeChatId) return;
    try {
        await req('DELETE', `/api/v1/chat/sessions/${activeChatId}`);
        activeChatId = null;
        setState({ activeChatId: null });
        hideChatArea();
        await loadSessions();
        showToast('Conversation deleted');
    } catch (err) {
        showToast('Failed to delete conversation', 'error');
    }
}

function highlightSession(id) {
    document.querySelectorAll('.chat-session-item').forEach(item => {
        item.classList.toggle('active', item.dataset.id === id);
    });
}

function showChatArea(session, messages) {
    document.getElementById('chat-main-header').hidden = false;
    document.getElementById('chat-input-area').hidden = false;
    document.getElementById('chat-title').textContent = session.title || 'New Conversation';

    const badge = document.getElementById('chat-provider-badge');
    badge.hidden = true;

    // Load memory recap
    loadSessionMemory(session.id);

    const container = document.getElementById('chat-messages');
    if (!messages || messages.length === 0) {
        container.innerHTML = '<div class="chat-empty-state"><i class="fas fa-comment-dots"></i><p>Send a message to start the conversation</p></div>';
    } else {
        container.innerHTML = '';
        messages.forEach(msg => container.appendChild(renderMessage(msg)));
        scrollToBottom();
    }

    document.getElementById('chat-input').focus();
}

async function loadSessionMemory(sessionId) {
    const badge = document.getElementById('chat-memory-badge');
    const banner = document.getElementById('chat-memory-banner');
    const content = document.getElementById('chat-memory-content');
    if (!badge) return;

    badge.hidden = true;
    if (banner) banner.hidden = true;

    try {
        const data = await req('GET', `/api/v1/sessions/${sessionId}/summary`);
        if (data && data.summary) {
            badge.hidden = false;
            if (content) content.textContent = data.summary;
        }
    } catch (e) {
        // Silently ignore if no summary exists
    }
}

function hideChatArea() {
    document.getElementById('chat-main-header').hidden = true;
    document.getElementById('chat-input-area').hidden = true;
    const memoryBadge = document.getElementById('chat-memory-badge');
    if (memoryBadge) memoryBadge.hidden = true;
    const memoryBanner = document.getElementById('chat-memory-banner');
    if (memoryBanner) memoryBanner.hidden = true;
    document.getElementById('chat-messages').innerHTML =
        '<div class="chat-empty-state"><i class="fas fa-comments"></i><p>Select a conversation or start a new one</p></div>';
}

async function sendMessage() {
    const input = document.getElementById('chat-input');
    const content = input.value.trim();
    if (!content || !activeChatId) return;

    const sendBtn = document.getElementById('chat-send-btn');
    input.value = '';
    input.style.height = 'auto';
    sendBtn.disabled = true;

    // Optimistic user bubble
    const container = document.getElementById('chat-messages');
    // Clear empty state if present
    const emptyState = container.querySelector('.chat-empty-state');
    if (emptyState) emptyState.remove();

    const userBubble = renderMessage({ role: 'user', content });
    container.appendChild(userBubble);
    scrollToBottom();

    // Typing indicator
    const typing = document.createElement('div');
    typing.className = 'typing-indicator';
    typing.innerHTML = '<span></span><span></span><span></span>';
    container.appendChild(typing);
    scrollToBottom();

    try {
        const data = await req('POST', `/api/v1/chat/sessions/${activeChatId}/messages`, { content });

        typing.remove();

        // Assistant bubble with provider label
        const assistantBubble = renderMessage(data.message, data.provider);
        container.appendChild(assistantBubble);
        scrollToBottom();

        // Update provider badge
        if (data.provider) {
            const badge = document.getElementById('chat-provider-badge');
            badge.textContent = data.provider;
            badge.hidden = false;
        }

        // Update session title in sidebar if changed
        if (data.session && data.session.title) {
            document.getElementById('chat-title').textContent = data.session.title;
            const item = document.querySelector(`.chat-session-item[data-id="${activeChatId}"] .chat-session-item-title`);
            if (item) item.textContent = data.session.title;
        }
    } catch (err) {
        typing.remove();
        let msg = 'Failed to send message';
        if (err.status === 429) msg = 'Rate limited — please try again';
        else if (err.status === 503) msg = 'AI provider unavailable';
        showToast(msg, 'error');
    } finally {
        sendBtn.disabled = false;
        input.focus();
    }
}

function renderMessage(msg, provider) {
    const wrapper = document.createElement('div');

    const bubble = document.createElement('div');
    bubble.className = `chat-bubble chat-bubble--${msg.role}`;
    bubble.innerHTML = renderMarkdown(msg.content);
    wrapper.appendChild(bubble);

    if (msg.role === 'assistant' && provider) {
        const label = document.createElement('div');
        label.className = 'chat-provider-label';
        label.textContent = provider;
        wrapper.appendChild(label);
    }

    return wrapper;
}

function renderMarkdown(text) {
    let html = escapeHTML(text);
    // Code blocks: ```...```
    html = html.replace(/```(\w*)\n([\s\S]*?)```/g, '<pre><code>$2</code></pre>');
    // Inline code: `...`
    html = html.replace(/`([^`]+)`/g, '<code>$1</code>');
    // Bold: **...**
    html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
    // Italic: *...*
    html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');
    // Line breaks
    html = html.replace(/\n/g, '<br>');
    return html;
}

function escapeHTML(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

function scrollToBottom() {
    const container = document.getElementById('chat-messages');
    container.scrollTop = container.scrollHeight;
}

function formatTime(iso) {
    if (!iso) return '';
    const d = new Date(iso);
    const now = new Date();
    const diff = now - d;

    if (diff < 86400000) {
        return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }
    if (diff < 604800000) {
        return d.toLocaleDateString([], { weekday: 'short' });
    }
    return d.toLocaleDateString([], { month: 'short', day: 'numeric' });
}
