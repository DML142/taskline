package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const migrationsSourceURL = "file://migrations"

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseCommand(args []string) (string, error) {
	if len(args) == 1 {
		switch args[0] {
		case "up", "down", "version":
			return args[0], nil
		}
	}
	return "", fmt.Errorf("command must be one of: up, down, version")
}

func run(args []string, stdout io.Writer) (runErr error) {
	command, err := parseCommand(args)
	if err != nil {
		return err
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	migrationDatabaseURL, err := pgxMigrationURL(databaseURL)
	if err != nil {
		return errors.New("DATABASE_URL must be a valid PostgreSQL connection string")
	}
	migrator, err := migrate.New(migrationsSourceURL, migrationDatabaseURL)
	if err != nil {
		return errors.New("initialize migrations")
	}
	defer func() {
		sourceErr, databaseErr := migrator.Close()
		if runErr == nil && (sourceErr != nil || databaseErr != nil) {
			runErr = errors.New("close migrations")
		}
	}()

	switch command {
	case "up":
		_, _, versionErr := migrator.Version()
		if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) && !isEmptyMigrationNoOp(versionErr, err) {
			return errors.New("apply migrations")
		}
	case "down":
		if err := migrator.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return errors.New("rollback migrations")
		}
	case "version":
		version, dirty, err := migrator.Version()
		if errors.Is(err, migrate.ErrNilVersion) {
			_, err = fmt.Fprintln(stdout, "none")
			return err
		}
		if err != nil {
			return errors.New("read migration version")
		}
		if dirty {
			_, err = fmt.Fprintf(stdout, "%d (dirty)\n", version)
		} else {
			_, err = fmt.Fprintln(stdout, version)
		}
		if err != nil {
			return errors.New("write migration version")
		}
	}

	return nil
}

func isEmptyMigrationNoOp(versionErr, migrationErr error) bool {
	var pathError *fs.PathError
	return errors.Is(versionErr, migrate.ErrNilVersion) &&
		errors.As(migrationErr, &pathError) &&
		pathError.Op == "first" &&
		errors.Is(pathError.Err, fs.ErrNotExist)
}

func pgxMigrationURL(databaseURL string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return "", errors.New("unsupported database URL scheme")
	}
	parsed.Scheme = "pgx5"
	return parsed.String(), nil
}
