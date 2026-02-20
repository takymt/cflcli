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
  - [x] domain, email, space key, output は default プロファイルを参照する
  - [x] Default space keyという表記はやめて Space Key にする
  - [x] Profile name -> Profile Name, Confluence domain -> Domain, Email Address -> Email, Output format -> Output
  - [x] `cfl config init <name>` の場合は Profile Name を `<name>` なものとして設定を省略する
  - [x] `cfl config edit <name>` の場合は、デフォルト値が `<name>` プロファイルな `cfl config init` 相当の挙動とする
- [x] `cfl config list` — profile 一覧表示
- [x] `cfl config show` — 現在の profile 詳細表示
  - [x] `output` の表示を追加
- [x] `cfl config delete <name>` — profile 削除
  - [x] current の削除はエラー
  - [x] `--force` オプション時は current 削除可能

### プロファイル切り替え

- [x] `cfl use` — インタラクティブに profile 選択
- [x] `cfl use <name>` — 指定 profile に切り替え

### ページ操作

- [x] `cfl page list [--space-id ID] [--limit N]`
  - [x] `list` `--space-id` `--limit`
  - [x] `--cursor` ページング
    - [x] IF `--cursor` が未指定 THEN `/pages` リクエストに `cursor` クエリを付与しない
    - [x] IF `--cursor` が指定される THEN `/pages` リクエストに `cursor=<value>` を付与する
    - [x] IF `--output json` THEN 返却された `next` をそのまま出力し次回の `--cursor` に利用できる
    - [x] IF `--cursor` が無効または期限切れで API が 4xx を返す THEN 再取得が必要だと分かるエラーを表示して終了する
  - [x] `--status` ページ状態フィルタ
    - [x] IF `--status` が未指定 THEN `status=current` のみを付与して取得する
    - [x] IF `--status` が指定される THEN カンマ区切りの値を分割し `status` クエリを複数付与する
    - [x] IF `--status` が許可値以外または空要素を含む THEN バリデーションエラーにする
      - 許可値: `current`, `archived`, `deleted`, `trashed`
    - [x] IF `--status` を明示指定して `--output table` THEN `STATUS` 列を表示する
    - [x] IF `--status` 未指定で `--output table` THEN `STATUS` 列を表示しない
  - [x] `--sort` 並び順
    - [x] IF `--sort` が未指定 THEN `/pages` リクエストに `sort` クエリを付与しない
    - [x] IF `--sort` が指定される THEN 指定値を `sort` クエリとして付与する
    - [x] IF `--sort` が許可値以外 THEN バリデーションエラーにする
      - 許可値: `id`, `-id`, `created-date`, `-created-date`, `modified-date`, `-modified-date`, `title`, `-title`
- [x] `cfl page get <page-id>`
- [x] `cfl page create --title TITLE --body-file FILE [--parent-id ID]`
- [x] `cfl page update <page-id> --title TITLE --body-file FILE`（versionは自動解決）
- [x] `cfl page delete <page-id>`

## Phase 1.1: Markdown 変換品質改善

### Markdown -> Confluence storage（`--body-format markdown`）

- [x] 変換パイプラインを固定する
  - [x] 前処理 -> Markdown(GFM) -> HTML->storage -> edit互換後処理
- [x] 見出し `#` `##` `###` `####` を正しく変換する
- [x] 箇条書き（ネスト含む）を edit/view で同等表示にする
  - [x] list 項目間や親子 list 前後で不要な soft 改行を入れない
- [x] 番号付きリストを正しく変換する
- [x] リンク `[text](url)` を Confluence storage link へ変換する
- [x] 画像 `![alt](url)` を Confluence storage image へ変換する
- [x] タスクリストを `<ac:task-list>` で厳密生成する
  - [x] 許可記法は `- [ ]` と `- [x]` のみ
  - [x] `- [×]` はタスクリストとして扱わない
- [x] 引用（`>` とネスト引用 `>>`）を保持する
- [x] 区切り線 `---` を変換する
- [x] 強調記法を変換する（italic, bold, strike, inline code, escape）
- [x] fenced code block を Confluence code macro に変換する
  - [x] language 未指定時は `text`
  - [x] code block 末尾の余計な改行を除去して edit 上の行数ずれを防ぐ
- [x] URL 単独行をリンクカード（block card）として変換する
- [x] `:emoji_id:` は Confluence 絵文字セット準拠で変換する
- [x] 折りたたみ（expand macro）をサポートする
  - [x] storage の初期状態は未展開（collapsed）
- [x] underline は Markdown 独自記法を追加しない
  - [x] 生 HTML の `<u>...</u>` は許容する
- [x] 変換の回帰テストを追加する（`internal/body`）

## Phase 1.2: Frontmatter 対応（最小）

### Frontmatter -> create/update 連携

- [x] Markdown frontmatter の `title` を解釈する
  - [x] `--title` 未指定時は frontmatter `title` を採用する
  - [x] `--title` 指定時は frontmatter `title` より優先する
  - [x] frontmatter ブロックは本文から除外して storage 変換する
  - [x] frontmatter の終端 `---` が無い場合はエラーにする
- [x] `cfl page create` に frontmatter `title` を適用する
- [x] `cfl page update` に frontmatter `title` を適用する
- [x] create/update のテストを追加する（`cmd/page_test.go`）

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
