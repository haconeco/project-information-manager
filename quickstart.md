# Quickstart

このドキュメントは「新しくプロダクトを作り、MCPを立ち上げ、設計情報や設計ルールを記述・管理する」ための最小手順をまとめます。開発手順や実装者向けの手続きは含みません。

## 導入手順（設定のみ）

このセクションの設定だけを先に行えば、MCPは利用可能になります。具体的な使い方は後述の「2. 使い方」を参照してください。

### 1) MCPクライアント設定を反映（必須）

MCPクライアントに `pim-server` を stdio で起動させ、3ツールのみを利用する設定にします。

```
# 例: MCPクライアント設定の概念例
mcp_servers:
  - name: project-information-manager
	 transport: stdio
	 command: /path/to/pim-server
	 args: []
	 tools: [stock_manage, state_manage, context_search]
```

### 2) プロジェクトのルールに運用ルールを追加（必須）

新規プロジェクトの system prompt / agent rules / skills に、以下のルールを**そのまま**追加します。
（例: rules.md / agent_rules.md など）

```
## セッション内での設計・タスク状態管理ルール（必須）
- MCP Server の project-information-manager を必ず利用して管理する。
- すべての設計・ルール・方針の追加/変更は、必ず stock_manage で登録/更新する。
- すべての進行中のタスク・課題・変更は、必ず state_manage で登録/更新/アーカイブする。
- 情報参照は context_search を優先し、詳細が必要な場合のみ stock_manage read / state_manage read を使う。
- MCPに未登録の設計や状態を会話内で発見した場合、必ずMCPへ追記/更新を要求する。
```

## 0. 事前準備

- MCPクライアント（例: Claude Code / Cursor など、MCP stdio を利用できるもの）
- `pim-server` バイナリのパス（実行可能になっていること）
- 新規プロジェクトのID（例: `proj-foo`）

## 1. MCPサーバー起動

MCPクライアントの設定に `pim-server` を登録し、stdio モードで起動します。設定を反映したらクライアントを再起動し、ツール一覧に以下の3つが表示されることを確認してください。

- `stock_manage`（静的情報: 設計、ルール、管理方針など）
- `state_manage`（動的情報: タスク、課題、変更など）
- `context_search`（Stock/State 横断検索）

## 2. 使い方（AIエージェントとユーザの共同作業）

以下は**AIエージェントとユーザが会話しながら進める作業**です。ユーザは「登録したい内容」を伝え、AIエージェントがMCPツールを呼び出して登録・更新します。

### 2.1 新規プロダクトのベース情報を登録（Stock）

新しくプロダクトを始めるときは、まず **プロダクトのゴール、設計方針、ルール** などの静的情報を `stock_manage` で登録します。

**やり取り例**

- ユーザ: 「このプロダクトのゴールは○○。P0で登録して」
- AIエージェント: `stock_manage` の `create` でゴールを登録し、作成結果（Summary View）を提示

#### 2.1.1 プロダクトゴールの登録（P0）

- **目的**: もっとも重要な指針を P0 として登録し、検索・参照の優先度を上げます。
- **推奨カテゴリ**: `requirement` または `management`

`stock_manage` の `create` アクションで以下を登録します。

- `projectId`: 任意のプロダクトID（例: `proj-foo`）
- `category`: `requirement` または `management`
- `priority`: `P0`
- `title`: プロダクトゴール
- `content`: ゴールの詳細（Markdown）
- `tags`: 任意

#### 2.1.2 設計方針の登録（P1/P2）

- **目的**: 概要設計・方式設計・基本設計を段階的に記述
- **推奨カテゴリ**: `design` / `architecture`
- **推奨優先度**: P1（概要）→ P2（詳細）

`stock_manage` の `create` で順に登録します。

#### 2.1.3 開発ルール・テスト方針の登録（P1/P2）

- **推奨カテゴリ**: `rules` / `test`
- **内容例**: テスト設計方針、CI/CD、レビュー基準、命名規約

こちらも `stock_manage` の `create` で登録します。

### 2.2 プロジェクトの動的情報を管理（State）

タスクや課題など、進行中の状態は `state_manage` で管理します。

**やり取り例**

- ユーザ: 「API設計レビューをタスクとして登録して。優先度はP1」
- AIエージェント: `state_manage` の `create` でタスクを登録し、作成結果を提示

#### 2.2.1 タスクの作成

`state_manage` の `create` で登録します。

- `type`: `task`
- `status`: `open` / `in_progress`
- `priority`: P0〜P3
- `title`, `description`: 具体的に記述

#### 2.2.2 課題・変更の管理

- **課題**: `type=issue`
- **変更**: `type=change`

進捗に応じて `update` で `status` や `resolution` を更新します。完了後は `archive` でアーカイブします。

### 2.3 情報の検索と活用

#### 2.3.1 横断検索（推奨）

`context_search` を使い、Stock/State をまとめて検索します。AI Agent に文脈を渡す際は、この結果（Summary View）を優先利用します。

- **例**: 「API設計方針を確認したい」「現在のP0課題を確認したい」

**やり取り例**

- ユーザ: 「今のP0課題と設計方針をまとめて教えて」
- AIエージェント: `context_search` で横断検索し、Summary View を提示

#### 2.3.2 詳細の取得

`context_search` の結果から必要なIDを選び、以下で全文を取得します。

- Stock: `stock_manage` の `read`
- State: `state_manage` の `read`

## 5. 最小運用ルール（推奨）

- P0は「プロダクトの北極星」だけに絞る
- 設計情報は **概要→詳細** の順で Stock に積み上げる
- State は「短いタイトル + 具体的な説明 + 完了条件」を記述する
- `context_search` で概要を集め、必要なときだけ `read` する

## 6. 典型フロー（まとめ）

1. MCPクライアントから `pim-server` を起動
2. `stock_manage` でゴール・設計・ルールを登録
3. `state_manage` でタスク・課題を作成/更新
4. `context_search` で必要情報を集約
5. `read` で必要な詳細のみ取得
