package main

import (
	"go/ast"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator_ScanPackages(t *testing.T) {
	t.Parallel()

	gen := NewGenerator()
	err := gen.ScanPackages([]string{"./testdata/simple"})
	require.NoError(t, err)
	assert.NotEmpty(t, gen.packages)
}

func TestGenerator_FindMarkedStructs(t *testing.T) {
	t.Parallel()

	gen := NewGenerator()
	err := gen.ScanPackages([]string{"./testdata/simple"})
	require.NoError(t, err)

	structs := gen.FindMarkedStructs()
	assert.Len(t, structs, 1)
	assert.Equal(t, "TestStruct", structs[0].Name)
}

func TestGenerator_GenerateResetMethod(t *testing.T) {
	t.Parallel()

	gen := NewGenerator()
	err := gen.ScanPackages([]string{"./testdata/simple"})
	require.NoError(t, err)

	structs := gen.FindMarkedStructs()
	require.Len(t, structs, 1)

	code, err := gen.GenerateResetMethod(structs[0])
	require.NoError(t, err)

	content := string(code)
	assert.Contains(t, content, "func (rs *TestStruct) Reset()")
	assert.Contains(t, content, "rs.Counter = 0")
	assert.Contains(t, content, "rs.Name = \"\"")
	assert.Contains(t, content, "rs.Active = false")
	assert.Contains(t, content, "rs.Tags = rs.Tags[:0]")
	assert.Contains(t, content, "clear(rs.Data)")
	assert.Contains(t, content, "rs.Values = rs.Values[:0]")
}

func TestGenerator_GenerateCode(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(testFile, []byte(`package test

// MyStruct is a test struct.
// generate:reset
type MyStruct struct {
	Value int
	Name  string
}
`), 0o644)
	require.NoError(t, err)

	gen := NewGenerator()
	err = gen.ScanPackages([]string{tmpDir})
	require.NoError(t, err)

	structs := gen.FindMarkedStructs()
	require.Len(t, structs, 1)

	structs[0].Directory = tmpDir
	structs[0].Package = "test"

	err = gen.GenerateCode(structs)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "reset.gen.go")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "package test")
	assert.Contains(t, contentStr, "*MyStruct) Reset()")
	assert.Contains(t, contentStr, "rs.Value = 0")
}

func TestGenerator_NoStructsFound(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(testFile, []byte(`package test

type MyStruct struct {
	Value int
}
`), 0o644)
	require.NoError(t, err)

	gen := NewGenerator()
	err = gen.ScanPackages([]string{tmpDir})
	require.NoError(t, err)

	structs := gen.FindMarkedStructs()
	assert.Len(t, structs, 0)
}

func TestGenerateIdentReset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{"int", "int", " = 0", false},
		{"int64", "int64", " = 0", false},
		{"float64", "float64", " = 0", false},
		{"string", "string", ` = ""`, false},
		{"bool", "bool", " = false", false},
		{"byte", "byte", " = 0", false},
		{"custom struct", "CustomStruct", ".Reset()", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := generateIdentReset(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGeneratePointerFieldReset(t *testing.T) {
	t.Parallel()

	ident := &ast.Ident{Name: "int"}
	code, err := generatePointerFieldReset("Field", ident)
	assert.NoError(t, err)
	assert.NotEmpty(t, code)
	assert.Contains(t, code, "rs.Field")

	customIdent := &ast.Ident{Name: "CustomStruct"}
	code, err = generatePointerFieldReset("Field", customIdent)
	assert.NoError(t, err)
	assert.Contains(t, code, "rs.Field.Reset()")
}

func TestHasGenerateResetComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		comments []string
		expected bool
	}{
		{"has marker", []string{"// generate:reset"}, true},
		{"has marker with prefix", []string{"// some comment", "// generate:reset"}, true},
		{"no marker", []string{"// some comment"}, false},
		{"empty", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var doc *ast.CommentGroup
			if tt.comments != nil {
				list := make([]*ast.Comment, len(tt.comments))
				for i, c := range tt.comments {
					list[i] = &ast.Comment{Text: c}
				}
				doc = &ast.CommentGroup{List: list}
			}

			got := hasGenerateResetComment(doc)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestExtractFields(t *testing.T) {
	t.Parallel()

	gen := NewGenerator()
	err := gen.ScanPackages([]string{"./testdata/simple"})
	require.NoError(t, err)

	structs := gen.FindMarkedStructs()
	require.Len(t, structs, 1)

	fields := structs[0].Fields
	assert.Len(t, fields, 6)

	fieldNames := make([]string, len(fields))
	for i, f := range fields {
		fieldNames[i] = f.Name
	}
	assert.Contains(t, fieldNames, "Counter")
	assert.Contains(t, fieldNames, "Name")
	assert.Contains(t, fieldNames, "Active")
	assert.Contains(t, fieldNames, "Tags")
	assert.Contains(t, fieldNames, "Data")
	assert.Contains(t, fieldNames, "Values")
}

func TestGenerator_MultipleStructs(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(testFile, []byte(`package test

// FirstStruct is first.
// generate:reset
type FirstStruct struct {
	Value int
}

// SecondStruct is second.
// generate:reset
type SecondStruct struct {
	Count int
}
`), 0o644)
	require.NoError(t, err)

	gen := NewGenerator()
	err = gen.ScanPackages([]string{tmpDir})
	require.NoError(t, err)

	structs := gen.FindMarkedStructs()
	assert.Len(t, structs, 2)

	for _, s := range structs {
		s.Directory = tmpDir
		s.Package = "test"
	}

	err = gen.GenerateCode(structs)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "reset.gen.go")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "FirstStruct) Reset()")
	assert.Contains(t, contentStr, "SecondStruct) Reset()")
	assert.Contains(t, contentStr, "rs.Value = 0")
	assert.Contains(t, contentStr, "rs.Count = 0")
}

func TestGenerator_SkipTestFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_test.go")
	err := os.WriteFile(testFile, []byte(`package test

// TestStruct is a test struct.
// generate:reset
type TestStruct struct {
	Value int
}
`), 0o644)
	require.NoError(t, err)

	gen := NewGenerator()
	err = gen.ScanPackages([]string{tmpDir})
	require.NoError(t, err)

	structs := gen.FindMarkedStructs()
	assert.Len(t, structs, 0)
}

