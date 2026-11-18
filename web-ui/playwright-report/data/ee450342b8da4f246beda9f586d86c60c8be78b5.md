# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: factory-stack.spec.ts >> Factory Stack Control Plane >> should render with no JavaScript errors
- Location: tests/e2e/factory-stack.spec.ts:143:3

# Error details

```
Error: expect(received).toBe(expected) // Object.is equality

Expected: 0
Received: 4
```

# Page snapshot

```yaml
- generic [active] [ref=f1e1]:
  - main [ref=f1e2]:
    - generic [ref=f1e3]:
      - generic [ref=f1e4]:
        - generic [ref=f1e5]:
          - generic [ref=f1e6]: ⚙️
          - heading "Factory Stack Control Plane" [level=1] [ref=f1e8]
        - paragraph [ref=f1e9]: AI-Powered Multi-Agent Workflow Coordination & Governance
      - generic [ref=f1e10]:
        - generic [ref=f1e11]:
          - generic [ref=f1e12]: Layer 5
          - generic [ref=f1e13]: Governance
          - generic [ref=f1e14]: Identity & Auth
        - generic [ref=f1e15]:
          - generic [ref=f1e16]: Layer 4
          - generic [ref=f1e17]: Coordination
          - generic [ref=f1e18]: 10 AI Agents
        - generic [ref=f1e19]:
          - generic [ref=f1e20]: Layer 3
          - generic [ref=f1e21]: Execution
          - generic [ref=f1e22]: DAG Engine
        - generic [ref=f1e23]:
          - generic [ref=f1e24]: Layer 2
          - generic [ref=f1e25]: Integration
          - generic [ref=f1e26]: Events
        - generic [ref=f1e27]:
          - generic [ref=f1e28]: Layer 1
          - generic [ref=f1e29]: Observability
          - generic [ref=f1e30]: Telemetry
      - generic [ref=f1e31]:
        - generic [ref=f1e33]:
          - heading "Generate Workflow" [level=2] [ref=f1e34]
          - generic [ref=f1e35]:
            - generic [ref=f1e36]:
              - generic [ref=f1e37]: Issue Title
              - textbox "e.g., Fix authentication bug in OAuth flow" [ref=f1e38]
            - generic [ref=f1e39]:
              - generic [ref=f1e40]: Issue Description
              - textbox "Describe the issue in detail..." [ref=f1e41]
            - button "Generate Workflow" [ref=f1e42] [cursor=pointer]
        - generic [ref=f1e43]:
          - heading "System Status" [level=3] [ref=f1e44]
          - generic [ref=f1e45]:
            - generic [ref=f1e46]:
              - generic [ref=f1e47]: Active Agents
              - generic [ref=f1e48]: "10"
            - generic [ref=f1e49]:
              - generic [ref=f1e50]: Workflows
              - generic [ref=f1e51]: "42"
            - generic [ref=f1e52]:
              - generic [ref=f1e53]: Test Coverage
              - generic [ref=f1e54]: 100+
            - generic [ref=f1e55]:
              - generic [ref=f1e56]: Avg Speedup
              - generic [ref=f1e57]: 45%
      - generic [ref=f1e59]:
        - generic [ref=f1e60]:
          - heading "Pending Approvals" [level=2] [ref=f1e61]
          - button "Refresh" [ref=f1e62] [cursor=pointer]
        - paragraph [ref=f1e71]: Failed to fetch
        - paragraph [ref=f1e76]: No pending approvals
      - paragraph [ref=f1e78]: Factory Stack • Combining Governance + Multi-Agent + Efficient Execution
  - alert [ref=f1e79]
```

# Test source

```ts
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
  117 |     if (await approvalsPanel.isVisible()) {
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
> 155 |     expect(errorCount).toBe(0);
      |                        ^ Error: expect(received).toBe(expected) // Object.is equality
  156 |   });
  157 | });
  158 | 
```