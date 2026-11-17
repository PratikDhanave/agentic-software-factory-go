# Visual Walkthrough: Triage UI in Action

This document shows exactly what you'll see when using the triage feature from the web UI.

---

## Screen 1: Dashboard Home Page

After starting both backend and frontend, visit `http://localhost:3000`

```
╔════════════════════════════════════════════════════════════════════════╗
║ Agentic Software Factory                                      v1.0.0 ⚙️  │
║ Production-grade Agentic Software Engineering patterns for Go          │
╠════════════════════════════════════════════════════════════════════════╣
║                                                                        │
║ 🎯 LAUNCHPAD SECTION                                                  │
║ ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                  │
║ │🤖 Code       │  │📝 API Design │  │🧪 Verify &   │                  │
║ │Generator     │  │              │  │Test          │                  │
║ └──────────────┘  └──────────────┘  └──────────────┘                  │
║ ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                  │
║ │🔄 Refactor   │  │🏗️ Architecture
│  │📊 Benchmark  │                  │
║ │Code          │  │Review         │  │              │                  │
║ └──────────────┘  └──────────────┘  └──────────────┘                  │
║                                                                        │
║ ⚡ AUTOMATIC GITHUB ISSUE TRIAGE SECTION  ← NEW!                      │
║ Analyze GitHub issues, categorize them, and assess readiness...      │
║                                                                        │
║ ┌────────────────────────────────────────────────────────────────┐   │
║ │ GitHub Owner:                                                  │   │
║ │ [                                        ]                     │   │
║ │                                                                │   │
║ │ Repository:                                                    │   │
║ │ [                                        ]                     │   │
║ │                                                                │   │
║ │ Issue Number:                                                  │   │
║ │ [    ]                                                         │   │
║ │                                                                │   │
║ │ GitHub Personal Access Token:                                  │   │
║ │ [●●●●●●●●●●●●●] [Show] [Load] [Save]                         │   │
║ │ Token needs scopes: repo, write:discussion                     │   │
║ │                                                                │   │
║ │ [🚀 Run Triage]                                               │   │
║ └────────────────────────────────────────────────────────────────┘   │
║                                                                        │
╚════════════════════════════════════════════════════════════════════════╝
```

---

## Screen 2: Filling in Details

User fills in the form with their GitHub repository information:

```
╔════════════════════════════════════════════════════════════════════════╗
║ ⚡ AUTOMATIC GITHUB ISSUE TRIAGE                                       │
║                                                                        │
║ ┌────────────────────────────────────────────────────────────────┐   │
║ │ GitHub Owner:                                                  │   │
║ │ [PratikDhanave                                    ]             │   │
║ │                                                                │   │
║ │ Repository:                                                    │   │
║ │ [agentic-software-factory-go                      ]             │   │
║ │                                                                │   │
║ │ Issue Number:                                                  │   │
║ │ [1     ]                                                       │   │
║ │                                                                │   │
║ │ GitHub Personal Access Token:                                  │   │
║ │ [ghp_ZyC66ZdK1r3F5qfIFDH7gxsbYrVuQY3N] [Show] [Load] [Save]  │   │
║ │ Token needs scopes: repo, write:discussion                     │   │
║ │                                                                │   │
║ │ [🚀 Run Triage]  ← User clicks here                           │   │
║ └────────────────────────────────────────────────────────────────┘   │
│                                                                        │
╚════════════════════════════════════════════════════════════════════════╝
```

---

## Screen 3: Running Triage (Progress)

After clicking "Run Triage", the UI shows real-time progress:

```
╔════════════════════════════════════════════════════════════════════════╗
║ ⚡ AUTOMATIC GITHUB ISSUE TRIAGE                                       │
║                                                                        │
║ ┌────────────────────────────────────────────────────────────────┐   │
║ │ GitHub Owner:   [PratikDhanave                    ]             │   │
║ │ Repository:     [agentic-software-factory-go      ]             │   │
║ │ Issue Number:   [1     ]                                       │   │
║ │ Token:          [●●●●●●●●●●●●●] [Show] [Load] [Save]         │   │
║ │                                                                │   │
║ │ Processing...                                                  │   │
║ │                                          45%                    │   │
║ │ ████████████░░░░░░░░░░░░░░░░░░░░░░░░  (animated progress)     │   │
║ │                                                                │   │
║ │ [⏳ Analyzing...]                                              │   │
║ └────────────────────────────────────────────────────────────────┘   │
│                                                                        │
╚════════════════════════════════════════════════════════════════════════╝
```

