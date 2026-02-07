# project information manager

本プロジェクトで作成するアプリケーションは、AI Agentによるプロダクト開発を支援するために、コンテキスト情報を高度に管理する仕組みを提供する。
MCPとSkills、RAGによって構成されることを想定する。

## 主な機能

プロダクトごとに下記を実現するために、agent rule, agent workflow, agent skills, MCP の設定を自動構築するための汎用的な機能を提供する。

* プロダクトのゴール、想定課題、解決策、想定作業、生み出す価値を記述し、プロダクトの進行している方向の妥当性をチェック
* プロダクトを管理するにあたっての管理手段を定義し、管理状態をチェック
  * スクラム型: ユーザストーリー、プロダクトバックログ、スプリントバックログ、インクリメント など
  * ITIL v3型: インシデント管理、問題管理、変更管理、リリース管理、構成管理、ナレッジ管理 など
* プロジェクトで構築するソリューションの設計を管理
  * ソリューションの機能要件と、これに対応する「概要設計」を管理
  * ソリューションの被機能要件と、これに対応する「方式設計（基盤方式, アーキテクチャ設計, コスト予想・削減指針）」を管理
  * 概要設計をさらに詳細化した、業務ロジックや実際の実装方針、ロジック内容、データ設計を定義する「基本設計」を管理
  * 方式設計をさらに詳細化した、基盤環境設計を管理。基盤構築用リソースの配置方針やセキュリティ維持の考え方、コスト低減のための工夫を記述。
  * CI/CDアーキテクチャ、監視・通知などの運用設計
  * 各要件事項と設計、実装を紐づける管理番号を付与
* プロジェクトで行うべきタスクと優先度、その進行状況を管理
* 発生している課題と解決手段、解決状況を管理
* 開発ルールを定義。設計に従ったテスト実装の作成、TDDによる開発フローやアプリケーションのアーキテクチャ定義、環境管理・ビルド用ツールなど
* 階層的テスト設計を定義
  * 自動単体テスト
  * 自動連結テスト
  * 自動E2Eテスト
  * UI/UXテスト
* LLMに投入する入力トークンが、システムプロンプトやMCP、Skills、本アプリケーションが提供する情報、Session内コンテキスト、依頼情報などからどの程度の割合で構成されているかを可視化し、過剰なMCP, Skills, システムプロンプトを入力してしまっていることを検出しやすくする

## 設計思想

プロジェクトごとに異なる上記の管理方針を順次定義するインタフェースを提供。定義された管理指針に従ってツールを選定・構築する。
特に、静的に定義する情報（Stocks）と、プロダクト開発プロジェクトの状態管理（States）を個別に管理する。

* StocksはWikiに記述されるもののように、プロダクトの定義や設計情報、ルールなどを記述。
* Statesはチケット管理形式で、各トピックについての状態と対処を記述し、完了したらアーカイブする（アーカイブ時に重要情報をStockに記述する）
* Stock, Stateは双方インデックスを設定し、必要な条件下で読み出せるようにする。
  * このために各Stocksは個別のSkillsとして保存する
    * Stocksの情報の参照優先度は、より上位の設計資料を優先とし、詳細の設計情報は低優先度。低優先度のものは状況に合っている場合のみ参照され、高優先度のものはできるだけ多くの場合に参照されるようにする
  * 各Statesは単一のSkills配下で管理する。下記を実現するSkillとする
    * archive: 不要なStateのアーカイブ（普段は参照されない状態にする）
    * create: 追加のStateの作成
    * update: 既存stateの状態管理・情報追加

## システム構成

### 技術選定

