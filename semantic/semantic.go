package semantic

import (
	"fmt"
	"strings"

	"github.com/derekmartinsdev/sieve/ast"
)

type Analyzer struct {
	tables map[string]*TableInfo
	errors []string
}

type TableInfo struct {
	Name   string
	Fields map[string]FieldInfo
	Source bool // true if from s3
}

type FieldInfo struct {
	Name  string
	Type  string
	Alias string
}

func New() *Analyzer {
	return &Analyzer{
		tables: make(map[string]*TableInfo),
	}
}

func (a *Analyzer) Errors() []string { return a.errors }

func (a *Analyzer) Analyze(prog *ast.Program) bool {
	a.errors = nil

	// First pass: register all tables
	for _, stmt := range prog.Statements {
		switch s := stmt.(type) {
		case *ast.SourceStatement:
			a.registerSource(s)
		case *ast.AssignmentStatement:
			a.registerAssignment(s)
		}
	}

	// Second pass: validate references
	for _, stmt := range prog.Statements {
		switch s := stmt.(type) {
		case *ast.SourceStatement:
			a.validateSource(s)
		case *ast.AssignmentStatement:
			a.validateAssignment(s)
		}
	}

	return len(a.errors) == 0
}

func (a *Analyzer) registerSource(s *ast.SourceStatement) {
	info := &TableInfo{
		Name:   s.Name,
		Fields: make(map[string]FieldInfo),
		Source: s.S3 != nil,
	}

	for _, ext := range s.Extracts {
		for _, f := range ext.Fields {
			fieldName := f.Alias
			if fieldName == "" {
				fieldName = f.SourcePath
			}
			info.Fields[fieldName] = FieldInfo{
				Name:  fieldName,
				Type:  f.Type.BaseType,
				Alias: f.Alias,
			}
		}
	}

	a.tables[s.Name] = info
}

func (a *Analyzer) registerAssignment(s *ast.AssignmentStatement) {
	if join, ok := s.Value.(*ast.JoinStatement); ok {
		info := &TableInfo{
			Name:   s.Name,
			Fields: make(map[string]FieldInfo),
		}

		for _, sel := range join.Selects {
			fieldName := sel.Alias
			if fieldName == "" {
				path := sel.Path
				if path == "" {
					path = sel.Source
				}
				fieldName = path
			}
			info.Fields[fieldName] = FieldInfo{
				Name:  fieldName,
				Type:  sel.FieldType.BaseType,
				Alias: sel.Alias,
			}
		}

		a.tables[s.Name] = info
	}
}

func (a *Analyzer) validateSource(s *ast.SourceStatement) {
	// Validate that from references exist
	if s.S3 == nil {
		// referencing another table - check it exists
		if _, ok := a.tables[s.Name]; !ok {
			a.errors = append(a.errors,
				fmt.Sprintf("table '%s' referenced but not defined", s.Name))
		}
	} else {
		// validate S3 source has required fields
		if s.S3.Bucket == "" {
			a.errors = append(a.errors,
				fmt.Sprintf("source '%s' missing bucket", s.Name))
		}
		if s.S3.Prefix == "" {
			a.errors = append(a.errors,
				fmt.Sprintf("source '%s' missing prefix", s.Name))
		}
		if s.S3.Format == "" {
			a.errors = append(a.errors,
				fmt.Sprintf("source '%s' missing format", s.Name))
		}
	}
}

func (a *Analyzer) validateAssignment(s *ast.AssignmentStatement) {
	if join, ok := s.Value.(*ast.JoinStatement); ok {
		// validate tables exist
		leftTable, leftOk := a.tables[join.Left]
		rightTable, rightOk := a.tables[join.Right]

		if !leftOk {
			a.errors = append(a.errors,
				fmt.Sprintf("table '%s' not found in join", join.Left))
		}
		if !rightOk {
			a.errors = append(a.errors,
				fmt.Sprintf("table '%s' not found in join", join.Right))
		}

		// validate select fields exist in source tables
		for _, sel := range join.Selects {
			if sel.Source != "" {
				var sourceTable *TableInfo
				if sel.Source == join.Left {
					sourceTable = leftTable
				} else if sel.Source == join.Right {
					sourceTable = rightTable
				}

				if sourceTable != nil {
					fieldName := sel.Path
					if fieldName == "" {
						fieldName = sel.Source
					}
					if _, ok := sourceTable.Fields[fieldName]; !ok {
						// could be a path like party.name, check partial match
						parts := strings.Split(sel.Path, ".")
						found := false
						for _, f := range sourceTable.Fields {
							if strings.HasPrefix(f.Name, parts[0]) {
								found = true
								break
							}
						}
						if !found {
							a.errors = append(a.errors,
								fmt.Sprintf("field '%s' not found in table '%s'", fieldName, sel.Source))
						}
					}
				}
			}
		}
	}
}

// GetTableInfo returns registered table info
func (a *Analyzer) GetTableInfo(name string) (*TableInfo, bool) {
	info, ok := a.tables[name]
	return info, ok
}

// ResolveField resolves a field path like "party.name" to a table+field
func (a *Analyzer) ResolveField(tableName string, fieldPath string) (*FieldInfo, error) {
	table, ok := a.tables[tableName]
	if !ok {
		return nil, fmt.Errorf("table '%s' not found", tableName)
	}

	// direct match
	if f, ok := table.Fields[fieldPath]; ok {
		return &f, nil
	}

	// prefix match (e.g., "name" matches "party.name")
	for _, f := range table.Fields {
		if strings.HasSuffix(f.Name, fieldPath) || strings.HasSuffix(f.Alias, fieldPath) {
			return &f, nil
		}
	}

	return nil, fmt.Errorf("field '%s' not found in table '%s'", fieldPath, tableName)
}