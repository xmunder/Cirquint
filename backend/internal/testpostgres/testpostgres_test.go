package testpostgres

import "testing"

func TestSchemaNameFromScopeSanitizesPackagePath(t *testing.T) {
	t.Parallel()

	if got, want := schemaNameFromScope("cmd/api"), "test_cmd_api"; got != want {
		t.Fatalf("schemaNameFromScope() = %q, want %q", got, want)
	}

	if got, want := schemaNameFromScope("internal/uploads"), "test_internal_uploads"; got != want {
		t.Fatalf("schemaNameFromScope() = %q, want %q", got, want)
	}
}

func TestConnectionURLAddsSearchPathAndPreservesExistingQuery(t *testing.T) {
	t.Parallel()

	got, err := connectionURL("postgres://postgres:postgres@127.0.0.1:55432/cirquint_test?sslmode=disable", "test_cmd_api")
	if err != nil {
		t.Fatalf("connectionURL() error = %v", err)
	}

	want := "postgres://postgres:postgres@127.0.0.1:55432/cirquint_test?search_path=test_cmd_api&sslmode=disable"
	if got != want {
		t.Fatalf("connectionURL() = %q, want %q", got, want)
	}
}
