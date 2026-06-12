package migrate

// ColumnDef describes a named column in declaration order.
type ColumnDef struct {
	// Name is the column name.
	Name string
	// Column is the column definition.
	Column *ColumnBuilder
}

// ColumnList stores ordered column definitions.
type ColumnList struct {
	items []ColumnDef
}

// Columns creates an empty ordered column list.
func Columns() *ColumnList {
	return &ColumnList{}
}

// Add appends a named column definition and returns the list for chaining.
func (c *ColumnList) Add(name string, column *ColumnBuilder) *ColumnList {
	if c == nil {
		c = Columns()
	}
	c.items = append(c.items, ColumnDef{Name: name, Column: column})
	return c
}

// Items returns a copy of the ordered column definitions.
func (c *ColumnList) Items() []ColumnDef {
	if c == nil {
		return nil
	}
	return append([]ColumnDef(nil), c.items...)
}
