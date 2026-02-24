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
  - [x] `--title` と frontmatter `title` の同時指定はエラーにする
  - [x] frontmatter ブロックは本文から除外して storage 変換する
  - [x] frontmatter の終端 `---` が無い場合はエラーにする
- [x] `cfl page create` に frontmatter `title` を適用する
- [x] `cfl page update` に frontmatter `title` を適用する
- [x] create/update のテストを追加する（`cmd/page_test.go`）

## Phase 1.3: ローカル画像解決と Mermaid 画像化

### create/update の Markdown アセット解決

- [x] `cfl page create/update` の `--body-format markdown` にアセット解決ステップを追加する
  - [x] `--body-format storage` は既存どおり変換対象外にする
- [x] `--assets-root` オプションを追加する（default: `--body-file` のディレクトリ）
  - [x] IF 画像パスが `http://` または `https://` THEN 既存どおり URL 画像として扱う
  - [x] IF 画像パスが `./` `../` または bare path THEN `--body-file` のディレクトリ基準で解決する
  - [x] IF 画像パスが `/` 始まり THEN OS ルートではなく `--assets-root` 基準で解決する
  - [x] IF 解決先ファイルが存在しない THEN エラーで終了する

### ローカル画像 -> Confluence 添付

- [x] ローカル画像を page 添付へアップロードできるようにする（Confluence REST API v1 attachments）
- [x] 本文は URL 画像ではなく `ri:attachment` 参照へ変換する
- [x] 添付ファイル名は元ファイル名（basename）を維持する
  - [x] IF 同一 Markdown 内で basename が衝突する THEN エラーで終了する
  - [x] IF 既存添付に同名がある THEN 新規追加ではなく添付の新バージョン更新として扱う

### Mermaid 画像化

- [x] mermaid fenced code block を画像化する（内蔵レンダラ）
- [x] `cfl page create/update` に `--no-render-mermaid` オプションを追加する
  - [x] デフォルトでは mermaid 画像化を有効にする
  - [x] IF `--no-render-mermaid` 指定時 THEN mermaid は code block として保持する
- [x] 生成画像を page 添付にアップロードし、本文を `ri:attachment` 参照へ変換する

## Phase 1.4: migrate export と冪等性

### migrate export（Git SSOT 移行）

- [x] `cfl migrate export` を追加する
  - [x] `--space-id` `--space-key` `--root-page-id` `--out` `--attachments-dir` をサポートする
  - [x] `--space-id` と `--space-key` は排他指定にする
  - [x] ページツリーを Markdown + frontmatter（`page-id`, `title`, `parent-id`, `space-key`）で出力する
- [x] 添付ファイルの保存先を migrate 専用サブディレクトリにする
  - [x] default は `attachments/_migrate` にする
  - [x] `--attachments-dir` で保存先を変更可能にする
- [x] `ri:attachment` は一律 `attachments` 配下の Markdown 画像リンクへ変換する
  - [x] 元の Markdown 画像パス復元は行わない

### Macro 方針（migrate）

- [x] Mermaid macro は ```mermaid``` fenced code block に変換する
- [x] 未対応 macro は削除せず HTML コメントとして残す
  - [x] macro 名と raw storage を追跡できる形式にする

### 変換冪等性

- [x] `markdown -> storage -> markdown -> storage` の冪等性を検討する
- [x] 冪等性検証用の受け入れケースを定義する

## Phase 2: 拡張機能

### CLI UX

- [x] `config init/edit` の対話プロンプトに `Domain` / `Assets Root` の例値を表示する

### 配布・インストール

- [x] `go install github.com/takymt/cflcli/cmd/cfl@latest` で `cfl` バイナリ名でインストールできるようにする

### migrate export 品質改善

- [x] `StorageToMarkdown` で基本的な HTML（見出し/リスト/リンク/強調/引用/区切り線/タスクリスト）を Markdown 記法へ変換する
- [x] Confluence `code` macro（mermaid 以外）を fenced code block に変換する
- [x] `:::` 系（details/info/success/memo/warn/error）の roundtrip 冪等性を characterization test で可視化する
- [x] `expand/info/tip/note/warning` と ADF panel(note) を `:::` DSL（details/info/success/memo/warn）へ逆変換する

### migrate export UX / パフォーマンス改善

#### 実時間短縮

- [ ] 添付メタデータ取得をページ単位に集約して API 呼び出し回数を削減する
  - [ ] 現状の filename ごとの添付一覧取得をやめ、ページ単位の添付一覧から `filename -> downloadURL` を引けるようにする
- [ ] `cfl migrate export` のページ処理を並列化する（worker pool）
  - [ ] `--concurrency` オプションを追加する（default は安全寄り）
  - [ ] 出力ファイル内容と最終結果の順序は安定化する
- [ ] 添付ダウンロードを並列化する
  - [ ] `--attachment-concurrency` オプションを追加する
  - [ ] レート制限を考慮して過剰並列を避ける
- [ ] 一時的な API エラー（429 / 5xx）に対する retry/backoff を導入する
- [ ] 再実行短縮のための `--resume` または `--skip-existing` を検討・導入する
- [ ] 大きい添付でのメモリ負荷を下げるため、添付ダウンロードのストリーム書き込みを検討する

#### 待機負荷軽減（体感 UX）

- [ ] `migrate export` 実行中の進捗を `stderr` に表示する（`stdout` は最終結果専用）
  - [ ] IF `--output json` THEN 進捗は `stderr` のみに出力する
- [ ] フェーズ別進捗を表示する（例: page list / plan build / page export / attachment download）
- [ ] ページ進捗を表示する（件数カウンタ + 現在の page id/title/path）
- [ ] 長時間無出力を避けるため、定期 heartbeat を表示する
- [ ] 可能なら ETA / throughput（pages/sec, attachments/sec）を表示する
- [ ] 進捗表示モードを切り替えられるようにする（例: `--progress auto|plain|off`）
- [ ] 実行前確認用の `--plan-only` / `--dry-run` を検討する（対象件数と出力先の確認）
- [ ] 最終サマリにフェーズ別所要時間と件数を表示する

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

### コメント

- [ ] `cfl comment list --page-id <page-id>`
- [ ] `cfl comment create --page-id <page-id> --body "テキスト"`
- [ ] `cfl comment delete <comment-id>`

### git 同期

- [ ] `cfl sync` — git リポジトリ内の Markdown ファイルと Confluence ページの双方向同期
