package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolate points the config at a throwaway directory, so no test can read or
// write the developer's real session.
func isolate(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	return filepath.Join(base, "journal")
}

func modeOf(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Mode().Perm()
}

func TestPathIsYAML(t *testing.T) {
	isolate(t)
	if got := filepath.Base(Path()); got != "config.yml" {
		t.Fatalf("config file is %q, want config.yml (CLI-STANDARD §6.1)", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	isolate(t)

	want := Config{URL: "https://journal.example.com", Token: "tok-123"}
	if err := Save(want); err != nil {
		t.Fatalf("save: %v", err)
	}

	raw, err := os.ReadFile(Path())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(raw) != "url: https://journal.example.com\ntoken: tok-123\n" {
		t.Fatalf("on-disk form is %q, want plain YAML with url and token keys", raw)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got != want {
		t.Fatalf("round trip gave %+v, want %+v", got, want)
	}
}

func TestSavePermissions(t *testing.T) {
	dir := isolate(t)

	if err := Save(Config{URL: DefaultURL, Token: "tok"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	if got := modeOf(t, Path()); got != 0o600 {
		t.Fatalf("config mode is %#o, want 0600", got)
	}
	if got := modeOf(t, dir); got != 0o700 {
		t.Fatalf("config dir mode is %#o, want 0700", got)
	}
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	isolate(t)

	got, err := Load()
	if err != nil {
		t.Fatalf("load with no config must not error, got %v", err)
	}
	if got.URL != DefaultURL || got.Token != "" {
		t.Fatalf("got %+v, want the default URL and no token", got)
	}
}

// CLI-STANDARD §6.2: a credential file found group- or world-readable is
// tightened on read, because installs predate the rule.
func TestLoadTightensLoosePermissions(t *testing.T) {
	isolate(t)

	if err := Save(Config{URL: DefaultURL, Token: "tok"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := os.Chmod(Path(), 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Token != "tok" {
		t.Fatalf("token is %q, want it read back intact", got.Token)
	}
	if got := modeOf(t, Path()); got != 0o600 {
		t.Fatalf("mode is %#o after load, want it tightened to 0600", got)
	}
}

func TestClearKeepsURL(t *testing.T) {
	isolate(t)

	if err := Save(Config{URL: "https://journal.example.com", Token: "tok"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := Clear(); err != nil {
		t.Fatalf("clear: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Token != "" {
		t.Fatalf("token is %q after clear, want empty", got.Token)
	}
	if got.URL != "https://journal.example.com" {
		t.Fatalf("url is %q after clear, want it kept", got.URL)
	}
}

func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"":                              "",
		"  ":                            "",
		"journal.example.com":           "https://journal.example.com",
		"https://journal.example.com/":  "https://journal.example.com",
		"http://127.0.0.1:8080":         "http://127.0.0.1:8080",
		" https://journal.example.com ": "https://journal.example.com",
	}
	for in, want := range cases {
		if got := NormalizeURL(in); got != want {
			t.Errorf("NormalizeURL(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestClearSurvivesAnUnparseableFile pins the property that matters most about
// logout: a credential that cannot be parsed must still be removed. Refusing
// because the YAML is malformed leaves a working token exactly where somebody
// tried to delete it.
func TestClearSurvivesAnUnparseableFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if err := os.MkdirAll(filepath.Join(dir, "journal"), 0o700); err != nil {
		t.Fatal(err)
	}
	corrupt := []byte("url: https://journal.facile.studio\ntoken: still-valid\n  bad: [unclosed\n")
	if err := os.WriteFile(Path(), corrupt, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Clear(); err != nil {
		t.Fatalf("Clear on an unparseable file: %v", err)
	}

	data, err := os.ReadFile(Path())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "still-valid") {
		t.Fatalf("the token survived logout: %s", data)
	}
}

// TestSaveTightensAnExistingLooseFile guards the gap between OpenFile's perm
// argument, which applies only at creation, and an existing file that already
// carries a looser mode.
func TestSaveTightensAnExistingLooseFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if err := os.MkdirAll(filepath.Join(dir, "journal"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path(), []byte("url: https://old.example.com\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Save(Config{URL: "https://new.example.com", Token: "t"}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(Path())
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode is %o after Save, want 600", got)
	}
}
