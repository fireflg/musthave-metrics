package main

import (
	"go/ast"
	"go/types"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
)

var (
	OsExitAnalyzer = &analysis.Analyzer{
		Name:     "osexitanalyzer",
		Doc:      "disallows direct os.Exit() calls in main function",
		URL:      "https://github.com/fireflg/go-musthave-metrics-tpl",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      runOsExit,
	}

	NamingAnalyzer = &analysis.Analyzer{
		Name:     "naminganalyzer",
		Doc:      "checks naming conventions (PascalCase for exported identifiers)",
		URL:      "https://github.com/fireflg/go-musthave-metrics-tpl",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      runNaming,
	}

	ContextAnalyzer = &analysis.Analyzer{
		Name:     "contextanalyzer",
		Doc:      "checks for context.Context in functions that should have it",
		URL:      "https://github.com/fireflg/go-musthave-metrics-tpl",
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      runContext,
	}
)

type osExitCallVisitor struct {
	file     *ast.File
	pass     *analysis.Pass
	isMain   bool
	funcName string
}

func (v *osExitCallVisitor) visit(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.FuncDecl:
		v.isMain = n.Name.Name == "main"
		v.funcName = n.Name.Name
		return true
	case *ast.CallExpr:
		if v.isMain {
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					if obj, ok := v.pass.TypesInfo.Uses[ident]; ok {
						if pkgName, ok := obj.(*types.PkgName); ok {
							if pkgName.Imported().Path() == "os" && sel.Sel.Name == "Exit" {
								v.pass.Reportf(n.Pos(), "os.Exit() is not allowed in main, use log.Fatalf or return with error code")
							}
						}
					}
				}
			}
		}
	}
	return true
}

func runOsExit(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Files {
		visitor := &osExitCallVisitor{file: f, pass: pass}
		ast.Inspect(f, visitor.visit)
	}
	return nil, nil
}

type namingVisitor struct {
	pass     *analysis.Pass
	pascalRE *regexp.Regexp
	camelRE  *regexp.Regexp
}

func (v *namingVisitor) visit(node ast.Node) {
	switch n := node.(type) {
	case *ast.FuncDecl:
		if len(n.Name.Name) > 0 && isUpper(n.Name.Name[0]) {
			if !v.pascalRE.MatchString(n.Name.Name) {
				v.pass.Reportf(n.Pos(), "exported function %q should use PascalCase", n.Name.Name)
			}
		}
	case *ast.TypeSpec:
		if len(n.Name.Name) > 0 && isUpper(n.Name.Name[0]) {
			if !v.pascalRE.MatchString(n.Name.Name) {
				v.pass.Reportf(n.Pos(), "exported type %q should use PascalCase", n.Name.Name)
			}
		}
	case *ast.ValueSpec:
		for _, name := range n.Names {
			if len(name.Name) > 0 && isUpper(name.Name[0]) {
				if !v.pascalRE.MatchString(name.Name) && !strings.Contains(name.Name, "_") {
					v.pass.Reportf(n.Pos(), "exported %s %q should use PascalCase", kindOf(n), name.Name)
				}
			}
		}
	}
}

func isUpper(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func kindOf(n *ast.ValueSpec) string {
	if n.Type != nil {
		return "variable"
	}
	return "constant"
}

func runNaming(pass *analysis.Pass) (interface{}, error) {
	pascalRE, err := regexp.Compile(`^[A-Z][a-zA-Z0-9]*$`)
	if err != nil {
		return nil, err
	}
	camelRE, err := regexp.Compile(`^[a-z][a-zA-Z0-9]*$`)
	if err != nil {
		return nil, err
	}

	v := &namingVisitor{
		pass:     pass,
		pascalRE: pascalRE,
		camelRE:  camelRE,
	}

	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			v.visit(n)
			return true
		})
	}
	return nil, nil
}

type contextVisitor struct {
	pass         *analysis.Pass
	info         *types.Info
	knownCtxFunc map[string]bool
}

func (v *contextVisitor) isContextType(pass *analysis.Pass, t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "context" && obj.Name() == "Context"
}

func (v *contextVisitor) funcHasContextParam(pass *analysis.Pass, decl *ast.FuncDecl) bool {
	if decl.Type.Params == nil {
		return false
	}
	for _, field := range decl.Type.Params.List {
		for _, name := range field.Names {
			if name.Name == "ctx" || name.Name == "context" {
				if v.info != nil && field.Type != nil {
					if basic, ok := field.Type.(*ast.StarExpr); ok {
						if sel, ok := basic.X.(*ast.SelectorExpr); ok {
							if ident, ok := sel.X.(*ast.Ident); ok {
								if ident.Name == "context" && sel.Sel.Name == "Context" {
									return true
								}
							}
						}
					}
				}
			}
		}
	}
	return false
}

func (v *contextVisitor) visit(node ast.Node) {
	switch n := node.(type) {
	case *ast.FuncDecl:
		if v.shouldHaveContext(n) && !v.hasContextParam(n) {
			v.pass.Reportf(n.Pos(), "function %q should have context.Context parameter", n.Name.Name)
		}
	}
}

func (v *contextVisitor) shouldHaveContext(decl *ast.FuncDecl) bool {
	if decl.Name == nil {
		return false
	}
	name := decl.Name.Name

	dbPatterns := []string{"DB", "Tx", "Query", "Exec", "Insert", "Update", "Delete", "Select"}
	for _, pattern := range dbPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}

	return false
}

func (v *contextVisitor) hasContextParam(decl *ast.FuncDecl) bool {
	if decl.Type.Params == nil {
		return false
	}
	for _, field := range decl.Type.Params.List {
		if ident, ok := field.Type.(*ast.Ident); ok {
			if ident.Name == "Context" {
				return true
			}
		}
		if star, ok := field.Type.(*ast.StarExpr); ok {
			if sel, ok := star.X.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					if ident.Name == "context" && sel.Sel.Name == "Context" {
						return true
					}
				}
			}
		}
	}
	return false
}

func runContext(pass *analysis.Pass) (interface{}, error) {
	v := &contextVisitor{
		pass:         pass,
		info:         pass.TypesInfo,
		knownCtxFunc: make(map[string]bool),
	}

	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			v.visit(n)
			return true
		})
	}
	return nil, nil
}
