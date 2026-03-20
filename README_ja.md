# TokMan 🌸

**世界最先軽のトークン削減システム** — 14層の研究ベース圧縮パイプラインで95-99%のトークン削減を実現。

## 圧縮性能

| 入力サイズ | オリジナル | 最終 | 削減率 |
|------------|-----------|------|--------|
| 小 (100行) | 982トークン | 44トークン | **95.5%** |
| 中 (1000行) | 9,737トークン | 52トークン | **99.5%** |
| 大 (5000行) | 49,437トークン | 63トークン | **99.9%** |

## 機能

- 🧠 **14層圧縮パイプライン** — 研究ベースのトークン削減 (95-99%)
- 🔧 **Gitコマンド** — `status`, `diff`, `log`, `add`, `commit`, `push`をフィルタリング
- 🐳 **インフラ** — Docker、kubectl、AWS CLIのコンパクトな出力
- 📦 **パッケージマネージャ** — npm、pnpm、pip、cargoをコンパクトに
- 🧪 **テスト** — Go、pytest、vitest、jest、playwrightの結果を集約
- 📊 **トークン追跡** — SQLiteベースのトークン節約メトリクス
- 🔄 **シェル統合** — フックによる自動コマンド書き換え
- 💰 **経済分析** — 支出と節約の比較

## インストール

```bash
git clone https://github.com/GrayCodeAI/tokman.git
cd tokman
go build -o tokman ./cmd/tokman
sudo mv tokman /usr/local/bin/
```

## クイックスタート

```bash
# TokManを初期化
tokman init

# トークン節約を確認
tokman status

# 包括的な分析
tokman gain

# ラップされたコマンドを使用
tokman git status
tokman ls
tokman go test ./...
```

## 使用例

### Git Status (77%削減)
```bash
$ tokman git status
🌿 main (origin/main)
📝 M internal/filter/pipeline.go
📝 M internal/filter/h2o.go
❓ internal/filter/stream.go
```

### Docker PS (83%削減)
```bash
$ tokman docker ps
🐳 nginx:latest    → web-server   (2h)  0.0.0.0:80
🐳 redis:alpine    → cache-server (3h)  0.0.0.0:6379
```

## 14層圧縮パイプライン

| 層 | 名前 | 研究 | 圧縮率 |
|----|------|------|--------|
| 1 | エントロピーフィルタ | Selective Context (Mila 2023) | 2-3x |
| 2 | パープレキシティプルーニング | LLMLingua (Microsoft 2023) | 20x |
| 3 | ゴール駆動選択 | SWE-Pruner (Shanghai 2025) | 14.8x |
| 4-9 | 研究ベース圧縮 | 複数の論文 | 4-30x |
| 10 | 予算 | 業界標準 | 保証 |
| 11-14 | 高度な圧縮 | MemGPT、ProCut、H2O | 30x+ |

## 主要コマンド

| コマンド | 説明 |
|---------|------|
| `tokman init` | 初期化とシェルフックのインストール |
| `tokman status` | 節約のクイックサマリー |
| `tokman gain` | グラフ付きの包括的な分析 |
| `tokman git status` | フィルタリングされたリポジトリの状態 |
| `tokman go test` | 結果を集約したGoテスト |
| `tokman docker ps` | コンパクトなDockerコンテナ |
| `tokman discover` | 見逃した節約の発見 |

## ライセンス

MIT License — [LICENSE](LICENSE)を参照。
