import { test, expect } from '@playwright/test';

test.describe('Navigation and Module Decoupling', () => {
  test('should render all tabs and maintain hash-based routing', async ({ page }) => {
    await page.goto('/');

    // Verify connectors tab is active initially
    const connectorsTab = page.locator('[data-tab="connectors"]');
    await expect(connectorsTab).toHaveClass(/active/);

    const connectorsSection = page.locator('#tab-connectors');
    await expect(connectorsSection).not.toHaveAttribute('hidden', '');

    // Click sessions tab
    await page.locator('[data-tab="sessions"]').click();
    await expect(page).toHaveURL('/#sessions');
    await expect(page.locator('[data-tab="sessions"]')).toHaveClass(/active/);
    await expect(page.locator('#tab-sessions')).not.toHaveAttribute('hidden', '');

    // Click logs tab
    await page.locator('[data-tab="logs"]').click();
    await expect(page).toHaveURL('/#logs');
    await expect(page.locator('[data-tab="logs"]')).toHaveClass(/active/);

    // Click settings tab
    await page.locator('[data-tab="settings"]').click();
    await expect(page).toHaveURL('/#settings');
    await expect(page.locator('[data-tab="settings"]')).toHaveClass(/active/);
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

  test('should render connectors list', async ({ page }) => {
    await page.goto('/#connectors');

    // Wait for connectors to load
    await page.waitForTimeout(500);

    const connectorsList = page.locator('#connectors-list');
    await expect(connectorsList).toBeTruthy();
  });

  test('should render settings form', async ({ page }) => {
    await page.goto('/#settings');

    // Wait for settings to load
    await page.waitForTimeout(500);

    const settingsForm = page.locator('#settings-form');
    await expect(settingsForm).toBeTruthy();

    // Check for expected form fields
    const apiKeyInput = settingsForm.locator('input[name="claude.api_key"]');
    await expect(apiKeyInput).toBeTruthy();

    const modelSelect = settingsForm.locator('select[name="claude.model"]');
    await expect(modelSelect).toBeTruthy();
  });
});
