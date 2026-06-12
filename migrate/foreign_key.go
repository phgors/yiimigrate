package migrate

// ForeignKeyAction describes ON DELETE and ON UPDATE foreign key actions.
type ForeignKeyAction string

const (
	// NoAction emits NO ACTION for a foreign key action.
	NoAction ForeignKeyAction = "NO ACTION"
	// Restrict emits RESTRICT for a foreign key action.
	Restrict ForeignKeyAction = "RESTRICT"
	// Cascade emits CASCADE for a foreign key action.
	Cascade ForeignKeyAction = "CASCADE"
	// SetNull emits SET NULL for a foreign key action.
	SetNull ForeignKeyAction = "SET NULL"
	// SetDefault emits SET DEFAULT for a foreign key action.
	SetDefault ForeignKeyAction = "SET DEFAULT"
)

func (a ForeignKeyAction) sql() string {
	return string(a)
}
