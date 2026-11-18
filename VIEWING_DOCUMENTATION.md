# How to View Documentation Properly

The documentation includes Mermaid diagrams which require specific viewers. This guide explains how to view them correctly.

## ✅ Recommended Viewers (Full Support)

### 1. GitHub (Best Option)
- **Platform**: github.com
- **Support**: Full Mermaid support
- **How**: Visit the repo and browse files directly
- **URL**: `https://github.com/PratikDhanave/agentic-software-factory-go`
- **Files to view**:
  - `README_DETAILED.md`
  - `docs/API_FLOWS.md`
  - `docs/DIAGRAMS_INDEX.md`
  - `docs/OBSERVABILITY_GUIDE.md`

### 2. Visual Studio Code (Local)
- **Install**: Free
- **Markdown Preview**: Built-in (Markdown Preview Enhanced)
- **Mermaid Support**: Install "Markdown Preview Mermaid Support" extension
- **How**:
  ```bash
  code README_DETAILED.md
  # Then press Ctrl+Shift+V (or Cmd+Shift+V) for preview
  ```

### 3. Obsidian
- **Install**: Free or paid
- **Mermaid Support**: Built-in with plugins
- **How**:
  1. Download Obsidian
  2. Open the project folder as a vault
  3. Click on files to preview

### 4. GitLab (If hosted there)
- **Platform**: gitlab.com
- **Support**: Full Mermaid support
- **How**: Same as GitHub

### 5. Notion
- **Platform**: notion.so
- **Support**: Limited Mermaid support
- **How**: Copy markdown and paste into Notion

## ⚠️ Viewers That Don't Support Mermaid

| Viewer | Support | Workaround |
|--------|---------|-----------|
| Plain text editor | ❌ No | Use GitHub or VS Code |
| Readme.so | ❌ No | Use GitHub directly |
| StackEdit | ⚠️ Limited | Use GitHub |
| Standard markdown | ❌ No | Install extension |
| Some email clients | ❌ No | Not recommended |

## 🔧 Setup Instructions

### VS Code (Recommended for Local)

```bash
# 1. Install VS Code
# Download from https://code.visualstudio.com/

# 2. Open project
cd /path/to/agentic-software-factory-go
code .

# 3. Install Markdown extensions
# - Open Extensions (Ctrl+Shift+X)
# - Search for "Markdown Preview Mermaid Support"
# - Click Install

# 4. View documentation
# - Open README_DETAILED.md
# - Press Ctrl+Shift+V to preview
```

### Online Markdown Editor

Use these free online tools to view markdown with Mermaid:

1. **Markdown Online Editor**
   - URL: https://pandao.github.io/editor.md/
   - Copy markdown content and paste

2. **Mermaid Live**
   - URL: https://mermaid.live/
   - Copy individual diagrams

3. **HackMD**
   - URL: https://hackmd.io/
   - Paste markdown content

## 📝 Documentation Files

### Core Documentation

| File | Size | Purpose | View With |
|------|------|---------|-----------|
| `README_DETAILED.md` | 29KB | Complete architecture guide with 25+ diagrams | GitHub ⭐ or VS Code |
| `docs/API_FLOWS.md` | 16KB | API operation flows with 9+ diagrams | GitHub ⭐ or VS Code |
| `docs/DIAGRAMS_INDEX.md` | 13KB | Diagram index and reference | GitHub ⭐ or any editor |
| `docs/OBSERVABILITY_GUIDE.md` | 6.7KB | Observability system documentation | GitHub ⭐ or any editor |
| `DOCUMENTATION_SUMMARY.md` | 8.6KB | Documentation overview | GitHub ⭐ or any editor |

### Supporting Documentation

| File | Purpose |
|------|---------|
| `docs/CLEAN_ARCHITECTURE_SUMMARY.md` | Architecture phases and design |
| `docs/ARCHITECTURE.md` | Architecture deep dive |
| `README.md` | Main project documentation |

## 🎯 Viewing Path

### Quick Start (5 minutes)
1. Open GitHub
2. View `README_DETAILED.md` → System Architecture section
3. Look at first 4 diagrams

### Complete Review (30 minutes)
1. Start with `docs/DIAGRAMS_INDEX.md` for overview
2. Follow "Quick Reference" section for topics of interest
3. Jump to corresponding diagrams in `README_DETAILED.md` or `docs/API_FLOWS.md`

### Deep Dive (1-2 hours)
1. Read `DOCUMENTATION_SUMMARY.md` for overview
2. Follow recommended learning path
3. Study each section with diagrams
4. Review code examples

## 🔍 If Diagrams Not Rendering

### Check Your Viewer
```bash
# GitHub should support all Mermaid diagrams
# If not rendering on GitHub:
1. Refresh the page (Ctrl+R or Cmd+R)
2. Clear browser cache
3. Try a different browser
4. Check GitHub status: https://www.githubstatus.com/
```

### Try Alternative Viewer
```bash
# If GitHub isn't working:
1. Download the files locally
2. Open in VS Code with Mermaid extension
3. Or upload to HackMD/MarkdownLive
```

### Common Issues

**Issue**: "Diagram not showing"
- **Solution**: Refresh page or try different browser

**Issue**: "Downloaded .md file won't render"
- **Solution**: Open in VS Code with Mermaid extension

**Issue**: "GitHub shows raw markdown"
- **Solution**: Make sure you're viewing the file, not the raw version

## 📚 Content Organization

### By Topic

**Architecture**:
- `README_DETAILED.md` → System Architecture section
- `docs/CLEAN_ARCHITECTURE_SUMMARY.md`

**Data Flows**:
- `README_DETAILED.md` → Data Flow Diagrams section
- `docs/API_FLOWS.md` → Webhook Processing

**Sequences & Timing**:
- `README_DETAILED.md` → Message Sequence Diagrams section
- `docs/API_FLOWS.md` → Complete flows

**Observability**:
- `README_DETAILED.md` → Observability Architecture section
- `docs/OBSERVABILITY_GUIDE.md` → Full guide

**Performance**:
- `docs/API_FLOWS.md` → Performance Metrics section
- `docs/DIAGRAMS_INDEX.md` → Performance section

## 🎓 Recommended Learning Order

1. **Start Here**: `docs/DIAGRAMS_INDEX.md`
   - Overview of all 35+ diagrams
   - Quick reference by topic

2. **System Design**: `README_DETAILED.md`
   - System Architecture (#1-4)
   - Clean Architecture (#3)

3. **Data Flows**: `README_DETAILED.md`
   - GitHub Webhook Flow (#5)
   - Job Lifecycle (#7)

4. **Implementation**: `docs/API_FLOWS.md`
   - Webhook Processing (#19)
   - Error Scenarios (#21-22)

5. **Observability**: `docs/OBSERVABILITY_GUIDE.md`
   - Complete system overview
   - Implementation details

## 🆘 Support

If diagrams still aren't rendering:

1. **Check your platform**: Does it support Mermaid?
2. **Try GitHub**: Most reliable for Mermaid support
3. **Try VS Code**: Install Mermaid extension
4. **Copy diagrams**: View individual diagrams on mermaid.live

---

**Last Updated**: November 18, 2026  
**Recommended Viewer**: GitHub (Primary) or VS Code (Local)  
**Status**: All diagrams validated and working