func TestGenerator_SkipExistingGenFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(testFile, []byte(`package test

// TestStruct is a test struct.
// generate:reset
type TestStruct struct {
	Value int
}
`), 0o644)
	require.NoError(t, err)

	genFile := filepath.Join(tmpDir, "reset.gen.go")
	err = os.WriteFile(genFile, []byte(`package test

// existing
`), 0o644)
	require.NoError(t, err)

	gen := NewGenerator()
	err = gen.ScanPackages([]string{tmpDir})
	require.NoError(t, err)

	structs := gen.FindMarkedStructs()
	assert.Len(t, structs, 1)
}

func TestGetGeneratedFiles(t *testing.T) {
	t.Parallel()

	gen := NewGenerator()
	files := gen.GetGeneratedFiles()
	assert.NotNil(t, files)
	assert.Len(t, files, 0)
}

func TestGenerator_PointerFields(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(testFile, []byte(`package test

// PointerStruct is a test.
// generate:reset
type PointerStruct struct {
	IntPtr    *int
	StrPtr    *string
	NestedPtr *SubStruct
}

// SubStruct is a sub struct.
// generate:reset
type SubStruct struct {
	Value int
}
`), 0o644)
	require.NoError(t, err)

	gen := NewGenerator()
	err = gen.ScanPackages([]string{tmpDir})
	require.NoError(t, err)

	structs := gen.FindMarkedStructs()
	require.Len(t, structs, 2)

	for _, s := range structs {
		s.Directory = tmpDir
		s.Package = "test"
	}

	err = gen.GenerateCode(structs)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "reset.gen.go")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "IntPtr != nil")
	assert.Contains(t, contentStr, "StrPtr != nil")
	assert.Contains(t, contentStr, "NestedPtr != nil")
	assert.Contains(t, contentStr, "NestedPtr.Reset()")
	assert.Contains(t, contentStr, "*rs.IntPtr = 0")
}

func TestGenerator_MapField(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	err := os.WriteFile(testFile, []byte(`package test

// MapStruct is a test.
// generate:reset
type MapStruct struct {
	Data map[string]int
}
`), 0o644)
	require.NoError(t, err)

	gen := NewGenerator()
	err = gen.ScanPackages([]string{tmpDir})
	require.NoError(t, err)

	structs := gen.FindMarkedStructs()
	require.Len(t, structs, 1)

	structs[0].Directory = tmpDir
	structs[0].Package = "test"

	err = gen.GenerateCode(structs)
	require.NoError(t, err)

	outputFile := filepath.Join(tmpDir, "reset.gen.go")
	content, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "clear(rs.Data)")
	assert.Contains(t, contentStr, "MapStruct) Reset()")
}

func TestGenerateResetMethodBody(t *testing.T) {
	t.Parallel()

	gen := NewGenerator()
	err := gen.ScanPackages([]string{"./testdata/simple"})
	require.NoError(t, err)

	structs := gen.FindMarkedStructs()
	require.Len(t, structs, 1)

	code, err := gen.GenerateResetMethod(structs[0])
	require.NoError(t, err)

	content := string(code)
	assert.Contains(t, content, "func (rs *TestStruct) Reset()")
	assert.Contains(t, content, "rs.Counter = 0")
	assert.Contains(t, content, "rs.Tags = rs.Tags[:0]")
	assert.Contains(t, content, "clear(rs.Data)")
}
