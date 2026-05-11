# Contributing to ghs

`ghs` への貢献ありがとうございます。本ドキュメントでは開発フロー・ブランチ運用・コミット規約をまとめます。

## 🌿 ブランチ運用 (GitHub Flow)

```
main (常にリリース可能)
 ├─ feat/owner-filter        ← 機能追加
 ├─ fix/clone-error-handling ← バグ修正
 └─ chore/deps               ← 依存更新
```

- `main` は **常にリリース可能** な状態を維持
- 直接 push せず、必ず PR を経由
- `main` への push / マージは **CI 通過必須**
- マージ方式: **Squash merge** (履歴を線形に保つ)

### ブランチ命名

| 種類 | プレフィックス | 例 |
|---|---|---|
| 新機能 | `feat/` | `feat/repo-filter` |
| バグ修正 | `fix/` | `fix/exit-code-on-cancel` |
| リファクタ | `refactor/` | `refactor/extract-prompter` |
| ドキュメント | `docs/` | `docs/installation` |
| 雑務・依存更新 | `chore/` | `chore/deps-update` |
| テスト | `test/` | `test/pkg-github-coverage` |

## ✍️ コミット規約 (Conventional Commits)

```
<type>: <subject>

[optional body]

[optional footer(s)]
```

### type

| type | 用途 | リリースノートでの扱い |
|---|---|---|
| `feat` | 新機能 | Features |
| `fix` | バグ修正 | Bug Fixes |
| `refactor` | 機能を変えないリファクタ | Others |
| `docs` | ドキュメントのみ | (非表示) |
| `test` | テストのみ | (非表示) |
| `chore` | ビルド・依存・雑務 | (非表示) |
| `ci` | CI/CD 設定 | (非表示) |
| `perf` | パフォーマンス改善 | Performance |

### 例

```
feat: Owner 選択にフィルタ入力を追加
fix: gh auth token 失敗時の握りつぶしを修正
docs: README に Homebrew インストール手順を追記
refactor: pkg/ui のラベル生成ロジックを純粋関数化
chore(deps): go-github を v60 → v61 に更新
```

### Breaking change

非互換変更がある場合は `!` を type の後ろに置き、body に `BREAKING CHANGE:` を記載:

```
feat!: --base-dir フラグを必須化

BREAKING CHANGE: 以前はデフォルトで $HOME/ghs を使っていたが
明示的な指定を要求するようにした。
```

## 🛠 開発フロー

```bash
# 1. リポジトリをクローン (Fork 不要、Collaborator なら直接)
git clone git@github.com:ray-on-code/ghs.git
cd ghs

# 2. ブランチを切る
git checkout -b feat/your-feature

# 3. 開発 + 検証
make check         # fmt-check + vet + test

# 4. コミット (Conventional Commits)
git commit -m "feat: ..."

# 5. push & PR
git push -u origin feat/your-feature
gh pr create
```

### PR 前チェックリスト

- [ ] `make check` がローカルで通る
- [ ] 新規ロジックには対応するテストを追加
- [ ] 公開 API を変更した場合は README / `docs/` を更新
- [ ] コミットメッセージが Conventional Commits に従っている

## 🧪 ローカル開発

詳細は `README.md` の「開発」セクションを参照してください。

主要コマンド:

```bash
make build      # bin/ghs を生成
make test       # 全テスト実行
make cover-html # カバレッジを HTML で可視化
make ci         # CI と同じパイプライン (fmt-check + vet + test-race + cover)
```

## 🏷 リリースフロー

メンテナ向け手順は `RELEASING.md` を参照してください。
概要:

1. `main` に変更をマージ
2. `git tag vX.Y.Z`
3. `git push origin vX.Y.Z`
4. GitHub Actions が GoReleaser を起動し、自動で:
   - GitHub Releases にバイナリをアップロード
   - Homebrew Tap (`ray-on-code/homebrew-tap`) に formula を push

## 🤖 Dependabot

- gomod と github-actions の更新が **週次** で自動 PR されます (`.github/dependabot.yml`)
- 依存更新 PR は CI が通れば積極的にマージしてください
- メジャー更新は破壊的変更を伴うことが多いので注意深くレビュー

## 質問・相談

[Issues](https://github.com/ray-on-code/ghs/issues) でお気軽にどうぞ。
