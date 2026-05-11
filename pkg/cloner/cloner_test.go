package cloner

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- 純粋関数のテスト -------------------------------------------------------

func TestResolveDestination(t *testing.T) {
	got := ResolveDestination("/base", "octo", "hello")
	if got.BaseDir != "/base" || got.Host != "github.com" || got.Owner != "octo" || got.Repo != "hello" {
		t.Errorf("unexpected destination: %+v", got)
	}
}

func TestDestination_FullPath(t *testing.T) {
	dest := Destination{BaseDir: "/base", Host: "github.com", Owner: "octo", Repo: "hello"}
	got := dest.FullPath()
	want := filepath.Join("/base", "github.com", "octo", "hello")
	if got != want {
		t.Errorf("FullPath = %q, want %q", got, want)
	}
}

func TestCloneURL(t *testing.T) {
	dest := Destination{Host: "github.com", Owner: "octo", Repo: "hello"}
	if got := CloneURL(dest, false); got != "https://github.com/octo/hello.git" {
		t.Errorf("HTTPS URL = %q", got)
	}
	if got := CloneURL(dest, true); got != "git@github.com:octo/hello.git" {
		t.Errorf("SSH URL = %q", got)
	}
}

// --- exec モックを使った Clone のテスト ------------------------------------

// withFakeExec はテストの間 execLookPath / execCommand を差し替えます。
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

// fakeSuccess は何も実行せず exit 0 を返す exec.Cmd を生成します (シェルの :)。
func fakeSuccess(name string, args ...string) *exec.Cmd {
	return exec.Command("true")
}

// fakeFail は exit 1 を返す exec.Cmd を生成します。
func fakeFail(name string, args ...string) *exec.Cmd {
	return exec.Command("false")
}

func TestClone_GitNotInstalled(t *testing.T) {
	withFakeExec(t,
		func(file string) (string, error) {
			return "", errors.New("not found")
		},
		fakeSuccess,
	)

	err := Clone(Destination{BaseDir: t.TempDir(), Host: "github.com", Owner: "o", Repo: "r"}, Options{})
	if !errors.Is(err, ErrGitNotInstalled) {
		t.Errorf("expected ErrGitNotInstalled, got %v", err)
	}
}

func TestClone_AlreadyExists(t *testing.T) {
	base := t.TempDir()
	full := filepath.Join(base, "github.com", "o", "r")
	if err := os.MkdirAll(full, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	withFakeExec(t,
		func(file string) (string, error) { return "/usr/bin/git", nil },
		func(name string, args ...string) *exec.Cmd {
			t.Errorf("execCommand should not be called when destination already exists")
			return exec.Command("true")
		},
	)

	err := Clone(Destination{BaseDir: base, Host: "github.com", Owner: "o", Repo: "r"}, Options{})
	if !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestClone_Success_CallsGitCloneWithCorrectArgs(t *testing.T) {
	base := t.TempDir()

	var got struct {
		name string
		args []string
	}

	withFakeExec(t,
		func(file string) (string, error) { return "/usr/bin/git", nil },
		func(name string, args ...string) *exec.Cmd {
			got.name = name
			got.args = args
			return exec.Command("true")
		},
	)

	dest := Destination{BaseDir: base, Host: "github.com", Owner: "o", Repo: "r"}
	if err := Clone(dest, Options{}); err != nil {
		t.Fatalf("Clone returned error: %v", err)
	}

	if got.name != "git" {
		t.Errorf("command name = %q, want git", got.name)
	}
	wantURL := "https://github.com/o/r.git"
	wantDest := filepath.Join(base, "github.com", "o", "r")
	wantArgs := []string{"clone", wantURL, wantDest}
	if !equalSlices(got.args, wantArgs) {
		t.Errorf("args = %v, want %v", got.args, wantArgs)
	}

	// 親ディレクトリが作成されているか
	if _, err := os.Stat(filepath.Dir(wantDest)); err != nil {
		t.Errorf("parent dir was not created: %v", err)
	}
}

func TestClone_UseSSH(t *testing.T) {
	base := t.TempDir()
	var capturedURL string
	withFakeExec(t,
		func(file string) (string, error) { return "/usr/bin/git", nil },
		func(name string, args ...string) *exec.Cmd {
			if len(args) >= 2 {
				capturedURL = args[1]
			}
			return exec.Command("true")
		},
	)

	dest := Destination{BaseDir: base, Host: "github.com", Owner: "o", Repo: "r"}
	if err := Clone(dest, Options{UseSSH: true}); err != nil {
		t.Fatalf("Clone returned error: %v", err)
	}
	if capturedURL != "git@github.com:o/r.git" {
		t.Errorf("captured URL = %q, want SSH form", capturedURL)
	}
}

func TestClone_PropagatesGitFailure(t *testing.T) {
	base := t.TempDir()
	withFakeExec(t,
		func(file string) (string, error) { return "/usr/bin/git", nil },
		fakeFail,
	)
	dest := Destination{BaseDir: base, Host: "github.com", Owner: "o", Repo: "r"}
	err := Clone(dest, Options{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "git clone") {
		t.Errorf("error message should mention git clone, got %v", err)
	}
}

func TestClone_CapturesOutputToProvidedWriter(t *testing.T) {
	base := t.TempDir()

	// 出力ありの fake コマンド
	withFakeExec(t,
		func(file string) (string, error) { return "/usr/bin/git", nil },
		func(name string, args ...string) *exec.Cmd {
			return exec.Command("sh", "-c", "echo hello-stdout; echo hello-stderr >&2")
		},
	)

	var stdout, stderr bytes.Buffer
	dest := Destination{BaseDir: base, Host: "github.com", Owner: "o", Repo: "r"}
	err := Clone(dest, Options{Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		t.Fatalf("Clone returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "hello-stdout") {
		t.Errorf("stdout not captured, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "hello-stderr") {
		t.Errorf("stderr not captured, got %q", stderr.String())
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
