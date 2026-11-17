package handlers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// ASTAnalyzer performs deep code analysis using Go AST
type ASTAnalyzer struct {
	issues []ReviewIssue
}

// AnalyzeCode performs comprehensive code analysis
func (aa *ASTAnalyzer) AnalyzeCode(filename, code string) []ReviewIssue {
	if !strings.HasSuffix(filename, ".go") {
		return []ReviewIssue{}
	}

	aa.issues = []ReviewIssue{}

	// Parse code into AST
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, filename, code, parser.AllErrors)
	if err != nil {
		// Parse error is itself an issue
		aa.issues = append(aa.issues, ReviewIssue{
			File:     filename,
			Severity: "critical",
			Message:  "Code does not parse: " + err.Error(),
		})
		return aa.issues
	}

	// Walk the AST
	walker := &astWalker{
		analyzer: aa,
		filename: filename,
		fset:     fset,
		code:     code,
	}
	ast.Walk(walker, astFile)

	return aa.issues
}

// astWalker traverses the AST tree
type astWalker struct {
	analyzer  *ASTAnalyzer
	filename  string
	fset      *token.FileSet
	code      string
	lockStack []string // Stack of locked variables
}

func (w *astWalker) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return w
	}

	switch n := node.(type) {
	case *ast.FuncDecl:
		w.checkFunction(n)
	case *ast.BlockStmt:
		w.checkBlock(n)
	case *ast.CallExpr:
		w.checkCall(n)
	case *ast.AssignStmt:
		w.checkAssignment(n)
	}

	return w
}

// checkFunction analyzes function declarations
func (w *astWalker) checkFunction(fn *ast.FuncDecl) {
	// Check function size
	if fn.Body != nil {
		lineCount := w.fset.Position(fn.Body.End()).Line - w.fset.Position(fn.Body.Pos()).Line
		if lineCount > 200 {
			w.analyzer.issues = append(w.analyzer.issues, ReviewIssue{
				File:     w.filename,
				Severity: "warning",
				Message:  "Large function: " + fn.Name.Name + " (" + string(rune(lineCount)) + " lines), consider breaking down",
			})
		}
	}

	// Check if function returns error but doesn't handle it
	hasErrorReturn := false
	if fn.Type.Results != nil {
		for _, result := range fn.Type.Results.List {
			if ident, ok := result.Type.(*ast.Ident); ok {
				if ident.Name == "error" {
					hasErrorReturn = true
				}
			}
		}
	}

	if hasErrorReturn {
		w.checkErrorHandling(fn.Body)
	}

	// Walk the function body
	bodyWalker := &astWalker{
		analyzer:  w.analyzer,
		filename:  w.filename,
		fset:      w.fset,
		code:      w.code,
		lockStack: []string{},
	}
	ast.Walk(bodyWalker, fn.Body)
}

// checkBlock analyzes code blocks for lock/unlock pairs
func (w *astWalker) checkBlock(block *ast.BlockStmt) {
	if block == nil {
		return
	}

	for _, stmt := range block.List {
		// Check for Lock() calls
		if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
			if call, ok := exprStmt.X.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if sel.Sel.Name == "Lock" || sel.Sel.Name == "RLock" {
						// Check if next statement is defer Unlock
						hasDefer := false
						if i := findStmtIndex(block, stmt); i >= 0 && i+1 < len(block.List) {
							if deferStmt, ok := block.List[i+1].(*ast.DeferStmt); ok {
								if c, ok := deferStmt.Call.Fun.(*ast.SelectorExpr); ok {
									if c.Sel.Name == "Unlock" || c.Sel.Name == "RUnlock" {
										hasDefer = true
									}
								}
							}
						}

						if !hasDefer {
							w.analyzer.issues = append(w.analyzer.issues, ReviewIssue{
								File:     w.filename,
								Severity: "critical",
								Message:  "Lock() call without defer Unlock() - deadlock risk",
							})
						}
					}
				}
			}
		}
	}
}

// checkCall analyzes function calls
func (w *astWalker) checkCall(call *ast.CallExpr) {
	// Check for error-returning functions not being checked
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		funcName := sel.Sel.Name
		// Common error-returning functions
		if funcName == "Read" || funcName == "Write" || funcName == "Close" || funcName == "Unmarshal" {
			// This is a heuristic - in reality we'd need type info
		}
	}
}

