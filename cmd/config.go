package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ray-on-code/ghs/pkg/config"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configInitCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "設定ファイルを操作します",
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "設定ファイルのパスを表示します",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.ConfigFilePath()
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), path)
		return nil
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "設定ファイルのテンプレートを ~/.config/ghs/config.toml に生成します",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.EnsureConfigFile()
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "📝 設定ファイルを準備しました: %s\n", path)
		return nil
	},
}
