// Package auth は GitHub のアクセストークンを解決するロジックを提供します。
//
// 解決の優先順位は次のとおりです:
//  1. gh CLI (`gh auth token`)
//  2. 設定ファイル (~/.config/ghs/config.toml の github_token)
package auth

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrTokenNotFound は GitHub トークンを取得できなかったときに返されます。
var ErrTokenNotFound = errors.New("github のアクセストークンが見つかりません")

// Source はトークンの取得元を表します。
type Source string

const (
	// SourceGHCLI は gh CLI からトークンを取得したことを表します。
	SourceGHCLI Source = "gh CLI"
	// SourceConfig は設定ファイルからトークンを取得したことを表します。
	SourceConfig Source = "config.toml"
)

// Result はトークン解決の結果を表します。
type Result struct {
	Token  string
	Source Source
}

// ResolveToken は優先順位に従って GitHub トークンを取得します。
// fallbackToken には設定ファイルから読み込んだトークンを渡します。
func ResolveToken(fallbackToken string) (*Result, error) {
	if token, err := tokenFromGHCLI(); err == nil && token != "" {
		return &Result{Token: token, Source: SourceGHCLI}, nil
	} else if err != nil && !errors.Is(err, errGHNotInstalled) && !errors.Is(err, errGHNotAuthenticated) {
		// 想定外のエラーは握りつぶさず警告として返すために包んで返す
		return nil, fmt.Errorf("gh CLI からのトークン取得中にエラーが発生しました: %w", err)
	}

	if strings.TrimSpace(fallbackToken) != "" {
		return &Result{Token: strings.TrimSpace(fallbackToken), Source: SourceConfig}, nil
	}
	return nil, ErrTokenNotFound
}

var (
	errGHNotInstalled     = errors.New("gh CLI がインストールされていません")
	errGHNotAuthenticated = errors.New("gh CLI で認証されていません")
)

// tokenFromGHCLI は `gh auth token` を呼び出してトークンを取得します。
func tokenFromGHCLI() (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", errGHNotInstalled
	}

	cmd := exec.Command("gh", "auth", "token")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrText := stderr.String()
		if strings.Contains(stderrText, "not logged into") ||
			strings.Contains(stderrText, "no oauth token") ||
			strings.Contains(stderrText, "Not logged in") {
			return "", errGHNotAuthenticated
		}
		return "", fmt.Errorf("gh auth token の実行に失敗しました: %w: %s", err, strings.TrimSpace(stderrText))
	}

	token := strings.TrimSpace(stdout.String())
	if token == "" {
		return "", errGHNotAuthenticated
	}
	return token, nil
}
