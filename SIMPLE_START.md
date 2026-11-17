# 🚀 SIMPLE START - 30 SECONDS

## Step 1: Get Token
Go to: https://github.com/settings/tokens
- Click "Generate new token (classic)"
- Check: `repo` and `write:discussion`
- Copy token

## Step 2: Terminal 1
```bash
export GITHUB_TOKEN=ghp_your_token_here
./asf-web localhost:8080
```

## Step 3: Terminal 2
```bash
cd web-ui && npm run dev
```

## Step 4: Open Browser
```
http://localhost:3000
```

---

## 💡 How to Use

```
1. Enter GitHub Owner (e.g., your_username)
2. Enter Repository (e.g., your_repo)
3. Enter Issue Number (e.g., 1)
4. Token: Auto-loaded ✓
5. Click: Run Triage
6. See results (color-coded)
```

---

## 🎯 What Happens

```
Issue → Analyzed → Categorized → Labeled → Posted to GitHub
```

That's it! 🎉
