package cmd

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ray-on-code/ghs/pkg/auth"
	"github.com/ray-on-code/ghs/pkg/cloner"
	"github.com/ray-on-code/ghs/pkg/config"
	ghapi "github.com/ray-on-code/ghs/pkg/github"
)

// --- fakes ----------------------------------------------------------------

type fakeLister struct {
	owners        []string
	reposByOw     map[string][]ghapi.Repository
	listOwnersErr error
	listReposErr  error
}

func (f *fakeLister) ListOwners(ctx context.Context) ([]string, error) {
	if f.listOwnersErr != nil {
		return nil, f.listOwnersErr
	}
	return f.owners, nil
}
func (f *fakeLister) ListRepositories(ctx context.Context, owner string) ([]ghapi.Repository, error) {
	if f.listReposErr != nil {
		return nil, f.listReposErr
	}
	return f.reposByOw[owner], nil
}

type fakePrompter struct {
	ownerToReturn string
	repoToReturn  *ghapi.Repository
	ownersSeen    []string
	reposSeen     []ghapi.Repository
	ownerErr      error
	repoErr       error
}

func (f *fakePrompter) SelectOwner(owners []string) (string, error) {
	f.ownersSeen = owners
	if f.ownerErr != nil {
		return "", f.ownerErr
	}
	return f.ownerToReturn, nil
}
func (f *fakePrompter) SelectRepository(repos []ghapi.Repository) (*ghapi.Repository, error) {
	f.reposSeen = repos
	if f.repoErr != nil {
		return nil, f.repoErr
	}
	return f.repoToReturn, nil
}
func (f *fakePrompter) ConfirmOverwrite(path string) (bool, error) { return false, nil }

// --- helpers --------------------------------------------------------------

func newTestApp(t *testing.T) (*App, *bytes.Buffer, *fakeLister, *fakePrompter, *struct {
	called bool
	dest   cloner.Destination
	opts   cloner.Options
	err    error
}) {
	t.Helper()
	out := &bytes.Buffer{}
	lister := &fakeLister{
		owners: []string{"me", "myorg"},
		reposByOw: map[string][]ghapi.Repository{
			"me":    {{Owner: "me", Name: "alpha", FullName: "me/alpha"}},
			"myorg": {{Owner: "myorg", Name: "beta", FullName: "myorg/beta"}},
		},
	}
	prompter := &fakePrompter{
		ownerToReturn: "myorg",
		repoToReturn:  &ghapi.Repository{Owner: "myorg", Name: "beta", FullName: "myorg/beta"},
	}
	cloneRec := &struct {
		called bool
		dest   cloner.Destination
		opts   cloner.Options
		err    error
	}{}

	app := &App{
		Cfg: &config.Config{
			GitHubToken:  "",
			CloneBaseDir: t.TempDir(),
		},
		ResolveToken: func(fallback string) (*auth.Result, error) {
			return &auth.Result{Token: "ghp_test", Source: auth.SourceGHCLI}, nil
		},
		NewLister: func(ctx context.Context, token string) RepoLister {
			return lister
		},
		Prompter: prompter,
		Clone: func(d cloner.Destination, o cloner.Options) error {
			cloneRec.called = true
			cloneRec.dest = d
			cloneRec.opts = o
			return cloneRec.err
		},
		Stdout: out,
	}
	return app, out, lister, prompter, cloneRec
}

// --- tests ----------------------------------------------------------------

func TestApp_Run_HappyPath(t *testing.T) {
	app, out, lister, prompter, rec := newTestApp(t)

	if err := app.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if got := strings.Join(lister.owners, ","); got != "me,myorg" {
		t.Errorf("owners list = %s", got)
	}
	if strings.Join(prompter.ownersSeen, ",") != "me,myorg" {
		t.Errorf("prompter did not receive correct owners: %v", prompter.ownersSeen)
	}
	if !rec.called {
		t.Fatal("Clone was not called")
	}
	wantPath := filepath.Join(app.Cfg.CloneBaseDir, "github.com", "myorg", "beta")
	if rec.dest.FullPath() != wantPath {
		t.Errorf("clone dest = %q, want %q", rec.dest.FullPath(), wantPath)
	}
	if rec.opts.UseSSH {
		t.Errorf("UseSSH should be false by default")
	}

	output := out.String()
	for _, s := range []string{"🚀", "🔑", "🔍", "✅ Owner: myorg", "📦", "🎉"} {
		if !strings.Contains(output, s) {
			t.Errorf("expected output to contain %q, got %q", s, output)
		}
	}
}

func TestApp_Run_UsesSSHFlag(t *testing.T) {
	app, _, _, _, rec := newTestApp(t)
	app.UseSSH = true

	if err := app.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !rec.opts.UseSSH {
		t.Errorf("expected UseSSH to be propagated to Clone options")
	}
}

func TestApp_Run_BaseDirOverride(t *testing.T) {
	app, _, _, _, rec := newTestApp(t)
	override := t.TempDir()
	app.BaseDirOver = override

	if err := app.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	want := filepath.Join(override, "github.com", "myorg", "beta")
	if rec.dest.FullPath() != want {
		t.Errorf("clone dest = %q, want %q", rec.dest.FullPath(), want)
	}
}

func TestApp_Run_TokenNotFound(t *testing.T) {
	app, _, _, _, _ := newTestApp(t)
	app.ResolveToken = func(string) (*auth.Result, error) {
		return nil, auth.ErrTokenNotFound
	}

	err := app.Run(context.Background())
	if !errors.Is(err, auth.ErrTokenNotFound) {
		t.Errorf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestApp_Run_ListOwnersError(t *testing.T) {
	app, _, lister, _, _ := newTestApp(t)
	lister.listOwnersErr = errors.New("boom")

	err := app.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected error containing 'boom', got %v", err)
	}
}

func TestApp_Run_NoRepositoriesForOwner(t *testing.T) {
	app, _, lister, _, _ := newTestApp(t)
	lister.reposByOw["myorg"] = nil

	err := app.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "リポジトリが見つかりません") {
		t.Errorf("expected 'リポジトリが見つかりません' error, got %v", err)
	}
}

func TestApp_Run_CloneErrorAlreadyExists(t *testing.T) {
	app, _, _, _, rec := newTestApp(t)
	rec.err = cloner.ErrAlreadyExists

	err := app.Run(context.Background())
	if !errors.Is(err, cloner.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists chain, got %v", err)
	}
}

func TestApp_Run_CloneErrorGitNotInstalled(t *testing.T) {
	app, _, _, _, rec := newTestApp(t)
	rec.err = cloner.ErrGitNotInstalled

	err := app.Run(context.Background())
	if !errors.Is(err, cloner.ErrGitNotInstalled) {
		t.Errorf("expected ErrGitNotInstalled chain, got %v", err)
	}
}

func TestApp_Run_PrompterCancel(t *testing.T) {
	app, _, _, prompter, rec := newTestApp(t)
	prompter.ownerErr = errors.New("interrupt")

	err := app.Run(context.Background())
	if err == nil {
		t.Fatal("expected error from cancelled prompt")
	}
	if rec.called {
		t.Errorf("Clone should not be called when prompt is cancelled")
	}
}