**Progress stages** (visible in sequence):
1. ✓ Connecting to GitHub (10%)
2. ✓ Fetching issue details (25%)
3. → **Analyzing issue content** (45%)
4. ⏳ Generating implementation spec (65%)
5. ⏳ Applying labels to issue (80%)
6. ⏳ Posting triage comment (90%)
7. ⏳ Complete (100%)

---

## Screen 4: Results Displayed

After triage completes (takes ~5-10 seconds):

```
╔════════════════════════════════════════════════════════════════════════╗
║ ⚡ AUTOMATIC GITHUB ISSUE TRIAGE                                       │
║                                                                        │
║ ┌────────────────────────────────────────────────────────────────┐   │
║ │ [Form fields above, now completed]                             │   │
║ │                                                                │   │
║ │ ✅ TRIAGE RESULT                                              │   │
║ │                                                                │   │
║ │ ┌──────────────────────┐ ┌──────────────────────┐             │   │
║ │ │ Category             │ │ Priority             │             │   │
║ │ │ feature              │ │ medium               │             │   │
║ │ └──────────────────────┘ └──────────────────────┘             │   │
║ │                                                                │   │
║ │ ┌──────────────────────┐ ┌──────────────────────┐             │   │
║ │ │ Ready to Implement   │ │ Labels to Apply      │             │   │
║ │ │ ✅ Yes               │ │ ready-to-implement   │             │   │
║ │ │                      │ │ category-feature     │             │   │
║ │ │                      │ │ priority-medium      │             │   │
║ │ └──────────────────────┘ └──────────────────────┘             │   │
║ │                                                                │   │
║ │ Analysis:                                                      │   │
║ │ Has clear acceptance criteria, sufficient context, and no     │   │
║ │ blockers                                                       │   │
║ │                                                                │   │
║ │ Generated Spec:                                                │   │
║ │ <details>                                                      │   │
║ │   ▶ Show YAML Spec                                            │   │
║ │ </details>                                                     │   │
║ │                                                                │   │
║ │ [View on GitHub] [Copy Spec]  ← Action buttons                │   │
║ └────────────────────────────────────────────────────────────────┘   │
│                                                                        │
╚════════════════════════════════════════════════════════════════════════╝
```

---

## Screen 5: Expanded Spec View

User clicks "Show YAML Spec" to see details:

```
╔════════════════════════════════════════════════════════════════════════╗
║ Generated Spec:                                                        │
║ ▼ Hide YAML Spec                                                       │
║                                                                        │
║ ┌────────────────────────────────────────────────────────────────┐   │
║ │ id: "issue-1"                                                  │   │
║ │ version: "1.0"                                                 │   │
║ │ title: "Add thread-safe cache service"                         │   │
║ │ description: "Complete user management system"                 │   │
║ │                                                                │   │
║ │ intent:                                                         │   │
║ │   goal: "Implement the requirement described in the issue"     │   │
║ │   context: "GitHub issue: Add thread-safe cache service"       │   │
║ │   success_criteria:                                             │   │
║ │     - "Code compiles without errors"                           │   │
║ │     - "Implementation matches issue requirements"              │   │
║ │     - "Tests pass if applicable"                               │   │
║ │     - "No new warnings or errors introduced"                   │   │
║ │                                                                │   │
║ │ constraints:                                                    │   │
║ │   - id: "c1"                                                   │   │
║ │     type: "must"                                               │   │
║ │     description: "Follow Go best practices and conventions"    │   │
║ │   - id: "c2"                                                   │   │
║ │     type: "must"                                               │   │
║ │     description: "Include appropriate error handling"          │   │
║ │   - id: "c3"                                                   │   │
║ │     type: "should"                                             │   │
║ │     description: "Add unit tests for new functionality"        │   │
║ │                                                                │   │
║ │ steps:                                                          │   │
║ │   - id: "s1"                                                   │   │
║ │     name: "Analyze requirements"                               │   │
║ │   - id: "s2"                                                   │   │
║ │     name: "Implement solution"                                 │   │
║ │   - id: "s3"                                                   │   │
║ │     name: "Test thoroughly"                                    │   │
║ │   - id: "s4"                                                   │   │
║ │     name: "Review code quality"                                │   │
║ │                                                                │   │
║ └────────────────────────────────────────────────────────────────┘   │
│                                                                        │
╚════════════════════════════════════════════════════════════════════════╝
```

