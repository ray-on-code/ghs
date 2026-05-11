package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// newTestClient はテスト用に BaseURL を httptest サーバへ差し替えた Client を返します。
func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c := New(context.Background(), "test-token")
	u, err := url.Parse(srv.URL + "/")
	if err != nil {
		t.Fatalf("url.Parse failed: %v", err)
	}
	c.api.BaseURL = u
	c.api.UploadURL = u
	return c, srv
}

func TestCurrentUser(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); !strings.Contains(got, "test-token") {
			t.Errorf("Authorization header = %q, want token to be sent", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"login":"octo"}`)
	})

	c, _ := newTestClient(t, mux)
	got, err := c.CurrentUser(context.Background())
	if err != nil {
		t.Fatalf("CurrentUser error: %v", err)
	}
	if got != "octo" {
		t.Errorf("CurrentUser = %q, want octo", got)
	}
}

func TestCurrentUser_NoLogin(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{}`)
	})
	c, _ := newTestClient(t, mux)
	if _, err := c.CurrentUser(context.Background()); err == nil {
		t.Fatal("expected error when login is missing")
	}
}

func TestListOwners_IncludesSelfAndSortedOrgs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"login":"octo"}`)
	})
	mux.HandleFunc("/user/orgs", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"login":"zeta"},{"login":"alpha"},{"login":"middle"}]`)
	})

	c, _ := newTestClient(t, mux)
	got, err := c.ListOwners(context.Background())
	if err != nil {
		t.Fatalf("ListOwners error: %v", err)
	}
	want := []string{"octo", "alpha", "middle", "zeta"}
	if !equalStrings(got, want) {
		t.Errorf("ListOwners = %v, want %v", got, want)
	}
}

func TestListOwners_Pagination(t *testing.T) {
	mux := http.NewServeMux()
	// srv の URL を後で参照したいので、サーバ作成より先にハンドラを定義し、
	// 共通の baseURL ポインタを使う構造にする。
	var baseURL string
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"login":"octo"}`)
	})
	mux.HandleFunc("/user/orgs", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "" || page == "1" {
			w.Header().Set("Link",
				`<`+baseURL+`/user/orgs?page=2>; rel="next", `+
					`<`+baseURL+`/user/orgs?page=2>; rel="last"`)
			fmt.Fprint(w, `[{"login":"org-a"}]`)
			return
		}
		fmt.Fprint(w, `[{"login":"org-b"}]`)
	})

	c, srv := newTestClient(t, mux)
	baseURL = srv.URL

	got, err := c.ListOwners(context.Background())
	if err != nil {
		t.Fatalf("ListOwners error: %v", err)
	}
	want := []string{"octo", "org-a", "org-b"}
	if !equalStrings(got, want) {
		t.Errorf("ListOwners (paginated) = %v, want %v", got, want)
	}
}

func TestListRepositories_ForSelf(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"login":"octo"}`)
	})
	mux.HandleFunc("/user/repos", func(w http.ResponseWriter, r *http.Request) {
		// オプションが反映されていること
		if got := r.URL.Query().Get("affiliation"); got != "owner" {
			t.Errorf("affiliation = %q, want owner", got)
		}
		fmt.Fprint(w, `[
			{
				"name":"r1","full_name":"octo/r1","description":"desc",
				"private":true,"fork":false,"archived":false,
				"owner":{"login":"octo"},
				"html_url":"https://github.com/octo/r1",
				"ssh_url":"git@github.com:octo/r1.git",
				"clone_url":"https://github.com/octo/r1.git"
			},
			{
				"name":"r2","full_name":"octo/r2",
				"owner":{"login":"octo"}
			}
		]`)
	})

	c, _ := newTestClient(t, mux)
	repos, err := c.ListRepositories(context.Background(), "octo")
	if err != nil {
		t.Fatalf("ListRepositories error: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("len(repos) = %d, want 2", len(repos))
	}
	r1 := repos[0]
	if r1.Owner != "octo" || r1.Name != "r1" || r1.FullName != "octo/r1" {
		t.Errorf("r1 = %+v", r1)
	}
	if !r1.Private || r1.Fork || r1.Archived {
		t.Errorf("flag mapping failed: %+v", r1)
	}
	if r1.Description != "desc" {
		t.Errorf("description = %q", r1.Description)
	}
	if r1.SSHURL != "git@github.com:octo/r1.git" {
		t.Errorf("ssh url = %q", r1.SSHURL)
	}
}

func TestListRepositories_ForOrg(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"login":"octo"}`)
	})
	mux.HandleFunc("/orgs/myorg/repos", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"name":"x","full_name":"myorg/x","owner":{"login":"myorg"}}]`)
	})

	c, _ := newTestClient(t, mux)
	repos, err := c.ListRepositories(context.Background(), "myorg")
	if err != nil {
		t.Fatalf("ListRepositories error: %v", err)
	}
	if len(repos) != 1 || repos[0].Owner != "myorg" || repos[0].Name != "x" {
		t.Errorf("got %+v", repos)
	}
}

func TestListRepositories_APIError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"login":"octo"}`)
	})
	mux.HandleFunc("/orgs/myorg/repos", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	})

	c, _ := newTestClient(t, mux)
	if _, err := c.ListRepositories(context.Background(), "myorg"); err == nil {
		t.Fatal("expected error from 404 response")
	}
}

func TestToRepository_NilSafe(t *testing.T) {
	got := toRepository(nil)
	if got.Owner != "" || got.Name != "" {
		t.Errorf("toRepository(nil) should return zero value, got %+v", got)
	}
}

func equalStrings(a, b []string) bool {
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
