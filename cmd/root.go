// Package cmd は ghs の Cobra コマンド定義を集約します。
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ray-on-code/ghs/pkg/auth"
	"github.com/ray-on-code/ghs/pkg/cloner"
	"github.com/ray-on-code/ghs/pkg/config"
	ghapi "github.com/ray-on-code/ghs/pkg/github"
	"github.com/ray-on-code/ghs/pkg/ui"
)

// 共通フラグ
var (
	flagUseSSH      bool
	flagBaseDirOver string
)

// Execute はルートコマンドを実行します。 main から呼び出されます。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "ghs",
	Short: "ghs はインタラクティブな GitHub リポジトリクローンツール (ghq のインタラクティブ拡張) です",
	Long: `ghs (GitHub Select) は、自分や所属組織の GitHub リポジトリを
インタラクティブに選択してクローンするツールです。

  1. gh CLI または config.toml から GitHub トークンを取得します
  2. Owner (自分 / 組織) を選択します
  3. リポジトリを選択 (フィルタ可能) します
  4. ghq 準拠のパス ($base/github.com/Owner/Repo) にクローンします`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInteractive(cmd.Context())
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagUseSSH, "ssh", false, "クローン URL に SSH (git@github.com:...) を使う")
	rootCmd.PersistentFlags().StringVar(&flagBaseDirOver, "base-dir", "", "クローン先のベースディレクトリを上書きする (例: ~/src)")
}

func runInteractive(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	fmt.Println("🚀 ghs を起動しています...")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	authResult, err := auth.ResolveToken(cfg.GitHubToken)
	if err != nil {
		if errors.Is(err, auth.ErrTokenNotFound) {
			path, _ := config.ConfigFilePath()
			return fmt.Errorf(`%w
ヒント:
  - GitHub CLI をインストールして `+"`gh auth login`"+` を実行する
  - もしくは %s に github_token を設定する`, err, path)
		}
		// gh CLI 周りの想定外エラーは警告して fallback 動作も試みる
		fmt.Fprintf(os.Stderr, "⚠️  %v\n", err)
		if cfg.GitHubToken != "" {
			authResult = &auth.Result{Token: cfg.GitHubToken, Source: auth.SourceConfig}
		} else {
			return auth.ErrTokenNotFound
		}
	}

	fmt.Printf("🔑 トークンを取得しました (source: %s)\n", authResult.Source)

	client := ghapi.New(ctx, authResult.Token)

	fmt.Println("🔍 Owner 候補 (自分 / 所属組織) を取得しています...")
	owners, err := client.ListOwners(ctx)
	if err != nil {
		return err
	}

	owner, err := ui.SelectOwner(owners)
	if err != nil {
		return err
	}
	fmt.Printf("✅ Owner: %s\n", owner)

	fmt.Printf("📚 %s のリポジトリ一覧を取得しています...\n", owner)
	repos, err := client.ListRepositories(ctx, owner)
	if err != nil {
		return err
	}
	if len(repos) == 0 {
		return fmt.Errorf("%s にはリポジトリが見つかりませんでした", owner)
	}

	repo, err := ui.SelectRepository(repos)
	if err != nil {
		return err
	}
	fmt.Printf("✅ Repository: %s\n", repo.FullName)

	baseDir := cfg.CloneBaseDir
	if flagBaseDirOver != "" {
		baseDir = flagBaseDirOver
	}

	dest := cloner.ResolveDestination(baseDir, repo.Owner, repo.Name)
	fmt.Printf("📦 クローン先: %s\n", dest.FullPath())

	if err := cloner.Clone(dest, cloner.Options{UseSSH: flagUseSSH}); err != nil {
		switch {
		case errors.Is(err, cloner.ErrGitNotInstalled):
			return fmt.Errorf("%w: git をインストールしてから再実行してください", err)
		case errors.Is(err, cloner.ErrAlreadyExists):
			return fmt.Errorf("%w: 既存のディレクトリを削除/移動するか、別の場所を指定してください", err)
		default:
			return err
		}
	}

	fmt.Printf("🎉 クローン完了: %s\n", dest.FullPath())
	return nil
}
