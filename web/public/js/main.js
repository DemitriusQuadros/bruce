/* Bootstrap application */

import { start as startRouter } from './router.js';
import { init as initConnectors } from './modules/connectors.js';
import { init as initSessions } from './modules/sessions.js';
import { init as initLogs } from './modules/logs.js';
import { init as initSettings } from './modules/settings.js';

document.addEventListener('DOMContentLoaded', async () => {
    try {
        // Initialize all modules synchronously to set up event listeners
        initConnectors();
        initSessions();
        initLogs();
        initSettings();

        // Allow async operations (API calls) to complete
        // Sessions must load so Logs dropdown can populate
        await new Promise(resolve => setTimeout(resolve, 0));

        // Start router last — triggers initial tab activation
        startRouter();
    } catch (err) {
        console.error('Failed to initialize:', err);
    }
});
