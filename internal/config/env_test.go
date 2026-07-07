package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("TEST_AFI_DOTENV=from-file\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	os.Unsetenv("TEST_AFI_DOTENV")
	LoadDotEnv(path)
	if got := os.Getenv("TEST_AFI_DOTENV"); got != "from-file" {
		t.Fatalf("expected from-file, got %q", got)
	}

	t.Setenv("TEST_AFI_DOTENV", "from-shell")
	LoadDotEnv(path)
	if got := os.Getenv("TEST_AFI_DOTENV"); got != "from-shell" {
		t.Fatalf("expected existing env to win, got %q", got)
	}
}
