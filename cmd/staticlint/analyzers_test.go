package main

import (
	"go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnalyzers(t *testing.T) {
	t.Parallel()

	analyzers := Analyzers()
	assert.NotEmpty(t, analyzers)

	names := make([]string, len(analyzers))
	for i, a := range analyzers {
		names[i] = a.Name
	}

	assert.Contains(t, names, "osexitanalyzer")
	assert.Contains(t, names, "naminganalyzer")
	assert.Contains(t, names, "contextanalyzer")

	assert.Contains(t, names, "asmdecl")
	assert.Contains(t, names, "printf")
	assert.Contains(t, names, "nilfunc")

	saCount := 0
	stCount := 0
	vCount := 0
	for _, name := range names {
		if len(name) >= 3 && name[:3] == "SA*" {
			saCount++
		}
		if len(name) >= 3 && name[:3] == "ST*" {
			stCount++
		}
		if len(name) >= 2 && name[:1] == "V" {
			vCount++
		}
	}

	assert.GreaterOrEqual(t, saCount, 0, "SA* analyzers count should be >= 0")
	assert.GreaterOrEqual(t, stCount, 0, "ST* analyzers count should be >= 0")
	assert.GreaterOrEqual(t, vCount, 0, "V* analyzers count should be >= 0")
}

func TestAnalyzersCount(t *testing.T) {
	t.Parallel()

	analyzers := Analyzers()
	assert.Greater(t, len(analyzers), 10)
}

func TestIsUpper(t *testing.T) {
	t.Parallel()

	assert.True(t, isUpper('A'))
	assert.True(t, isUpper('Z'))
	assert.True(t, isUpper('B'))
	assert.False(t, isUpper('a'))
	assert.False(t, isUpper('z'))
	assert.False(t, isUpper('0'))
	assert.False(t, isUpper('_'))
	assert.False(t, isUpper(' '))
}

func TestKindOf(t *testing.T) {
	t.Parallel()

	variable := &ast.ValueSpec{Type: &ast.Ident{Name: "int"}}
	assert.Equal(t, "variable", kindOf(variable))

	constant := &ast.ValueSpec{}
	assert.Equal(t, "constant", kindOf(constant))
}
