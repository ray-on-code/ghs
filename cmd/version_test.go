package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCmd_PrintsVersion(t *testing.T) {
	orig := Version
	Version = "v9.9.9-test"
	t.Cleanup(func() { Version = orig })

	buf := &bytes.Buffer{}
	versionCmd.SetOut(buf)

	// サブコマンドの Run を直接呼ぶ (rootCmd のフローを経由しない)。
	versionCmd.Run(versionCmd, []string{})

	got := buf.String()
	if !strings.Contains(got, "ghs v9.9.9-test") {
		t.Errorf("output = %q, want to contain ghs v9.9.9-test", got)
	}
}
