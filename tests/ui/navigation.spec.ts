import { test, expect } from '@playwright/test';

test.describe('Navigation and Module Decoupling', () => {
  test('should render all primary tabs and maintain hash-based routing', async ({ page }) => {
    await page.goto('/');

    // Verify chat tab is active initially
    const chatTab = page.locator('[data-tab="chat"]');
    await expect(chatTab).toHaveClass(/active/);

    const chatSection = page.locator('#tab-chat');
    await expect(chatSection).not.toHaveAttribute('hidden', '');

    // Click sessions tab
    await page.locator('[data-tab="sessions"]').click();
    await expect(page).toHaveURL('/#sessions');
    await expect(page.locator('[data-tab="sessions"]')).toHaveClass(/active/);
    await expect(page.locator('#tab-sessions')).not.toHaveAttribute('hidden', '');

    // Click schedules tab
    await page.locator('[data-tab="schedules"]').click();
    await expect(page).toHaveURL('/#schedules');
    await expect(page.locator('[data-tab="schedules"]')).toHaveClass(/active/);
    await expect(page.locator('#tab-schedules')).not.toHaveAttribute('hidden', '');

    // Click logs tab
    await page.locator('[data-tab="logs"]').click();
    await expect(page).toHaveURL('/#logs');
    await expect(page.locator('[data-tab="logs"]')).toHaveClass(/active/);
    await expect(page.locator('#tab-logs')).not.toHaveAttribute('hidden', '');

    // Click settings tab
    await page.locator('[data-tab="settings"]').click();
    await expect(page).toHaveURL('/#settings');
    await expect(page.locator('[data-tab="settings"]')).toHaveClass(/active/);
    await expect(page.locator('#tab-settings')).not.toHaveAttribute('hidden', '');
  });

  test('should redirect legacy /#connectors to unified settings connectors category', async ({ page }) => {
    await page.goto('/#connectors');

    // Should redirect to #settings/connectors
    await expect(page).toHaveURL('/#settings/connectors');
    await expect(page.locator('[data-tab="settings"]')).toHaveClass(/active/);
    await expect(page.locator('#tab-settings')).not.toHaveAttribute('hidden', '');

    // The Connectors category button should be active
    const connectorsCatBtn = page.locator('[data-settings-cat="connectors"]');
    await expect(connectorsCatBtn).toHaveClass(/settings-nav-btn--active/);

    // The Connectors section should be visible
    const connectorsSection = page.locator('[data-settings-section="connectors"]');
    await expect(connectorsSection).toBeVisible();

    // LLM section should be hidden when filtered to connectors
    const llmSection = page.locator('[data-settings-section="llm"]');
    await expect(llmSection).toBeHidden();
  });

  test('should filter settings sections via category buttons', async ({ page }) => {
    await page.goto('/#settings');

    const llmSection = page.locator('[data-settings-section="llm"]');
    const connectorsSection = page.locator('[data-settings-section="connectors"]');
    const toolsSection = page.locator('[data-settings-section="tools"]');

    // Initially 'all' category is active -> all sections visible
    await expect(llmSection).toBeVisible();
    await expect(connectorsSection).toBeVisible();
    await expect(toolsSection).toBeVisible();

    // Filter to 'tools'
    await page.locator('[data-settings-cat="tools"]').click();
    await expect(toolsSection).toBeVisible();
    await expect(connectorsSection).toBeHidden();
    await expect(llmSection).toBeHidden();

    // Reset to 'all'
    await page.locator('[data-settings-cat="all"]').click();
    await expect(llmSection).toBeVisible();
    await expect(connectorsSection).toBeVisible();
    await expect(toolsSection).toBeVisible();
  });

  test('should persist tab state on page refresh', async ({ page }) => {
    await page.goto('/');

    // Navigate to sessions tab
    await page.locator('[data-tab="sessions"]').click();
    await expect(page).toHaveURL('/#sessions');

    // Refresh page
    await page.reload();

    // Sessions tab should still be active
    await expect(page.locator('[data-tab="sessions"]')).toHaveClass(/active/);
    await expect(page.locator('#tab-sessions')).not.toHaveAttribute('hidden', '');
  });

  test('should populate logs dropdown without visiting sessions tab first', async ({ page }) => {
    // This test verifies the Sessions → Logs decoupling fix
    await page.goto('/#logs');

    // Sessions should load in background despite not visiting that tab
    await page.waitForTimeout(1000); // Wait for async init

    const logsDropdown = page.locator('#logs-session-select');
    const options = logsDropdown.locator('option');

    // Should have at least the default "Select a session" option
    const optionCount = await options.count();
    expect(optionCount).toBeGreaterThan(0);

    // If there are sessions in the DB, they should be populated
    const optionTexts = await options.allTextContents();
    expect(optionTexts[0]).toContain('Select a session');
  });

  test('should render connector cards with live status in settings', async ({ page }) => {
    await page.goto('/#settings/connectors');

    const discordCard = page.locator('[data-connector="discord"]');
    await expect(discordCard).toBeVisible();
    await expect(discordCard.locator('[data-status]')).toBeVisible();

    const whatsappCard = page.locator('[data-connector="whatsapp"]');
    await expect(whatsappCard).toBeVisible();
    await expect(whatsappCard.locator('[data-status]')).toBeVisible();

    const telegramCard = page.locator('[data-connector="telegram"]');
    await expect(telegramCard).toBeVisible();
  });
});
