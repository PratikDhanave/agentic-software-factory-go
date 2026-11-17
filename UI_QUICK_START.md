# UI Quick Start: Automatic Triage (5 Minutes)

## 🚀 TL;DR - Get Running NOW

### Terminal 1: Backend
```bash
./asf-web localhost:8080
# Wait for: "ASF Web Server starting on localhost:8080"
```

### Terminal 2: Frontend
```bash
cd web-ui
npm run dev
# Wait for: "✓ Ready in X.Xs"
```

### Browser
Open **http://localhost:3000**

Scroll down to **"Automatic GitHub Issue Triage"** section

---

## 📋 5-Minute Workflow

### Step 1: Prepare Your GitHub Token (1 min)

Go to: https://github.com/settings/tokens

1. Click "Generate new token (classic)"
2. Name: `Factory Triage`
3. Check scopes:
   - ✅ `repo`
   - ✅ `write:discussion`
4. Generate → Copy token

### Step 2: Fill Form (30 sec)

On dashboard, under "Automatic GitHub Issue Triage":

```
GitHub Owner:     your_username
Repository:       your_repo
Issue Number:     1 (or any issue number)
GitHub Token:     ghp_... (paste)
```

### Step 3: Run Triage (20 sec)

Click **"Run Triage"** → Wait for progress bar

```
████████████░░░░░░ 65% Generating implementation spec
```

### Step 4: View Results (1 min)

See:
- ✅ Category (feature/bug/refactor/doc)
- ✅ Priority (critical/high/medium/low)
- ✅ Ready to Implement (Yes/No)
- ✅ Labels that were applied
- ✅ Generated YAML spec

---

## 🎯 What Happens Behind the Scenes

```
Your UI Action              → Backend API            → GitHub
─────────────────────────────────────────────────────────────
1. Fill form               1. Fetch issue via API   1. Issue fetched
2. Click "Run Triage"      2. Analyze content      2. Labels applied
3. See progress            3. Generate spec        3. Comment posted
4. Review results          4. Post results         4. Ready for code gen
```

---

## 💡 Smart Tips

### Save Your Token
```
1. Fill in token field
2. Click "Save"
3. Next time: Click "Load" (auto-fills!)
```

### Batch Multiple Issues
```
1. Triage issue #1 → See results
2. Change issue number to #2
3. Click "Run Triage" again
4. Results update automatically
```

### Quick Copy-to-Code-Gen
```
1. "Copy Spec" button copies generated spec
2. Scroll to LaunchPad (above triage section)
3. Paste spec into "Code Generator" field
4. Click "Run Agent"
5. Code generated automatically
```

---

## 📊 Understanding Results

### Category
What type of issue?
- `feature` = New capability
- `bug` = Problem to fix
- `refactor` = Code improvement
- `documentation` = Docs/wiki

### Priority
How urgent?
- `critical` = Blocks work
- `high` = Important
- `medium` = Standard
- `low` = Nice-to-have

### Ready to Implement
Can we code it?
- `✅ Yes` = Clear requirements, go generate code
- `⏳ No` = Needs more info, check GitHub issue for details

### Labels
Tags applied to issue:
- `ready-to-implement` = Triage passed
- `category-feature` = Type identifier
- `priority-high` = Urgency marker
- `needs-info` = If not ready

---

## 🔧 Troubleshooting

### "HTTP 401" Error
**Problem**: Bad token
**Fix**: Generate new token at github.com/settings/tokens

### "404: Issue not found"
**Problem**: Wrong owner/repo/issue
**Fix**: Check GitHub URL format: github.com/OWNER/REPO/issues/NUMBER

### No Progress/Stuck
**Problem**: Network issue
**Fix**: Refresh page, try again

### Token Not Saving
**Problem**: Browser cache disabled
**Fix**: Manually paste token each time or use password manager

---

## 📁 New Files Added

