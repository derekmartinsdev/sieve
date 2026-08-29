package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	IDENT  = "IDENT"
	INT    = "INT"
	FLOAT  = "FLOAT"
	STRING = "STRING"

	DOT      = "."
	COMMA    = ","
	LPAREN   = "("
	RPAREN   = ")"
	ASSIGN   = "="
	PIPE     = "|"
	ASTERISK = "*"

	ARROW = "->"
	JOIN  = "/\\"

	FROM        = "FROM"
	S3          = "S3"
	BUCKET      = "BUCKET"
	REGION      = "REGION"
	PREFIX      = "PREFIX"
	FORMAT      = "FORMAT"
	DELTA       = "DELTA"
	EXTRACT     = "EXTRACT"
	JSON        = "JSON"
	MESSAGE     = "MESSAGE"
	ARRAY       = "ARRAY"
	TYPE_STRING = "TYPE_STRING"
	BIGINT      = "BIGINT"
	TYPE_DATE   = "TYPE_DATE"
	DECIMAL     = "DECIMAL"
	TO          = "TO"
	MODE        = "MODE"
	OVERWRITE   = "OVERWRITE"
	APPEND      = "APPEND"
	MERGE       = "MERGE"
	PARTITIONED = "PARTITIONED"
	BY          = "BY"
	DEFAULT_KW  = "DEFAULT"
	CAST        = "CAST"
	REPLACE     = "REPLACE"
	HASH        = "HASH"
	TRANSFORM   = "TRANSFORM"
	LEFT        = "LEFT"
	RIGHT       = "RIGHT"
	INNER       = "INNER"
	SELECT      = "SELECT"
	EXPLODE     = "EXPLODE"
	YEAR        = "YEAR"
	MONTH       = "MONTH"
	DAY         = "DAY"
	OR          = "OR"
)

var keywords = map[string]TokenType{
	"from":        FROM,
	"s3":          S3,
	"bucket":      BUCKET,
	"region":      REGION,
	"prefix":      PREFIX,
	"format":      FORMAT,
	"delta":       DELTA,
	"extract":     EXTRACT,
	"json":        JSON,
	"select":      SELECT,
	"explode":     EXPLODE,
	"array":       ARRAY,
	"string":      TYPE_STRING,
	"bigint":      BIGINT,
	"date":        TYPE_DATE,
	"decimal":     DECIMAL,
	"to":          TO,
	"mode":        MODE,
	"overwrite":   OVERWRITE,
	"append":      APPEND,
	"merge":       MERGE,
	"partitioned": PARTITIONED,
	"by":          BY,
	"default":     DEFAULT_KW,
	"cast":        CAST,
	"replace":     REPLACE,
	"hash":        HASH,
	"transform":   TRANSFORM,
	"left":        LEFT,
	"right":       RIGHT,
	"inner":       INNER,
	"year":        YEAR,
	"month":       MONTH,
	"day":         DAY,
	"or":          OR,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
