package migrate

// Row represents a database row for DML helpers.
type Row map[string]any

// Expression represents a raw SQL expression that should not be parameter-bound.
type Expression string

// Expr creates a raw SQL expression.
func Expr(sql string) Expression {
	return Expression(sql)
}
