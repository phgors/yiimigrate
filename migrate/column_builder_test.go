package migrate

import "testing"

func TestColumnBuilderIsImmutable(t *testing.T) {
	migration := NewMigrationContext(nil, MySQLDialect{})

	base := migration.String(64)
	required := base.NotNull()
	optional := base.Null().DefaultValue("guest")

	if base == required {
		t.Fatalf("NotNull() returned the original builder")
	}
	if base == optional {
		t.Fatalf("Null() returned the original builder")
	}

	assertColumnSQL(t, base, "varchar(64)")
	assertColumnSQL(t, required, "varchar(64) NOT NULL")
	assertColumnSQL(t, optional, "varchar(64) NULL DEFAULT 'guest'")
}

func TestColumnBuilderRendersIntegerTypes(t *testing.T) {
	migration := NewMigrationContext(nil, MySQLDialect{})
	tests := []struct {
		name    string
		builder *ColumnBuilder
		want    string
	}{
		{name: "primary key", builder: migration.PrimaryKey(), want: "int NOT NULL AUTO_INCREMENT PRIMARY KEY"},
		{name: "big primary key", builder: migration.BigPrimaryKey(), want: "bigint NOT NULL AUTO_INCREMENT PRIMARY KEY"},
		{name: "unsigned primary key", builder: migration.UnsignedPrimaryKey(), want: "int UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY"},
		{name: "unsigned big primary key", builder: migration.UnsignedBigPrimaryKey(), want: "bigint UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY"},
		{name: "tiny integer", builder: migration.TinyInteger(), want: "tinyint"},
		{name: "tiny integer length", builder: migration.TinyInteger(3), want: "tinyint(3)"},
		{name: "small integer", builder: migration.SmallInteger(), want: "smallint"},
		{name: "integer", builder: migration.Integer(), want: "int"},
		{name: "big integer", builder: migration.BigInteger(), want: "bigint"},
		{name: "unsigned tiny integer", builder: migration.UnsignedTinyInteger(), want: "tinyint UNSIGNED"},
		{name: "unsigned small integer", builder: migration.UnsignedSmallInteger(), want: "smallint UNSIGNED"},
		{name: "unsigned integer", builder: migration.UnsignedInteger(), want: "int UNSIGNED"},
		{name: "unsigned big integer", builder: migration.UnsignedBigInteger(), want: "bigint UNSIGNED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertColumnSQL(t, tt.builder, tt.want)
		})
	}
}

func TestColumnBuilderRendersTextAndBinaryTypes(t *testing.T) {
	migration := NewMigrationContext(nil, MySQLDialect{})
	tests := []struct {
		name    string
		builder *ColumnBuilder
		want    string
	}{
		{name: "string default", builder: migration.String(), want: "varchar(255)"},
		{name: "string size", builder: migration.String(128), want: "varchar(128)"},
		{name: "char", builder: migration.Char(36), want: "char(36)"},
		{name: "text", builder: migration.Text(), want: "text"},
		{name: "tiny text", builder: migration.TinyText(), want: "tinytext"},
		{name: "medium text", builder: migration.MediumText(), want: "mediumtext"},
		{name: "long text", builder: migration.LongText(), want: "longtext"},
		{name: "binary default", builder: migration.Binary(), want: "varbinary(255)"},
		{name: "binary length", builder: migration.Binary(16), want: "varbinary(16)"},
		{name: "tiny blob", builder: migration.TinyBlob(), want: "tinyblob"},
		{name: "medium blob", builder: migration.MediumBlob(), want: "mediumblob"},
		{name: "long blob", builder: migration.LongBlob(), want: "longblob"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertColumnSQL(t, tt.builder, tt.want)
		})
	}
}

func TestColumnBuilderRendersNumericAndDateTypes(t *testing.T) {
	migration := NewMigrationContext(nil, MySQLDialect{})
	tests := []struct {
		name    string
		builder *ColumnBuilder
		want    string
	}{
		{name: "boolean", builder: migration.Boolean(), want: "tinyint(1)"},
		{name: "float", builder: migration.Float(), want: "float"},
		{name: "float precision", builder: migration.Float(10, 2), want: "float(10,2)"},
		{name: "double", builder: migration.Double(), want: "double"},
		{name: "double precision", builder: migration.Double(12, 4), want: "double(12,4)"},
		{name: "decimal", builder: migration.Decimal(10, 2), want: "decimal(10,2)"},
		{name: "money", builder: migration.Money(19, 4), want: "decimal(19,4)"},
		{name: "date", builder: migration.Date(), want: "date"},
		{name: "datetime", builder: migration.DateTime(), want: "datetime"},
		{name: "datetime precision", builder: migration.DateTime(3), want: "datetime(3)"},
		{name: "time", builder: migration.Time(), want: "time"},
		{name: "time precision", builder: migration.Time(6), want: "time(6)"},
		{name: "timestamp", builder: migration.Timestamp(), want: "timestamp"},
		{name: "timestamp precision", builder: migration.Timestamp(0), want: "timestamp(0)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertColumnSQL(t, tt.builder, tt.want)
		})
	}
}

func TestColumnBuilderRendersSpecialTypes(t *testing.T) {
	migration := NewMigrationContext(nil, MySQLDialect{})
	tests := []struct {
		name    string
		builder *ColumnBuilder
		want    string
	}{
		{name: "json", builder: migration.Json(), want: "json"},
		{name: "uuid", builder: migration.UUID(), want: "char(36)"},
		{name: "enum", builder: migration.Enum("draft", "published"), want: "enum('draft','published')"},
		{name: "enum escapes quotes", builder: migration.Enum("reader's pick"), want: "enum('reader''s pick')"},
		{name: "set", builder: migration.Set("read", "write"), want: "set('read','write')"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertColumnSQL(t, tt.builder, tt.want)
		})
	}
}

func TestColumnBuilderRendersChainClauses(t *testing.T) {
	migration := NewMigrationContext(nil, MySQLDialect{})

	assertColumnSQL(t,
		migration.String(64).
			NotNull().
			Unique().
			DefaultValue("admin").
			Comment("用户名").
			Charset("utf8mb4").
			Collate("utf8mb4_bin").
			Check("username <> ''").
			After("id").
			Append("VISIBLE"),
		"varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL UNIQUE DEFAULT 'admin' COMMENT '用户名' CHECK (username <> '') VISIBLE AFTER `id`",
	)

	assertColumnSQL(t,
		migration.Timestamp(0).Null().DefaultExpression("CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP").First(),
		"timestamp(0) NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP FIRST",
	)

	assertColumnSQL(t,
		migration.Decimal(10, 2).Unsigned().NotNull().DefaultValue(0),
		"decimal(10,2) UNSIGNED NOT NULL DEFAULT 0",
	)

	assertColumnSQL(t,
		migration.Integer().GeneratedAs("JSON_EXTRACT(metadata, '$.status')").Stored(),
		"int GENERATED ALWAYS AS (JSON_EXTRACT(metadata, '$.status')) STORED",
	)

	assertColumnSQL(t,
		migration.String(32).GeneratedAs("LOWER(email)").Virtual(),
		"varchar(32) GENERATED ALWAYS AS (LOWER(email)) VIRTUAL",
	)
}

func assertColumnSQL(t *testing.T, builder *ColumnBuilder, want string) {
	t.Helper()
	got := MySQLDialect{}.columnSQL(builder)
	if got != want {
		t.Fatalf("columnSQL() = %q, want %q", got, want)
	}
}
