// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package modernize

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
	"golang.org/x/tools/internal/analysisinternal"
)

type ifWalker struct {
	info *types.Info
	call *ast.CallExpr // the call inside IfStmt

	// varName is the variable name used in IfStmt Init
	varName   string
	bodyEdits []analysis.TextEdit
}

// allEdits return all text edits for an IfStmt.
func (iw *ifWalker) allEdits() []analysis.TextEdit {
	sel := iw.call.Fun.(*ast.SelectorExpr).Sel

	return append([]analysis.TextEdit{
		{
			Pos:     iw.call.Fun.Pos(),
			End:     iw.call.Fun.Pos(),
			NewText: []byte(fmt.Sprintf("%s,ok :=", iw.varName)),
		},
		{
			Pos:     sel.Pos(),
			End:     sel.End(), // keep the original arguments
			NewText: []byte("CutPrefix"),
		},
		{
			Pos:     iw.call.End(),
			End:     iw.call.End(),
			NewText: []byte(";ok"),
		},
	}, iw.bodyEdits...)
}

func (iw *ifWalker) Visit(node ast.Node) (w ast.Visitor) {
	switch clause := node.(type) {
	case *ast.CallExpr:
		obj1 := typeutil.Callee(iw.info, clause)
		if !analysisinternal.IsFunctionNamed(obj1, "strings", "TrimPrefix") &&
			!analysisinternal.IsFunctionNamed(obj1, "bytes", "TrimPrefix") {
			return iw
		}
		var (
			s0   = iw.call.Args[0]
			pre0 = iw.call.Args[1]
			s    = clause.Args[0]
			pre  = clause.Args[1]
		)
		// check whether the obj1 uses the exact the same argument with strings.HasPrefix
		if equalSyntax(s0, s) && equalSyntax(pre0, pre) {
			iw.bodyEdits = append(iw.bodyEdits, analysis.TextEdit{
				Pos:     clause.Pos(),
				End:     clause.End(),
				NewText: []byte(iw.varName),
			})
		}

	default:
	}
	return iw
}

