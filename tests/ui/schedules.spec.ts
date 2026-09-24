import { test, expect } from '@playwright/test';

test.describe('Schedules & Ambient Watches Dashboard', () => {
  test('should navigate to schedules tab and display toolbar', async ({ page }) => {
    await page.goto('/');

    // Click Schedules tab button
    const schedulesTab = page.locator('[data-tab="schedules"]');
    await expect(schedulesTab).toBeVisible();
    await schedulesTab.click();

    // Verify hash routing and tab activation
    await expect(page).toHaveURL('/#schedules');
    await expect(schedulesTab).toHaveClass(/active/);

    const schedulesSection = page.locator('#tab-schedules');
    await expect(schedulesSection).not.toHaveAttribute('hidden', '');

    // Verify toolbar elements
    await expect(page.locator('[data-testid="schedules-title"]')).toContainText('Scheduled Tasks & Ambient Watches');
    await expect(page.locator('[data-testid="schedules-filter"]')).toBeVisible();
    await expect(page.locator('[data-testid="new-task-btn"]')).toBeVisible();
  });

  test('should open, interact with, and close the New Task modal', async ({ page }) => {
    await page.goto('/#schedules');

    const modalBackdrop = page.locator('[data-testid="schedules-modal-backdrop"]');
    await expect(modalBackdrop).toHaveAttribute('hidden', '');

    // Open modal
    await page.locator('[data-testid="new-task-btn"]').click();
    await expect(modalBackdrop).not.toHaveAttribute('hidden', '');

    const modal = page.locator('[data-testid="schedules-modal"]');
    await expect(modal).toBeVisible();

    // Title input should be present and focused
    const titleInput = page.locator('[data-testid="task-title-input"]');
    await expect(titleInput).toBeVisible();

    // Verify type switching changes schedule label and hint
    const typeSelect = page.locator('[data-testid="task-type-select"]');
    const scheduleLabel = page.locator('#task-schedule-label');
    const scheduleInput = page.locator('[data-testid="task-schedule-input"]');

    // Default is cron
    await expect(scheduleLabel).toContainText('Schedule Expression');
    await expect(scheduleInput).toHaveAttribute('placeholder', '0 9 * * 1-5');

    // Switch to watch
    await typeSelect.selectOption('watch');
    await expect(scheduleLabel).toContainText('Poll Interval (Minutes)');
    await expect(scheduleInput).toHaveAttribute('placeholder', '30');

    // Switch back to cron
    await typeSelect.selectOption('cron');
    await expect(scheduleLabel).toContainText('Schedule Expression');

    // Close via cancel button
    await page.locator('[data-testid="task-modal-cancel-btn"]').click();
    await expect(modalBackdrop).toHaveAttribute('hidden', '');

    // Open again and close via Escape key
    await page.locator('[data-testid="new-task-btn"]').click();
    await expect(modalBackdrop).not.toHaveAttribute('hidden', '');
    await page.keyboard.press('Escape');
    await expect(modalBackdrop).toHaveAttribute('hidden', '');
  });

  test('should render task cards with badges, countdowns, and handle run-now action', async ({ page }) => {
    // Intercept GET /api/v1/proactive-tasks with mock data
    await page.route('**/api/v1/proactive-tasks', async route => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([
            {
              id: 'tsk-cron-1',
              session_id: 'default',
              connector_type: 'discord',
              channel_id: 'chan-123',
              target_connector: 'whatsapp',
              target_channel_id: '5511999999999',
              title: 'Daily Morning Standup Briefing',
              task_type: 'cron',
              schedule_expr: '0 9 * * 1-5',
              timezone: 'America/Sao_Paulo',
              prompt_condition: 'Summarize today meetings and urgent pull requests',
              is_active: true,
              next_run_at: new Date(Date.now() + 3600 * 1000).toISOString(),
              last_run_at: new Date(Date.now() - 86400 * 1000).toISOString(),
              created_at: new Date().toISOString(),
            },
            {
              id: 'tsk-watch-1',
              session_id: 'default',
              connector_type: 'web',
              channel_id: 'web',
              target_connector: 'telegram',
              target_channel_id: 'tg-group-1',
              title: 'VIP Inbox Watcher',
              task_type: 'watch',
              schedule_expr: '15',
              timezone: 'America/Sao_Paulo',
              prompt_condition: 'Alert immediately if email from boss arrives',
              is_active: false,
              next_run_at: new Date(Date.now() + 900 * 1000).toISOString(),
              last_run_at: null,
              created_at: new Date().toISOString(),
            },
          ]),
        });
      } else {
        await route.continue();
      }
    });

    // Intercept POST /api/v1/proactive-tasks/tsk-cron-1/run
    await page.route('**/api/v1/proactive-tasks/tsk-cron-1/run', async route => {
      await route.fulfill({
        status: 202,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'execution enqueued successfully', task_id: 'tsk-cron-1' }),
      });
    });

    await page.goto('/#schedules');

    // Wait for cards to render
    const cards = page.locator('[data-testid="schedule-card"]');
    await expect(cards).toHaveCount(2);

    // Verify first card details
    const firstCard = cards.first();
    await expect(firstCard.locator('[data-testid="task-title"]')).toContainText('Daily Morning Standup Briefing');
    await expect(firstCard.locator('.pill-badge--cron')).toBeVisible();
    await expect(firstCard.locator('.pill-badge--active')).toBeVisible();

    // Verify second card (watch, paused)
    const secondCard = cards.nth(1);
    await expect(secondCard.locator('[data-testid="task-title"]')).toContainText('VIP Inbox Watcher');
    await expect(secondCard.locator('.pill-badge--watch')).toBeVisible();
    await expect(secondCard.locator('.pill-badge--paused')).toBeVisible();

    // Click "Run Now" on first card and verify toast feedback
    const runBtn = firstCard.locator('[data-testid="task-run-btn"]');
    await runBtn.click();

    const toast = page.locator('.toast--visible');
    await expect(toast).toBeVisible();
    await expect(toast).toContainText('Daily Morning Standup Briefing');
  });

  test('should filter cards by active, paused, cron, and watch', async ({ page }) => {
    // Intercept GET with both types and statuses
    await page.route('**/api/v1/proactive-tasks', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          {
            id: 'tsk-1',
            title: 'Task Active Cron',
            task_type: 'cron',
            schedule_expr: '0 9 * * *',
            prompt_condition: 'Test prompt',
            is_active: true,
            next_run_at: new Date(Date.now() + 3600000).toISOString(),
          },
          {
            id: 'tsk-2',
            title: 'Task Paused Watch',
            task_type: 'watch',
            schedule_expr: '30',
            prompt_condition: 'Test prompt',
            is_active: false,
            next_run_at: new Date(Date.now() + 3600000).toISOString(),
          },
        ]),
      });
    });

    await page.goto('/#schedules');
    const filterSelect = page.locator('[data-testid="schedules-filter"]');
    const cards = page.locator('[data-testid="schedule-card"]');

    // All: 2 cards
    await expect(cards).toHaveCount(2);

    // Active: 1 card
    await filterSelect.selectOption('active');
    await expect(cards).toHaveCount(1);
    await expect(cards.first().locator('[data-testid="task-title"]')).toContainText('Task Active Cron');

    // Paused: 1 card
    await filterSelect.selectOption('paused');
    await expect(cards).toHaveCount(1);
    await expect(cards.first().locator('[data-testid="task-title"]')).toContainText('Task Paused Watch');

    // Cron: 1 card
    await filterSelect.selectOption('cron');
    await expect(cards).toHaveCount(1);
    await expect(cards.first().locator('[data-testid="task-title"]')).toContainText('Task Active Cron');

    // Watch: 1 card
    await filterSelect.selectOption('watch');
    await expect(cards).toHaveCount(1);
    await expect(cards.first().locator('[data-testid="task-title"]')).toContainText('Task Paused Watch');
  });

  test('should open edit modal pre-filled and update a task via PATCH', async ({ page }) => {
    let patchCalled = false;
    let patchPayload: any = null;

    const mockTask = {
      id: 'tsk-edit-1',
      title: 'Original Title',
      task_type: 'cron',
      schedule_expr: '0 9 * * *',
      timezone: 'America/Sao_Paulo',
      prompt_condition: 'Original instruction prompt',
      target_connector: 'discord',
      target_channel_id: 'chan-dev',
      is_active: true,
      next_run_at: new Date(Date.now() + 3600000).toISOString(),
    };

    await page.route('**/api/v1/proactive-tasks', async route => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([mockTask]),
        });
      }
    });

    await page.route('**/api/v1/proactive-tasks/tsk-edit-1', async route => {
      if (route.request().method() === 'PATCH') {
        patchCalled = true;
        patchPayload = JSON.parse(route.request().postData() || '{}');
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ ...mockTask, ...patchPayload }),
        });
      }
    });

    await page.goto('/#schedules');

    // Click Edit button
    const editBtn = page.locator('[data-testid="task-edit-btn"]').first();
    await expect(editBtn).toBeVisible();
    await editBtn.click();

    // Verify modal is open in Edit mode
    const modal = page.locator('[data-testid="schedules-modal"]');
    await expect(modal).toBeVisible();
    await expect(page.locator('#schedules-modal-title')).toContainText('Edit Proactive Task');
    await expect(page.locator('[data-testid="task-modal-submit-btn"]')).toContainText('Save Changes');

    // Verify pre-filled inputs
    await expect(page.locator('[data-testid="task-title-input"]')).toHaveValue('Original Title');
    await expect(page.locator('[data-testid="task-schedule-input"]')).toHaveValue('0 9 * * *');
    await expect(page.locator('[data-testid="task-prompt-input"]')).toHaveValue('Original instruction prompt');

    // Modify title and prompt
    await page.locator('[data-testid="task-title-input"]').fill('Updated Title');
    await page.locator('[data-testid="task-prompt-input"]').fill('Updated instruction prompt');

    // Submit
    await page.locator('[data-testid="task-modal-submit-btn"]').click();

    // Modal should close
    const backdrop = page.locator('[data-testid="schedules-modal-backdrop"]');
    await expect(backdrop).toHaveAttribute('hidden', '');

    expect(patchCalled).toBe(true);
    expect(patchPayload.title).toBe('Updated Title');
    expect(patchPayload.prompt_condition).toBe('Updated instruction prompt');
  });
});

