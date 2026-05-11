// Package cloner はリポジトリのクローン先パス決定と `git clone` の実行を担当します。
package cloner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ErrAlreadyExists はクローン先ディレクトリが既に存在する場合に返されます。
var ErrAlreadyExists = errors.New("クローン先のディレクトリは既に存在します")

// ErrGitNotInstalled は git コマンドが見つからない場合に返されます。
var ErrGitNotInstalled = errors.New("git コマンドが見つかりません")

// Destination はクローン先のパス情報を表します。
type Destination struct {
	BaseDir string // 例: /Users/foo/src
	Host    string // 例: github.com
	Owner   string
	Repo    string
}

// FullPath はクローン先の絶対パスを返します ($base/github.com/Owner/Repo)。
func (d Destination) FullPath() string {
	return filepath.Join(d.BaseDir, d.Host, d.Owner, d.Repo)
}

// ResolveDestination は ghq 準拠のクローン先パスを構築します。
func ResolveDestination(baseDir, owner, repo string) Destination {
	return Destination{
		BaseDir: baseDir,
		Host:    "github.com",
		Owner:   owner,
		Repo:    repo,
	}
}

// Options は Clone の追加オプションです。
type Options struct {
	// AllowExisting が true の場合、既存ディレクトリでもエラーにせずスキップします。
	AllowExisting bool
	// UseSSH が true の場合、 git@github.com:Owner/Repo.git をクローン URL として利用します。
	UseSSH bool
}

// Clone は git clone を実行します。
//
// - 既にクローン先ディレクトリが存在する場合は ErrAlreadyExists を返します
//   （Options.AllowExisting が true の場合のみ呼び出し側に判断を委ねます）。
// - git コマンドが存在しない場合は ErrGitNotInstalled を返します。
func Clone(dest Destination, opts Options) error {
	if _, err := exec.LookPath("git"); err != nil {
		return ErrGitNotInstalled
	}

	full := dest.FullPath()
	if info, err := os.Stat(full); err == nil {
		if info.IsDir() && !opts.AllowExisting {
			return fmt.Errorf("%w: %s", ErrAlreadyExists, full)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("クローン先の状態確認に失敗しました: %w", err)
	}

	parent := filepath.Dir(full)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("親ディレクトリの作成に失敗しました: %w", err)
	}

	url := cloneURL(dest, opts.UseSSH)
	cmd := exec.Command("git", "clone", url, full)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone の実行に失敗しました: %w", err)
	}
	return nil
}

func cloneURL(dest Destination, useSSH bool) string {
	if useSSH {
		return fmt.Sprintf("git@%s:%s/%s.git", dest.Host, dest.Owner, dest.Repo)
	}
	return fmt.Sprintf("https://%s/%s/%s.git", dest.Host, dest.Owner, dest.Repo)
}
