package codegen

import (
	"fmt"
	"io"
	"strings"

	"github.com/derekmartinsdev/sieve/ast"
)

func schemaFromAs(as string) string {
	if as == "" || as == "array" {
		return "array<string>"
	}
	if as == "struct" {
		return "struct<>"
	}
	return "array<string>"
}

func splitFields(fields []*ast.FieldDef) (regular, computed []*ast.FieldDef) {
	for _, f := range fields {
		if f.ComputedExpr != nil {
			computed = append(computed, f)
		} else {
			regular = append(regular, f)
		}
	}
	return
}

func buildAliasMap(fields []*ast.FieldDef) map[string]string {
	m := make(map[string]string)
	for _, f := range fields {
		src := f.Source
		if src == "" {
			src = f.Name
		}
		alias := f.Alias
		if alias == "" {
			alias = f.Name
		}
		m[src] = alias
	}
	return m
}

func emitComputedColumn(w io.Writer, varName string, f *ast.FieldDef, nameMap map[string]string) error {
	if f.ComputedExpr == nil {
		return nil
	}
	left := f.ComputedExpr.Left
	right := f.ComputedExpr.Right
	if a, ok := nameMap[left]; ok {
		left = a
	}
	if a, ok := nameMap[right]; ok {
		right = a
	}
	expr := fmt.Sprintf("(F.col(%q) %s F.col(%q))", left, f.ComputedExpr.Operator, right)
	if f.DataType != "" {
		expr += fmt.Sprintf(".cast(%q)", f.DataType)
	}
	_, err := fmt.Fprintf(w, "%s = %s.withColumn(%q, %s)\n", varName, varName, f.Alias, expr)
	return err
}

func splitExplodePath(path string) (col, jsonPath string) {
	i := strings.Index(path, ".")
	if i < 0 {
		return path, "$"
	}
	return path[:i], "$." + path[i+1:]
}

func lastPathPart(path string) string {
	i := strings.LastIndex(path, ".")
	if i < 0 {
		return path
	}
	return path[i+1:]
}

func buildDerivedFieldExpr(f *ast.FieldDef) string {
	alias := f.Alias
	if alias == "" {
		alias = f.Name
	}
	expr := fmt.Sprintf("F.col(%q).alias(%q)", f.Name, alias)
	if f.DataType != "" {
		expr += fmt.Sprintf(".cast(%q)", f.DataType)
	}
	return expr
}

func buildJsonFieldExpr(f *ast.FieldDef, jsonCol string) string {
	col := f.JsonColumn
	if col == "" {
		col = jsonCol
	}
	source := f.Source
	if source == "" {
		source = f.Name
	}
	alias := f.Alias
	if alias == "" {
		alias = f.Name
	}
	expr := fmt.Sprintf("F.get_json_object(F.col(%q), %q).alias(%q)", col, "$."+source, alias)
	if f.DataType != "" {
		expr += fmt.Sprintf(".cast(%q)", f.DataType)
	}
	return expr
}