// The stringscutprefix offers a fix to replace an if statement which
// calls to the 2 patterns below with strings.CutPrefix.
//
// Patterns:
//
//  1. if strings.HasPrefix(s, pre) { use(strings.TrimPrefix(s, pre) }
//     => if after, ok := strings.CutPrefix(s, pre); ok { use(after) }
//  2. if varName := strings.TrimPrefix(s, pre); varName != s { use(varName) }
//     => if after, ok := strings.CutPrefix(s, pre); ok { use(after) }
//
// Variants:
// - bytes.HasPrefix usage as pattern 1.
func stringscutprefix(pass *analysis.Pass) {
	if !analysisinternal.Imports(pass.Pkg, "strings") &&
		!analysisinternal.Imports(pass.Pkg, "bytes") {
		return
	}

	const (
		category     = "stringscutprefix"
		message      = "if statement can be modernized using strings.CutPrefix"
		fixedMessage = "Replace if statement with CutPrefix"
	)
	info := pass.TypesInfo
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	// CutPrefix was added since go1.20
	for curFile := range filesUsing(inspect, pass.TypesInfo, "go1.20") {
		for curIfStmt := range curFile.Preorder((*ast.IfStmt)(nil)) {
			ifStmt := curIfStmt.Node().(*ast.IfStmt)

			// findNewVarName tries to find a new non-defined variable name to
			// avoid overriding a variable defined outside.
			// Otherwise, the assignment inside if stmt won't take effect in the next stmt.
			findNewVarName := func(varName string) string {
				scope := info.Scopes[ifStmt]
				if scope == nil {
					return varName
				}
				// defined checks whether a variable has been defined before the if stmt,
				// and all its parent scopes.
				defined := func(varName string) bool {
					_, obj := scope.LookupParent(varName, ifStmt.Pos())
					return obj != nil
				}

				newName := varName
				// check whether the selected variable has been defined.
				if defined(newName) {
					// we will try at most 10 time and will use the last one
					// even though it conflicts with the variable defined before.
					for i := range 10 {
						newName = fmt.Sprintf("%s%d", varName, i)
						if !defined(newName) {
							break
						}
					}
				}
				return newName
			}

			// pattern1
			if call, ok := ifStmt.Cond.(*ast.CallExpr); ok {
				obj := typeutil.Callee(info, call)
				if !analysisinternal.IsFunctionNamed(obj, "strings", "HasPrefix") &&
					!analysisinternal.IsFunctionNamed(obj, "bytes", "HasPrefix") {
					return
				}
				varName := findNewVarName("after")

				if ifStmt.Init != nil &&
					analysisinternal.IsFunctionNamed(obj, "strings", "TrimPrefix") ||
					analysisinternal.IsFunctionNamed(obj, "bytes", "TrimPrefix") {
					assign, ok := ifStmt.Init.(*ast.AssignStmt)
					if ok {
						varName = assign.Lhs[0].(*ast.Ident).Name
					}
				}

				w := &ifWalker{
					varName: varName,
					info:    info,
					call:    call,
				}
				ast.Walk(w, ifStmt.Body)
				if w.bodyEdits != nil {
					pass.Report(analysis.Diagnostic{
						// highlight at string.HasPrefix,
						// if strings.HasPrefix(s, pre)
						// -->
						// if leftVar, ok := strings.CutPrefix(s, pre); ok
						Pos:      call.Pos(),
						End:      call.End(),
						Category: category,
						Message:  message,
						SuggestedFixes: []analysis.SuggestedFix{{
							Message:   fixedMessage,
							TextEdits: w.allEdits(),
						}},
					})
				}
			}

			// pattern2
			if bin, ok := ifStmt.Cond.(*ast.BinaryExpr); ok && bin.Op == token.NEQ {
				// handle case:
				// if after := strings.TrimPrefix(s, pre); after != s
				//
				// but it doesn't handle case:
				// if after, another := strings.TrimPrefix(s, pre), ""; after != s
				if ifStmt.Init != nil && isSimpleAssign(ifStmt.Init) {
					// isSimpleAssign guards the assert so checking is unnecessary.
					assign := ifStmt.Init.(*ast.AssignStmt)
					if trimPrefixCall, ok := assign.Rhs[0].(*ast.CallExpr); ok {
						definedVar := assign.Lhs[0]
						obj := typeutil.Callee(info, trimPrefixCall)
						if analysisinternal.IsFunctionNamed(obj, "strings", "TrimPrefix") &&
							// ensure the defined var and the first arg of TrimPrefix are used to compare,
							// the other cases are not suitable for this case.
							(equalSyntax(definedVar, bin.X) && equalSyntax(trimPrefixCall.Args[0], bin.Y) ||
								(equalSyntax(definedVar, bin.Y) && equalSyntax(trimPrefixCall.Args[0], bin.X))) {

							varName := definedVar.(*ast.Ident).Name
							var additionalEdits []analysis.TextEdit
							// we should find another name rather than override it inside if stmt,
							// otherwise the later siblings of if stmt cannot access the desired value
							// of this variable.
							if assign.Tok == token.ASSIGN {
								varName = findNewVarName(varName)
								pos := ifStmt.Body.End()
								if l := len(ifStmt.Body.List); l != 0 {
									pos = ifStmt.Body.List[0].Pos()
								}
								// assign the original value with the new defined value
								additionalEdits = append(additionalEdits, analysis.TextEdit{
									Pos:     pos,
									End:     pos,
									NewText: fmt.Appendf(nil, "%s = %s\n", definedVar.(*ast.Ident).Name, varName),
								})
							}

							// because users has defined his own variable, we should reuse this.
							pass.Report(analysis.Diagnostic{
								// highlight from the init and the condition end
								Pos:      ifStmt.Init.Pos(),
								End:      ifStmt.Cond.End(),
								Category: category,
								Message:  message,
								SuggestedFixes: []analysis.SuggestedFix{{
									Message: fixedMessage,
									TextEdits: append([]analysis.TextEdit{
										{
											Pos:     ifStmt.Pos(),
											End:     trimPrefixCall.Fun.End(),
											NewText: []byte(fmt.Sprintf("if %s,ok := strings.CutPrefix", varName)),
										},
										{
											Pos:     ifStmt.Cond.Pos(),
											End:     ifStmt.Cond.End(),
											NewText: []byte("ok"),
										},
									}, additionalEdits...),
								}},
							})
						}
					}
				}
			}
		}
	}
}
