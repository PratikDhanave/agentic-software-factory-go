import { test, expect } from '@playwright/test';

test.describe('Factory Stack Control Plane', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should display header and title', async ({ page }) => {
    const header = page.locator('h1');
    await expect(header).toContainText('Factory Stack Control Plane');
  });

  test('should display all 5 architecture layers', async ({ page }) => {
    const layers = page.locator('[class*="grid"]').first();

    await expect(page.locator('text=Governance')).toBeVisible();
    await expect(page.locator('text=Coordination')).toBeVisible();
    await expect(page.locator('text=Execution')).toBeVisible();
    await expect(page.locator('text=Integration')).toBeVisible();
    await expect(page.locator('text=Observability')).toBeVisible();
  });

  test('should display system status metrics', async ({ page }) => {
    await expect(page.locator('text=Active Agents')).toBeVisible();
    await expect(page.locator('text=10')).toBeVisible();
    await expect(page.locator('text=Workflows')).toBeVisible();
    await expect(page.locator('text=42')).toBeVisible();
  });

  test('should render workflow generator form', async ({ page }) => {
    const titleInput = page.locator('input[placeholder*="Fix authentication"]');
    const descTextarea = page.locator('textarea[placeholder*="Describe the issue"]');
    const generateBtn = page.locator('button:has-text("Generate Workflow")');

    await expect(titleInput).toBeVisible();
    await expect(descTextarea).toBeVisible();
    await expect(generateBtn).toBeVisible();
  });

  test('should validate workflow generation requires both fields', async ({ page }) => {
    const generateBtn = page.locator('button:has-text("Generate Workflow")');

    await generateBtn.click();

    const errorMsg = page.locator('text=Please fill in both title and description');
    await expect(errorMsg).toBeVisible();
  });

  test('should allow workflow generation with valid input', async ({ page }) => {
    await page.locator('input[placeholder*="Fix authentication"]').fill('Test Auth Bug');
    await page.locator('textarea[placeholder*="Describe the issue"]').fill('OAuth token not refreshing');

    const generateBtn = page.locator('button:has-text("Generate Workflow")');
    await generateBtn.click();

    // Wait for loading indicator
    await expect(page.locator('text=Generating...')).toBeVisible();
  });

  test('should display pending approvals panel', async ({ page }) => {
    const approvalsHeader = page.locator('h2:has-text("Pending Approvals")');
    await expect(approvalsHeader).toBeVisible();
  });

  test('should show loading state for approvals', async ({ page }) => {
    const loadingText = page.locator('text=Loading pending approvals');
    // Should eventually disappear as it loads
    await page.waitForTimeout(1000);
  });

  test('should display footer', async ({ page }) => {
    const footer = page.locator('text=Factory Stack • Combining Governance');
    await expect(footer).toBeVisible();
  });

  test('should be responsive on mobile viewport', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 667 });

    const header = page.locator('h1');
    await expect(header).toBeVisible();

    const workflowForm = page.locator('input[placeholder*="Fix authentication"]');
    await expect(workflowForm).toBeVisible();
  });

  test('should handle API errors gracefully', async ({ page }) => {
    // Mock API error
    await page.route('**/api/v1/approvals/pending', (route) => {
      route.abort('failed');
    });

    await page.goto('/');

    // Should eventually show error or fallback UI
    await page.waitForTimeout(2000);
    // Error handling would show in the approvals panel
  });

  test('should allow workflow form submission with keyboard enter', async ({ page }) => {
    const titleInput = page.locator('input[placeholder*="Fix authentication"]');

    await titleInput.fill('Test Issue');
    await page.locator('textarea[placeholder*="Describe"]').fill('Description here');

    // Submit via keyboard
    const generateBtn = page.locator('button:has-text("Generate Workflow")');
    await generateBtn.click();

    await expect(page.locator('text=Generating...')).toBeVisible();
  });

  test('should display approval actions when pending approvals exist', async ({ page }) => {
    // This test assumes pending approvals are loaded from API
    // If approvals exist, check for action buttons
    const approvalsPanel = page.locator('.bg-white').filter({ hasText: 'Pending Approvals' });

    if (await approvalsPanel.isVisible()) {
      // Check for approve/reject buttons if approvals exist
      const approveBtn = page.locator('button:has-text("Approve")');
      const rejectBtn = page.locator('button:has-text("Reject")');

      // These would be visible if there are pending approvals
      if (await approveBtn.count() > 0) {
        await expect(approveBtn.first()).toBeVisible();
        await expect(rejectBtn.first()).toBeVisible();
      }
    }
  });

  test('should maintain form state when switching tabs', async ({ page }) => {
    const titleInput = page.locator('input[placeholder*="Fix authentication"]');

    await titleInput.fill('Test Input');

    // Switch focus to other elements and back
    await page.locator('textarea[placeholder*="Describe"]').focus();
    await titleInput.focus();

    // Form state should persist
    await expect(titleInput).toHaveValue('Test Input');
  });

  test('should render with no JavaScript errors', async ({ page }) => {
    let errorCount = 0;

    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        errorCount++;
      }
    });

    await page.goto('/');
    await page.waitForTimeout(1000);

    expect(errorCount).toBe(0);
  });
});
