package codegen

import (
	"fmt"
	"io"
	"strings"

	"github.com/derekmartinsdev/sieve/ast"
)

func Generate(w io.Writer, prog *ast.Program) error {
	_, err := fmt.Fprint(w, "from pyspark.sql import SparkSession\n")
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, "import pyspark.sql.functions as F\n\n\n")
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, "spark = SparkSession.builder.appName(\"job\").getOrCreate()\n\n")
	if err != nil {
		return err
	}

	for _, stmt := range prog.Statements {
		sec, ok := stmt.(*ast.Section)
		if !ok {
			continue
		}
		if err := generateSection(w, sec); err != nil {
			return err
		}
	}

	return nil
}

func generateSection(w io.Writer, sec *ast.Section) error {
	varName := "df_" + sec.Name
	isDerived := false

	if sec.Source != nil {
		src := sec.Source
		if src.From == "s3" {
			_, err := fmt.Fprintf(w, "%s = spark.read.format(%q).load(%q)\n",
				varName, src.Format, "s3a://"+src.Bucket+"/"+src.Prefix)
			if err != nil {
				return err
			}
		} else if src.From != "" {
			_, err := fmt.Fprintf(w, "%s = df_%s\n", varName, src.From)
			if err != nil {
				return err
			}
			isDerived = true
		}
	}

	explodeCol := ""
	var allFields []*ast.FieldDef
	var jsonCol string

	for _, ext := range sec.Extracts {
		if ext.Explode != nil {
			col, jsonPath := splitExplodePath(ext.Explode.Path)
			explodeCol = "exploded"
			jsonCol = explodeCol
			schemaType := schemaFromAs(ext.Explode.As)
			if isDerived {
				_, err := fmt.Fprintf(w, "%s = %s.withColumn(%q, F.explode(F.from_json(F.col(%q), %q))).select(%q)\n",
					varName, varName, explodeCol, col, schemaType, explodeCol+".*")
				if err != nil {
					return err
				}
			} else {
				_, err := fmt.Fprintf(w, "%s = %s.withColumn(%q, F.explode(F.from_json(F.get_json_object(F.col(%q), %q), %q))).select(%q)\n",
					varName, varName, explodeCol, col, jsonPath, schemaType, explodeCol+".*")
				if err != nil {
					return err
				}
			}
			allFields = append(allFields, ext.Explode.Fields...)
		}
		if ext.JsonSelect != nil {
			if jsonCol == "" {
				jsonCol = ext.JsonSelect.Path
			}
			allFields = append(allFields, ext.JsonSelect.Fields...)
		}
		if ext.JsonExtract != nil {
			col, _ := splitExplodePath(ext.JsonExtract.Path)
			if isDerived {
				col = lastPathPart(ext.JsonExtract.Path)
			}
			explodeCol = "exploded"
			jsonCol = explodeCol
			schemaType := schemaFromAs(ext.JsonExtract.As)
			if isDerived {
				_, err := fmt.Fprintf(w, "%s = %s.withColumn(%q, F.explode(F.from_json(F.col(%q), %q))).select(%q)\n",
					varName, varName, explodeCol, col, schemaType, explodeCol+".*")
				if err != nil {
					return err
				}
			} else {
				_, jsonPath := splitExplodePath(ext.JsonExtract.Path)
				_, err := fmt.Fprintf(w, "%s = %s.withColumn(%q, F.explode(F.from_json(F.get_json_object(F.col(%q), %q), %q))).select(%q)\n",
					varName, varName, explodeCol, col, jsonPath, schemaType, explodeCol+".*")
				if err != nil {
					return err
				}
			}
			allFields = append(allFields, ext.JsonExtract.Fields...)
		}
	}

	if len(allFields) > 0 {
		regular, computed := splitFields(allFields)
		nameMap := buildAliasMap(allFields)

		if len(regular) > 0 {
			_, err := fmt.Fprintf(w, "%s = %s.select(\n", varName, varName)
			if err != nil {
				return err
			}
			for i, f := range regular {
				comma := ","
				if i == len(regular)-1 {
					comma = ""
				}
				if isDerived && f.JsonColumn != "exploded" {
					_, err := fmt.Fprintf(w, "    %s%s\n", buildDerivedFieldExpr(f), comma)
					if err != nil {
						return err
					}
				} else {
					_, err := fmt.Fprintf(w, "    %s%s\n", buildJsonFieldExpr(f, jsonCol), comma)
					if err != nil {
						return err
					}
				}
			}
			_, err = fmt.Fprint(w, ")\n")
			if err != nil {
				return err
			}
		}
		totalTransformSteps := len(computed)
		for _, tc := range sec.Transforms {
			totalTransformSteps += len(tc.Steps)
		}
		if totalTransformSteps >= 5 {
			_, err := fmt.Fprintf(w, "# %s %d withColumn calls detected. Consider merging into a single select().\n", "\u26a0\ufe0f", totalTransformSteps)
			if err != nil {
				return err
			}
		}
		for _, f := range computed {
			if err := emitComputedColumn(w, varName, f, nameMap); err != nil {
				return err
			}
		}
	}

	for _, join := range sec.Joins {
		if err := generateJoin(w, varName, join); err != nil {
			return err
		}
	}

	for _, tc := range sec.Transforms {
		if err := generateTransform(w, varName, &tc); err != nil {
			return err
		}
	}

	if sec.Select != nil && len(sec.Select.Fields) > 0 {
		if err := generateSelect(w, varName, sec.Select); err != nil {
			return err
		}
	}

	if sec.Sink != nil {
		if err := generateSink(w, varName, sec.Sink); err != nil {
			return err
		}
	}

	_, err := fmt.Fprint(w, "\n")
	return err
}

