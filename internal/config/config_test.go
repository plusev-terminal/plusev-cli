package config

import (
	"os"
	"path/filepath"
	"testing"
)

func setTempConfigDir(t *testing.T) {
	t.Helper()

	dir := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", dir)
}

func TestHostFromURL(t *testing.T) {
	cases := map[string]string{
		"https://terminal.example.com/extapi": "terminal.example.com-extapi",
		"https://host:8080/extapi":            "host-8080-extapi",
		"http://localhost:8080/extapi":        "localhost-8080-extapi",
		"terminal.example.com/extapi":         "terminal.example.com-extapi",
	}

	for in, want := range cases {
		got := HostFromURL(in)
		if got != want {
			t.Errorf("HostFromURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	setTempConfigDir(t)

	cfg := &Config{
		Label:     "prod",
		BaseURL:   "https://terminal.example.com/extapi",
		APIKey:    "key-123",
		APISecret: "secret-456",
	}

	path, err := Save(cfg)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("config file mode = %o, want 0600", mode)
	}

	all, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}

	got, ok := all["terminal.example.com_prod"]
	if !ok {
		t.Fatalf("LoadAll did not return the prod config: %#v", all)
	}

	if got.BaseURL != cfg.BaseURL || got.APIKey != cfg.APIKey || got.APISecret != cfg.APISecret {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, cfg)
	}

	if got.Label != "terminal.example.com_prod" {
		t.Errorf("Label = %q, want %q", got.Label, "terminal.example.com_prod")
	}
}

func TestLoadAllMissingDirIsEmpty(t *testing.T) {
	setTempConfigDir(t)

	all, err := LoadAll()
	if err != nil {
		t.Fatalf("LoadAll on missing dir: %v", err)
	}

	if len(all) != 0 {
		t.Errorf("expected empty map, got %#v", all)
	}
}

func TestSaveRejectsEmptyLabel(t *testing.T) {
	setTempConfigDir(t)

	_, err := Save(&Config{BaseURL: "https://x"})
	if err == nil {
		t.Fatal("expected error for empty label, got nil")
	}

	// Ensure no file was created.
	dir, _ := Dir()
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected no files written, found %d", len(entries))
	}
}

func TestDirUnderXDG(t *testing.T) {
	setTempConfigDir(t)

	want := filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "plusev")

	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir: %v", err)
	}

	if got != want {
		t.Errorf("Dir = %q, want %q", got, want)
	}
}
