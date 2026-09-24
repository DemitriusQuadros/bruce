/* Bootstrap application */

import { start as startRouter } from './router.js';
import { init as initChat } from './modules/chat.js';
import { init as initSessions } from './modules/sessions.js';
import { init as initLogs } from './modules/logs.js';
import { init as initSettings } from './modules/settings.js';
import { init as initSchedules } from './modules/schedules.js';
import { init as initArtifacts } from './modules/artifacts.js';

document.addEventListener('DOMContentLoaded', async () => {
    try {
        // Initialize all modules synchronously to set up event listeners
        initChat();
        initSessions();
        initArtifacts();
        initSchedules();
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
