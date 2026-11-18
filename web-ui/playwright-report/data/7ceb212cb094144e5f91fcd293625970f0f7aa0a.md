# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: factory-stack.spec.ts >> Factory Stack Control Plane >> should display approval actions when pending approvals exist
- Location: tests/e2e/factory-stack.spec.ts:112:3

# Error details

```
Error: locator.isVisible: Error: strict mode violation: locator('.bg-white').filter({ hasText: 'Pending Approvals' }) resolved to 2 elements:
    1) <body class="min-h-screen bg-white text-black antialiased">…</body> aka locator('body')
    2) <div class="bg-white rounded-lg shadow-md p-6">…</div> aka getByText('Pending Approvals Failed to')

Call log:
    - checking visibility of locator('.bg-white').filter({ hasText: 'Pending Approvals' })

```

# Page snapshot

```yaml
- generic [active] [ref=e1]:
  - main [ref=e2]:
    - generic [ref=e3]:
      - generic [ref=e4]:
        - generic [ref=e5]:
          - generic [ref=e6]: ⚙️
          - heading "Factory Stack Control Plane" [level=1] [ref=e8]
        - paragraph [ref=e9]: AI-Powered Multi-Agent Workflow Coordination & Governance
      - generic [ref=e10]:
        - generic [ref=e11]:
          - generic [ref=e12]: Layer 5
          - generic [ref=e13]: Governance
          - generic [ref=e14]: Identity & Auth
        - generic [ref=e15]:
          - generic [ref=e16]: Layer 4
          - generic [ref=e17]: Coordination
          - generic [ref=e18]: 10 AI Agents
        - generic [ref=e19]:
          - generic [ref=e20]: Layer 3
          - generic [ref=e21]: Execution
          - generic [ref=e22]: DAG Engine
        - generic [ref=e23]:
          - generic [ref=e24]: Layer 2
          - generic [ref=e25]: Integration
          - generic [ref=e26]: Events
        - generic [ref=e27]:
          - generic [ref=e28]: Layer 1
          - generic [ref=e29]: Observability
          - generic [ref=e30]: Telemetry
      - generic [ref=e31]:
        - generic [ref=e33]:
          - heading "Generate Workflow" [level=2] [ref=e34]
          - generic [ref=e35]:
            - generic [ref=e36]:
              - generic [ref=e37]: Issue Title
              - textbox "e.g., Fix authentication bug in OAuth flow" [ref=e38]
            - generic [ref=e39]:
              - generic [ref=e40]: Issue Description
              - textbox "Describe the issue in detail..." [ref=e41]
            - button "Generate Workflow" [ref=e42] [cursor=pointer]
        - generic [ref=e43]:
          - heading "System Status" [level=3] [ref=e44]
          - generic [ref=e45]:
            - generic [ref=e46]:
              - generic [ref=e47]: Active Agents
              - generic [ref=e48]: "10"
            - generic [ref=e49]:
              - generic [ref=e50]: Workflows
              - generic [ref=e51]: "42"
            - generic [ref=e52]:
              - generic [ref=e53]: Test Coverage
              - generic [ref=e54]: 100+
            - generic [ref=e55]:
              - generic [ref=e56]: Avg Speedup
              - generic [ref=e57]: 45%
      - generic [ref=e59]:
        - generic [ref=e60]:
          - heading "Pending Approvals" [level=2] [ref=e61]
          - button "Refresh" [ref=e62] [cursor=pointer]
        - paragraph [ref=e71]: Failed to fetch
        - paragraph [ref=e76]: No pending approvals
      - paragraph [ref=e78]: Factory Stack • Combining Governance + Multi-Agent + Efficient Execution
  - alert [ref=e79]
```

# Test source