---

## Screen 6: Copy Spec Action

User clicks "Copy Spec" button:

```
╔════════════════════════════════════════════════════════════════════════╗
║                                                                        │
║ [View on GitHub] [Copy Spec]                                           │
║                     ↑ Clicked!
║
║ ✅ CONFIRMATION DIALOG:
║
║ ┌─────────────────────────────────────────────────────────────┐
║ │                                                             │
║ │  Spec copied to clipboard!                                 │
║ │                                                             │
║ │  You can now:                                               │
║ │  1. Go to LaunchPad section                                │
║ │  2. Find "Code Generator" card                              │
║ │  3. Paste spec in the input field                           │
║ │  4. Click to generate code                                  │
║ │                                                             │
║ │  [OK]                                                       │
║ │                                                             │
║ └─────────────────────────────────────────────────────────────┘
│                                                                        │
╚════════════════════════════════════════════════════════════════════════╝
```

---

## Screen 7: View on GitHub

User clicks "View on GitHub" button:

**New browser tab opens showing:**

```
GitHub Issue #1
═══════════════════════════════════════════════════════════════════════

Add thread-safe cache service

Description:
Need a production-grade cache service that is safe for concurrent use.

Acceptance Criteria:
- Create a Cache type with sync.RWMutex protection
- Implement Get, Set, Delete operations
- Thread-safe operations using defer Unlock pattern
- Full test coverage (80%+)
- Proper error handling for all operations

Expected Behavior:
Users should be able to safely use the cache from multiple goroutines 
without data corruption.

---

## Automatic Triage Report  ← Posted by factory!

**Category:** feature
**Priority:** medium
**Status:** ✅ Ready to Implement

**Analysis:**
Has clear acceptance criteria, sufficient context, and no blockers

**Labels Applied:** ready-to-implement, priority-medium, category-feature

**Implementation Spec Generated:** Ready for code generation

_Analyzed by 7-Layer Automatic Triage Agent_

---

Labels: ready-to-implement, category-feature, priority-medium
```

---

## Screen 8: Using Copy Spec for Code Generation

After copying spec, user scrolls back up to LaunchPad:

```
╔════════════════════════════════════════════════════════════════════════╗
║ 🎯 LAUNCHPAD SECTION                                                  │
║                                                                        │
║ ┌────────────────────────────────────────────────────────────────┐   │
║ │ 🤖 Code Generator                                              │   │
║ │                                                                │   │
║ │ Agent ID: CodeGen                                              │   │
║ │                                                                │   │
║ │ LLD YAML (Specification):                                      │   │
║ │ ┌──────────────────────────────────────────────────────────┐  │   │
║ │ │ id: "issue-1"                                            │  │   │
║ │ │ version: "1.0"                                           │  │   │
║ │ │ title: "Add thread-safe cache service"                   │  │   │ ← Pasted!
║ │ │ [rest of spec...]                                        │  │   │
║ │ └──────────────────────────────────────────────────────────┘  │   │
║ │                                                                │   │
║ │ [Run Agent]  ← User clicks here                               │   │
║ │                                                                │   │
║ └────────────────────────────────────────────────────────────────┘   │
│                                                                        │
╚════════════════════════════════════════════════════════════════════════╝
```

---

## Screen 9: Code Generation Results

After code generation completes:

