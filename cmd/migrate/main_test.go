package main

import (
	"strings"
	"testing"

	"github.com/phgors/yiimigrate/migrate"
)

func TestResolveDBConfigDefaultsToMySQL(t *testing.T) {
	config, err := resolveDBConfig("")
	if err != nil {
		t.Fatalf("resolveDBConfig returned error: %v", err)
	}
	if config.driverName != "mysql" {
		t.Fatalf("driverName = %q, want mysql", config.driverName)
	}
	if _, ok := config.dialect.(migrate.MySQLDialect); !ok {
		t.Fatalf("dialect = %T, want migrate.MySQLDialect", config.dialect)
	}
}

func TestResolveDBConfigSupportsSQLiteAliases(t *testing.T) {
	tests := []string{"sqlite", "sqlite3", " SQLite3 "}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			config, err := resolveDBConfig(input)
			if err != nil {
				t.Fatalf("resolveDBConfig returned error: %v", err)
			}
			if config.driverName != "sqlite" {
				t.Fatalf("driverName = %q, want sqlite", config.driverName)
			}
			if _, ok := config.dialect.(migrate.SQLiteDialect); !ok {
				t.Fatalf("dialect = %T, want migrate.SQLiteDialect", config.dialect)
			}
		})
	}
}

func TestResolveDBConfigRejectsUnsupportedDialect(t *testing.T) {
	_, err := resolveDBConfig("postgres")
	if err == nil {
		t.Fatal("resolveDBConfig returned nil error")
	}
	if !strings.Contains(err.Error(), `unsupported DB_DIALECT "postgres"`) {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "mysql, sqlite, sqlite3") {
		t.Fatalf("error does not list supported dialects: %v", err)
	}
}
