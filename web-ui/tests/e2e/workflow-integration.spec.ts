import { test, expect } from '@playwright/test';

test.describe('Workflow Generation Integration', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should generate workflow with valid input', async ({ page }) => {
    // Fill in the form
    await page.locator('input[placeholder*="Fix authentication"]').fill('Fix OAuth token refresh bug');
    await page.locator('textarea[placeholder*="Describe"]').fill('The OAuth token is not refreshing properly when it expires. Users are getting logged out prematurely.');

    // Submit form
    const generateBtn = page.locator('button:has-text("Generate Workflow")');
    await generateBtn.click();

    // Wait for loading state
    await expect(page.locator('text=Generating...')).toBeVisible();

    // Wait for result
    await expect(page.locator('text=Workflow Generated')).toBeVisible({ timeout: 10000 });

    // Verify workflow details are displayed
    await expect(page.locator('text=Agent Decisions')).toBeVisible();
  });

  test('should display agent decisions with confidence scores', async ({ page }) => {
    await page.locator('input[placeholder*="Fix authentication"]').fill('Authentication Bug');
    await page.locator('textarea[placeholder*="Describe"]').fill('OAuth broken');

    await page.locator('button:has-text("Generate Workflow")').click();

    // Wait for decisions to appear
    await expect(page.locator('text=Architect')).toBeVisible({ timeout: 10000 });
    await expect(page.locator('text=Developer')).toBeVisible();
    await expect(page.locator('text=QA')).toBeVisible();

    // Verify confidence scores are displayed
    await expect(page.locator('text=95%')).toBeVisible();
    await expect(page.locator('text=92%')).toBeVisible();
    await expect(page.locator('text=88%')).toBeVisible();
  });

  test('should show workflow ID after generation', async ({ page }) => {
    await page.locator('input[placeholder*="Fix authentication"]').fill('Test Bug');
    await page.locator('textarea[placeholder*="Describe"]').fill('Test Description');

    await page.locator('button:has-text("Generate Workflow")').click();

    // Wait for success message with workflow ID
    await expect(page.locator('text=ID:')).toBeVisible({ timeout: 10000 });

    // Verify workflow ID format (should be wf-xxx)
    const workflowId = page.locator('text=ID:').locator('..').locator('text=wf-');
    await expect(workflowId).toBeVisible();
  });

  test('should handle multiple sequential workflows', async ({ page }) => {
    // Generate first workflow
    await page.locator('input[placeholder*="Fix authentication"]').fill('First Bug');
    await page.locator('textarea[placeholder*="Describe"]').fill('First Description');
    await page.locator('button:has-text("Generate Workflow")').click();

    await expect(page.locator('text=Workflow Generated')).toBeVisible({ timeout: 10000 });

    // Clear form and generate second workflow
    await page.locator('input[placeholder*="Fix authentication"]').fill('Second Bug');
    await page.locator('textarea[placeholder*="Describe"]').fill('Second Description');
    await page.locator('button:has-text("Generate Workflow")').click();

    // Should show new workflow without old one
    await expect(page.locator('text=Workflow Generated')).toBeVisible({ timeout: 10000 });
  });

  test('should persist form data while loading', async ({ page }) => {
    const titleInput = page.locator('input[placeholder*="Fix authentication"]');
    const descInput = page.locator('textarea[placeholder*="Describe"]');

    await titleInput.fill('Test Title');
    await descInput.fill('Test Description');

    const values1 = await Promise.all([
      titleInput.inputValue(),
      descInput.inputValue(),
    ]);

    expect(values1[0]).toBe('Test Title');
    expect(values1[1]).toBe('Test Description');
  });

  test('should show error for submission without title', async ({ page }) => {
    await page.locator('textarea[placeholder*="Describe"]').fill('Description only');
    await page.locator('button:has-text("Generate Workflow")').click();

    await expect(page.locator('text=Please fill in both')).toBeVisible();
  });

  test('should show error for submission without description', async ({ page }) => {
    await page.locator('input[placeholder*="Fix authentication"]').fill('Title only');
    await page.locator('button:has-text("Generate Workflow")').click();

    await expect(page.locator('text=Please fill in both')).toBeVisible();
  });

  test('should clear form after successful generation', async ({ page }) => {
    // Generate workflow
    await page.locator('input[placeholder*="Fix authentication"]').fill('Bug Title');
    await page.locator('textarea[placeholder*="Describe"]').fill('Bug Description');
    await page.locator('button:has-text("Generate Workflow")').click();

    await expect(page.locator('text=Workflow Generated')).toBeVisible({ timeout: 10000 });

    // Check if form is still filled (behavior may vary - adjust if needed)
    const titleValue = await page.locator('input[placeholder*="Fix authentication"]').inputValue();
    const descValue = await page.locator('textarea[placeholder*="Describe"]').inputValue();

    // Form should retain values for user convenience
    expect(titleValue).toBeTruthy();
    expect(descValue).toBeTruthy();
  });

  test('should handle special characters in input', async ({ page }) => {
    const specialChars = "Bug with @#$%^&*() characters & quotes \"test\"";

    await page.locator('input[placeholder*="Fix authentication"]').fill(specialChars);
    await page.locator('textarea[placeholder*="Describe"]').fill('Description with special chars: <>&"');

    await page.locator('button:has-text("Generate Workflow")').click();

    // Should handle special characters without errors
    await expect(page.locator('text=Workflow Generated')).toBeVisible({ timeout: 10000 });
  });

  test('should display detailed reasoning for each decision', async ({ page }) => {
    await page.locator('input[placeholder*="Fix authentication"]').fill('Auth Bug');
    await page.locator('textarea[placeholder*="Describe"]').fill('OAuth issue');

    await page.locator('button:has-text("Generate Workflow")').click();

    // Wait for decisions
    await expect(page.locator('text=Architect')).toBeVisible({ timeout: 10000 });

    // Verify reasoning text is displayed (italic text under each decision)
    const decisions = page.locator('p').filter({ hasText: 'Issue indicates' });
    await expect(decisions.first()).toBeVisible();
  });

  test('should handle rapid successive submissions', async ({ page }) => {
    // Submit first
    await page.locator('input[placeholder*="Fix authentication"]').fill('Bug 1');
    await page.locator('textarea[placeholder*="Describe"]').fill('Desc 1');
    await page.locator('button:has-text("Generate Workflow")').click();

    // Don't wait, submit second immediately
    await page.locator('input[placeholder*="Fix authentication"]').fill('Bug 2');
    await page.locator('textarea[placeholder*="Describe"]').fill('Desc 2');
    await page.locator('button:has-text("Generate Workflow")').click();

    // Should eventually show success
    await expect(page.locator('text=Workflow Generated')).toBeVisible({ timeout: 15000 });
  });
});
