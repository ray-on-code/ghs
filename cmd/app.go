package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ray-on-code/ghs/pkg/auth"
	"github.com/ray-on-code/ghs/pkg/cloner"
	"github.com/ray-on-code/ghs/pkg/config"
	ghapi "github.com/ray-on-code/ghs/pkg/github"
	"github.com/ray-on-code/ghs/pkg/ui"
)

// RepoLister は ghs のメインフローが GitHub から必要とする操作の最小限の契約です。
// consumer 側 (このパッケージ) でインターフェースを定義することで、
// テストでは fake、本番では *ghapi.Client に差し替えられます。
type RepoLister interface {
	ListOwners(ctx context.Context) ([]string, error)
	ListRepositories(ctx context.Context, owner string) ([]ghapi.Repository, error)
}

// TokenResolver はトークン解決ロジックの差し替えポイントです。
type TokenResolver func(fallbackToken string) (*auth.Result, error)

// ListerFactory はトークンを受け取って RepoLister を生成する関数の型です。
type ListerFactory func(ctx context.Context, token string) RepoLister

// CloneFunc はクローン処理の差し替えポイントです。
type CloneFunc func(dest cloner.Destination, opts cloner.Options) error

// App はメインフローの依存をまとめた DI コンテナです。
// テストでは各フィールドを fake に差し替えることでフロー全体を検証できます。
type App struct {
	Cfg          *config.Config
	ResolveToken TokenResolver
	NewLister    ListerFactory
	Prompter     ui.Prompter
	Clone        CloneFunc

	Stdout io.Writer

	// CLI フラグ由来の挙動。
	UseSSH      bool
	BaseDirOver string
}

// NewDefaultApp は本番用デフォルトの依存をセットした App を返します。
func NewDefaultApp(cfg *config.Config) *App {
	return &App{
		Cfg:          cfg,
		ResolveToken: auth.ResolveToken,
		NewLister: func(ctx context.Context, token string) RepoLister {
			return ghapi.New(ctx, token)
		},
		Prompter: ui.SurveyPrompter{},
		Clone:    cloner.Clone,
		Stdout:   os.Stdout,
	}
}

// Run はメインフロー (認証 → Owner選択 → リポジトリ選択 → クローン) を実行します。
func (a *App) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	out := a.Stdout
	if out == nil {
		out = os.Stdout
	}

	fmt.Fprintln(out, "🚀 ghs を起動しています...")

	authResult, err := a.ResolveToken(a.Cfg.GitHubToken)
	if err != nil {
		if errors.Is(err, auth.ErrTokenNotFound) {
			path, _ := config.ConfigFilePath()
			return fmt.Errorf(`%w
ヒント:
  - GitHub CLI をインストールして `+"`gh auth login`"+` を実行する
  - もしくは %s に github_token を設定する`, err, path)
		}
		// gh CLI 周りの想定外エラーは警告して fallback も試みる。
		fmt.Fprintf(out, "⚠️  %v\n", err)
		if a.Cfg.GitHubToken != "" {
			authResult = &auth.Result{Token: a.Cfg.GitHubToken, Source: auth.SourceConfig}
		} else {
			return auth.ErrTokenNotFound
		}
	}

	fmt.Fprintf(out, "🔑 トークンを取得しました (source: %s)\n", authResult.Source)

	lister := a.NewLister(ctx, authResult.Token)

	fmt.Fprintln(out, "🔍 Owner 候補 (自分 / 所属組織) を取得しています...")
	owners, err := lister.ListOwners(ctx)
	if err != nil {
		return err
	}

	owner, err := a.Prompter.SelectOwner(owners)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "✅ Owner: %s\n", owner)

	fmt.Fprintf(out, "📚 %s のリポジトリ一覧を取得しています...\n", owner)
	repos, err := lister.ListRepositories(ctx, owner)
	if err != nil {
		return err
	}
	if len(repos) == 0 {
		return fmt.Errorf("%s にはリポジトリが見つかりませんでした", owner)
	}

	repo, err := a.Prompter.SelectRepository(repos)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "✅ Repository: %s\n", repo.FullName)

	baseDir := a.Cfg.CloneBaseDir
	if a.BaseDirOver != "" {
		baseDir = a.BaseDirOver
	}

	dest := cloner.ResolveDestination(baseDir, repo.Owner, repo.Name)
	fmt.Fprintf(out, "📦 クローン先: %s\n", dest.FullPath())

	if err := a.Clone(dest, cloner.Options{UseSSH: a.UseSSH}); err != nil {
		switch {
		case errors.Is(err, cloner.ErrGitNotInstalled):
			return fmt.Errorf("%w: git をインストールしてから再実行してください", err)
		case errors.Is(err, cloner.ErrAlreadyExists):
			return fmt.Errorf("%w: 既存のディレクトリを削除/移動するか、別の場所を指定してください", err)
		default:
			return err
		}
	}

	fmt.Fprintf(out, "🎉 クローン完了: %s\n", dest.FullPath())
	return nil
}
