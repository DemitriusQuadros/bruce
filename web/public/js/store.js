/* Reactive Store — global state without circular imports */

const state = {
    connectors: [],
    sessions: [],
    selectedSessionId: null,
};

/**
 * Update state and dispatch events for changed keys
 */
export function setState(patch) {
    for (const [key, value] of Object.entries(patch)) {
        state[key] = value;
        // Dispatch custom event so listeners can react
        window.dispatchEvent(new CustomEvent('store:change', {
            detail: { key, value }
        }));
    }
}

/**
 * Subscribe to changes on a specific state key
 */
export function subscribe(key, callback) {
    window.addEventListener('store:change', (e) => {
        if (e.detail.key === key) {
            callback(e.detail.value);
        }
    });
}

/**
 * Get current state value for a key
 */
export function getState(key) {
    return state[key];
}
