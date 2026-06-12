package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseFieldsBuildsColumnCode(t *testing.T) {
	fields, err := ParseFields("user_id:unsignedBigInteger:notNull,title:string(128):notNull,content:longText,metadata:json,status:unsignedTinyInteger:notNull:default(10)")
	if err != nil {
		t.Fatalf("ParseFields() error = %v", err)
	}

	got := make([]string, 0, len(fields))
	for _, field := range fields {
		got = append(got, field.ColumnCode())
	}
	want := []string{
		`Add("user_id", m.UnsignedBigInteger().NotNull())`,
		`Add("title", m.String(128).NotNull())`,
		`Add("content", m.LongText().Null())`,
		`Add("metadata", m.Json().Null())`,
		`Add("status", m.UnsignedTinyInteger().NotNull().DefaultValue(10))`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("ColumnCode() = %#v, want %#v", got, want)
	}
}

func TestGenerateMigrationWritesFileAndRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	opts := Options{
		Name:          "create_article_table",
		Dir:           dir,
		PackageName:   "migrations",
		MigrateImport: "github.com/phgors/yiimigrate/migrate",
		Now:           func() time.Time { return time.Date(2026, 6, 13, 10, 30, 0, 0, time.UTC) },
		Fields:        "title:string(128):notNull,content:longText",
	}

	path, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if filepath.Base(path) != "m20260613_103000_create_article_table.go" {
		t.Fatalf("file name = %q", filepath.Base(path))
	}
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(contentBytes)
	for _, want := range []string{
		"type M20260613_103000CreateArticleTable struct{}",
		`return "m20260613_103000_create_article_table"`,
		`CreateTable("article", migrate.Columns().`,
		`Add("id", m.UnsignedBigPrimaryKey()).`,
		`Add("title", m.String(128).NotNull()).`,
		`Add("content", m.LongText().Null()),`,
		`DropTable("article").`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("generated content missing %q:\n%s", want, content)
		}
	}

	if _, err := Generate(opts); err == nil {
		t.Fatalf("Generate() overwrote existing file")
	}
}
