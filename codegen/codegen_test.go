package codegen

import (
	"strings"
	"testing"

	"github.com/derekmartinsdev/sieve/ast"
)

func generateTest(prog *ast.Program) string {
	var b strings.Builder
	if err := Generate(&b, prog); err != nil {
		panic(err)
	}
	return b.String()
}

func TestGenerateImports(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.Section{
				Name: "main",
				Source: &ast.Source{
					From:   "s3",
					Bucket: "b",
					Prefix: "p",
					Format: "delta",
				},
			},
		},
	}

	out := generateTest(prog)

	if !strings.Contains(out, "from pyspark.sql import SparkSession") {
		t.Error("output missing SparkSession import")
	}
	if !strings.Contains(out, "import pyspark.sql.functions as F") {
		t.Error("output missing pyspark functions import")
	}
}

func TestGenerateJsonSelectGetJsonObject(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.Section{
				Name: "events",
				Source: &ast.Source{
					From:   "s3",
					Bucket: "data",
					Prefix: "events",
					Format: "delta",
				},
				Extracts: []*ast.Extract{
					{
						JsonSelect: &ast.JsonSelect{
							Path: "message",
							Fields: []*ast.FieldDef{
								{Name: "id", Source: "id", JsonColumn: "message"},
								{Name: "name", Source: "name", JsonColumn: "message"},
							},
						},
					},
				},
			},
		},
	}

	out := generateTest(prog)

	if !strings.Contains(out, "F.get_json_object") {
		t.Error("output missing get_json_object call")
	}
}

func TestGenerateComputedColumns(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.Section{
				Name: "sales",
				Source: &ast.Source{
					From:   "s3",
					Bucket: "data",
					Prefix: "sales",
					Format: "delta",
				},
				Extracts: []*ast.Extract{
					{
						JsonSelect: &ast.JsonSelect{
							Path: "message",
							Fields: []*ast.FieldDef{
								{Name: "quantity", Source: "quantity", JsonColumn: "message"},
								{Name: "price", Source: "price", JsonColumn: "message"},
								{
									Name:         "total",
									Source:       "",
									Alias:        "total",
									ComputedExpr: &ast.BinaryExpr{Left: "quantity", Operator: "*", Right: "price"},
									JsonColumn:   "message",
								},
							},
						},
					},
				},
			},
		},
	}

	out := generateTest(prog)

	if !strings.Contains(out, "F.col") {
		t.Error("output missing F.col call")
	}
	if !strings.Contains(out, "(F.col(\"quantity\") * F.col(\"price\"))") {
		t.Error("output missing computed column pattern (F.col(q) * F.col(p))")
	}
	if !strings.Contains(out, "withColumn") {
		t.Error("output missing withColumn for computed column")
	}
}

func TestGenerateExplodeFromJson(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.Section{
				Name: "events",
				Source: &ast.Source{
					From:   "s3",
					Bucket: "data",
					Prefix: "events",
					Format: "delta",
				},
				Extracts: []*ast.Extract{
					{
						Explode: &ast.ExtractExplode{
							Path: "message.items",
							As:   "array",
							Fields: []*ast.FieldDef{
								{Name: "item_id", Source: "item_id", JsonColumn: "exploded"},
							},
						},
					},
				},
			},
		},
	}

	out := generateTest(prog)

	if !strings.Contains(out, "F.from_json") {
		t.Error("output missing from_json call")
	}
	if !strings.Contains(out, "F.explode") {
		t.Error("output missing explode call")
	}
}

func TestGenerateTransformChainsWithColumn(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.Section{
				Name: "clean",
				Source: &ast.Source{
					From:   "s3",
					Bucket: "data",
					Prefix: "raw",
					Format: "delta",
				},
				Transforms: []ast.TransformChain{
					{
						Alias: "user_id",
						Steps: []ast.TransformStep{
							{Type: "col", Args: []string{"raw_id"}},
							{Type: "hash"},
						},
					},
				},
			},
		},
	}

	out := generateTest(prog)

	if strings.Count(out, ".withColumn(") < 2 {
		t.Errorf("expected at least 2 withColumn calls, got %d\n%s", strings.Count(out, ".withColumn("), out)
	}
}

func TestGenerateSinkWriteFormatMode(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.Section{
				Name: "output",
				Source: &ast.Source{
					From:   "s3",
					Bucket: "data",
					Prefix: "raw",
					Format: "delta",
				},
				Sink: &ast.Sink{
					To:     "s3",
					Bucket: "output",
					Prefix: "processed",
					Format: "delta",
					Mode:   "overwrite",
				},
			},
		},
	}

	out := generateTest(prog)

	if !strings.Contains(out, ".write.format(\"delta\").mode(\"overwrite\").save(\"s3a://output/processed\")") {
		t.Errorf("output missing write.format.mode.save pattern\ngot:\n%s", out)
	}
}
