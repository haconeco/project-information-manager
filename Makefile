.PHONY: build test lint clean run

# ビルド
build:
	go build -o bin/pim-server ./cmd/pim-server

# テスト実行
test:
	go test ./... -v -count=1

# テスト（カバレッジ付き）
test-coverage:
	go test ./... -v -count=1 -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# リント
lint:
	go vet ./...

# クリーンアップ
clean:
	rm -rf bin/ coverage.out coverage.html

# 実行（stdio MCP）
run: build
	./bin/pim-server

# 依存関係整理
tidy:
	go mod tidy

# データディレクトリ初期化
init-data:
	mkdir -p data/stocks data/skills data/vectors
