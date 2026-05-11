# ghs — GitHub Select

[![CI](https://github.com/ray-on-code/ghs/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/ray-on-code/ghs/actions/workflows/ci.yml)
[![Release](https://github.com/ray-on-code/ghs/actions/workflows/release.yml/badge.svg)](https://github.com/ray-on-code/ghs/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ray-on-code/ghs)](https://goreportcard.com/report/github.com/ray-on-code/ghs)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

`ghs` は **「ghq のインタラクティブ拡張版」** を目指した Go 製 CLI ツールです。
自分や所属組織の GitHub リポジトリをターミナル UI で検索・選択し、
`ghq` 準拠のディレクトリ構成 (`$base/github.com/Owner/Repo`) にクローンします。

## ✨ 特徴

- 🔑 GitHub CLI (`gh auth token`) の認証情報をそのまま流用
- 🏢 自分のアカウントと所属組織を選択して切り替え
- 🔍 `survey.Select` のインクリメンタルフィルタでリポジトリを高速検索
- 📦 `ghq` 互換のディレクトリレイアウトでクローン
- ⚙️ `~/.config/ghs/config.toml` でベースディレクトリ等を設定可能

## 📥 インストール

### Homebrew (macOS / Linux)

```bash
brew install ray-on-code/tap/ghs
```

### GitHub Releases (バイナリ直接)

[Releases ページ](https://github.com/ray-on-code/ghs/releases) から
お使いの OS / アーキテクチャ向けのアーカイブをダウンロードし、
解凍した `ghs` バイナリを PATH の通った場所に配置してください。

```bash
# 例: macOS arm64
curl -L https://github.com/ray-on-code/ghs/releases/latest/download/ghs_Darwin_arm64.tar.gz \
  | tar -xz -C /usr/local/bin ghs
```

### Go Install

```bash
go install github.com/ray-on-code/ghs@latest
```

### ソースからビルド

```bash
git clone https://github.com/ray-on-code/ghs.git
cd ghs
make build   # bin/ghs に生成される
```

## 🚀 使い方

```bash
# 起動するだけで対話的に Owner → リポジトリ を選択 → クローンします
ghs

# SSH URL でクローンしたい場合
ghs --ssh

# ベースディレクトリを一時的に上書き
ghs --base-dir ~/work/ghs
```

### サブコマンド

| コマンド            | 説明                                                    |
| ------------------- | ------------------------------------------------------- |
| `ghs config init`   | `~/.config/ghs/config.toml` のテンプレートを生成        |
| `ghs config path`   | 設定ファイルのフルパスを表示                            |
| `ghs version`       | バージョン情報を表示                                    |

## 🔐 認証

トークンは以下の優先順位で解決されます。

1. **`gh auth token`** — GitHub CLI が認証済みであればそれを使用 (推奨)
2. **`~/.config/ghs/config.toml`** — 下記の `github_token` キー

```toml
# ~/.config/ghs/config.toml
github_token = "ghp_xxxxxxxxxxxxxxxxxxxx"
clone_base_dir = "~/ghs"
```

> 💡 GitHub CLI を使うのが最も簡単です。`brew install gh && gh auth login` を実行してください。

### 環境変数によるオーバーライド

`GHS_` プレフィックス付きの環境変数でも設定を上書きできます。

```bash
export GHS_CLONE_BASE_DIR="$HOME/work/ghs"
```

## 🗂 ディレクトリ構成

```
.
├── cmd/                # Cobra コマンド定義
│   ├── root.go
│   ├── config.go
│   └── version.go
├── pkg/
│   ├── auth/           # GitHub トークン解決ロジック
│   ├── config/         # Viper を用いた設定ロード/生成
│   ├── github/         # google/go-github の薄いラッパー
│   ├── ui/             # survey/v2 を用いた TUI
│   └── cloner/         # ghq 準拠のパス決定 + `git clone` 実行
└── main.go
```

## 🛠 動作要件

- Go 1.22 以上 (`go.mod` を参照)
- `git` コマンド
- (推奨) GitHub CLI (`gh`)

## 🧰 開発

開発に必要なタスクは `Makefile` に集約されています。

```bash
make            # ヘルプ表示
make build      # bin/ghs にビルド (VERSION 注入付き)
make test       # 全テスト実行
make test-race  # race detector 付きでテスト
make cover      # カバレッジ計測
make cover-html # coverage.html を生成
make fmt        # gofmt 適用
make vet        # go vet
make tidy       # go mod tidy
make check      # fmt-check + vet + test (PR 前推奨)
make ci         # CI 用一式 (fmt-check + vet + test-race + cover)
make clean      # 成果物削除
```

リント (`golangci-lint`) を使うには `make tools` でインストールしてから `make lint` を実行してください。

### コントリビューション

ブランチ運用・コミット規約は [CONTRIBUTING.md](./CONTRIBUTING.md) を参照してください。
リリース手順は [RELEASING.md](./RELEASING.md) を参照してください。

## 📜 ライセンス

MIT
