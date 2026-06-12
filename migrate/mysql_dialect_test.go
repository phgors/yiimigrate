package migrate

import "testing"

func TestMySQLDialectQuotesIndexColumns(t *testing.T) {
	dialect := MySQLDialect{}
	tests := []struct {
		name   string
		column string
		want   string
	}{
		{name: "simple column", column: "email", want: "`email`"},
		{name: "qualified column", column: "user.email", want: "`user`.`email`"},
		{name: "already quoted", column: "`email`", want: "`email`"},
		{name: "lower expression", column: "LOWER(email)", want: "LOWER(email)"},
		{name: "json expression", column: "JSON_EXTRACT(metadata, '$.slug')", want: "JSON_EXTRACT(metadata, '$.slug')"},
		{name: "prefix length", column: "title(32)", want: "`title`(32)"},
		{name: "direction", column: "created_at DESC", want: "`created_at` DESC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := dialect.QuoteIndexColumn(tt.column); got != tt.want {
				t.Fatalf("QuoteIndexColumn(%q) = %q, want %q", tt.column, got, tt.want)
			}
		})
	}
}

func TestMySQLDialectBuildsTableDDL(t *testing.T) {
	dialect := MySQLDialect{}
	ctx := NewMigrationContext(nil, dialect)
	columns := Columns().
		Add("id", ctx.UnsignedBigPrimaryKey()).
		Add("email", ctx.String(128).NotNull().Unique()).
		Add("metadata", ctx.Json().Null()).
		Add("created_at", ctx.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP"))

	assertSQL(t, dialect.CreateTable("user", columns, "ENGINE=InnoDB DEFAULT CHARSET=utf8mb4"), "CREATE TABLE `user` (`id` bigint UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY, `email` varchar(128) NOT NULL UNIQUE, `metadata` json NULL, `created_at` timestamp(0) NOT NULL DEFAULT CURRENT_TIMESTAMP) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4")
	assertSQL(t, dialect.DropTable("user"), "DROP TABLE `user`")
	assertSQL(t, dialect.RenameTable("user", "account"), "RENAME TABLE `user` TO `account`")
	assertSQL(t, dialect.TruncateTable("user"), "TRUNCATE TABLE `user`")
}

func TestMySQLDialectBuildsColumnDDL(t *testing.T) {
	dialect := MySQLDialect{}
	ctx := NewMigrationContext(nil, dialect)

	assertSQL(t, dialect.AddColumn("user", "email", ctx.String(128).NotNull().After("id")), "ALTER TABLE `user` ADD COLUMN `email` varchar(128) NOT NULL AFTER `id`")
	assertSQL(t, dialect.AlterColumn("user", "email", ctx.String(255).Null()), "ALTER TABLE `user` MODIFY COLUMN `email` varchar(255) NULL")
	assertSQL(t, dialect.DropColumn("user", "email"), "ALTER TABLE `user` DROP COLUMN `email`")
	assertSQL(t, dialect.RenameColumn("user", "email", "email_address"), "ALTER TABLE `user` RENAME COLUMN `email` TO `email_address`")
}

func TestMySQLDialectBuildsIndexAndKeyDDL(t *testing.T) {
	dialect := MySQLDialect{}

	assertSQL(t, dialect.CreateIndex("idx-user-email", "user", []string{"email"}, true), "CREATE UNIQUE INDEX `idx-user-email` ON `user` (`email`)")
	assertSQL(t, dialect.CreateIndex("idx-user-lower-email", "user", []string{"LOWER(email)"}, false), "CREATE INDEX `idx-user-lower-email` ON `user` (LOWER(email))")
	assertSQL(t, dialect.CreateIndex("idx-user-title", "user", []string{"title(32)", "created_at DESC"}, false), "CREATE INDEX `idx-user-title` ON `user` (`title`(32), `created_at` DESC)")
	assertSQL(t, dialect.DropIndex("idx-user-email", "user"), "DROP INDEX `idx-user-email` ON `user`")

	assertSQL(t, dialect.AddPrimaryKey("pk-user", "user", []string{"id"}), "ALTER TABLE `user` ADD CONSTRAINT `pk-user` PRIMARY KEY (`id`)")
	assertSQL(t, dialect.DropPrimaryKey("pk-user", "user"), "ALTER TABLE `user` DROP PRIMARY KEY")
}

func TestMySQLDialectBuildsForeignKeyDDL(t *testing.T) {
	dialect := MySQLDialect{}

	assertSQL(t,
		dialect.AddForeignKey(
			"fk-profile-user_id",
			"profile",
			[]string{"user_id"},
			"user",
			[]string{"id"},
			Cascade,
			Restrict,
		),
		"ALTER TABLE `profile` ADD CONSTRAINT `fk-profile-user_id` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`) ON DELETE CASCADE ON UPDATE RESTRICT",
	)
	assertSQL(t, dialect.DropForeignKey("fk-profile-user_id", "profile"), "ALTER TABLE `profile` DROP FOREIGN KEY `fk-profile-user_id`")
	assertSQL(t,
		dialect.AddForeignKey("fk-post-user_id", "post", []string{"user_id"}, "user", []string{"id"}, NoAction, SetNull),
		"ALTER TABLE `post` ADD CONSTRAINT `fk-post-user_id` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`) ON DELETE NO ACTION ON UPDATE SET NULL",
	)
}

func TestMySQLDialectBuildsCommentDDL(t *testing.T) {
	dialect := MySQLDialect{}

	assertSQL(t, dialect.AddCommentOnColumn("user", "email", "邮箱"), "ALTER TABLE `user` MODIFY COLUMN `email` COMMENT '邮箱'")
	assertSQL(t, dialect.DropCommentFromColumn("user", "email"), "ALTER TABLE `user` MODIFY COLUMN `email` COMMENT ''")
	assertSQL(t, dialect.AddCommentOnTable("user", "用户表"), "ALTER TABLE `user` COMMENT = '用户表'")
	assertSQL(t, dialect.DropCommentFromTable("user"), "ALTER TABLE `user` COMMENT = ''")
}

func TestColumnListPreservesDeclarationOrder(t *testing.T) {
	ctx := NewMigrationContext(nil, MySQLDialect{})
	columns := Columns().
		Add("id", ctx.Integer()).
		Add("title", ctx.String(128)).
		Add("body", ctx.LongText())

	items := columns.Items()
	if len(items) != 3 {
		t.Fatalf("len(Items()) = %d, want 3", len(items))
	}
	if items[0].Name != "id" || items[1].Name != "title" || items[2].Name != "body" {
		t.Fatalf("Items() order = %#v", []string{items[0].Name, items[1].Name, items[2].Name})
	}

	items[0].Name = "mutated"
	if got := columns.Items()[0].Name; got != "id" {
		t.Fatalf("Items() leaked internal slice, first name = %q", got)
	}
}

func assertSQL(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("SQL = %q, want %q", got, want)
	}
}
