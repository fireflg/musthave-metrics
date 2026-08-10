package main

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
)

func runAnalyzer(t *testing.T, a *analysis.Analyzer, src string) []string {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "src.go", src, parser.ParseComments)
	require.NoError(t, err)

	info := &types.Info{
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Types:      map[ast.Expr]types.TypeAndValue{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}
	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	pkg, err := conf.Check("testpkg", fset, []*ast.File{f}, info)
	require.NoError(t, err)

	var diags []string
	pass := &analysis.Pass{
		Analyzer:  a,
		Fset:      fset,
		Files:     []*ast.File{f},
		Pkg:       pkg,
		TypesInfo: info,
		Report:    func(d analysis.Diagnostic) { diags = append(diags, d.Message) },
		ResultOf:  map[*analysis.Analyzer]interface{}{},
	}

	_, err = a.Run(pass)
	require.NoError(t, err)
	return diags
}

func TestOsExitAnalyzer_ReportsExitInMain(t *testing.T) {
	diags := runAnalyzer(t, OsExitAnalyzer, `package main

import "os"

func main() {
	os.Exit(1)
}
`)

	require.Len(t, diags, 1)
	assert.Contains(t, diags[0], "os.Exit() is not allowed in main")
}

func TestOsExitAnalyzer_AllowsExitOutsideMain(t *testing.T) {
	diags := runAnalyzer(t, OsExitAnalyzer, `package main

import "os"

func shutdown() {
	os.Exit(1)
}

func main() {
	shutdown()
}
`)

	assert.Empty(t, diags)
}

func TestOsExitAnalyzer_IgnoresShadowedPackage(t *testing.T) {
	diags := runAnalyzer(t, OsExitAnalyzer, `package main

type fakeOS struct{}

func (fakeOS) Exit(int) {}

func main() {
	os := fakeOS{}
	os.Exit(1)
}
`)

	assert.Empty(t, diags, "вызов метода на переменной с именем os не должен считаться os.Exit")
}

func TestNamingAnalyzer(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "функция с подчёркиванием",
			src: `package a

func Bad_Name() {}
`,
			want: `exported function "Bad_Name" should use PascalCase`,
		},
		{
			name: "тип с подчёркиванием",
			src: `package a

type Bad_Type struct{}
`,
			want: `exported type "Bad_Type" should use PascalCase`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := runAnalyzer(t, NamingAnalyzer, tt.src)
			require.Len(t, diags, 1)
			assert.Equal(t, tt.want, diags[0])
		})
	}
}

func TestNamingAnalyzer_AcceptsValidNames(t *testing.T) {
	diags := runAnalyzer(t, NamingAnalyzer, `package a

type GoodType struct{}

func GoodFunc() {}

var GoodVar int = 1

const GoodConst = 2

func unexportedFunc() {}
`)

	assert.Empty(t, diags)
}

func TestNamingAnalyzer_SkipsNamesWithUnderscoreInValueSpec(t *testing.T) {
	diags := runAnalyzer(t, NamingAnalyzer, `package a

const MAX_SIZE = 10
`)

	assert.Empty(t, diags, "имена с подчёркиванием в const/var исключены намеренно")
}

func TestContextAnalyzer_ReportsMissingContext(t *testing.T) {
	diags := runAnalyzer(t, ContextAnalyzer, `package a

func QueryUsers() {}
`)

	require.Len(t, diags, 1)
	assert.Contains(t, diags[0], `function "QueryUsers" should have context.Context parameter`)
}

func TestContextAnalyzer_AcceptsContextParam(t *testing.T) {
	diags := runAnalyzer(t, ContextAnalyzer, `package a

import "context"

func QueryUsers(ctx context.Context) {}

func UpdateItem(ctx *context.Context) {}

func unrelated() {}
`)

	assert.Empty(t, diags)
}

func TestContextAnalyzer_IgnoresFunctionsWithoutDBPatterns(t *testing.T) {
	diags := runAnalyzer(t, ContextAnalyzer, `package a

func Helper() {}
`)

	assert.Empty(t, diags)
}

func TestContextVisitorHelpers(t *testing.T) {
	fset := token.NewFileSet()
	src := `package a

import "context"

func WithCtx(ctx context.Context) {}
func WithPtrCtx(ctx *context.Context) {}
func NoParams() {}
func Other(n int) {}
`
	f, err := parser.ParseFile(fset, "src.go", src, 0)
	require.NoError(t, err)

	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}}
	conf := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
	pkg, err := conf.Check("a", fset, []*ast.File{f}, info)
	require.NoError(t, err)

	v := &contextVisitor{info: info}

	decls := map[string]*ast.FuncDecl{}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok {
			decls[fn.Name.Name] = fn
		}
	}

	assert.True(t, v.hasContextParam(decls["WithCtx"]))
	assert.True(t, v.hasContextParam(decls["WithPtrCtx"]))
	assert.False(t, v.hasContextParam(decls["NoParams"]))
	assert.False(t, v.hasContextParam(decls["Other"]))

	assert.True(t, v.funcHasContextParam(nil, decls["WithPtrCtx"]))
	assert.False(t, v.funcHasContextParam(nil, decls["NoParams"]))

	ctxType := pkg.Scope().Lookup("WithCtx").Type().(*types.Signature).Params().At(0).Type()
	assert.True(t, v.isContextType(nil, ctxType))
	assert.False(t, v.isContextType(nil, types.Typ[types.Int]))
}

func TestIsUpperAndKindOf(t *testing.T) {
	assert.True(t, isUpper('A'))
	assert.False(t, isUpper('a'))

	assert.Equal(t, "variable", kindOf(&ast.ValueSpec{Type: ast.NewIdent("int")}))
	assert.Equal(t, "constant", kindOf(&ast.ValueSpec{}))
}
