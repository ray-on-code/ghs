# ghs — development Makefile
#
# 使い方:
#   make            # ヘルプを表示
#   make build      # バイナリをビルド
#   make test       # 全テスト実行
#   make ci         # CI 相当チェック (fmt-check + vet + test)
#
# 注意: Makefile のレシピ行は **タブ** インデント必須です。

# ---- 基本設定 --------------------------------------------------------------

GO          ?= go
PKG         ?= ./...
BIN_DIR     ?= bin
BIN_NAME    ?= ghs
BIN         := $(BIN_DIR)/$(BIN_NAME)

MODULE      := github.com/ray-on-code/ghs
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS     := -s -w -X $(MODULE)/cmd.Version=$(VERSION)
BUILD_FLAGS ?= -trimpath -ldflags '$(LDFLAGS)'

COVER_OUT   := coverage.out
COVER_HTML  := coverage.html

# golangci-lint を tools として固定バージョンで使う想定
GOLANGCI_LINT_VERSION ?= v1.61.0

# ---- ヘルプ (デフォルトターゲット) ----------------------------------------

.DEFAULT_GOAL := help

.PHONY: help
help: ## このヘルプを表示します
	@printf "\n\033[1mghs Makefile\033[0m — 開発タスク一覧\n\n"
	@awk 'BEGIN {FS = ":.*?## "} \
	     /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' \
	     $(MAKEFILE_LIST)
	@printf "\n変数を上書きできます (例: \033[33mmake build VERSION=1.0.0\033[0m)\n\n"

# ---- ビルド ----------------------------------------------------------------

.PHONY: build
build: ## バイナリを bin/ghs にビルド (VERSION 注入付き)
	@mkdir -p $(BIN_DIR)
	$(GO) build $(BUILD_FLAGS) -o $(BIN) .

.PHONY: install
install: ## $GOBIN に ghs をインストール
	$(GO) install $(BUILD_FLAGS) .

.PHONY: run
run: build ## ビルドしてそのまま実行 (引数は ARGS= で渡せる)
	@$(BIN) $(ARGS)

# ---- テスト ----------------------------------------------------------------

.PHONY: test
test: ## 全テストを実行
	$(GO) test -count=1 $(PKG)

.PHONY: test-v
test-v: ## 全テストを verbose で実行
	$(GO) test -v -count=1 $(PKG)

.PHONY: test-race
test-race: ## race detector 付きでテスト実行
	$(GO) test -race -count=1 $(PKG)

.PHONY: cover
cover: ## カバレッジを計測 (coverage.out 生成)
	$(GO) test -count=1 -coverprofile=$(COVER_OUT) $(PKG)
	@$(GO) tool cover -func=$(COVER_OUT) | tail -n 1

.PHONY: cover-html
cover-html: cover ## カバレッジを HTML で出力 (coverage.html 生成)
	$(GO) tool cover -html=$(COVER_OUT) -o $(COVER_HTML)
	@echo "📊 $(COVER_HTML) を生成しました"

.PHONY: cover-func
cover-func: cover ## 関数単位のカバレッジを表示
	$(GO) tool cover -func=$(COVER_OUT)

# ---- 静的解析・整形 --------------------------------------------------------

.PHONY: fmt
fmt: ## gofmt を実行 (差分があれば書き換え)
	$(GO) fmt $(PKG)

.PHONY: fmt-check
fmt-check: ## gofmt 差分があれば失敗 (CI 用)
	@out=$$(gofmt -l . | grep -v '^vendor/' || true); \
	if [ -n "$$out" ]; then \
		echo "❌ 以下のファイルに gofmt 差分があります:"; \
		echo "$$out"; \
		exit 1; \
	fi

.PHONY: vet
vet: ## go vet を実行
	$(GO) vet $(PKG)

.PHONY: lint
lint: ## golangci-lint を実行 (未インストールなら案内のみ)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run $(PKG); \
	else \
		echo "⚠️  golangci-lint がインストールされていません"; \
		echo "   インストール: make tools"; \
		exit 1; \
	fi

# ---- モジュール ------------------------------------------------------------

.PHONY: tidy
tidy: ## go mod tidy を実行
	$(GO) mod tidy

.PHONY: deps
deps: ## 依存ライブラリをダウンロード
	$(GO) mod download

# ---- ツール ----------------------------------------------------------------

.PHONY: tools
tools: ## 開発用ツール (golangci-lint 等) をインストール
	@echo "📦 golangci-lint $(GOLANGCI_LINT_VERSION) をインストールします..."
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

# ---- リリース --------------------------------------------------------------

.PHONY: release-snapshot
release-snapshot: ## GoReleaser でスナップショットビルド (タグ不要、dist/ に成果物)
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "⚠️  goreleaser が見つかりません"; \
		echo "   インストール: brew install goreleaser または make tools-release"; \
		exit 1; \
	fi
	goreleaser release --snapshot --clean

.PHONY: release-check
release-check: ## .goreleaser.yaml の構文検証
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "⚠️  goreleaser が見つかりません (make tools-release でインストール)"; \
		exit 1; \
	fi
	goreleaser check

.PHONY: tools-release
tools-release: ## goreleaser をインストール (Homebrew 経由)
	@if command -v brew >/dev/null 2>&1; then \
		brew install goreleaser; \
	else \
		echo "⚠️  Homebrew がありません。手動でインストールしてください: https://goreleaser.com/install/"; \
		exit 1; \
	fi

# ---- 集約タスク ------------------------------------------------------------

.PHONY: check
check: fmt-check vet test ## fmt-check + vet + test を実行

.PHONY: ci
ci: fmt-check vet test-race cover ## CI 用: fmt-check + vet + test-race + cover

# ---- クリーンアップ --------------------------------------------------------

.PHONY: clean
clean: ## ビルド成果物・カバレッジファイルを削除
	rm -rf $(BIN_DIR) dist $(COVER_OUT) $(COVER_HTML)
