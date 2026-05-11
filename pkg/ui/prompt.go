// Package ui はターミナル UI (TUI) のプロンプト処理を担当します。
//
// Prompter インターフェースを介して上位レイヤから利用されることで、
// テストでは fake 実装に差し替え可能です。本番では survey/v2 を用いた
// SurveyPrompter を利用します。
package ui

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"

	ghapi "github.com/ray-on-code/ghs/pkg/github"
)

// Prompter は対話的な選択 UI の契約です。
type Prompter interface {
	// SelectOwner はユーザに Owner (自分 or 組織) を選択させ、選択結果の文字列を返します。
	SelectOwner(owners []string) (string, error)
	// SelectRepository はユーザにリポジトリを選択させ、選択された Repository を返します。
	SelectRepository(repos []ghapi.Repository) (*ghapi.Repository, error)
	// ConfirmOverwrite は対象ディレクトリ上書き確認用のプロンプトです。
	ConfirmOverwrite(path string) (bool, error)
}

// SurveyPrompter は survey/v2 を用いた Prompter の本番実装です。
type SurveyPrompter struct{}

// 静的な型チェック (満たしていない場合はコンパイルエラーになる)。
var _ Prompter = SurveyPrompter{}

// SelectOwner はユーザーに Owner を選択させます。
func (SurveyPrompter) SelectOwner(owners []string) (string, error) {
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
func (SurveyPrompter) SelectRepository(repos []ghapi.Repository) (*ghapi.Repository, error) {
	if len(repos) == 0 {
		return nil, fmt.Errorf("選択可能なリポジトリが存在しません")
	}

	labels, lookup := BuildRepoLookup(repos)

	var selected string
	prompt := &survey.Select{
		Message:  "📦 クローンするリポジトリを選択してください:",
		Options:  labels,
		PageSize: 15,
		Help:     "入力して絞り込み (例: \"repo-name\")、↑↓ で移動、Enter で決定します。",
		Filter:   FilterByLabel,
	}
	if err := survey.AskOne(prompt, &selected, survey.WithValidator(survey.Required)); err != nil {
		return nil, fmt.Errorf("リポジトリの選択がキャンセルされました: %w", err)
	}
	return ResolveRepoSelection(selected, lookup)
}

// BuildRepoLookup はリポジトリリストから、表示用ラベルの配列と
// ラベル→リポジトリの逆引きマップを生成します。SurveyPrompter から
// 切り出した純粋関数で、単体テスト容易性のために公開しています。
func BuildRepoLookup(repos []ghapi.Repository) ([]string, map[string]*ghapi.Repository) {
	labels := make([]string, len(repos))
	lookup := make(map[string]*ghapi.Repository, len(repos))
	for i := range repos {
		repo := &repos[i]
		label := FormatRepoLabel(repo)
		labels[i] = label
		lookup[label] = repo
	}
	return labels, lookup
}

// ResolveRepoSelection は選択ラベルから対応する Repository を返します。
// 見つからない場合はエラーを返します。
func ResolveRepoSelection(label string, lookup map[string]*ghapi.Repository) (*ghapi.Repository, error) {
	repo, ok := lookup[label]
	if !ok {
		return nil, fmt.Errorf("選択結果の解決に失敗しました: %s", label)
	}
	return repo, nil
}

// FilterByLabel は survey.Select の Filter 関数として使う、大文字小文字を無視した部分一致フィルタです。
func FilterByLabel(filterValue, optValue string, _ int) bool {
	return strings.Contains(strings.ToLower(optValue), strings.ToLower(filterValue))
}

// ConfirmOverwrite は対象ディレクトリが既に存在する場合に上書き確認を行います。
func (SurveyPrompter) ConfirmOverwrite(path string) (bool, error) {
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

// FormatRepoLabel は survey で見やすいラベルを生成します。
// 公開しているのは単体テストのためです (純粋関数のため副作用なし)。
func FormatRepoLabel(r *ghapi.Repository) string {
	if r == nil {
		return ""
	}
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
