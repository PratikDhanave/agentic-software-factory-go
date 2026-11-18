package workspace

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileArtifact represents a single file operation produced by an agent.
type FileArtifact struct {
	Path    string `json:"path"`
	Action  string `json:"action"` // "create", "modify", "delete"
	Content string `json:"content"`
}

// Workspace represents an isolated task working directory where generated code
// can be materialized, compiled, and tested without mutating the live repo.
type Workspace struct {
	TaskID       string            `json:"task_id"`
	Dir          string            `json:"dir"`
	Strategy     string            `json:"strategy"` // "worktree", "copy", "module"
	FilesWritten []string          `json:"files_written"`
	OriginalSnap map[string]string `json:"-"`
}

// WorkspaceManager provisions and cleans up isolated task workspaces.
type WorkspaceManager struct {
	sourceRoot     string
	workspacesRoot string
}

func NewWorkspaceManager(sourceRoot string) *WorkspaceManager {
	return &WorkspaceManager{
		sourceRoot:     sourceRoot,
		workspacesRoot: filepath.Join(sourceRoot, ".asf", "workspaces"),
	}
}

// CreateModuleWorkspace creates a fast, isolated standalone Go module workspace
// ideal for spec-driven component generation and self-healing build/test loops.
func (m *WorkspaceManager) CreateModuleWorkspace(taskID, modulePath string) (*Workspace, error) {
	wsDir, err := m.prepareDir(taskID)
	if err != nil {
		return nil, err
	}
	if modulePath == "" {
		modulePath = "asf/workspace/" + sanitizeID(taskID)
	}
	goMod := fmt.Sprintf("module %s\n\ngo 1.23\n", modulePath)
	if err := os.WriteFile(filepath.Join(wsDir, "go.mod"), []byte(goMod), 0644); err != nil {
		return nil, fmt.Errorf("writing workspace go.mod: %w", err)
	}
	return &Workspace{
		TaskID:       taskID,
		Dir:          wsDir,
		Strategy:     "module",
		OriginalSnap: make(map[string]string),
	}, nil
}

// CreateRepoWorkspace provisions an isolated copy of sourceRoot using `git worktree`
// when available, falling back to a filtered directory copy.
func (m *WorkspaceManager) CreateRepoWorkspace(ctx context.Context, taskID string) (*Workspace, error) {
	wsDir, err := m.prepareDir(taskID)
	if err != nil {
		return nil, err
	}

	if m.tryGitWorktree(ctx, wsDir) == nil {
		return &Workspace{
			TaskID:       taskID,
			Dir:          wsDir,
			Strategy:     "worktree",
			OriginalSnap: make(map[string]string),
		}, nil
	}

	if err := copyFilteredTree(m.sourceRoot, wsDir); err != nil {
		return nil, fmt.Errorf("copying workspace tree: %w", err)
	}

	return &Workspace{
		TaskID:       taskID,
		Dir:          wsDir,
		Strategy:     "copy",
		OriginalSnap: make(map[string]string),
	}, nil
}

func (m *WorkspaceManager) prepareDir(taskID string) (string, error) {
	safeID := sanitizeID(taskID)
	if safeID == "" {
		safeID = "task"
	}
	if err := os.MkdirAll(m.workspacesRoot, 0755); err != nil {
		return "", fmt.Errorf("creating workspaces root: %w", err)
	}
	wsDir := filepath.Join(m.workspacesRoot, safeID)
	_ = os.RemoveAll(wsDir)
	if err := os.MkdirAll(wsDir, 0755); err != nil {
		return "", fmt.Errorf("creating workspace dir: %w", err)
	}
	return wsDir, nil
}

func (m *WorkspaceManager) tryGitWorktree(ctx context.Context, wsDir string) error {
	gitDir := filepath.Join(m.sourceRoot, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return err
	}
	// Remove empty target dir before git worktree add
	_ = os.RemoveAll(wsDir)
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "--detach", wsDir, "HEAD")
	cmd.Dir = m.sourceRoot
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = os.MkdirAll(wsDir, 0755)
		return fmt.Errorf("git worktree add failed: %w (%s)", err, string(out))
	}
	return nil
}