```
╔════════════════════════════════════════════════════════════════════════╗
║ 🤖 Code Generator - Results                                            │
║                                                                        │
║ ┌────────────────────────────────────────────────────────────────┐   │
║ │                                                                │   │
║ │ Generated Code:                                                │   │
║ │ <CodeViewer>                                                   │   │
║ │ ┌──────────────────────────────────────────────────────────┐  │   │
║ │ │ package cache                                            │  │   │
║ │ │                                                          │  │   │
║ │ │ import (                                                 │  │   │
║ │ │   "errors"                                               │  │   │
║ │ │   "sync"                                                 │  │   │
║ │ │ )                                                         │  │   │
║ │ │                                                          │  │   │
║ │ │ type Cache struct {                                      │  │   │
║ │ │   mu    sync.RWMutex                                     │  │   │
║ │ │   store map[string]interface{}                           │  │   │
║ │ │ }                                                         │  │   │
║ │ │                                                          │  │   │
║ │ │ func NewCache() *Cache {                                 │  │   │
║ │ │   return &Cache{                                         │  │   │
║ │ │     store: make(map[string]interface{}),                 │  │   │
║ │ │   }                                                       │  │   │
║ │ │ }                                                         │  │   │
║ │ │                                                          │  │   │
║ │ │ func (c *Cache) Get(key string) (interface{}, error) {  │  │   │
║ │ │   c.mu.RLock()                                           │  │   │
║ │ │   defer c.mu.RUnlock()                                   │  │   │
║ │ │   // ...                                                 │  │   │
║ │ │ }                                                         │  │   │
║ │ │                                                          │  │   │
║ │ │ [Code continues... scroll to see more]                  │  │   │
║ │ └──────────────────────────────────────────────────────────┘  │   │
║ │                                                                │   │
║ │ Quality Score:                                                 │   │
║ │ ████████████████████░░░░░░░░░░░░░░░░░  92/100 ✅ Excellent  │   │
║ │                                                                │   │
║ │ Issues Found: 0                                                │   │
║ │ Tests Passing: 12/12 ✅                                        │   │
║ │ Acceptance Met: ✅ Yes                                         │   │
║ │                                                                │   │
║ │ [Copy Code] [Download] [Create PR]                            │   │
║ │                                                                │   │
║ └────────────────────────────────────────────────────────────────┘   │
│                                                                        │
╚════════════════════════════════════════════════════════════════════════╝
```

---

## End-to-End Timeline

```
┌─────────────────────────────────────────────────────────────────────┐
│ COMPLETE WORKFLOW TIMELINE                                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│ ⏱️  0:00  - Open dashboard at localhost:3000                        │
│            ↓                                                        │
│ ⏱️  0:10  - Fill in triage form (owner, repo, issue, token)         │
│            ↓                                                        │
│ ⏱️  0:15  - Click "Run Triage"                                       │
│            ↓                                                        │
│ ⏱️  0:20  - See progress bar updating (10% → 100%)                 │
│            ↓                                                        │
│ ⏱️  0:28  - Results appear on screen                                │
│     ├─ Category: feature                                            │
│     ├─ Priority: medium                                             │
│     ├─ Ready: ✅ Yes                                                │
│     └─ Labels: ready-to-implement, priority-medium, category-feat...│
│            ↓                                                        │
│ ⏱️  0:30  - Click "Copy Spec"                                       │
│            ↓                                                        │
│ ⏱️  0:45  - Scroll to LaunchPad, paste spec                         │
│            ↓                                                        │
│ ⏱️  0:50  - Click "Run Agent" to generate code                      │
│            ↓                                                        │
│ ⏱️  1:05  - Generated code with 92% quality score displayed         │
│            ↓                                                        │
│ ⏱️  1:10  - Copy code, create local branch, commit, push            │
│            ↓                                                        │
│ ⏱️  1:25  - Create pull request on GitHub                           │
│            ↓                                                        │
│ ✅ 1:30  - COMPLETE! PR ready for team review                      │
│                                                                     │
│ TOTAL TIME: 1.5 minutes (automated) + 15 minutes (manual PR setup) │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Summary

The UI makes it super easy to:

1. ✅ **Analyze** - Triage any GitHub issue (5 inputs)
2. ✅ **Review** - See categorization and readiness
3. ✅ **Generate** - Spec → Code in one dashboard
4. ✅ **Create** - PR from generated code

All from the browser, with beautiful progress indicators and results.

No command-line needed! 🎉
