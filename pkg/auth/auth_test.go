package auth

import (
	"errors"
	"os/exec"
	"testing"
)

func withFakeExec(t *testing.T, lookPath func(string) (string, error), cmd func(name string, args ...string) *exec.Cmd) {
	t.Helper()
	origLook, origCmd := execLookPath, execCommand
	execLookPath = lookPath
	execCommand = cmd
	t.Cleanup(func() {
		execLookPath = origLook
		execCommand = origCmd
	})
}

func TestResolveToken_GHCLISuccess(t *testing.T) {
	withFakeExec(t,
		func(file string) (string, error) { return "/usr/bin/gh", nil },
		func(name string, args ...string) *exec.Cmd {
			// echo を使ってトークンを stdout に出す。 cmd.Run() で stdout がキャプチャされる。
			return exec.Command("echo", "ghp_from_gh")
		},
	)

	got, err := ResolveToken("ghp_fallback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Token != "ghp_from_gh" {
		t.Errorf("token = %q, want ghp_from_gh", got.Token)
	}
	if got.Source != SourceGHCLI {
		t.Errorf("source = %q, want SourceGHCLI", got.Source)
	}
}

func TestResolveToken_GHCLINotInstalled_FallsBackToConfig(t *testing.T) {
	withFakeExec(t,
		func(file string) (string, error) {
			return "", errors.New("executable not found in $PATH")
		},
		func(name string, args ...string) *exec.Cmd {
			t.Errorf("execCommand should not be called when gh is not installed")
			return exec.Command("true")
		},
	)

	got, err := ResolveToken("ghp_fallback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Token != "ghp_fallback" {
		t.Errorf("token = %q, want ghp_fallback", got.Token)
	}
	if got.Source != SourceConfig {
		t.Errorf("source = %q, want SourceConfig", got.Source)
	}
}

func TestResolveToken_GHCLINotAuthenticated_FallsBack(t *testing.T) {
	withFakeExec(t,
		func(file string) (string, error) { return "/usr/bin/gh", nil },
		func(name string, args ...string) *exec.Cmd {
			// stderr に "not logged into" を吐いて非ゼロ終了する fake。
			return exec.Command("sh", "-c", "echo 'You are not logged into any GitHub hosts. Run gh auth login to authenticate.' >&2; exit 1")
		},
	)

	got, err := ResolveToken("ghp_fallback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source != SourceConfig {
		t.Errorf("source = %q, want SourceConfig (fallback)", got.Source)
	}
}

func TestResolveToken_AllSourcesEmpty(t *testing.T) {
	withFakeExec(t,
		func(file string) (string, error) {
			return "", errors.New("not found")
		},
		func(name string, args ...string) *exec.Cmd {
			return exec.Command("true")
		},
	)

	_, err := ResolveToken("   ") // ホワイトスペースのみは空扱い
	if !errors.Is(err, ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestResolveToken_EmptyStdout_FromGH_FallsBack(t *testing.T) {
	withFakeExec(t,
		func(file string) (string, error) { return "/usr/bin/gh", nil },
		func(name string, args ...string) *exec.Cmd {
			return exec.Command("true") // 標準出力なし、 exit 0
		},
	)

	got, err := ResolveToken("ghp_fallback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Source != SourceConfig {
		t.Errorf("source = %q, want SourceConfig", got.Source)
	}
}
