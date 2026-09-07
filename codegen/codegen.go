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
		if ext.Plain != nil {
			allFields = append(allFields, ext.Plain.Fields...)
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
			escaped := escapeRegexMeta(old)
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

func escapeRegexMeta(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '.', '*', '+', '?', '^', '$', '{', '}', '(', ')', '[', ']', '|', '\\':
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
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