// Cleanup removes an isolated workspace from disk (and prunes git worktree metadata if needed).
func (m *WorkspaceManager) Cleanup(ctx context.Context, ws *Workspace) error {
	if ws == nil || ws.Dir == "" {
		return nil
	}
	if ws.Strategy == "worktree" {
		cmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", ws.Dir)
		cmd.Dir = m.sourceRoot
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
		_ = cmd.Run()
	}
	return os.RemoveAll(ws.Dir)
}

// ApplyArtifacts safely writes or deletes files inside ws.Dir while preventing
// any path traversal outside the workspace boundary.
func (ws *Workspace) ApplyArtifacts(artifacts []FileArtifact) ([]string, error) {
	absRoot, err := filepath.Abs(ws.Dir)
	if err != nil {
		return nil, err
	}

	written := make([]string, 0, len(artifacts))
	seen := make(map[string]bool)

	for _, art := range artifacts {
		relClean, err := safeRelPath(absRoot, art.Path)
		if err != nil {
			return nil, err
		}
		target := filepath.Join(absRoot, relClean)

		if _, recorded := ws.OriginalSnap[relClean]; !recorded {
			if prev, err := os.ReadFile(target); err == nil {
				ws.OriginalSnap[relClean] = string(prev)
			} else {
				ws.OriginalSnap[relClean] = ""
			}
		}

		if strings.EqualFold(art.Action, "delete") {
			if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
				return nil, fmt.Errorf("deleting %s: %w", relClean, err)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return nil, fmt.Errorf("creating parent dir for %s: %w", relClean, err)
			}
			if err := os.WriteFile(target, []byte(art.Content), 0644); err != nil {
				return nil, fmt.Errorf("writing %s: %w", relClean, err)
			}
		}

		if !seen[relClean] {
			seen[relClean] = true
			written = append(written, relClean)
			ws.FilesWritten = appendUnique(ws.FilesWritten, relClean)
		}
	}

	return written, nil
}

// DiffSummary produces a concise unified-style change summary of all files touched in the workspace.
func (ws *Workspace) DiffSummary() map[string]interface{} {
	changes := make([]map[string]interface{}, 0, len(ws.FilesWritten))
	for _, rel := range ws.FilesWritten {
		target := filepath.Join(ws.Dir, rel)
		afterBytes, err := os.ReadFile(target)
		status := "modified"
		before := ws.OriginalSnap[rel]
		if err != nil {
			status = "deleted"
		} else if before == "" {
			status = "created"
		}
		after := string(afterBytes)
		changes = append(changes, map[string]interface{}{
			"path":        rel,
			"status":      status,
			"lines_added": countLines(after),
			"preview":     truncate(after, 1200),
		})
	}
	return map[string]interface{}{
		"workspace_dir": ws.Dir,
		"strategy":      ws.Strategy,
		"files_changed": len(changes),
		"changes":       changes,
	}
}

func safeRelPath(absRoot, rawPath string) (string, error) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "", fmt.Errorf("artifact file path is empty")
	}
	if filepath.IsAbs(trimmed) {
		return "", fmt.Errorf("absolute artifact path rejected: %s", rawPath)
	}
	clean := filepath.Clean(trimmed)
	if clean == "." || strings.HasPrefix(clean, "..") || strings.Contains(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path traversal rejected: %s", rawPath)
	}
	if clean == ".git" || strings.HasPrefix(clean, ".git"+string(filepath.Separator)) {
		return "", fmt.Errorf("modifying .git metadata is forbidden: %s", rawPath)
	}
	joined := filepath.Join(absRoot, clean)
	rel, err := filepath.Rel(absRoot, joined)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("artifact escapes workspace root: %s", rawPath)
	}
	return rel, nil
}

func copyFilteredTree(srcRoot, dstRoot string) error {
	skipDirs := map[string]bool{
		".git":         true,
		".asf":         true,
		"node_modules": true,
		".next":        true,
		"vendor":       true,
	}

	return filepath.WalkDir(srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil || rel == "." {
			return err
		}
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dstRoot, rel), 0755)
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstRoot, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func sanitizeID(id string) string {
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n... [truncated]"
}
