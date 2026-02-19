# cfl - TODO

## Phase 0: プロジェクト基盤

- [x] go mod init
- [x] mise.toml（ツールバージョン + タスク定義）
- [x] .golangci.yml
- [x] lefthook.yml
- [x] .gitignore
- [x] main.go（最小限のエントリーポイント）
- [x] mise run all が通ることを確認

## Phase 1: コア機能（MVP）

### 初期化

- [x] `cfl init`
  - [x] IF 未初期化 THEN 設定ファイル/設定ディレクトリを作成する (`config init`から移譲)
  - [x] IF 既に初期化済み THEN 何も破壊的変更をしない
  - [x] IF 作成に成功 THEN 成功メッセージを出して終了する（exit=0）
  - [x] IF 作成に失敗 THEN エラーメッセージを出して終了する（exit!=0）

### 設定管理

- [x] `cfl config init` — 対話式で新しい profile を作成
- [x] `cfl config list` — profile 一覧表示
- [x] `cfl config show` — 現在の profile 詳細表示
  - [x] `output` の表示を追加
- [x] `cfl config delete <name>` — profile 削除

### プロファイル切り替え

- [x] `cfl use` — インタラクティブに profile 選択
- [x] `cfl use <name>` — 指定 profile に切り替え

### ページ操作

- [x] `cfl page list [--space-id ID] [--limit N]`
  - [x] `list` `--space-id` `--limit`
  - [ ] `--cursor` ページング
  - [ ] `--status` ページ状態フィルタ
  - [ ] `--sort` 並び順
- [ ] `cfl page get <page-id>`
- [ ] `cfl page create --title TITLE --body-file FILE [--parent-id ID]`
- [ ] `cfl page update <page-id> --title TITLE --body-file FILE --version N`
- [ ] `cfl page delete <page-id>`

## Phase 2: 拡張機能

### ページ補助操作

- [ ] `cfl page children <page-id>` — 子ページ一覧
- [ ] `cfl page label list <page-id>` — ラベル一覧
- [ ] `cfl page label add <page-id> --name <label>` — ラベル追加
- [ ] `cfl page label remove <page-id> --name <label>` — ラベル削除

### フォルダ操作

- [ ] `cfl folder list [--space-id ID]`
- [ ] `cfl folder get <folder-id>`
- [ ] `cfl folder create --name NAME [--parent-id ID] [--space-id ID]`
- [ ] `cfl folder update <folder-id> --name NAME`
- [ ] `cfl folder delete <folder-id>`
- [ ] `cfl folder children <folder-id>`

### 添付ファイル

- [ ] `cfl attachment list --page-id <page-id>`
- [ ] `cfl attachment upload --page-id <page-id> --file <path>`
- [ ] `cfl attachment download <attachment-id> --output <path>`
- [ ] `cfl attachment delete <attachment-id>`

### ブログ記事

- [ ] `cfl blog list [--space-id ID]`
- [ ] `cfl blog get <blog-id>`
- [ ] `cfl blog create --space-id ID --title TITLE --body-file FILE`
- [ ] `cfl blog update <blog-id> --title TITLE --body-file FILE --version N`
- [ ] `cfl blog delete <blog-id>`

### コメント

- [ ] `cfl comment list --page-id <page-id>`
- [ ] `cfl comment create --page-id <page-id> --body "テキスト"`
- [ ] `cfl comment delete <comment-id>`

### リッチ表記

- [ ] Markdown → Confluence storage format (XHTML) 変換
- [ ] Markdown → ADF (Atlassian Document Format) 変換
- [ ] 画像・テーブル等のリッチ表現対応

### git 同期

- [ ] `cfl sync` — git リポジトリ内の Markdown ファイルと Confluence ページの双方向同期
