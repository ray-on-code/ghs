# Releasing ghs

メンテナ向けのリリース手順をまとめます。
日常のリリースは「タグを切るだけ」ですが、**初回セットアップ** だけ少し作業が必要です。

## 🏁 初回セットアップ (一度だけ)

### 1. Homebrew Tap リポジトリを作成

`brew install ray-on-code/tap/ghs` を実現するため、別リポジトリが必要です。

```bash
# GitHub 上で空のパブリックリポジトリを作成
gh repo create ray-on-code/homebrew-tap --public \
  --description "Homebrew Tap for ray-on-code tools" \
  --confirm

# ローカルで初期化 (空コミットだけでも OK)
git clone git@github.com:ray-on-code/homebrew-tap.git
cd homebrew-tap
git commit --allow-empty -m "chore: initial commit"
git push -u origin main
```

リポジトリ名は **必ず `homebrew-<tap名>`** にしてください (Homebrew の規約)。
今回は `homebrew-tap` なので、ユーザは `brew install ray-on-code/tap/ghs` でインストールできます。

### 2. Fine-grained Personal Access Token を作成

GoReleaser が `homebrew-tap` に formula を push するためのトークンが必要です。

1. https://github.com/settings/personal-access-tokens/new にアクセス
2. **Resource owner**: `ray-on-code` を選択
3. **Repository access** → **Only select repositories** → `ghs` と `homebrew-tap` を選ぶ
4. **Repository permissions**:
   - `Contents`: **Read and write**
   - `Metadata`: Read-only (デフォルト)
   - `Pull requests`: Read and write (オプション、PR ベースで brew を更新したい場合)
5. 有効期限: 1 年以内 (推奨 90 日 〜 1 年)
6. 生成されたトークンをコピー (再表示不可)

### 3. シークレットを `ghs` リポジトリに登録

```bash
# トークン文字列を環境変数に入れて gh CLI で登録
gh secret set HOMEBREW_TAP_TOKEN --repo ray-on-code/ghs
# プロンプトでトークンを貼り付け
```

または GitHub UI から:
`Settings` → `Secrets and variables` → `Actions` → `New repository secret`
- Name: `HOMEBREW_TAP_TOKEN`
- Value: 上記で生成したトークン

### 4. ブランチ保護ルールを設定

```bash
# main を保護: PR 必須 + CI 通過必須
gh api repos/ray-on-code/ghs/branches/main/protection \
  --method PUT \
  --input - <<EOF
{
  "required_status_checks": {
    "strict": true,
    "contexts": ["test (ubuntu-latest)", "test (macos-latest)", "build (ubuntu-latest)"]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": null,
  "restrictions": null,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "required_linear_history": true
}
EOF
```

または UI から:
`Settings` → `Branches` → `Add rule` で `main` に対し:
- ✅ Require a pull request before merging
- ✅ Require status checks to pass
  - `test (ubuntu-latest)`, `test (macos-latest)`, `build (ubuntu-latest)` を必須化
- ✅ Require linear history
- ❌ Allow force pushes
- ❌ Allow deletions

## 🚀 日常のリリース手順

### 1. リリース内容の確認

```bash
git checkout main
git pull
git log $(git describe --tags --abbrev=0)..HEAD --oneline
```

Conventional Commits に従ったコミットメッセージがあれば、GoReleaser が自動で
リリースノートを生成します。

### 2. ローカル検証 (任意だが推奨)

```bash
# GoReleaser を実際に走らせず、成果物だけ生成して確認
make release-snapshot

# dist/ 配下に各 OS のアーカイブが生成される
ls -la dist/
```

### 3. タグを切って push

```bash
# Semantic Versioning に従って
git tag -a v1.0.0 -m "v1.0.0"
git push origin v1.0.0
```

これだけで GitHub Actions が自動で:
- バイナリをクロスコンパイル (linux / darwin / windows × amd64 / arm64)
- GitHub Releases にアップロード
- `ray-on-code/homebrew-tap` の `Formula/ghs.rb` を更新する PR/コミットを push

### 4. リリース後の確認

```bash
# Release ページを開いて確認
gh release view v1.0.0 --repo ray-on-code/ghs --web

# Homebrew Tap が更新されているか
gh repo view ray-on-code/homebrew-tap --web

# 実際にインストールできるかテスト
brew untap ray-on-code/tap 2>/dev/null || true
brew install ray-on-code/tap/ghs
ghs version
```

### 5. Go Install の動作確認

```bash
go install github.com/ray-on-code/ghs@v1.0.0
ghs version   # → "ghs v1.0.0" になっていればOK (ldflags 注入は go install では効かないため "dev" のままになる場合あり)
```

> ⚠️ 補足: `go install` 経由では `cmd.Version` に ldflags が注入されないため、
> `ghs version` は `"ghs dev"` と表示されることがあります。
> これは Go の `runtime/debug.ReadBuildInfo()` で git tag を読み取る実装に
> 切り替えることで改善できますが、当面は GitHub Releases / Homebrew 経由のインストールを推奨します。

## 🔄 リリースのやり直し (失敗時)

タグを誤って push してしまった場合:

```bash
# GitHub Release を削除
gh release delete v1.0.0 --yes --repo ray-on-code/ghs

# リモートタグを削除
git push origin :refs/tags/v1.0.0

# ローカルタグを削除
git tag -d v1.0.0

# 修正後、再度タグを打って push
```

## 📋 Versioning Policy

`ghs` は **Semantic Versioning 2.0.0** に従います:

- **MAJOR** (x.0.0): 破壊的変更
- **MINOR** (0.x.0): 後方互換な機能追加
- **PATCH** (0.0.x): 後方互換なバグ修正

v0.x.x の間は **MINOR を破壊的変更にも使う** ことがあります (initial development phase)。
v1.0.0 以降は厳格に SemVer を守ります。

## 🆘 トラブルシュート

### GoReleaser がトークン関連で失敗する

`Resource not accessible by integration` というエラーが出る場合:
- `HOMEBREW_TAP_TOKEN` が `ghs` リポジトリの Secrets に登録されているか確認
- PAT のスコープが `homebrew-tap` リポジトリの **Contents: Read and write** を含むか確認
- PAT の有効期限が切れていないか確認

### Homebrew formula が古いまま

```bash
brew update
brew upgrade ray-on-code/tap/ghs
```

それでも古い場合は tap をリセット:

```bash
brew untap ray-on-code/tap
brew install ray-on-code/tap/ghs
```
