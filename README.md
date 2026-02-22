> [!WARNING]
> この README は暫定版（AI 生成）です。まずは「使い始められること」を優先して整理しています。後で人間が内容・表現を整備しますので、少し荒い部分があってもいったんご容赦ください。

# cfl

`cfl` は Confluence Cloud 向けの CLI です。Confluence REST API v2 を中心に、日常運用で使うページ操作と、Markdown ベースの更新フローを扱いやすくすることを目的にしています。

## まず何ができる？

現時点で主にできること:

- プロファイル管理（複数環境の切り替え）
- ページ一覧 / 取得 / 作成 / 更新 / 削除
- Markdown から Confluence storage format への変換（`create` / `update`）
- Markdown frontmatter（`title`, `parent-id`）の取り込み
- Markdown 内のローカル画像を Confluence 添付へアップロード
- Mermaid fenced code block の画像化（デフォルト有効）
- Confluence ページを Markdown + 添付ファイルとしてエクスポート（`migrate export`）
- 出力形式の切り替え（`table` / `json`）

## 最短スタート（Quick Start）

### 1. ビルド

```bash
go build -o cfl .
```

`mise` を使う場合:

```bash
mise run build
```

### 2. API トークンを環境変数に設定

`cfl` は Basic Auth を使います。

- ユーザー: profile に設定したメールアドレス
- パスワード: `CFL_API_TOKEN`

```bash
export CFL_API_TOKEN="your_confluence_api_token"
```

### 3. 初期設定（最初の profile を作る）

```bash
./cfl init
```

`cfl init` は未初期化時に `default` profile を対話式で作成します。

### 4. （任意）profile を追加・切り替え

```bash
./cfl config init work
./cfl config init personal
./cfl use work
```

`cfl config use work` でも切り替えできます。

### 5. まず一覧を取る

profile に `space_key` を設定していれば `--space-key` を省略できます。

```bash
./cfl page list --limit 25
```

明示的に指定する場合:

```bash
./cfl page list --space-key TEST --limit 25
```

## よく使う操作（ユーザー導線）

### 1. ページを探す（一覧・絞り込み・ページング）

```bash
./cfl page list --space-key TEST --status current,archived --sort -modified-date --limit 50
```

- `--status`: `current`, `archived`, `deleted`, `trashed`（カンマ区切り）
- `--sort`: `id`, `-id`, `created-date`, `-created-date`, `modified-date`, `-modified-date`, `title`, `-title`
- `--cursor`: 前回結果の `next` を使って続き取得

`json` 出力で `next` を取りたい場合:

```bash
./cfl -o json page list --space-key TEST
```

### 2. ページ本文（storage format）を取得する

```bash
./cfl page get 123456 > page.storage.xhtml
```

`page get` は Confluence storage format を標準出力へそのまま出します（`-o json/table` は使いません）。

### 3. Markdown からページを作成する

```bash
./cfl page create \
  --space-key TEST \
  --title "Release Notes" \
  --body-file ./docs/release-notes.md
```

- `--body-format` のデフォルトは `markdown`
- `--body-format storage` を指定すると storage format をそのまま送信
- Markdown 内のローカル画像は添付ファイルとしてアップロードされ、本文は `ri:attachment` 参照に変換
- Mermaid fenced code block はデフォルトで画像化（`--no-render-mermaid` で無効化）

### 4. 既存ページを更新する

```bash
./cfl page update 123456 \
  --title "Release Notes v2" \
  --body-file ./docs/release-notes.md
```

- ページバージョン番号は自動で解決します
- 同時更新衝突（version conflict）時は再取得してリトライが必要です

### 5. ページを削除する

```bash
./cfl page delete 123456
```

### 6. Confluence から Markdown にエクスポートする（移行用）

スペース全体:

```bash
./cfl migrate export --space-key TEST --out ./export
```

特定ページ配下のサブツリーのみ:

```bash
./cfl migrate export \
  --space-key TEST \
  --root-page-id 123456 \
  --out ./export
```

- 出力は Markdown + frontmatter（`page-id`, `title`, `parent-id`, `space-key`）
- 添付ファイルは既定で `attachments/_migrate` 配下に保存
- `--attachments-dir` で変更可能

## Markdown 運用のポイント

### Frontmatter（`create` / `update`）

Markdown の先頭に frontmatter を置くと、`--title` / `--parent-id` の代わりに使えます。

```markdown
---
title: Weekly Update
parent-id: "123456"
---

# Summary

- Item 1
- Item 2
```

挙動:

- `--title` 未指定時は frontmatter の `title` を使用
- `--parent-id` 未指定時は frontmatter の `parent-id` を使用
- フラグと frontmatter の同時指定はエラー
- frontmatter ブロックは本文変換前に取り除かれます
- `parent-id`, `parent_id`, `parentid` を受け付けます

### ローカル画像と `--assets-root`

Markdown 内の画像パス解決ルール（`--body-format markdown` のとき）:

- `http://` / `https://`: 外部 URL として扱う
- `./` / `../` / bare path: `--body-file` のあるディレクトリ基準
- `/` 始まり: OS ルートではなく `--assets-root` 基準

profile に `assets_root` を設定しておくと、`--assets-root` の省略時に使われます。

## 設定ファイル（profiles）

設定ファイルの場所:

- `$XDG_CONFIG_HOME/cflcli/config.toml`
- 既定: `~/.config/cflcli/config.toml`

例:

```toml
current = "work"

[[profiles]]
name = "work"
domain = "your-domain.atlassian.net"
user = "you@example.com"
space_key = "TEST"
assets_root = "/Users/you/docs"
output = "table"

[[profiles]]
name = "personal"
domain = "my-site.atlassian.net"
user = "me@example.com"
space_key = "DEV"
output = "json"
```

主な管理コマンド:

```bash
./cfl config init [name]    # 対話式で作成
./cfl config edit <name>    # 対話式で編集
./cfl config list           # 一覧
./cfl config show           # 現在の profile を表示
./cfl config delete <name>  # 削除
./cfl use [name]            # 切り替え（引数なしで対話式）
```

## 出力形式

グローバルフラグ:

- `-o, --output`: `table`（既定） / `json`
- `-p, --profile`: 一時的に profile を切り替え
- `-v, --verbose`: 詳細出力

例:

```bash
./cfl -o json page list --space-key TEST
```

注意:

- `page get` は storage format 本文をそのまま出力するコマンドで、`--output` の影響を受けません

## コマンド一覧（現時点）

```text
cfl
├── init
├── config
│   ├── init / edit / use / list / show / delete
├── use
├── page
│   ├── list / get / create / update / delete
└── migrate
    └── export
```

未実装/今後の予定（labels, folder, attachment, comment, sync など）は `TODO.md` を参照してください。

## 開発者向け

### 必要ツール

- Go `1.26.0`（`go.mod` / `mise.toml` 準拠）
- `mise`（任意）

### よく使うコマンド

```bash
mise run all        # build + fmt + lint + test
mise run fmt
mise run lint
mise run test
mise run test-it    # integration tests (build tag)
mise run test-live  # live Confluence tests (CFL_LIVE=1)
mise run cover
mise run build
```

テスト方針の詳細は `TESTING.md` を参照してください。

## API / 実装メモ

- ベース: Confluence Cloud REST API v2
- 添付ファイルの一部操作は v1 endpoint を利用（ローカル画像アップロードのため）
- ページングは cursor ベース（`limit` + `cursor` / レスポンスの `next`）
