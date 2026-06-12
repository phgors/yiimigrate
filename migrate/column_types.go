package migrate

import "fmt"

// PrimaryKey returns an auto-incrementing integer primary key column.
func (m *MigrationContext) PrimaryKey() *ColumnBuilder {
	return newColumnBuilder("int").NotNull().AutoIncrement().PrimaryKey()
}

// BigPrimaryKey returns an auto-incrementing bigint primary key column.
func (m *MigrationContext) BigPrimaryKey() *ColumnBuilder {
	return newColumnBuilder("bigint").NotNull().AutoIncrement().PrimaryKey()
}

// UnsignedPrimaryKey returns an auto-incrementing unsigned integer primary key column.
func (m *MigrationContext) UnsignedPrimaryKey() *ColumnBuilder {
	return newColumnBuilder("int").Unsigned().NotNull().AutoIncrement().PrimaryKey()
}

// UnsignedBigPrimaryKey returns an auto-incrementing unsigned bigint primary key column.
func (m *MigrationContext) UnsignedBigPrimaryKey() *ColumnBuilder {
	return newColumnBuilder("bigint").Unsigned().NotNull().AutoIncrement().PrimaryKey()
}

// TinyInteger returns a tinyint column.
func (m *MigrationContext) TinyInteger(length ...int) *ColumnBuilder {
	return newColumnBuilder(integerType("tinyint", length...))
}

// SmallInteger returns a smallint column.
func (m *MigrationContext) SmallInteger(length ...int) *ColumnBuilder {
	return newColumnBuilder(integerType("smallint", length...))
}

// Integer returns an int column.
func (m *MigrationContext) Integer(length ...int) *ColumnBuilder {
	return newColumnBuilder(integerType("int", length...))
}

// BigInteger returns a bigint column.
func (m *MigrationContext) BigInteger(length ...int) *ColumnBuilder {
	return newColumnBuilder(integerType("bigint", length...))
}

// UnsignedTinyInteger returns an unsigned tinyint column.
func (m *MigrationContext) UnsignedTinyInteger(length ...int) *ColumnBuilder {
	return m.TinyInteger(length...).Unsigned()
}

// UnsignedSmallInteger returns an unsigned smallint column.
func (m *MigrationContext) UnsignedSmallInteger(length ...int) *ColumnBuilder {
	return m.SmallInteger(length...).Unsigned()
}

// UnsignedInteger returns an unsigned int column.
func (m *MigrationContext) UnsignedInteger(length ...int) *ColumnBuilder {
	return m.Integer(length...).Unsigned()
}

// UnsignedBigInteger returns an unsigned bigint column.
func (m *MigrationContext) UnsignedBigInteger(length ...int) *ColumnBuilder {
	return m.BigInteger(length...).Unsigned()
}

// String returns a varchar column.
func (m *MigrationContext) String(size ...int) *ColumnBuilder {
	length := firstOrDefault(size, 255)
	return newColumnBuilder(fmt.Sprintf("varchar(%d)", length))
}

// Char returns a char column.
func (m *MigrationContext) Char(size int) *ColumnBuilder {
	return newColumnBuilder(fmt.Sprintf("char(%d)", size))
}

// Text returns a text column.
func (m *MigrationContext) Text() *ColumnBuilder {
	return newColumnBuilder("text")
}

// TinyText returns a tinytext column.
func (m *MigrationContext) TinyText() *ColumnBuilder {
	return newColumnBuilder("tinytext")
}

// MediumText returns a mediumtext column.
func (m *MigrationContext) MediumText() *ColumnBuilder {
	return newColumnBuilder("mediumtext")
}

// LongText returns a longtext column.
func (m *MigrationContext) LongText() *ColumnBuilder {
	return newColumnBuilder("longtext")
}

// Binary returns a varbinary column.
func (m *MigrationContext) Binary(length ...int) *ColumnBuilder {
	size := firstOrDefault(length, 255)
	return newColumnBuilder(fmt.Sprintf("varbinary(%d)", size))
}

// TinyBlob returns a tinyblob column.
func (m *MigrationContext) TinyBlob() *ColumnBuilder {
	return newColumnBuilder("tinyblob")
}

// MediumBlob returns a mediumblob column.
func (m *MigrationContext) MediumBlob() *ColumnBuilder {
	return newColumnBuilder("mediumblob")
}

// LongBlob returns a longblob column.
func (m *MigrationContext) LongBlob() *ColumnBuilder {
	return newColumnBuilder("longblob")
}

// Boolean returns a tinyint(1) column.
func (m *MigrationContext) Boolean() *ColumnBuilder {
	return newColumnBuilder("tinyint(1)")
}

// Float returns a float column.
func (m *MigrationContext) Float(precision ...int) *ColumnBuilder {
	return newColumnBuilder(precisionType("float", precision...))
}

// Double returns a double column.
func (m *MigrationContext) Double(precision ...int) *ColumnBuilder {
	return newColumnBuilder(precisionType("double", precision...))
}

// Decimal returns a decimal column.
func (m *MigrationContext) Decimal(precision, scale int) *ColumnBuilder {
	return newColumnBuilder(fmt.Sprintf("decimal(%d,%d)", precision, scale))
}

// Money returns a decimal column suitable for monetary values.
func (m *MigrationContext) Money(precision, scale int) *ColumnBuilder {
	return m.Decimal(precision, scale)
}

// Date returns a date column.
func (m *MigrationContext) Date() *ColumnBuilder {
	return newColumnBuilder("date")
}

// DateTime returns a datetime column.
func (m *MigrationContext) DateTime(precision ...int) *ColumnBuilder {
	return newColumnBuilder(temporalType("datetime", precision...))
}

// Time returns a time column.
func (m *MigrationContext) Time(precision ...int) *ColumnBuilder {
	return newColumnBuilder(temporalType("time", precision...))
}

// Timestamp returns a timestamp column.
func (m *MigrationContext) Timestamp(precision ...int) *ColumnBuilder {
	return newColumnBuilder(temporalType("timestamp", precision...))
}

// Json returns a json column.
func (m *MigrationContext) Json() *ColumnBuilder {
	return newColumnBuilder("json")
}

// UUID returns a char(36) column for UUID strings.
func (m *MigrationContext) UUID() *ColumnBuilder {
	return newColumnBuilder("char(36)")
}

// Enum returns an enum column.
func (m *MigrationContext) Enum(values ...string) *ColumnBuilder {
	return newColumnBuilder(enumSetType("enum", values))
}

// Set returns a set column.
func (m *MigrationContext) Set(values ...string) *ColumnBuilder {
	return newColumnBuilder(enumSetType("set", values))
}

func integerType(name string, length ...int) string {
	if len(length) == 0 {
		return name
	}
	return fmt.Sprintf("%s(%d)", name, length[0])
}

func precisionType(name string, precision ...int) string {
	switch len(precision) {
	case 0:
		return name
	case 1:
		return fmt.Sprintf("%s(%d)", name, precision[0])
	default:
		return fmt.Sprintf("%s(%d,%d)", name, precision[0], precision[1])
	}
}

func temporalType(name string, precision ...int) string {
	if len(precision) == 0 {
		return name
	}
	return fmt.Sprintf("%s(%d)", name, precision[0])
}

func enumSetType(name string, values []string) string {
	out := name + "("
	for i, value := range values {
		if i > 0 {
			out += ","
		}
		out += quoteSQLString(value)
	}
	return out + ")"
}

func firstOrDefault(values []int, fallback int) int {
	if len(values) == 0 {
		return fallback
	}
	return values[0]
}
