package main

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"strings"

	"soln-teachermodule/database"

	"github.com/go-sql-driver/mysql"
)

// showcaseDB names a throwaway MariaDB database built from the schema plus the generated
// showcase data, and the dedicated user the app connects as.
type showcaseDB struct {
	Name, User, Pass string
	Extra            string // more SQL to run after the showcase data (test fixtures), or ""
}

// loadShowcaseDatabase drops and recreates db.Name from soln_db.sql + testdata/showcase_seed.sql
// (+ db.Extra), creates db.User with access to it, points DBUSER/DBPASS/DBNAME/DBHOST/DBPORT at it
// and runs database.InitializeDatabase. rootDSN must be allowed to create databases and users, and
// both SQL files are read relative to the working directory (the repo root). It returns the
// server's host:port. Shared by the integration tests (main_test.go) and the static export
// (export.go); it never touches the dev database.
func loadShowcaseDatabase(rootDSN string, db showcaseDB) (string, error) {
	cfg, err := mysql.ParseDSN(rootDSN)
	if err != nil {
		return "", fmt.Errorf("parsing root DSN: %w", err)
	}
	host, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return "", fmt.Errorf("root DSN address %q: %w", cfg.Addr, err)
	}
	cfg.DBName = ""
	cfg.MultiStatements = true
	root, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return "", err
	}
	defer root.Close()
	if _, err := root.Exec(fmt.Sprintf(`
		DROP DATABASE IF EXISTS %[1]s;
		CREATE DATABASE %[1]s;
		CREATE USER IF NOT EXISTS '%[2]s'@'%%' IDENTIFIED BY '%[3]s';
		ALTER USER '%[2]s'@'%%' IDENTIFIED BY '%[3]s';
		GRANT ALL PRIVILEGES ON %[1]s.* TO '%[2]s'@'%%';
		FLUSH PRIVILEGES;`, db.Name, db.User, db.Pass)); err != nil {
		return "", fmt.Errorf("creating %s: %w", db.Name, err)
	}

	cfg.DBName = db.Name
	loader, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return "", err
	}
	defer loader.Close()
	loader.SetMaxOpenConns(1)
	schema, err := os.ReadFile("soln_db.sql")
	if err != nil {
		return "", err
	}
	// soln_db.sql targets the dev database by name; point it at this one instead.
	schemaSQL := strings.NewReplacer(
		"CREATE DATABASE IF NOT EXISTS soln_db;", "",
		"USE soln_db;", "USE "+db.Name+";",
	).Replace(string(schema))
	if _, err := loader.Exec(schemaSQL); err != nil {
		return "", fmt.Errorf("loading soln_db.sql: %w", err)
	}
	seed, err := os.ReadFile("testdata/showcase_seed.sql")
	if err != nil {
		return "", err
	}
	if _, err := loader.Exec(string(seed)); err != nil {
		return "", fmt.Errorf("loading showcase_seed.sql: %w", err)
	}
	if db.Extra != "" {
		if _, err := loader.Exec(db.Extra); err != nil {
			return "", fmt.Errorf("loading extra SQL: %w", err)
		}
	}

	for key, value := range map[string]string{
		"DBUSER": db.User, "DBPASS": db.Pass, "DBNAME": db.Name, "DBHOST": host, "DBPORT": port,
	} {
		os.Setenv(key, value)
	}
	if err := database.InitializeDatabase(); err != nil {
		return "", err
	}
	return cfg.Addr, nil
}
