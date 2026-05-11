# ghs — GitHub Select

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

```bash
go install github.com/ray-on-code/ghs@latest
```

リポジトリをクローンしてビルドする場合:

```bash
git clone https://github.com/ray-on-code/ghs.git
cd ghs
go build -o ./bin/ghs .
```

## 🚀 使い方

```bash
# 起動するだけで対話的に Owner → リポジトリ を選択 → クローンします
ghs

# SSH URL でクローンしたい場合
ghs --ssh

# ベースディレクトリを一時的に上書き
ghs --base-dir ~/work/src
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
clone_base_dir = "/Users/you/src"
```

> 💡 GitHub CLI を使うのが最も簡単です。`brew install gh && gh auth login` を実行してください。

### 環境変数によるオーバーライド

`GHS_` プレフィックス付きの環境変数でも設定を上書きできます。

```bash
export GHS_CLONE_BASE_DIR="$HOME/work/src"
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

## 📜 ライセンス

MIT