```ts
  17  |     await expect(page.locator('text=Coordination')).toBeVisible();
  18  |     await expect(page.locator('text=Execution')).toBeVisible();
  19  |     await expect(page.locator('text=Integration')).toBeVisible();
  20  |     await expect(page.locator('text=Observability')).toBeVisible();
  21  |   });
  22  | 
  23  |   test('should display system status metrics', async ({ page }) => {
  24  |     await expect(page.locator('text=Active Agents')).toBeVisible();
  25  |     await expect(page.locator('text=10')).toBeVisible();
  26  |     await expect(page.locator('text=Workflows')).toBeVisible();
  27  |     await expect(page.locator('text=42')).toBeVisible();
  28  |   });
  29  | 
  30  |   test('should render workflow generator form', async ({ page }) => {
  31  |     const titleInput = page.locator('input[placeholder*="Fix authentication"]');
  32  |     const descTextarea = page.locator('textarea[placeholder*="Describe the issue"]');
  33  |     const generateBtn = page.locator('button:has-text("Generate Workflow")');
  34  | 
  35  |     await expect(titleInput).toBeVisible();
  36  |     await expect(descTextarea).toBeVisible();
  37  |     await expect(generateBtn).toBeVisible();
  38  |   });
  39  | 
  40  |   test('should validate workflow generation requires both fields', async ({ page }) => {
  41  |     const generateBtn = page.locator('button:has-text("Generate Workflow")');
  42  | 
  43  |     await generateBtn.click();
  44  | 
  45  |     const errorMsg = page.locator('text=Please fill in both title and description');
  46  |     await expect(errorMsg).toBeVisible();
  47  |   });
  48  | 
  49  |   test('should allow workflow generation with valid input', async ({ page }) => {
  50  |     await page.locator('input[placeholder*="Fix authentication"]').fill('Test Auth Bug');
  51  |     await page.locator('textarea[placeholder*="Describe the issue"]').fill('OAuth token not refreshing');
  52  | 
  53  |     const generateBtn = page.locator('button:has-text("Generate Workflow")');
  54  |     await generateBtn.click();
  55  | 
  56  |     // Wait for loading indicator
  57  |     await expect(page.locator('text=Generating...')).toBeVisible();
  58  |   });
  59  | 
  60  |   test('should display pending approvals panel', async ({ page }) => {
  61  |     const approvalsHeader = page.locator('h2:has-text("Pending Approvals")');
  62  |     await expect(approvalsHeader).toBeVisible();
  63  |   });
  64  | 
  65  |   test('should show loading state for approvals', async ({ page }) => {
  66  |     const loadingText = page.locator('text=Loading pending approvals');
  67  |     // Should eventually disappear as it loads
  68  |     await page.waitForTimeout(1000);
  69  |   });
  70  | 
  71  |   test('should display footer', async ({ page }) => {
  72  |     const footer = page.locator('text=Factory Stack • Combining Governance');
  73  |     await expect(footer).toBeVisible();
  74  |   });
  75  | 
  76  |   test('should be responsive on mobile viewport', async ({ page }) => {
  77  |     await page.setViewportSize({ width: 375, height: 667 });
  78  | 
  79  |     const header = page.locator('h1');
  80  |     await expect(header).toBeVisible();
  81  | 
  82  |     const workflowForm = page.locator('input[placeholder*="Fix authentication"]');
  83  |     await expect(workflowForm).toBeVisible();
  84  |   });
  85  | 
  86  |   test('should handle API errors gracefully', async ({ page }) => {
  87  |     // Mock API error
  88  |     await page.route('**/api/v1/approvals/pending', (route) => {
  89  |       route.abort('failed');
  90  |     });
  91  | 
  92  |     await page.goto('/');
  93  | 
  94  |     // Should eventually show error or fallback UI
  95  |     await page.waitForTimeout(2000);
  96  |     // Error handling would show in the approvals panel
  97  |   });
  98  | 
  99  |   test('should allow workflow form submission with keyboard enter', async ({ page }) => {
  100 |     const titleInput = page.locator('input[placeholder*="Fix authentication"]');
  101 | 
  102 |     await titleInput.fill('Test Issue');
  103 |     await page.locator('textarea[placeholder*="Describe"]').fill('Description here');
  104 | 
  105 |     // Submit via keyboard
  106 |     const generateBtn = page.locator('button:has-text("Generate Workflow")');
  107 |     await generateBtn.click();
  108 | 
  109 |     await expect(page.locator('text=Generating...')).toBeVisible();
  110 |   });
  111 | 
  112 |   test('should display approval actions when pending approvals exist', async ({ page }) => {
  113 |     // This test assumes pending approvals are loaded from API
  114 |     // If approvals exist, check for action buttons
  115 |     const approvalsPanel = page.locator('.bg-white').filter({ hasText: 'Pending Approvals' });
  116 | 
> 117 |     if (await approvalsPanel.isVisible()) {
      |                              ^ Error: locator.isVisible: Error: strict mode violation: locator('.bg-white').filter({ hasText: 'Pending Approvals' }) resolved to 2 elements:
  118 |       // Check for approve/reject buttons if approvals exist
  119 |       const approveBtn = page.locator('button:has-text("Approve")');
  120 |       const rejectBtn = page.locator('button:has-text("Reject")');
  121 | 
  122 |       // These would be visible if there are pending approvals
  123 |       if (await approveBtn.count() > 0) {
  124 |         await expect(approveBtn.first()).toBeVisible();
  125 |         await expect(rejectBtn.first()).toBeVisible();
  126 |       }
  127 |     }
  128 |   });
  129 | 
  130 |   test('should maintain form state when switching tabs', async ({ page }) => {
  131 |     const titleInput = page.locator('input[placeholder*="Fix authentication"]');
  132 | 
  133 |     await titleInput.fill('Test Input');
  134 | 
  135 |     // Switch focus to other elements and back
  136 |     await page.locator('textarea[placeholder*="Describe"]').focus();
  137 |     await titleInput.focus();
  138 | 
  139 |     // Form state should persist
  140 |     await expect(titleInput).toHaveValue('Test Input');
  141 |   });
  142 | 
  143 |   test('should render with no JavaScript errors', async ({ page }) => {
  144 |     let errorCount = 0;
  145 | 
  146 |     page.on('console', (msg) => {
  147 |       if (msg.type() === 'error') {
  148 |         errorCount++;
  149 |       }
  150 |     });
  151 | 
  152 |     await page.goto('/');
  153 |     await page.waitForTimeout(1000);
  154 | 
  155 |     expect(errorCount).toBe(0);
  156 |   });
  157 | });
  158 | 
```