```
web-ui/src/components/
└── TriagePanel.tsx                    ← Main UI component (400 lines)

Documentation:
├── UI_QUICK_START.md                  ← This file
├── UI_TRIAGE_GUIDE.md                 ← Detailed walkthrough
├── UI_WALKTHROUGH.md                  ← Visual mockups
├── TRIAGE_EXAMPLE.md                  ← API examples
└── HUMAN_IN_THE_LOOP.md              ← Oversight & feedback

Backend:
├── cmd/asf-web/handlers/triage.go    ← Triage logic (300 lines)
└── cmd/asf-web/main.go               ← Added 2 endpoints

Updated:
└── web-ui/src/app/page.tsx           ← Added TriagePanel section
```

---

## 🎓 Learning Path

1. **First time**: Read this file (5 min)
2. **Try it**: Run the 5-minute workflow above
3. **Deep dive**: Read `UI_TRIAGE_GUIDE.md` (10 min)
4. **Understand API**: Read `TRIAGE_EXAMPLE.md` (10 min)
5. **Master feedback**: Read `HUMAN_IN_THE_LOOP.md` (10 min)

---

## ✨ Full Workflow (with Code Generation)

```
┌─────────────┐
│  Dashboard  │
└──────┬──────┘
       │
       ▼
┌──────────────────────┐
│ 1. Fill Triage Form  │ (30 sec)
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ 2. Run Triage        │ (5-10 sec)
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ 3. View Results      │ (1 min reading)
│    Copy Spec         │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ 4. Paste in LaunchPad│ (1 min)
│ 5. Generate Code     │ (10-15 sec)
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ 6. Review Code       │ (2-5 min)
│ 7. Create PR         │
└──────────────────────┘

TOTAL: ~20 minutes for complete issue → PR workflow!
```

---

## 🎯 Real Example

### GitHub Issue Created:
```
Title: Add thread-safe cache

Body:
# Cache Service

## Acceptance Criteria
- Thread-safe operations
- Get, Set, Delete methods
- Full test coverage

## Steps
1. Create Cache struct with RWMutex
2. Implement methods
3. Add tests
```

### Dashboard Triage:
```
Owner: your_username
Repo: your_repo
Issue: 42
Token: ghp_...

[Run Triage]
→ ████████████░░░░░░ 65%
→ Complete!

Results:
Category: feature ✅
Priority: medium ✅
Ready: ✅ Yes
```

### Copy Spec & Generate Code:
```
Paste spec into LaunchPad Code Generator

[Run Agent]
→ Code generated (92% score)
→ 12/12 tests passing
```

### Create PR:
```
Copy code
Push to GitHub
Create pull request
Link to issue #42
```

**DONE!** Issue analyzed → Code generated → PR ready! 🚀

---

## 🤔 FAQ

**Q: Do I need to use the CLI anymore?**
A: Nope! UI does everything. CLI is optional for advanced users.

**Q: Can I save multiple tokens?**
A: Browser local storage saves one. Or use different browser profiles.

**Q: What if triage takes too long?**
A: Might be network issue. Refresh and retry. Should be <10 seconds.

**Q: Can I see what GitHub labels were applied?**
A: Yes! Click "View on GitHub" to see the issue with labels and comment.

**Q: Where does the generated spec come from?**
A: Backend analyzes issue content and creates YAML automatically.

**Q: Can I modify the spec before generating code?**
A: Yes! Copy spec, modify locally, paste modified version back.

---

## 🏁 Next Steps

1. ✅ Start backend and frontend
2. ✅ Try triage on your first GitHub issue
3. ✅ Generate code from the spec
4. ✅ Create your first AI-assisted PR!

Happy building! 🎉

Questions? Check:
- `UI_TRIAGE_GUIDE.md` for detailed walkthrough
- `UI_WALKTHROUGH.md` for visual mockups
- `HUMAN_IN_THE_LOOP.md` for feedback mechanisms
