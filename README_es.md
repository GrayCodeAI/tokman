# TokMan 🌸

**Sistema Avanzado de Reducción de Tokens** — Pipeline de compresión de 14 capas con reducción de tokens del 95-99%.

## Rendimiento de Compresión

| Entrada | Original | Final | Reducción |
|---------|----------|-------|-----------|
| Pequeña (100 líneas) | 982 tokens | 44 tokens | **95.5%** |
| Mediana (1000 líneas) | 9,737 tokens | 52 tokens | **99.5%** |
| Grande (5000 líneas) | 49,437 tokens | 63 tokens | **99.9%** |

## Características

- 🧠 **Pipeline de 14 capas** — Reducción de tokens basada en investigación (95-99%)
- 🔧 **Comandos Git** — `status`, `diff`, `log`, `add`, `commit`, `push` filtrados
- 🐳 **Infraestructura** — Docker, kubectl, AWS CLI con salida compacta
- 📦 **Gestores de paquetes** — npm, pnpm, pip, cargo compactos
- 🧪 **Tests** — Go, pytest, vitest, jest, playwright con resultados agregados
- 📊 **Seguimiento de tokens** — Métricas SQLite sobre tokens ahorrados
- 🔄 **Integración shell** — Reescritura automática de comandos vía hooks
- 💰 **Análisis económico** — Comparación de gasto vs ahorro

## Instalación

```bash
git clone https://github.com/GrayCodeAI/tokman.git
cd tokman
go build -o tokman ./cmd/tokman
sudo mv tokman /usr/local/bin/
```

## Inicio Rápido

```bash
# Inicializar TokMan
tokman init

# Ver ahorros de tokens
tokman status

# Análisis completo
tokman gain

# Usar comandos envueltos
tokman git status
tokman ls
tokman go test ./...
```

## Ejemplos

### Git Status (77% reducción)
```bash
$ tokman git status
🌿 main (origin/main)
📝 M internal/filter/pipeline.go
📝 M internal/filter/h2o.go
❓ internal/filter/stream.go
```

### Docker PS (83% reducción)
```bash
$ tokman docker ps
🐳 nginx:latest    → web-server   (2h)  0.0.0.0:80
🐳 redis:alpine    → cache-server (3h)  0.0.0.0:6379
```

## Pipeline de 14 Capas

| Capa | Nombre | Investigación | Compresión |
|------|--------|---------------|------------|
| 1 | Filtro de Entropía | Selective Context (Mila 2023) | 2-3x |
| 2 | Poda de Perplejidad | LLMLingua (Microsoft 2023) | 20x |
| 3 | Selección por Objetivo | SWE-Pruner (Shanghai 2025) | 14.8x |
| 4 | Preservación AST | LongCodeZip (NUS 2025) | 4-8x |
| 5-9 | Compresión investigativa | Varios papers | 4-30x |
| 10 | Presupuesto | Estándar industrial | Garantizado |
| 11-14 | Compresión avanzada | MemGPT, ProCut, H2O | 30x+ |

## Comandos Principales

| Comando | Descripción |
|---------|-------------|
| `tokman init` | Inicializar e instalar hook shell |
| `tokman status` | Resumen rápido de ahorros |
| `tokman gain` | Análisis completo con gráficos |
| `tokman git status` | Estado del repositorio filtrado |
| `tokman go test` | Tests Go con resultados agregados |
| `tokman docker ps` | Contenedores Docker compactos |
| `tokman discover` | Encontrar ahorros perdidos |

## Licencia

MIT License — ver [LICENSE](LICENSE).
