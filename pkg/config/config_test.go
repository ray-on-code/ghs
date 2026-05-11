package config

import (
	"os"
	"path/filepath"
	"testing"
)

// setupIsolatedHome は HOME / XDG_CONFIG_HOME を一時ディレクトリに切り替え、
// 設定ファイル探索が外部環境の影響を受けないようにします。
func setupIsolatedHome(t *testing.T) (home, xdg string) {
	t.Helper()
	home = t.TempDir()
	xdg = filepath.Join(home, ".config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	return home, xdg
}

func TestConfigDir_RespectsXDG(t *testing.T) {
	_, xdg := setupIsolatedHome(t)

	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir returned error: %v", err)
	}
	want := filepath.Join(xdg, "ghs")
	if got != want {
		t.Errorf("ConfigDir = %q, want %q", got, want)
	}
}

func TestConfigDir_FallbackToHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir returned error: %v", err)
	}
	want := filepath.Join(home, ".config", "ghs")
	if got != want {
		t.Errorf("ConfigDir = %q, want %q", got, want)
	}
}

func TestLoad_DefaultsWhenNoConfigFile(t *testing.T) {
	home, _ := setupIsolatedHome(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	wantDir := filepath.Join(home, "ghs")
	if cfg.CloneBaseDir != wantDir {
		t.Errorf("CloneBaseDir = %q, want %q", cfg.CloneBaseDir, wantDir)
	}
	if cfg.GitHubToken != "" {
		t.Errorf("GitHubToken = %q, want empty", cfg.GitHubToken)
	}
}

func TestLoad_ReadsTomlFileAndExpandsTilde(t *testing.T) {
	home, _ := setupIsolatedHome(t)

	path, err := EnsureConfigFile()
	if err != nil {
		t.Fatalf("EnsureConfigFile failed: %v", err)
	}
	content := `github_token = "ghp_xxx"
clone_base_dir = "~/ghs"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.GitHubToken != "ghp_xxx" {
		t.Errorf("GitHubToken = %q, want %q", cfg.GitHubToken, "ghp_xxx")
	}
	wantDir := filepath.Join(home, "ghs")
	if cfg.CloneBaseDir != wantDir {
		t.Errorf("CloneBaseDir = %q, want %q (tilde should be expanded)", cfg.CloneBaseDir, wantDir)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	setupIsolatedHome(t)
	t.Setenv("GHS_GITHUB_TOKEN", "from-env")
	t.Setenv("GHS_CLONE_BASE_DIR", "/tmp/explicit")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.GitHubToken != "from-env" {
		t.Errorf("GitHubToken = %q, want from-env", cfg.GitHubToken)
	}
	if cfg.CloneBaseDir != "/tmp/explicit" {
		t.Errorf("CloneBaseDir = %q, want /tmp/explicit", cfg.CloneBaseDir)
	}
}

func TestEnsureConfigFile_CreatesTemplateAndIsIdempotent(t *testing.T) {
	_, xdg := setupIsolatedHome(t)

	path, err := EnsureConfigFile()
	if err != nil {
		t.Fatalf("EnsureConfigFile failed: %v", err)
	}
	wantPath := filepath.Join(xdg, "ghs", "config.toml")
	if path != wantPath {
		t.Errorf("path = %q, want %q", path, wantPath)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("config file was not created: %v", err)
	}
	if info.Size() == 0 {
		t.Errorf("config file should not be empty")
	}

	// もう一度呼んでも上書きされないこと (テンプレートを書き換えていないことを確認)
	modTimeBefore := info.ModTime()
	if _, err := EnsureConfigFile(); err != nil {
		t.Fatalf("EnsureConfigFile (2nd call) failed: %v", err)
	}
	info2, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after 2nd call failed: %v", err)
	}
	if !info2.ModTime().Equal(modTimeBefore) {
		t.Errorf("config file should not be overwritten on repeat call")
	}
}

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"no tilde", "/abs/path", "/abs/path"},
		{"tilde only", "~", home},
		{"tilde slash", "~/ghs", filepath.Join(home, "ghs")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := expandHome(tc.in)
			if err != nil {
				t.Fatalf("expandHome error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
