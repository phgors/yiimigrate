package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
	_ "modernc.org/sqlite"

	"github.com/phgors/yiimigrate/internal/generator"
	"github.com/phgors/yiimigrate/migrate"
	"github.com/phgors/yiimigrate/migrations"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: migrate <up|down|redo|history|new|mark|to|create>")
	}
	switch args[0] {
	case "create":
		return runCreate(args[1:])
	case "up", "down", "redo", "history", "new", "mark", "to":
		return runDBCommand(args[0], args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runCreate(args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fields := fs.String("fields", "", "comma-separated fields")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: migrate create NAME [--fields=...]")
	}
	path, err := generator.Generate(generator.Options{
		Name:          fs.Arg(0),
		Dir:           "migrations",
		PackageName:   "migrations",
		MigrateImport: "github.com/phgors/yiimigrate/migrate",
		Fields:        *fields,
	})
	if err != nil {
		return err
	}
	fmt.Printf("created %s\n", path)
	return nil
}

type dbConfig struct {
	driverName string
	dialect    migrate.Dialect
}

func resolveDBConfig(name string) (dbConfig, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "mysql":
		return dbConfig{driverName: "mysql", dialect: migrate.MySQLDialect{}}, nil
	case "sqlite", "sqlite3":
		return dbConfig{driverName: "sqlite", dialect: migrate.SQLiteDialect{}}, nil
	case "postgres", "postgresql":
		return dbConfig{driverName: "postgres", dialect: migrate.PostgreSQLDialect{}}, nil
	case "sqlserver", "mssql":
		return dbConfig{driverName: "sqlserver", dialect: migrate.SQLServerDialect{}}, nil
	default:
		return dbConfig{}, fmt.Errorf("unsupported DB_DIALECT %q; supported values: mysql, sqlite, postgres, sqlserver", name)
	}
}

func runDBCommand(command string, args []string) error {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		return errors.New("DB_DSN is required")
	}
	config, err := resolveDBConfig(os.Getenv("DB_DIALECT"))
	if err != nil {
		return err
	}
	db, err := sql.Open(config.driverName, dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	migrator := migrate.NewMigrator(db, config.dialect, migrations.All())
	if table := os.Getenv("MIGRATE_TABLE"); table != "" {
		migrator.MigrationTable = table
	}
	migrator.DryRun = envBool("MIGRATE_DRY_RUN")

	ctx := context.Background()
	switch command {
	case "up":
		_, err = migrator.Up(ctx, optionalInt(args, 0))
	case "down":
		_, err = migrator.Down(ctx, optionalInt(args, 1))
	case "redo":
		_, _, err = migrator.Redo(ctx, optionalInt(args, 1))
	case "history":
		history, historyErr := migrator.History(ctx, optionalInt(args, 10))
		if historyErr != nil {
			return historyErr
		}
		for _, item := range history {
			fmt.Printf("%s %d\n", item.Version, item.ApplyTime)
		}
	case "new":
		pending, pendingErr := migrator.New(ctx, optionalInt(args, 10))
		if pendingErr != nil {
			return pendingErr
		}
		for _, migration := range pending {
			fmt.Println(migration.Name())
		}
	case "mark":
		if len(args) != 1 {
			return errors.New("usage: migrate mark VERSION")
		}
		err = migrator.Mark(ctx, args[0])
	case "to":
		if len(args) != 1 {
			return errors.New("usage: migrate to VERSION")
		}
		err = migrator.To(ctx, args[0])
	}
	return err
}

func optionalInt(args []string, fallback int) int {
	if len(args) == 0 {
		return fallback
	}
	value, err := strconv.Atoi(args[0])
	if err != nil {
		return fallback
	}
	return value
}

func envBool(name string) bool {
	value := os.Getenv(name)
	return value == "1" || value == "true" || value == "TRUE" || value == "yes"
}
