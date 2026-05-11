// Package ui はターミナル UI (TUI) のプロンプト処理を担当します。
package ui

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	ghapi "github.com/ray-on-code/ghs/pkg/github"
)

// SelectOwner はユーザーに Owner (自分 or 組織) を選択させます。
func SelectOwner(owners []string) (string, error) {
	if len(owners) == 0 {
		return "", fmt.Errorf("選択可能な owner が存在しません")
	}

	var selected string
	prompt := &survey.Select{
		Message:  "🔍 クローン元の Owner を選択してください:",
		Options:  owners,
		Default:  owners[0],
		PageSize: 15,
		Help:     "入力して絞り込み、↑↓ で移動、Enter で決定します。",
	}
	if err := survey.AskOne(prompt, &selected, survey.WithValidator(survey.Required)); err != nil {
		return "", fmt.Errorf("owner の選択がキャンセルされました: %w", err)
	}
	return selected, nil
}

// SelectRepository はユーザーにリポジトリを選択させます。
// survey.Select の標準のフィルタ機能（タイプして絞り込み）を活用します。
func SelectRepository(repos []ghapi.Repository) (*ghapi.Repository, error) {
	if len(repos) == 0 {
		return nil, fmt.Errorf("選択可能なリポジトリが存在しません")
	}

	labels := make([]string, len(repos))
	lookup := make(map[string]*ghapi.Repository, len(repos))
	for i := range repos {
		repo := &repos[i]
		label := formatRepoLabel(repo)
		labels[i] = label
		lookup[label] = repo
	}

	var selected string
	prompt := &survey.Select{
		Message:  "📦 クローンするリポジトリを選択してください:",
		Options:  labels,
		PageSize: 15,
		Help:     "入力して絞り込み (例: \"repo-name\")、↑↓ で移動、Enter で決定します。",
		Filter: func(filterValue string, optValue string, optIndex int) bool {
			return strings.Contains(strings.ToLower(optValue), strings.ToLower(filterValue))
		},
	}
	if err := survey.AskOne(prompt, &selected, survey.WithValidator(survey.Required)); err != nil {
		return nil, fmt.Errorf("リポジトリの選択がキャンセルされました: %w", err)
	}
	repo, ok := lookup[selected]
	if !ok {
		return nil, fmt.Errorf("選択結果の解決に失敗しました: %s", selected)
	}
	return repo, nil
}

// ConfirmOverwrite は対象ディレクトリが既に存在する場合に上書き確認を行います。
func ConfirmOverwrite(path string) (bool, error) {
	var ok bool
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("⚠️  %s は既に存在します。中身を確認して続行しますか? (No で中止)", path),
		Default: false,
	}
	if err := survey.AskOne(prompt, &ok); err != nil {
		return false, err
	}
	return ok, nil
}

// formatRepoLabel は survey で見やすいラベルを生成します。
func formatRepoLabel(r *ghapi.Repository) string {
	flags := make([]string, 0, 3)
	if r.Private {
		flags = append(flags, "private")
	}
	if r.Fork {
		flags = append(flags, "fork")
	}
	if r.Archived {
		flags = append(flags, "archived")
	}

	label := r.FullName
	if label == "" {
		label = fmt.Sprintf("%s/%s", r.Owner, r.Name)
	}
	if len(flags) > 0 {
		label = fmt.Sprintf("%s [%s]", label, strings.Join(flags, ","))
	}
	if r.Description != "" {
		desc := r.Description
		const max = 60
		if len(desc) > max {
			desc = desc[:max] + "…"
		}
		label = fmt.Sprintf("%s — %s", label, desc)
	}
	return label
}
