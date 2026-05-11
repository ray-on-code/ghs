// Package github は go-github を薄くラップし、ghs から必要な操作のみを公開します。
package github

import (
	"context"
	"fmt"
	"sort"

	gh "github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

// Client は GitHub API クライアントのラッパーです。
type Client struct {
	api *gh.Client
}

// New は OAuth2 トークンで認証された Client を返します。
func New(ctx context.Context, token string) *Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	return &Client{api: gh.NewClient(tc)}
}

// Repository は UI 側で扱いやすいよう必要な情報だけを抽出した構造体です。
type Repository struct {
	Owner       string
	Name        string
	FullName    string
	Description string
	Private     bool
	Fork        bool
	Archived    bool
	HTMLURL     string
	SSHURL      string
	CloneURL    string
}

// CurrentUser は認証済みユーザーの login (username) を返します。
func (c *Client) CurrentUser(ctx context.Context) (string, error) {
	user, _, err := c.api.Users.Get(ctx, "")
	if err != nil {
		return "", fmt.Errorf("認証ユーザー情報の取得に失敗しました: %w", err)
	}
	if user.Login == nil {
		return "", fmt.Errorf("認証ユーザーの login が取得できませんでした")
	}
	return *user.Login, nil
}

// ListOwners は自分のユーザー名と、所属しているすべての組織のスラッグをまとめて返します。
// 先頭は必ず認証ユーザー自身です。
func (c *Client) ListOwners(ctx context.Context) ([]string, error) {
	me, err := c.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	orgs, err := c.listOrganizations(ctx)
	if err != nil {
		return nil, err
	}

	owners := make([]string, 0, len(orgs)+1)
	owners = append(owners, me)
	owners = append(owners, orgs...)
	return owners, nil
}

// listOrganizations は認証ユーザーが所属している組織を全件取得します。
func (c *Client) listOrganizations(ctx context.Context) ([]string, error) {
	opt := &gh.ListOptions{PerPage: 100}
	var names []string
	for {
		orgs, resp, err := c.api.Organizations.List(ctx, "", opt)
		if err != nil {
			return nil, fmt.Errorf("組織一覧の取得に失敗しました: %w", err)
		}
		for _, o := range orgs {
			if o.Login != nil {
				names = append(names, *o.Login)
			}
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	sort.Strings(names)
	return names, nil
}

// ListRepositories は指定オーナーのリポジトリ一覧を取得します。
// owner が認証ユーザー自身の場合は private 含む全件、それ以外（組織）も同様に取得を試みます。
func (c *Client) ListRepositories(ctx context.Context, owner string) ([]Repository, error) {
	me, err := c.CurrentUser(ctx)
	if err != nil {
		return nil, err
	}

	if owner == me {
		return c.listAuthenticatedUserRepos(ctx)
	}
	return c.listOrgRepos(ctx, owner)
}

func (c *Client) listAuthenticatedUserRepos(ctx context.Context) ([]Repository, error) {
	opt := &gh.RepositoryListByAuthenticatedUserOptions{
		ListOptions: gh.ListOptions{PerPage: 100},
		Affiliation: "owner",
		Sort:        "updated",
		Direction:   "desc",
	}
	var repos []Repository
	for {
		page, resp, err := c.api.Repositories.ListByAuthenticatedUser(ctx, opt)
		if err != nil {
			return nil, fmt.Errorf("自分のリポジトリ一覧取得に失敗しました: %w", err)
		}
		for _, r := range page {
			repos = append(repos, toRepository(r))
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return repos, nil
}

func (c *Client) listOrgRepos(ctx context.Context, org string) ([]Repository, error) {
	opt := &gh.RepositoryListByOrgOptions{
		ListOptions: gh.ListOptions{PerPage: 100},
		Sort:        "updated",
		Direction:   "desc",
	}
	var repos []Repository
	for {
		page, resp, err := c.api.Repositories.ListByOrg(ctx, org, opt)
		if err != nil {
			return nil, fmt.Errorf("組織 %s のリポジトリ一覧取得に失敗しました: %w", org, err)
		}
		for _, r := range page {
			repos = append(repos, toRepository(r))
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return repos, nil
}

func toRepository(r *gh.Repository) Repository {
	repo := Repository{}
	if r == nil {
		return repo
	}
	if r.Owner != nil && r.Owner.Login != nil {
		repo.Owner = *r.Owner.Login
	}
	if r.Name != nil {
		repo.Name = *r.Name
	}
	if r.FullName != nil {
		repo.FullName = *r.FullName
	}
	if r.Description != nil {
		repo.Description = *r.Description
	}
	if r.Private != nil {
		repo.Private = *r.Private
	}
	if r.Fork != nil {
		repo.Fork = *r.Fork
	}
	if r.Archived != nil {
		repo.Archived = *r.Archived
	}
	if r.HTMLURL != nil {
		repo.HTMLURL = *r.HTMLURL
	}
	if r.SSHURL != nil {
		repo.SSHURL = *r.SSHURL
	}
	if r.CloneURL != nil {
		repo.CloneURL = *r.CloneURL
	}
	return repo
}
