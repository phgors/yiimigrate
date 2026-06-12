package migrations

import (
	"context"

	"github.com/phgors/yiimigrate/migrate"
)

// M20260613_120000CreateArticleTable demonstrates a Yii2-style Go migration.
type M20260613_120000CreateArticleTable struct{}

// Name returns the migration version.
func (M20260613_120000CreateArticleTable) Name() string {
	return "m20260613_120000_create_article_table"
}

// Up applies the migration.
func (M20260613_120000CreateArticleTable) Up(ctx context.Context, m *migrate.MigrationContext) error {
	exists, err := m.RowExists(ctx, "article", "slug = ?", "hello")
	if err != nil {
		return err
	}

	plan := m.Schema().
		CreateTableIfNotExists(ctx, "article", migrate.Columns().
			Add("id", m.UnsignedBigPrimaryKey()).
			Add("user_id", m.UnsignedBigInteger().NotNull()).
			Add("title", m.String(128).NotNull().Unique()).
			Add("slug", m.String(128).NotNull().Unique()).
			Add("content", m.LongText().Null()).
			Add("metadata", m.Json().Null()).
			Add("status", m.UnsignedTinyInteger().NotNull().DefaultValue(10)).
			Add("created_at", m.Timestamp(0).NotNull().DefaultExpression("CURRENT_TIMESTAMP")),
			"ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		).
		CreateIndexIfNotExists(ctx, "idx-article-user_id", "article", []string{"user_id"}, false).
		AddForeignKeyIfNotExists(ctx, "fk-article-user_id", "article", []string{"user_id"}, "user", []string{"id"}, migrate.Cascade, migrate.Cascade)

	if !exists {
		plan.Insert("article", migrate.Row{
			"user_id":  1,
			"title":    "Hello",
			"slug":     "hello",
			"content":  "Hello world",
			"metadata": migrate.Expr("JSON_OBJECT('source', 'migration')"),
			"status":   10,
		})
	}

	return plan.Exec(ctx)
}

// Down reverts the migration.
func (M20260613_120000CreateArticleTable) Down(ctx context.Context, m *migrate.MigrationContext) error {
	return m.Schema().
		DropForeignKeyIfExists(ctx, "fk-article-user_id", "article").
		DropIndexIfExists(ctx, "idx-article-user_id", "article").
		DropTableIfExists(ctx, "article").
		Exec(ctx)
}
