package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version はビルド時に -ldflags で書き換えられる想定のバージョン文字列です。
var Version = "dev"

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "ghs のバージョンを表示します",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "ghs %s\n", Version)
	},
}
