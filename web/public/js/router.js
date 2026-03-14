/* Hash-based router for tab navigation */

const tabCallbacks = {};
const cleanupCallbacks = {};
let previousTab = null;

/**
 * Register a tab's activation callback and optional cleanup callback
 */
export function registerTab(name, onActivate, onCleanup = null) {
    tabCallbacks[name] = onActivate;
    if (onCleanup) {
        cleanupCallbacks[name] = onCleanup;
    }
}

/**
 * Get the currently active tab name
 */
export function currentTab() {
    const hash = window.location.hash.slice(1) || 'chat';
    return hash;
}

/**
 * Navigate to a tab by name
 */
export function navigate(name) {
    window.location.hash = name;
}

/**
 * Activate a tab and hide others
 */
function activateTab(name) {
    // Call cleanup for previous tab if it exists
    if (previousTab && cleanupCallbacks[previousTab]) {
        cleanupCallbacks[previousTab]();
    }

    // Hide all sections
    document.querySelectorAll('.tab-section').forEach(section => {
        section.hidden = true;
    });

    // Remove active from all buttons
    document.querySelectorAll('.tab-button').forEach(button => {
        button.classList.remove('active');
    });

    // Show selected section
    const section = document.getElementById(`tab-${name}`);
    if (section) {
        section.hidden = false;
    }

    // Mark button active
    const button = document.querySelector(`[data-tab="${name}"]`);
    if (button) {
        button.classList.add('active');
    }

    // Call the tab's registered callback if it exists
    if (tabCallbacks[name]) {
        tabCallbacks[name]();
    }

    // Update previous tab tracker
    previousTab = name;
}

/**
 * Start the router — wire clicks and hash changes
 */
export function start() {
    // Wire tab button clicks
    document.querySelectorAll('.tab-button').forEach(button => {
        button.addEventListener('click', () => {
            navigate(button.dataset.tab);
        });
    });

    // Listen for hash changes
    window.addEventListener('hashchange', () => {
        activateTab(currentTab());
    });

    // Activate initial tab
    activateTab(currentTab());
}
