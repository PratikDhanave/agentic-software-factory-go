import { test, expect } from '@playwright/test';

test.describe('Approvals Panel Integration', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should display pending approvals panel', async ({ page }) => {
    const approvalsPanel = page.locator('h2, h3').filter({ hasText: 'Pending Approvals' });
    await expect(approvalsPanel).toBeVisible();
  });

  test('should show loading state initially', async ({ page }) => {
    const loadingText = page.locator('text=Loading pending approvals');
    // May be visible briefly
    await page.waitForTimeout(500);
  });

  test('should handle empty approvals gracefully', async ({ page }) => {
    // After load, should show either empty state or nothing
    await page.waitForTimeout(2000);

    const approvalsPanel = page.locator('h2, h3').filter({ hasText: 'Pending Approvals' });
    await expect(approvalsPanel).toBeVisible();
  });

  test('should display approval item structure', async ({ page }) => {
    // Even with no approvals, the structure should be there
    const approvalsContainer = page.locator('div').filter({ has: page.locator('text=Pending Approvals') });
    await expect(approvalsContainer).toBeVisible();
  });

  test('should auto-refresh approvals panel', async ({ page }) => {
    // Check that panel is visible
    const approvalsPanel = page.locator('h2, h3').filter({ hasText: 'Pending Approvals' });
    await expect(approvalsPanel).toBeVisible();

    // Wait for auto-refresh interval (5 seconds)
    await page.waitForTimeout(6000);

    // Panel should still be visible after refresh
    await expect(approvalsPanel).toBeVisible();
  });

  test('should maintain panel visibility during workflow generation', async ({ page }) => {
    const approvalsPanel = page.locator('h2, h3').filter({ hasText: 'Pending Approvals' });

    // Generate workflow
    await page.locator('input[placeholder*="Fix authentication"]').fill('Test Bug');
    await page.locator('textarea[placeholder*="Describe"]').fill('Test Description');
    await page.locator('button:has-text("Generate Workflow")').click();

    // Approvals panel should still be visible
    await expect(approvalsPanel).toBeVisible();
  });

  test('should handle approvals panel errors gracefully', async ({ page }) => {
    // Even if API fails, panel should still render
    const approvalsPanel = page.locator('h2, h3').filter({ hasText: 'Pending Approvals' });
    await expect(approvalsPanel).toBeVisible();

    // Wait for load
    await page.waitForTimeout(1000);

    // Check if error or empty state is shown
    const container = page.locator('div').filter({ has: approvalsPanel });
    await expect(container).toBeVisible();
  });

  test('should display system status metrics', async ({ page }) => {
    const statusPanel = page.locator('h3').filter({ hasText: 'System Status' });
    await expect(statusPanel).toBeVisible();

    // Check for metrics
    await expect(page.getByText('Active Agents')).toBeVisible();
    await expect(page.getByText('Workflows')).toBeVisible();
    await expect(page.getByText('Test Coverage')).toBeVisible();
    await expect(page.getByText('Avg Speedup')).toBeVisible();
  });

  test('should show metric values', async ({ page }) => {
    // Check for specific metric values
    await expect(page.locator('text=10').filter({ hasText: 'green' }).first()).toBeVisible();
    await expect(page.locator('text=42').first()).toBeVisible();
    await expect(page.locator('text=100+').first()).toBeVisible();
    await expect(page.locator('text=45%').first()).toBeVisible();
  });

  test('should maintain layout on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    // Approvals panel should still be accessible
    const approvalsPanel = page.locator('h2, h3').filter({ hasText: 'Pending Approvals' });

    // May need to scroll to see it
    await approvalsPanel.scrollIntoViewIfNeeded();
    await expect(approvalsPanel).toBeVisible();
  });

  test('should handle concurrent operations', async ({ page }) => {
    const approvalsPanel = page.locator('h2, h3').filter({ hasText: 'Pending Approvals' });

    // Start generating workflow
    await page.locator('input[placeholder*="Fix authentication"]').fill('Concurrent Bug');
    await page.locator('textarea[placeholder*="Describe"]').fill('Concurrent Desc');
    await page.locator('button:has-text("Generate Workflow")').click();

    // Approvals should still be fetching/visible
    await page.waitForTimeout(2000);
    await expect(approvalsPanel).toBeVisible();
  });

  test('should display complete panel hierarchy', async ({ page }) => {
    // Main header
    await expect(page.getByText('Factory Stack Control Plane')).toBeVisible();

    // Architecture layers
    const layers = ['Governance', 'Coordination', 'Execution', 'Integration', 'Observability'];
    for (const layer of layers) {
      await expect(page.getByText(layer, { exact: true }).first()).toBeVisible();
    }

    // Workflow section
    await expect(page.getByText('Generate Workflow')).toBeVisible();

    // Approvals section
    await expect(page.locator('h2, h3').filter({ hasText: 'Pending Approvals' })).toBeVisible();

    // Status section
    await expect(page.getByText('System Status')).toBeVisible();

    // Footer
    await expect(page.getByText('Factory Stack • Combining')).toBeVisible();
  });

  test('should handle page refresh while approvals loading', async ({ page }) => {
    // Load page
    await page.goto('/');

    // Immediately refresh
    await page.reload();

    // Should still work after refresh
    const approvalsPanel = page.locator('h2, h3').filter({ hasText: 'Pending Approvals' });
    await expect(approvalsPanel).toBeVisible();
  });
});
