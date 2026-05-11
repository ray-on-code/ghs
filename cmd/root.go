// Package cmd は ghs の Cobra コマンド定義を集約します。
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ray-on-code/ghs/pkg/config"
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
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		app := NewDefaultApp(cfg)
		app.UseSSH = flagUseSSH
		app.BaseDirOver = flagBaseDirOver
		return app.Run(cmd.Context())
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagUseSSH, "ssh", false, "クローン URL に SSH (git@github.com:...) を使う")
	rootCmd.PersistentFlags().StringVar(&flagBaseDirOver, "base-dir", "", "クローン先のベースディレクトリを上書きする (例: ~/ghs)")
}