| カテゴリ | 技術 | 選定理由 |
|---|---|---|
| 実装言語 | **Go** | 高速なバイナリ生成、goroutineによる並行処理、CGO不要でのクロスコンパイル容易性 |
| MCP SDK | [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) v1.2+ | MCP公式SDK（Google協力開発）。MCP spec 2025-11-25対応 |
| LLM SDK | [anthropics/anthropic-sdk-go](https://github.com/anthropics/anthropic-sdk-go) v1.21+ | Anthropic公式。Skills API対応（`betaskill.go`）。Bedrock/Vertex対応 |
| LLM SDK (補助) | [sashabaranov/go-openai](https://github.com/sashabaranov/go-openai) | OpenAI互換API用。マルチプロバイダー対応のため |
| ベクトルDB | [philippgille/chromem-go](https://github.com/philippgille/chromem-go) | 組み込み型ベクトルDB。CGO不要。外部サービス不要でローカル完結 |
| RDB | [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) | Pure Go SQLite実装。CGO不要。States永続化用 |
| テスト | `testing` 標準パッケージ + [stretchr/testify](https://github.com/stretchr/testify) | テーブル駆動テスト + アサーション強化 |
| ビルド | `go build` + Makefile | シンプルなビルドパイプライン |

#### 言語選定の補足（Go vs Rust）

Rustも検討したが、以下の理由でGoを採用した:

* **Anthropic公式SDK**: Go版はv1.21でSkills API対応済み。Rust版は公式SDK未公開（GitHub 404）
* **開発速度**: Goのシンプルな文法と高速コンパイルにより、プロトタイピングから本番投入までが高速
* **CGO不要のSQLite**: `modernc.org/sqlite`によりCリンク不要。クロスコンパイルとCI/CDが容易
* **並行処理**: goroutineにより、MCPリクエスト処理やLLM API呼び出しの並行実行が言語レベルで自然に記述可能
* 将来的にパフォーマンスクリティカルなベクトル検索部分のみRust/WASM化する選択肢は残す

### アーキテクチャ概要

```
┌─────────────────────────────────────────────────────────────────────┐
│                        AI Agent (Claude等)                         │
└──────────────────────────────┬──────────────────────────────────────┘
                               │ MCP Protocol (stdio / SSE)
┌──────────────────────────────▼──────────────────────────────────────┐
│                     MCP Server (Go binary)                         │
│                                                                     │
│  MCPツール層 (ファサードパターン: 3ツールのみ)                        │
│  ┌──────────────────┐ ┌──────────────────┐ ┌────────────────────┐  │
│  │ stock_manage     │ │ state_manage     │ │ context_search     │  │
│  │ action:          │ │ action:          │ │ - RAG統合検索       │  │
│  │  create/read/    │ │  create/read/    │ │ - Stock+State横断   │  │
│  │  list/update/    │ │  update/archive/ │ │ - サマリビュー返却   │  │
│  │  search          │ │  list/search     │ │                    │  │
│  └────────┬─────────┘ └────────┬─────────┘ └─────────┬──────────┘  │
│           │                    │                      │             │
│  ┌────────▼────────────────────▼──────────────────────▼───────────┐ │
│  │                    Domain Service Layer                        │ │
│  │  ┌────────────────┐ ┌────────────────┐ ┌────────────────────┐ │ │
│  │  │ StockService   │ │ StateService   │ │ ContextService     │ │ │
│  │  │ - CRUD         │ │ - CRUD         │ │ - RAG横断検索       │ │ │
│  │  │ - Summary View │ │ - Lifecycle    │ │ - コンテキスト集約   │ │ │
│  │  │ - Full View    │ │ - Archive      │ │ - トークン見積      │ │ │
│  │  └───────┬────────┘ └───────┬────────┘ └────────┬───────────┘ │ │
│  └──────────┼──────────────────┼───────────────────┼─────────────┘ │
│             │                  │                   │               │
│  ┌──────────▼──────────────────▼───────────────────▼─────────────┐ │
│  │                   Repository Layer                            │ │
│  │  ┌─────────────────┐ ┌──────────────────┐                    │ │
│  │  │ StockRepository │ │ StateRepository  │                    │ │
│  │  │ (File System)   │ │ (SQLite)         │                    │ │
│  │  └────────┬────────┘ └────────┬─────────┘                    │ │
│  └───────────┼───────────────────┼──────────────────────────────┘ │
│              │                   │                                 │
│  ┌───────────▼───────────────────▼──────────────────────────────┐ │
│  │                     RAG Engine                               │ │
│  │  chromem-go (組み込みベクトルDB)                               │ │
│  │  - Stock/Stateのベクトルインデックス生成                        │ │
│  │  - 優先度加重付きセマンティック検索                              │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                    LLM Gateway                               │ │
│  │  anthropic-sdk-go / go-openai                                │ │
│  │  - トークン使用量追跡                                          │ │
│  └──────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘

永続化ストレージ:
┌─────────────────────┐  ┌──────────────┐
│ stocks/              │  │ states.db    │
│  ├── {project}/      │  │ (SQLite)     │
│  │   ├── design/     │  │ - states     │
│  │   ├── rules/      │  │ - archives   │
│  │   ├── management/ │  │ - indexes    │
│  │   └── index.json  │  │              │
│  └── ...             │  └──────────────┘
└─────────────────────┘
```

### コンテキスト最適化設計

本システムの目的は「AI Agentにプロジェクト情報を効率的に提供する」ことであるため、MCP経由で消費されるコンテキストトークンを最小化する設計が不可欠。

#### 課題と対策

| # | 課題 | 影響 | 対策 |
|---|---|---|---|
| ① | MCPツール数が多い（旧: 13ツール） | ツール定義だけで3,000-5,000トークン消費 | **ファサードパターンで3ツールに統合**。action引数でCRUD操作を指定 |
| ② | レスポンスがフルJSON | list/searchで全Content/Description返却 | **Summary View / Full View** の2段階設計。list/searchはSummary（ID, Title, Priority, UpdatedAt）のみ返却。readで全文取得 |
| ③ | P0/P1常時ロードがスケールしない | P0/P1のStock増加で7,500-30,000トークン常時消費 | **P0/P1はツールdescriptionに要約のみ記載**。全文はreadアクションで取得 |
| ④ | 1:1 Stock→Skill生成 | Stock増加=ツール数爆発。本アプリの目的と矛盾 | **1:1マッピング廃止**。SkillService→ContextServiceに変更。RAGで横断検索→サマリ集約して返却 |
| ⑤ | State Descriptionフル返却 | 一覧取得で大量トークン消費 | ②と同じSummary View対策 |

#### Summary View と Full View

```
Summary View (list/search レスポンス):
┌──────────────────────────────────────────┐
│ { "id": "STK-DESIGN-001",               │
│   "title": "API設計方針",                │
│   "category": "design",                 │
│   "priority": "P0",                     │
│   "tags": ["api", "rest"],              │
│   "updated_at": "2025-01-15T..." }      │
└──────────────────────────────────────────┘
→ 1件あたり ~100トークン

Full View (read レスポンス):
┌──────────────────────────────────────────┐
│ { "id": "STK-DESIGN-001",               │
│   "title": "API設計方針",                │
│   "content": "# API設計方針\n\n...",     │
│   ... 全フィールド }                      │
└──────────────────────────────────────────┘
→ 必要なときだけ取得
```

### Skills 動的生成機構（改訂）

[Anthropic Skills API](https://github.com/anthropics/claude-cookbooks/tree/main/skills) の設計パターンを参考にしつつ、**1:1 Stock→Skill マッピングは採用しない**。

#### 旧設計の問題

Stock 1件ごとにSkillファイル（= MCPツール）を生成すると、Stock数の増加がそのままツール数の爆発を招き、本アプリの「コンテキストを効率管理する」という目的と矛盾する。

#### 改訂後の設計: ContextService

StockとStateの情報をRAGベースで横断検索し、必要な情報だけを集約してAI Agentに提供する `context_search` ツールを1本のみ公開する。

```
Agent が context_search ツールを呼び出し
       │
       ▼
ContextService.Search()
  - クエリをベクトル検索（chromem-go）
  - Stock / State の両方から関連ドキュメントを検索
  - Priority による重み付けスコアリング
  - 上位N件のSummary Viewを返却
       │
       ▼
Agent が詳細を知りたい場合
  → stock_manage action=read / state_manage action=read で個別取得
```

#### 優先度ベースのコンテキスト制御（改訂）

| 優先度 | 対象 | ロード方式 | トークン消費 |
|---|---|---|---|
| P0 (最高) | プロダクトゴール、アーキテクチャ方針 | **ツールdescriptionに1行要約を記載**。全文はreadで取得 | ~50トークン/件 (要約) |
| P1 (高) | 概要設計、管理方針 | context_search の結果で優先的に上位表示 | 検索時のみ |
| P2 (中) | 基本設計、方式設計 | RAG検索でオンデマンドロード | 検索時のみ |
| P3 (低) | 詳細実装メモ、過去の議事録 | 明示的なクエリ時のみロード | 検索時のみ |

**変更点**: 旧設計のP0/P1「常時フル内容ロード」を廃止。P0情報は `stock_manage` ツールのdescription内に1行要約として埋め込み（~50トークン）、Agent が必要に応じてreadで全文取得する方式に変更。

### コンポーネント構成

```
project-information-manager/
├── cmd/
│   └── pim-server/
│       └── main.go                 # エントリポイント（MCPサーバー起動）
├── internal/
│   ├── domain/                     # ドメインモデル
│   │   ├── stock.go                # Stock エンティティ + StockSummary
│   │   ├── state.go                # State エンティティ + StateSummary
│   │   ├── project.go              # Project エンティティ
│   │   └── errors.go               # ドメインエラー定義
│   ├── service/                    # ビジネスロジック
│   │   ├── stock_service.go        # Stock CRUD + Summary/Full View
│   │   ├── state_service.go        # State ライフサイクル管理 + Summary/Full View
│   │   ├── context_service.go      # RAG横断検索・コンテキスト集約
│   │   ├── rag_service.go          # RAG検索・インデックス管理
│   │   └── token_tracker.go        # トークン使用量追跡
│   ├── repository/                 # 永続化層
│   │   ├── interfaces.go           # リポジトリインターフェース定義
│   │   ├── stock_repository.go     # Stock リポジトリ（ファイルシステム）
│   │   ├── state_repository.go     # State リポジトリ（SQLite）
│   │   ├── vector_repository.go    # ベクトルインデックス（chromem-go）
│   │   └── repositories.go        # リポジトリ初期化・集約
│   ├── mcp/                        # MCPサーバー・ツール定義
│   │   ├── server.go               # MCPサーバー初期化・起動
│   │   ├── tools_stock.go          # stock_manage ファサードツール
│   │   ├── tools_state.go          # state_manage ファサードツール
│   │   └── tools_context.go        # context_search 統合検索ツール
│   ├── llm/                        # LLM統合
│   │   ├── gateway.go              # LLMプロバイダー抽象化
│   │   ├── anthropic.go            # Anthropic Claude連携
│   │   └── openai.go               # OpenAI連携
│   └── config/                     # 設定管理
│       └── config.go               # アプリケーション設定
├── configs/
│   └── default.yaml                # デフォルト設定ファイル
├── data/                           # ランタイムデータ（.gitignore対象）
│   ├── stocks/                     # Stockファイル格納
│   ├── states.db                   # SQLiteデータベース
│   └── vectors/                    # ベクトルインデックス
├── Makefile                        # ビルド・テスト・リントコマンド
├── go.mod
├── go.sum
└── README.md
```

### データモデル

#### Stock

```go
type Stock struct {
    ID          string         // 管理番号 (例: "STK-DESIGN-001")
    ProjectID   string         // 所属プロジェクトID
    Category    StockCategory  // design | rules | management | architecture
    Priority    Priority       // P0 | P1 | P2 | P3
    Title       string         // タイトル
    Content     string         // Markdown形式の本文
    Tags        []string       // 検索用タグ
    References  []string       // 関連Stock/StateのID
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type StockCategory string
const (
    CategoryDesign       StockCategory = "design"        // 設計情報
    CategoryRules        StockCategory = "rules"         // 開発ルール
    CategoryManagement   StockCategory = "management"    // 管理方針
    CategoryArchitecture StockCategory = "architecture"  // 方式設計
    CategoryRequirement  StockCategory = "requirement"   // 要件定義
    CategoryTest         StockCategory = "test"          // テスト設計
)
```

#### State

```go
type State struct {
    ID          string       // 管理番号 (例: "STA-TASK-042")
    ProjectID   string       // 所属プロジェクトID
    Type        StateType    // task | issue | incident | change
    Status      StateStatus  // open | in_progress | resolved | archived
    Priority    Priority     // P0 | P1 | P2 | P3
    Title       string       // タイトル
    Description string       // 詳細説明
    Resolution  string       // 解決内容（resolved/archived時）
    Tags        []string     // 検索用タグ
    References  []string     // 関連Stock/StateのID
    CreatedAt   time.Time
    UpdatedAt   time.Time
    ArchivedAt  *time.Time   // アーカイブ日時（nilなら未アーカイブ）
}

type StateStatus string
const (
    StatusOpen       StateStatus = "open"
    StatusInProgress StateStatus = "in_progress"
    StatusResolved   StateStatus = "resolved"
    StatusArchived   StateStatus = "archived"
)
```

#### Skill（生成されるSkillのメタデータ）

> **Note**: 旧設計のSkillエンティティは廃止。1:1 Stock→Skill マッピングを行わず、ContextServiceによるRAG横断検索に置き換え。

### MCP ツール定義（改訂: 3ツール ファサードパターン）

旧設計の13ツールを3ツールに統合。ツールスキーマによるコンテキスト消費を ~3,000-5,000トークン → ~800トークン に削減。

| ツール名 | 説明 | actionパラメータ | 主な入力パラメータ |
|---|---|---|---|
| `stock_manage` | Stock（静的プロジェクト情報）の管理 | `create`, `read`, `list`, `update`, `search` | action別: projectId, stockId, category, priority, title, content, query等 |
| `state_manage` | State（動的状態情報）の管理 | `create`, `read`, `update`, `archive`, `list`, `search` | action別: projectId, stateId, type, status, description, query等 |
| `context_search` | Stock+State横断のRAG検索 | ― | query, projectId, limit? |

#### レスポンス形式

- **list / search アクション**: Summary View（ID, Title, Priority, Category/Type, Tags, UpdatedAt のみ）
- **read アクション**: Full View（全フィールド）
- **create / update / archive アクション**: 操作結果のSummary View

### デプロイ形態

#### Phase 1: ローカル実行型（現在のスコープ）

```
開発者のマシン
┌─────────────────────────────────────────┐
│ AI Agent (Claude Code / Cursor等)       │
│      │ stdio                            │
│      ▼                                  │
│ pim-server (Go binary)                  │
│      │                                  │
│      ├── stocks/  (ローカルファイル)      │
│      ├── skills/  (ローカルファイル)      │
│      ├── states.db (SQLite)             │
│      └── vectors/ (chromem-go)          │
└─────────────────────────────────────────┘
```

* 単一バイナリで配布（`go build`で生成）
* MCP stdioトランスポートでAI Agentと通信
* すべてのデータはローカルファイルシステム + SQLiteに永続化
* 外部サービス依存なし（LLM API呼び出しを除く）

#### Phase 2: クラウドベース・マルチユーザー（将来）

```
┌────────────┐    ┌────────────┐
│ Agent A    │    │ Agent B    │
└─────┬──────┘    └─────┬──────┘
      │ SSE/StreamableHTTP│
      ▼                  ▼
┌─────────────────────────────┐
│ pim-server (HTTP/gRPC)      │
│   ├── Auth / RBAC           │
│   ├── Multi-tenant Router   │
│   └── API Gateway           │
├─────────────────────────────┤
│ Storage Backend             │
│   ├── PostgreSQL (States)   │
│   ├── Object Storage(Stocks)│
│   └── Managed Vector DB     │
└─────────────────────────────┘
```

### 将来的な外部ツール連携アーキテクチャ

Phase 1ではローカル完結とし、将来的にJira / GitHub Issues連携を追加する際は、以下のLocal Cache + 差分同期パターンを採用する。

```
外部ツール (Jira / GitHub Issues)
       │
       ▼  API Sync (定期 or Webhook)
┌──────────────────────────────┐
│ External Adapter Layer       │
│  - Jira Adapter              │
│  - GitHub Issues Adapter     │
│  - 差分検出・部分同期         │
│  - Conflict Resolution       │
└──────────┬───────────────────┘
           ▼
┌──────────────────────────────┐
│ Local Cache (SQLite)         │
│  - 外部チケットのローカル複製  │
│  - 最終同期タイムスタンプ      │
│  - 変更フラグ                 │
└──────────┬───────────────────┘
           ▼
┌──────────────────────────────┐
│ Cache Search Engine          │
│  - ローカルキャッシュ全文検索  │
│  - ベクトル検索               │
│  - State/Stockとの相互参照    │
└──────────────────────────────┘

同期方式:
1. 初回: 全量取得 → ローカルキャッシュ構築
2. 定常: タイムスタンプベースの差分取得（トークン消費を最小化）
3. オンデマンド: 特定チケットの最新情報を部分取得
4. 書き戻し: ローカルで更新したStateを外部ツールに反映
```

