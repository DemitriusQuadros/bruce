/* Hash-based router for tab navigation */

const tabCallbacks = {};

/**
 * Register a tab's activation callback
 */
export function registerTab(name, onActivate) {
    tabCallbacks[name] = onActivate;
}

/**
 * Get the currently active tab name
 */
export function currentTab() {
    const hash = window.location.hash.slice(1) || 'connectors';
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
