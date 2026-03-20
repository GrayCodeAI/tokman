# TokMan 🌸

**世界最先进的令牌压缩系统** — 14层研究级压缩流水线，实现95-99%的令牌节省。

## 压缩性能

| 输入大小 | 原始 | 最终 | 节省 |
|----------|------|------|------|
| 小型 (100行) | 982 令牌 | 44 令牌 | **95.5%** |
| 中型 (1000行) | 9,737 令牌 | 52 令牌 | **99.5%** |
| 大型 (5000行) | 49,437 令牌 | 63 令牌 | **99.9%** |

## 功能

- 🧠 **14层压缩流水线** — 研究级令牌压缩 (95-99%)
- 🔧 **Git命令** — 过滤的 `status`, `diff`, `log`, `add`, `commit`, `push`
- 🐳 **基础设施** — Docker、kubectl、AWS CLI 紧凑输出
- 📦 **包管理器** — npm、pnpm、pip、cargo 压缩
- 🧪 **测试** — Go、pytest、vitest、jest、playwright 结果聚合
- 📊 **令牌追踪** — SQLite 数据库记录节省指标
- 🔄 **Shell集成** — 通过钩子自动重写命令
- 💰 **经济分析** — 支出与节省对比

## 安装

```bash
git clone https://github.com/GrayCodeAI/tokman.git
cd tokman
go build -o tokman ./cmd/tokman
sudo mv tokman /usr/local/bin/
```

## 快速开始

```bash
# 初始化 TokMan
tokman init

# 查看令牌节省
tokman status

# 完整分析
tokman gain

# 使用封装的命令
tokman git status
tokman ls
tokman go test ./...
```

## 使用示例

### Git Status (节省77%)
```bash
$ tokman git status
🌿 main (origin/main)
📝 M internal/filter/pipeline.go
📝 M internal/filter/h2o.go
❓ internal/filter/stream.go
```

### Docker PS (节省83%)
```bash
$ tokman docker ps
🐳 nginx:latest    → web-server   (2h)  0.0.0.0:80
🐳 redis:alpine    → cache-server (3h)  0.0.0.0:6379
```

## 14层压缩流水线

| 层 | 名称 | 研究 | 压缩 |
|----|------|------|------|
| 1 | 熵过滤 | Selective Context (Mila 2023) | 2-3x |
| 2 | 困惑度修剪 | LLMLingua (Microsoft 2023) | 20x |
| 3 | 目标驱动选择 | SWE-Pruner (Shanghai 2025) | 14.8x |
| 4-9 | 研究级压缩 | 多篇论文 | 4-30x |
| 10 | 预算 | 行业标准 | 保证 |
| 11-14 | 高级压缩 | MemGPT、ProCut、H2O | 30x+ |

## 主要命令

| 命令 | 说明 |
|------|------|
| `tokman init` | 初始化并安装 Shell 钩子 |
| `tokman status` | 快速查看节省摘要 |
| `tokman gain` | 包含图表的完整分析 |
| `tokman git status` | 过滤的仓库状态 |
| `tokman go test` | 结果聚合的 Go 测试 |
| `tokman docker ps` | 紧凑的 Docker 容器列表 |
| `tokman discover` | 发现错过的节省机会 |

## 许可证

MIT License — 参见 [LICENSE](LICENSE)。
