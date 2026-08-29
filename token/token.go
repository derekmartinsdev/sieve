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

	// Identifiers + literals
	IDENT    = "IDENT"
	STRING   = "STRING"
	NUMBER   = "NUMBER"
	DECIMAL  = "DECIMAL"

	// DSL Keywords
	POSITION       = "POSITION"
	TRADE          = "TRADE"
	PERACQUISITION = "PERACQUISITION"
	TAXES          = "TAXES"
	FROM           = "FROM"
	S3             = "S3"
	BUCKET         = "BUCKET"
	REGION         = "REGION"
	PREFIX         = "PREFIX"
	FORMAT         = "FORMAT"
	EXTRACT        = "EXTRACT"
	JSON           = "JSON"
	SELECT         = "SELECT"
	EXPLODE        = "EXPLODE"
	ARRAY          = "ARRAY"
	TO             = "TO"
	MODE           = "MODE"
	PARTITIONED    = "PARTITIONED"
	BY             = "BY"
	AND            = "AND"
	OR             = "OR"
	LEFT           = "LEFT"
	OVERWRITE      = "OVERWRITE"
	APPEND         = "APPEND"
	MERGE          = "MERGE"
	DELTA          = "DELTA"
	PARQUET        = "PARQUET"
	CSV            = "CSV"

	// Types
	STRING_TYPE  = "STRING_TYPE"
	BIGINT       = "BIGINT"
	DATE         = "DATE"
	DECIMAL_TYPE = "DECIMAL_TYPE"

	// Operators
	PIPE     = "PIPE"    // |
	JOIN     = "JOIN"    // /\
	ARROW    = "ARROW"   // ->
	ASSIGN   = "ASSIGN"  // =
	COMMA    = "COMMA"
	DOT      = "DOT"
	LPAREN   = "LPAREN"
	RPAREN   = "RPAREN"
	LBRACKET = "LBRACKET"
	RBRACKET = "RBRACKET"
	COLON    = "COLON"
	PLUS     = "PLUS"
	MINUS    = "MINUS"
	ASTERISK = "ASTERISK"
	SLASH    = "SLASH"
)

var Keywords = map[string]TokenType{
	"position":       POSITION,
	"trade":          TRADE,
	"perAcquisition": PERACQUISITION,
	"taxes":          TAXES,
	"from":           FROM,
	"s3":             S3,
	"bucket":         BUCKET,
	"region":         REGION,
	"prefix":         PREFIX,
	"format":         FORMAT,
	"extract":        EXTRACT,
	"json":           JSON,
	"select":         SELECT,
	"explode":        EXPLODE,
	"array":          ARRAY,
	"to":             TO,
	"mode":           MODE,
	"partitioned":    PARTITIONED,
	"by":             BY,
	"and":            AND,
	"or":             OR,
	"left":           LEFT,
	"overwrite":      OVERWRITE,
	"append":         APPEND,
	"merge":          MERGE,
	"delta":          DELTA,
	"parquet":        PARQUET,
	"csv":            CSV,
	"string":         STRING_TYPE,
	"bigint":         BIGINT,
	"date":           DATE,
	"decimal":        DECIMAL_TYPE,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := Keywords[ident]; ok {
		return tok
	}
	return IDENT
}