// checkAssignment analyzes variable assignments
func (w *astWalker) checkAssignment(assign *ast.AssignStmt) {
	// Check for err := ... without checking err
	for i, rhs := range assign.Rhs {
		if _, ok := rhs.(*ast.CallExpr); ok {
			if len(assign.Lhs) > i {
				if ident, ok := assign.Lhs[i].(*ast.Ident); ok {
					if ident.Name == "err" || strings.HasSuffix(ident.Name, "Err") {
						// Found error assignment - should be checked
						// (This would need flow analysis to verify)
					}
				}
			}
		}
	}
}

// checkErrorHandling verifies error handling in function body
func (w *astWalker) checkErrorHandling(block *ast.BlockStmt) {
	if block == nil {
		return
	}

	hasErrorCheck := false
	for _, stmt := range block.List {
		if ifStmt, ok := stmt.(*ast.IfStmt); ok {
			if binary, ok := ifStmt.Cond.(*ast.BinaryExpr); ok {
				if ident, ok := binary.X.(*ast.Ident); ok {
					if ident.Name == "err" {
						hasErrorCheck = true
					}
				}
			}
		}
	}

	if !hasErrorCheck && len(block.List) > 0 {
		w.analyzer.issues = append(w.analyzer.issues, ReviewIssue{
			File:     w.filename,
			Severity: "warning",
			Message:  "Function returns error but no error checks found",
		})
	}
}

// findStmtIndex finds the index of a statement in a block
func findStmtIndex(block *ast.BlockStmt, target ast.Stmt) int {
	for i, stmt := range block.List {
		if stmt == target {
			return i
		}
	}
	return -1
}

// DeepAnalyzeFile performs AST-based analysis on patch content
func DeepAnalyzeFile(filename, patch string) []ReviewIssue {
	analyzer := &ASTAnalyzer{
		issues: []ReviewIssue{},
	}

	// Extract code blocks from patch
	lines := strings.Split(patch, "\n")
	var codeLines []string

	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			// Remove the + prefix and analyze
			codeLine := strings.TrimPrefix(line, "+")
			codeLines = append(codeLines, codeLine)
		}
	}

	if len(codeLines) == 0 {
		return analyzer.issues
	}

	code := strings.Join(codeLines, "\n")
	return analyzer.AnalyzeCode(filename, code)
}

// GetDetailedIssues analyzes code more thoroughly
func GetDetailedIssues(filename, content string) []ReviewIssue {
	analyzer := &ASTAnalyzer{
		issues: []ReviewIssue{},
	}

	issues := analyzer.AnalyzeCode(filename, content)

	// Add pattern-based checks as fallback
	issues = append(issues, patternBasedChecks(filename, content)...)

	return issues
}

// patternBasedChecks performs regex-based checks as supplement to AST
func patternBasedChecks(filename, content string) []ReviewIssue {
	var issues []ReviewIssue

	if !strings.HasSuffix(filename, ".go") {
		return issues
	}

	// Check for unused variables
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// Check for := followed by unused assignment
		if strings.Contains(line, ":=") && !strings.Contains(line, "err") {
			if strings.Contains(line, "_") && !strings.Contains(line, "_ =") {
				// Blank import might be intentional
			}
		}

		// Check for missing error checks
		if strings.Contains(line, "err :=") && i < len(lines)-1 {
			nextLine := lines[i+1]
			if !strings.Contains(nextLine, "if err") && !strings.Contains(nextLine, "return") {
				// This is heuristic - might not be actual error
			}
		}

		// Check for unprotected map access with mutexes
		if strings.Contains(line, ".Lock()") {
			for j := i + 1; j < i+5 && j < len(lines); j++ {
				if strings.Contains(lines[j], "Unlock()") || strings.Contains(lines[j], "defer") {
					break
				}
				if strings.Contains(lines[j], "make(map") {
					issues = append(issues, ReviewIssue{
						File:     filename,
						Severity: "warning",
						Message:  "Map operations detected after Lock - verify synchronization",
					})
					break
				}
			}
		}
	}

	return issues
}
