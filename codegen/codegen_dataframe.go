package codegen

import (
	"fmt"
	"io"
	"strings"

	"github.com/derekmartinsdev/sieve/ast"
)

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