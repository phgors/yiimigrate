package migrate

// ColumnBuilder describes a database column definition.
type ColumnBuilder struct {
	sqlType           string
	nullable          *bool
	unsigned          bool
	primaryKey        bool
	autoIncrement     bool
	unique            bool
	defaultValue      any
	hasDefaultValue   bool
	defaultExpression string
	comment           string
	check             string
	after             string
	first             bool
	appendSQL         []string
	charset           string
	collation         string
	generatedAs       string
	generatedStorage  string
}

func newColumnBuilder(sqlType string) *ColumnBuilder {
	return &ColumnBuilder{sqlType: sqlType}
}

func (c *ColumnBuilder) clone() *ColumnBuilder {
	if c == nil {
		return &ColumnBuilder{}
	}
	next := *c
	if c.nullable != nil {
		nullable := *c.nullable
		next.nullable = &nullable
	}
	next.appendSQL = append([]string(nil), c.appendSQL...)
	return &next
}

// NotNull marks the column as NOT NULL.
func (c *ColumnBuilder) NotNull() *ColumnBuilder {
	next := c.clone()
	nullable := false
	next.nullable = &nullable
	return next
}

// Null marks the column as NULL.
func (c *ColumnBuilder) Null() *ColumnBuilder {
	next := c.clone()
	nullable := true
	next.nullable = &nullable
	return next
}

// Unsigned marks a numeric column as UNSIGNED.
func (c *ColumnBuilder) Unsigned() *ColumnBuilder {
	next := c.clone()
	next.unsigned = true
	return next
}

// PrimaryKey marks the column as PRIMARY KEY.
func (c *ColumnBuilder) PrimaryKey() *ColumnBuilder {
	next := c.clone()
	next.primaryKey = true
	return next
}

// AutoIncrement marks the column as AUTO_INCREMENT.
func (c *ColumnBuilder) AutoIncrement() *ColumnBuilder {
	next := c.clone()
	next.autoIncrement = true
	return next
}

// Unique marks the column as UNIQUE.
func (c *ColumnBuilder) Unique() *ColumnBuilder {
	next := c.clone()
	next.unique = true
	return next
}

// DefaultValue sets a literal DEFAULT value.
func (c *ColumnBuilder) DefaultValue(v any) *ColumnBuilder {
	next := c.clone()
	next.defaultValue = v
	next.hasDefaultValue = true
	next.defaultExpression = ""
	return next
}

// DefaultExpression sets a raw SQL DEFAULT expression.
func (c *ColumnBuilder) DefaultExpression(sql string) *ColumnBuilder {
	next := c.clone()
	next.defaultExpression = sql
	next.hasDefaultValue = false
	next.defaultValue = nil
	return next
}

// Comment sets a column comment.
func (c *ColumnBuilder) Comment(comment string) *ColumnBuilder {
	next := c.clone()
	next.comment = comment
	return next
}

// Check adds a CHECK expression.
func (c *ColumnBuilder) Check(expr string) *ColumnBuilder {
	next := c.clone()
	next.check = expr
	return next
}

// After positions the column after another column.
func (c *ColumnBuilder) After(column string) *ColumnBuilder {
	next := c.clone()
	next.after = column
	next.first = false
	return next
}

// First positions the column first.
func (c *ColumnBuilder) First() *ColumnBuilder {
	next := c.clone()
	next.first = true
	next.after = ""
	return next
}

// Append appends raw SQL to the column definition.
func (c *ColumnBuilder) Append(sql string) *ColumnBuilder {
	next := c.clone()
	if sql != "" {
		next.appendSQL = append(next.appendSQL, sql)
	}
	return next
}

// Charset sets a column character set.
func (c *ColumnBuilder) Charset(charset string) *ColumnBuilder {
	next := c.clone()
	next.charset = charset
	return next
}

// Collate sets a column collation.
func (c *ColumnBuilder) Collate(collation string) *ColumnBuilder {
	next := c.clone()
	next.collation = collation
	return next
}

// GeneratedAs marks the column as generated from an expression.
func (c *ColumnBuilder) GeneratedAs(expr string) *ColumnBuilder {
	next := c.clone()
	next.generatedAs = expr
	return next
}

// Stored marks a generated column as STORED.
func (c *ColumnBuilder) Stored() *ColumnBuilder {
	next := c.clone()
	next.generatedStorage = "STORED"
	return next
}

// Virtual marks a generated column as VIRTUAL.
func (c *ColumnBuilder) Virtual() *ColumnBuilder {
	next := c.clone()
	next.generatedStorage = "VIRTUAL"
	return next
}
