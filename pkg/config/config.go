// Package config はアプリケーションの設定ファイル管理を担当します。
//
// 設定ファイルは ~/.config/ghs/config.toml に配置されることを想定しており、
// Viper を介してロード・保存します。
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config はアプリケーション全体の設定を保持する構造体です。
type Config struct {
	// GitHubToken は config.toml に保存された GitHub のアクセストークンです。
	// gh CLI から取得できない場合のフォールバックとして利用されます。
	GitHubToken string `mapstructure:"github_token"`

	// CloneBaseDir はクローン先のベースディレクトリです。
	// 未設定の場合は $HOME/src が使われます。
	CloneBaseDir string `mapstructure:"clone_base_dir"`
}

const (
	defaultConfigDirName  = "ghs"
	defaultConfigFileName = "config"
	defaultConfigFileExt  = "toml"
)

// Load は設定ファイルを読み込み、 Config を返します。
// 設定ファイルが存在しない場合でもエラーにせず、デフォルト値を返します。
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName(defaultConfigFileName)
	v.SetConfigType(defaultConfigFileExt)

	configDir, err := ConfigDir()
	if err != nil {
		return nil, fmt.Errorf("設定ディレクトリの解決に失敗しました: %w", err)
	}
	v.AddConfigPath(configDir)

	// デフォルト値
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("HOME ディレクトリの解決に失敗しました: %w", err)
	}
	v.SetDefault("clone_base_dir", filepath.Join(home, "src"))
	v.SetDefault("github_token", "")

	// 環境変数 (GHS_GITHUB_TOKEN, GHS_CLONE_BASE_DIR) も任意で受け付ける
	v.SetEnvPrefix("GHS")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("設定のパースに失敗しました: %w", err)
	}

	// CloneBaseDir 内の ~ を展開する
	if expanded, err := expandHome(cfg.CloneBaseDir); err == nil {
		cfg.CloneBaseDir = expanded
	}

	return cfg, nil
}

// ConfigDir は ~/.config/ghs を返します。 XDG_CONFIG_HOME が設定されている場合はそれを優先します。
func ConfigDir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, defaultConfigDirName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", defaultConfigDirName), nil
}

// ConfigFilePath は設定ファイルの絶対パスを返します。
func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, defaultConfigFileName+"."+defaultConfigFileExt), nil
}

// EnsureConfigFile は設定ファイルが存在しない場合に空のテンプレートを生成します。
func EnsureConfigFile() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("設定ディレクトリの作成に失敗しました: %w", err)
	}
	path, err := ConfigFilePath()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(path); err == nil {
		return path, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	template := fmt.Sprintf(`# ghs configuration file
# gh CLI のトークンが優先されますが、未設定の場合はこちらが利用されます。
github_token = ""

# クローン先のベースディレクトリ (ghq 準拠: $base/github.com/Owner/Repo)
clone_base_dir = %q
`, filepath.Join(home, "src"))

	if err := os.WriteFile(path, []byte(template), 0o600); err != nil {
		return "", fmt.Errorf("設定ファイルの作成に失敗しました: %w", err)
	}
	return path, nil
}

// expandHome は先頭の ~ を $HOME に展開します。
func expandHome(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, path[1:]), nil
}