func schemaFromAs(as string) string {
	if as == "" {
		return "array<string>"
	}
	if as == "array" {
		return "array<string>"
	}
	return as
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

func generateSelect(w io.Writer, varName string, sel *ast.SelectStmt) error {
	if len(sel.Fields) == 0 {
		return nil
	}
	_, err := fmt.Fprintf(w, "%s = %s.select(\n", varName, varName)
	if err != nil {
		return err
	}
	for i, f := range sel.Fields {
		comma := ","
		if i == len(sel.Fields)-1 {
			comma = ""
		}
		expr := fmt.Sprintf("F.col(%q)", f.Name)
		if f.Function == "month" {
			expr = fmt.Sprintf("F.date_format(F.col(%q), \"yyyy-MM\")", f.Name)
		} else if f.Function == "day" {
			expr = fmt.Sprintf("F.dayofmonth(F.col(%q))", f.Name)
		} else if f.Function != "" {
			expr = fmt.Sprintf("F.%s(F.col(%q))", f.Function, f.Name)
		}
		if f.DataType != "" {
			expr += fmt.Sprintf(".cast(%q)", f.DataType)
		}
		alias := f.Name
		if f.Alias != "" {
			alias = f.Alias
		}
		_, err = fmt.Fprintf(w, "    %s.alias(%q)%s\n", expr, alias, comma)
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprint(w, ")\n")
	return err
}

func generateJoin(w io.Writer, varName string, join *ast.Join) error {
	joinType := "inner"
	if join.JoinType != "" {
		joinType = join.JoinType
	}
	rightDF := "df_" + join.Right
	leftAlias := join.LeftAlias
	if leftAlias == "" {
		leftAlias = join.Left
	}
	rightAlias := join.RightAlias
	if rightAlias == "" {
		rightAlias = join.Right
	}

	var condParts []string
	if join.Condition != nil {
		for i, leftRef := range join.Condition.LeftRefs {
			if i >= len(join.Condition.RightRefs) {
				break
			}
			rightRef := join.Condition.RightRefs[i]
			condParts = append(condParts,
				fmt.Sprintf("F.col(%q) == F.col(%q)", leftRef, rightRef))
		}
	}

	condition := strings.Join(condParts, " & ")
	if condition == "" {
		condition = "F.lit(True)"
	}

	_, err := fmt.Fprintf(w, "%s = %s.alias(%q).join(%s.alias(%q), %s, %q)\n",
		varName, varName, leftAlias, rightDF, rightAlias, condition, joinType)
	if err != nil {
		return err
	}

	if join.Select != nil && len(join.Select.Fields) > 0 {
		if err := generateSelect(w, varName, join.Select); err != nil {
			return err
		}
	}
	return nil
}

func generateSink(w io.Writer, varName string, sink *ast.Sink) error {
	mode := sink.Mode
	if mode == "" {
		mode = "overwrite"
	}

	var partitionClause string
	if len(sink.PartitionBy) > 0 {
		parts := make([]string, len(sink.PartitionBy))
		for i, p := range sink.PartitionBy {
			parts[i] = fmt.Sprintf("%q", p)
		}
		partitionClause = ".partitionBy(" + strings.Join(parts, ",") + ")"
	}

	_, err := fmt.Fprintf(w, "%s.write.format(%q).mode(%q)%s.save(%q)\n",
		varName, sink.Format, mode, partitionClause, "s3a://"+sink.Bucket+"/"+sink.Prefix)
	return err
}

func generateTransform(w io.Writer, varName string, tc *ast.TransformChain) error {
	alias := tc.Alias

	for _, step := range tc.Steps {
		var err error
		switch step.Type {
		case "col":
			source := step.Args[0]
			_, err = fmt.Fprintf(w, "%s = %s.withColumn(%q, F.col(%q))\n", varName, varName, alias, source)
		case "cast":
			castType := strings.Join(step.Args, "")
			_, err = fmt.Fprintf(w, "%s = %s.withColumn(%q, F.col(%q).cast(%q))\n", varName, varName, alias, alias, castType)
		case "default":
			val := step.Args[0]
			_, err = fmt.Fprintf(w, "%s = %s.withColumn(%q, F.coalesce(F.col(%q), %s))\n",
				varName, varName, alias, alias, formatLit(val))
		case "replace":
			old := step.Args[0]
			new := step.Args[1]
			escaped := strings.ReplaceAll(old, ".", "\\.")
			_, err = fmt.Fprintf(w, "%s = %s.withColumn(%q, F.regexp_replace(F.col(%q), %q, %q))\n",
				varName, varName, alias, alias, escaped, new)
		case "prefix":
			str := step.Args[0]
			_, err = fmt.Fprintf(w, "%s = %s.withColumn(%q, F.concat(F.lit(%q), F.col(%q)))\n",
				varName, varName, alias, str, alias)
		case "hash":
			_, err = fmt.Fprintf(w, "%s = %s.withColumn(%q, F.sha2(F.col(%q), 256))\n",
				varName, varName, alias, alias)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func formatLit(val string) string {
	if isNumeric(val) {
		return fmt.Sprintf("F.lit(%s)", val)
	}
	return fmt.Sprintf("F.lit(%q)", val)
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if (c >= '0' && c <= '9') || c == '.' || c == '-' {
			continue
		}
		return false
	}
	return true
}
