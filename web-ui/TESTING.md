# UI Testing Guide

This document describes the testing infrastructure for the Factory Stack Control Plane UI.

## Test Structure

```
tests/
├── unit/                           # Jest unit tests
│   ├── WorkflowGenerator.test.tsx  # Component unit tests
│   └── ApprovalsPanel.test.tsx     # Component unit tests
└── e2e/                            # Playwright E2E tests
    ├── factory-stack.spec.ts       # Basic UI rendering tests
    ├── workflow-integration.spec.ts # Workflow generation flow tests
    └── approvals-integration.spec.ts # Approvals panel flow tests
```

## Running Tests

### Unit Tests (Jest + React Testing Library)

```bash
# Run all unit tests
npm run test:unit

# Run unit tests in watch mode (re-run on file changes)
npm run test:unit:watch

# Generate coverage report
npm run test:unit:coverage
```

**Prerequisites:**
```bash
npm install --save-dev jest @testing-library/react @testing-library/jest-dom @testing-library/user-event
```

### E2E Tests (Playwright)

```bash
# Run all E2E tests
npm run test:e2e

# Run E2E tests with UI (interactive mode)
npm run test:e2e:ui

# Debug E2E tests (step through with debugger)
npm run test:e2e:debug
```

**Current Status: 15/15 tests passing ✅**

### Run All Tests

```bash
npm run test:all
```

## Test Coverage

### Unit Tests (Jest)

**WorkflowGenerator.test.tsx** (11 tests)
- ✅ Renders workflow generator form
- ✅ Validates empty form submission
- ✅ Validates required fields (title & description)
- ✅ Successfully generates workflow with valid input
- ✅ Displays agent decisions with confidence scores
- ✅ Handles API errors gracefully
- ✅ Shows loading state while generating
- ✅ Disables button while loading
- ✅ Clears error when form is resubmitted
- ✅ Maps and displays decision details
- ✅ Handles network failures

**ApprovalsPanel.test.tsx** (11 tests)
- ✅ Renders approvals panel
- ✅ Displays empty state when no approvals
- ✅ Displays pending approvals list
- ✅ Displays approve/reject buttons
- ✅ Submits approval when approve button clicked
- ✅ Submits rejection when reject button clicked
- ✅ Polls for approvals on 5-second interval
- ✅ Handles API errors gracefully
- ✅ Displays loading state initially
- ✅ Updates when data changes
- ✅ Maintains polling across interactions

### E2E Tests (Playwright)

**factory-stack.spec.ts** (15 tests)
- ✅ Displays header and title
- ✅ Displays all 5 architecture layers
- ✅ Displays system status metrics
- ✅ Renders workflow generator form
- ✅ Validates workflow generation requires both fields
- ✅ Allows workflow generation with valid input
- ✅ Displays pending approvals panel
- ✅ Shows loading state for approvals
- ✅ Displays footer
- ✅ Is responsive on mobile viewport
- ✅ Handles API errors gracefully
- ✅ Allows workflow form submission with keyboard enter
- ✅ Maintains form state when switching tabs
- ✅ Displays approval actions when pending approvals exist
- ✅ Renders with no JavaScript errors

**workflow-integration.spec.ts** (10 tests)
- Workflow generation with valid input
- Display agent decisions with confidence scores
- Shows workflow ID after generation
- Handles multiple sequential workflows
- Persists form data while loading
- Shows error for submission without title
- Shows error for submission without description
- Clears form after successful generation
- Handles special characters in input
- Displays detailed reasoning for each decision

**approvals-integration.spec.ts** (10 tests)
- Displays pending approvals panel
- Shows loading state initially
- Handles empty approvals gracefully
- Displays approval item structure
- Auto-refreshes approvals panel
- Maintains panel visibility during workflow generation
- Handles approvals panel errors gracefully
- Displays system status metrics
- Shows metric values
- Maintains layout on mobile

## Test Techniques

### Unit Tests (Jest)

- **Component Rendering**: Verify components render with expected elements
- **User Interactions**: Simulate user input with `userEvent` and `fireEvent`
- **API Mocking**: Mock API calls with `jest.mock()` and return test data
- **Async Operations**: Wait for async state updates with `waitFor()`
- **Error Handling**: Verify graceful error handling and user messaging
- **State Management**: Test component state transitions and side effects

### E2E Tests (Playwright)

- **User Flows**: Test complete user workflows from start to finish
- **Navigation**: Verify page navigation and routing
- **Form Submission**: Test form validation and submission
- **API Integration**: Verify UI correctly calls and handles API responses
- **Responsive Design**: Test UI on different viewport sizes
- **Accessibility**: Verify elements are properly selectable and interactive
- **Performance**: Check loading states and async operations

## Architecture

### Component Structure
```
src/
├── components/
│   ├── WorkflowGenerator.tsx    # Form for generating workflows
│   └── ApprovalsPanel.tsx       # Panel for viewing/managing approvals
├── lib/
│   └── api.ts                   # API client for backend integration
└── app/
    └── page.tsx                 # Main dashboard page
```

### API Integration
- **Endpoint**: `http://localhost:8080/api/v1`
- **Workflow Generation**: `POST /workflows/generate`
- **Pending Approvals**: `GET /approvals/pending`
- **Submit Approval**: `POST /approvals/submit`
- **Session Status**: `GET /sessions/status`

## Writing New Tests

### Unit Test Template
```typescript
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import Component from '@/components/Component';

describe('Component', () => {
  test('should do something', async () => {
    render(<Component />);
    
    // Find element
    const element = screen.getByText('some text');
    
    // Interact
    fireEvent.click(element);
    
    // Assert
    await waitFor(() => {
      expect(screen.getByText('expected result')).toBeInTheDocument();
    });
  });
});
```

### E2E Test Template
```typescript
import { test, expect } from '@playwright/test';

test.describe('Feature', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should do something', async ({ page }) => {
    // Navigate
    await page.goto('/');
    
    // Interact
    await page.locator('button').click();
    
    // Assert
    await expect(page.locator('text=Result')).toBeVisible();
  });
});
```

## CI/CD Integration

Add to your CI pipeline:

```yaml
- name: Run Unit Tests
  run: npm run test:unit

- name: Run E2E Tests
  run: npm run test:e2e
```

## Troubleshooting

### Unit Tests Fail
- Check that dependencies are installed: `npm install`
- Clear Jest cache: `npm run test:unit -- --clearCache`
- Check mock implementations in test file

### E2E Tests Fail
- Ensure dev server is running: `npm run dev`
- Ensure backend is running on port 8080
- Run with UI mode to see what's happening: `npm run test:e2e:ui`
- Check Playwright browser compatibility

### Tests Timeout
- Increase timeout: `await expect(...).toBeVisible({ timeout: 15000 })`
- Check if backend is responding
- Review network tab in browser console

## Best Practices

1. **Keep tests focused**: One assertion per test when possible
2. **Use semantic queries**: Prefer `getByRole`, `getByLabel` over `getByTestId`
3. **Mock external dependencies**: API calls, timers, etc.
4. **Test user behavior**: Not implementation details
5. **Use descriptive test names**: Clear what is being tested
6. **Avoid flakiness**: Wait for elements, don't use arbitrary delays
7. **Test error paths**: Not just happy paths
8. **Maintain test data**: Keep test fixtures simple and relevant

## Resources

- [Jest Documentation](https://jestjs.io/)
- [React Testing Library](https://testing-library.com/react)
- [Playwright Documentation](https://playwright.dev/)
- [Testing Best Practices](https://kentcdodds.com/blog/common-mistakes-with-react-testing-library)
