package testpostgres

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/lib/pq"
)

func Open(t *testing.T, scope string) *sql.DB {
	t.Helper()
	isolatedURL, schema := mustIsolatedURL(t, scope)

	db, err := sql.Open("postgres", isolatedURL)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	resetSchema(t, db, schema)
	applyMigrations(t, db)
	return db
}

func URL(t *testing.T, scope string) string {
	t.Helper()
	isolatedURL, _ := mustIsolatedURL(t, scope)
	return isolatedURL
}

func schemaNameFromScope(scope string) string {
	var b strings.Builder
	b.WriteString("test_")
	for _, r := range scope {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		default:
			b.WriteByte('_')
		}
	}
	return strings.TrimRight(b.String(), "_")
}

func connectionURL(databaseURL, schema string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func mustIsolatedURL(t *testing.T, scope string) (string, string) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if strings.TrimSpace(databaseURL) == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	schema := schemaNameFromScope(scope)
	isolatedURL, err := connectionURL(databaseURL, schema)
	if err != nil {
		t.Fatalf("build isolated postgres url: %v", err)
	}
	return isolatedURL, schema
}

func resetSchema(t *testing.T, db *sql.DB, schema string) {
	t.Helper()
	statement := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE; CREATE SCHEMA %s;", schema, schema)
	if _, err := db.ExecContext(context.Background(), statement); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
}

func applyMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, path := range []string{
		filepath.Join("..", "..", "migrations", "0001_initial_schema.sql"),
		filepath.Join("..", "..", "migrations", "0002_circuit_review_decisions.sql"),
	} {
		payload, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		for _, stmt := range strings.Split(string(payload), ";") {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.ExecContext(context.Background(), stmt); err != nil {
				t.Fatalf("apply migration %s: %v", path, err)
			}
		}
	}
}
