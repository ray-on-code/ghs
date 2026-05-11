package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupIsolatedHome(t *testing.T) (home, xdg string) {
	t.Helper()
	home = t.TempDir()
	xdg = filepath.Join(home, ".config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	return home, xdg
}

func TestConfigPathCmd(t *testing.T) {
	_, xdg := setupIsolatedHome(t)

	buf := &bytes.Buffer{}
	configPathCmd.SetOut(buf)

	if err := configPathCmd.RunE(configPathCmd, []string{}); err != nil {
		t.Fatalf("RunE error: %v", err)
	}
	want := filepath.Join(xdg, "ghs", "config.toml")
	if got := strings.TrimSpace(buf.String()); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

func TestConfigInitCmd_CreatesTemplate(t *testing.T) {
	_, xdg := setupIsolatedHome(t)

	buf := &bytes.Buffer{}
	configInitCmd.SetOut(buf)

	if err := configInitCmd.RunE(configInitCmd, []string{}); err != nil {
		t.Fatalf("RunE error: %v", err)
	}

	want := filepath.Join(xdg, "ghs", "config.toml")
	if !strings.Contains(buf.String(), want) {
		t.Errorf("output should mention %q, got %q", want, buf.String())
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("config file not created: %v", err)
	}
	if !strings.Contains(buf.String(), "📝") {
		t.Errorf("output should include emoji feedback, got %q", buf.String())
	}
}

func TestNewDefaultApp_HasAllDeps(t *testing.T) {
	app := NewDefaultApp(nil)
	if app == nil {
		t.Fatal("NewDefaultApp returned nil")
	}
	if app.ResolveToken == nil {
		t.Error("ResolveToken should be set")
	}
	if app.NewLister == nil {
		t.Error("NewLister should be set")
	}
	if app.Prompter == nil {
		t.Error("Prompter should be set")
	}
	if app.Clone == nil {
		t.Error("Clone should be set")
	}
	if app.Stdout == nil {
		t.Error("Stdout should be set")
	}
}
