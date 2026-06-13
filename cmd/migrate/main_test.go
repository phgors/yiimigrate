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

func TestResolveDBConfigSupportsPostgreSQLAliases(t *testing.T) {
	tests := []string{"postgres", "postgresql", " PostgreSQL "}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			config, err := resolveDBConfig(input)
			if err != nil {
				t.Fatalf("resolveDBConfig returned error: %v", err)
			}
			if config.driverName != "postgres" {
				t.Fatalf("driverName = %q, want postgres", config.driverName)
			}
			if _, ok := config.dialect.(migrate.PostgreSQLDialect); !ok {
				t.Fatalf("dialect = %T, want migrate.PostgreSQLDialect", config.dialect)
			}
		})
	}
}

func TestResolveDBConfigSupportsSQLServerAliases(t *testing.T) {
	tests := []string{"sqlserver", "mssql", " MSSQL "}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			config, err := resolveDBConfig(input)
			if err != nil {
				t.Fatalf("resolveDBConfig returned error: %v", err)
			}
			if config.driverName != "sqlserver" {
				t.Fatalf("driverName = %q, want sqlserver", config.driverName)
			}
			if _, ok := config.dialect.(migrate.SQLServerDialect); !ok {
				t.Fatalf("dialect = %T, want migrate.SQLServerDialect", config.dialect)
			}
		})
	}
}

func TestResolveDBConfigRejectsUnsupportedDialect(t *testing.T) {
	_, err := resolveDBConfig("oracle")
	if err == nil {
		t.Fatal("resolveDBConfig returned nil error")
	}
	if !strings.Contains(err.Error(), `unsupported DB_DIALECT "oracle"`) {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "mysql, sqlite, postgres, sqlserver") {
		t.Fatalf("error does not list supported dialects: %v", err)
	}
}
