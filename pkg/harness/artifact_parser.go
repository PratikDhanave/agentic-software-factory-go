package harness

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/workspace"
)

// ParseArtifacts extracts structured FileArtifact entries from an LLM response.
// Supported formats:
//  1. JSON envelope: {"files": [{"path": "foo.go", "action": "create", "content": "..."}]}
//  2. Fenced multi-file blocks:
//       ```go path=pkg/foo/foo.go
//       package foo
//       ```
//     or first-line comment `// file: pkg/foo/foo.go` / `// path: pkg/foo/foo.go`
//  3. Single/multi fenced code blocks without explicit paths (auto-named generated.go / generated_test.go).
func ParseArtifacts(output string) []workspace.FileArtifact {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil
	}

	// 1. Try JSON envelope first
	if jsonArtifacts := tryParseJSONArtifacts(trimmed); len(jsonArtifacts) > 0 {
		return jsonArtifacts
	}

	// 2. Try fenced code blocks
	if strings.Contains(trimmed, "```") {
		var artifacts []workspace.FileArtifact
		parts := strings.Split(trimmed, "```")
		unnamedIdx := 0

		for i := 1; i < len(parts); i += 2 {
			block := parts[i]
			newline := strings.IndexByte(block, '\n')
			if newline < 0 {
				continue
			}
			header := strings.TrimSpace(block[:newline])
			body := block[newline+1:]
			if strings.TrimSpace(body) == "" {
				continue
			}

			path := extractPathFromFenceHeader(header)
			if path == "" {
				path, body = extractLeadingFileComment(body)
			}
			if path == "" {
				// Skip pure json metadata fences if not code
				if strings.EqualFold(header, "json") {
					if a := tryParseJSONArtifacts(body); len(a) > 0 {
						artifacts = append(artifacts, a...)
						continue
					}
				}
				path = inferDefaultGoFilename(body, unnamedIdx)
				unnamedIdx++
			}

			artifacts = append(artifacts, workspace.FileArtifact{
				Path:    path,
				Action:  "create",
				Content: ensurePackageDeclaration(body),
			})
		}
		if len(artifacts) > 0 {
			return artifacts
		}
	}

	// 3. Raw Go source fallback
	if strings.Contains(trimmed, "package ") || strings.Contains(trimmed, "func ") {
		return []workspace.FileArtifact{
			{
				Path:    inferDefaultGoFilename(trimmed, 0),
				Action:  "create",
				Content: ensurePackageDeclaration(trimmed),
			},
		}
	}

	return nil
}

func tryParseJSONArtifacts(text string) []workspace.FileArtifact {
	start := strings.IndexByte(text, '{')
	end := strings.LastIndexByte(text, '}')
	if start < 0 || end <= start {
		return nil
	}
	var envelope struct {
		Files []workspace.FileArtifact `json:"files"`
	}
	if err := json.Unmarshal([]byte(text[start:end+1]), &envelope); err != nil || len(envelope.Files) == 0 {
		return nil
	}
	out := make([]workspace.FileArtifact, 0, len(envelope.Files))
	for _, f := range envelope.Files {
		if strings.TrimSpace(f.Path) == "" {
			continue
		}
		if f.Action == "" {
			f.Action = "create"
		}
		out = append(out, f)
	}
	return out
}

func extractPathFromFenceHeader(header string) string {
	fields := strings.Fields(header)
	for _, f := range fields {
		for _, prefix := range []string{"path=", "file=", "filename="} {
			if strings.HasPrefix(strings.ToLower(f), prefix) {
				return strings.Trim(f[len(prefix):], "\"'`")
			}
		}
		if strings.Contains(f, "/") || strings.HasSuffix(f, ".go") {
			return strings.Trim(f, "\"'`")
		}
	}
	return ""
}

func extractLeadingFileComment(body string) (string, string) {
	lines := strings.Split(body, "\n")
	if len(lines) == 0 {
		return "", body
	}
	first := strings.TrimSpace(lines[0])
	for _, prefix := range []string{"// file:", "// path:", "// filename:"} {
		if strings.HasPrefix(strings.ToLower(first), prefix) {
			p := strings.TrimSpace(first[len(prefix):])
			if p != "" {
				return p, strings.Join(lines[1:], "\n")
			}
		}
	}
	return "", body
}

func inferDefaultGoFilename(code string, idx int) string {
	if strings.Contains(code, "\"testing\"") || strings.Contains(code, "func Test") {
		if idx == 0 {
			return "generated_test.go"
		}
		return fmt.Sprintf("generated_%d_test.go", idx)
	}
	if idx == 0 {
		return "generated.go"
	}
	return fmt.Sprintf("generated_%d.go", idx)
}

func ensurePackageDeclaration(code string) string {
	trimmed := strings.TrimSpace(code)
	for _, line := range strings.Split(trimmed, "\n") {
		l := strings.TrimSpace(line)
		if l == "" || strings.HasPrefix(l, "//") {
			continue
		}
		if strings.HasPrefix(l, "package ") {
			return code
		}
		break
	}
	return "package generated\n\n" + code
}
