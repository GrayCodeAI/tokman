# TokMan 🌸

**Système Avancé de Réduction de Tokens** — Pipeline de compression en 14 couches avec réduction de 95-99%.

## Performance de Compression

| Entrée | Original | Final | Réduction |
|--------|----------|-------|-----------|
| Petite (100 lignes) | 982 tokens | 44 tokens | **95.5%** |
| Moyenne (1000 lignes) | 9,737 tokens | 52 tokens | **99.5%** |
| Grande (5000 lignes) | 49,437 tokens | 63 tokens | **99.9%** |

## Fonctionnalités

- 🧠 **Pipeline en 14 couches** — Réduction de tokens basée sur la recherche (95-99%)
- 🔧 **Commandes Git** — `status`, `diff`, `log`, `add`, `commit`, `push` filtrés
- 🐳 **Infrastructure** — Docker, kubectl, AWS CLI avec sortie compacte
- 📦 **Gestionnaires de paquets** — npm, pnpm, pip, cargo compacts
- 🧪 **Tests** — Go, pytest, vitest, jest, playwright avec résultats agrégés
- 📊 **Suivi des tokens** — Métriques SQLite sur les tokens économisés
- 🔄 **Intégration shell** — Réécriture automatique des commandes via hooks
- 💰 **Analyse économique** — Comparaison dépenses vs économies

## Installation

```bash
git clone https://github.com/GrayCodeAI/tokman.git
cd tokman
go build -o tokman ./cmd/tokman
sudo mv tokman /usr/local/bin/
```

## Démarrage Rapide

```bash
# Initialiser TokMan
tokman init

# Voir les économies de tokens
tokman status

# Analyse complète
tokman gain

# Utiliser les commandes enveloppées
tokman git status
tokman ls
tokman go test ./...
```

## Exemples

### Git Status (réduction 77%)
```bash
$ tokman git status
🌿 main (origin/main)
📝 M internal/filter/pipeline.go
📝 M internal/filter/h2o.go
❓ internal/filter/stream.go
```

### Docker PS (réduction 83%)
```bash
$ tokman docker ps
🐳 nginx:latest    → web-server   (2h)  0.0.0.0:80
🐳 redis:alpine    → cache-server (3h)  0.0.0.0:6379
```

## Pipeline en 14 Couches

| Couche | Nom | Recherche | Compression |
|--------|-----|-----------|-------------|
| 1 | Filtrage d'Entropie | Selective Context (Mila 2023) | 2-3x |
| 2 | Élagage de Perplexité | LLMLingua (Microsoft 2023) | 20x |
| 3 | Sélection par Objectif | SWE-Pruner (Shanghai 2025) | 14.8x |
| 4-9 | Compression avancée | Divers papers | 4-30x |
| 10 | Budget | Standard industriel | Garanti |
| 11-14 | Compression avancée | MemGPT, ProCut, H2O | 30x+ |

## Commandes Principales

| Commande | Description |
|----------|-------------|
| `tokman init` | Initialiser et installer le hook shell |
| `tokman status` | Résumé rapide des économies |
| `tokman gain` | Analyse complète avec graphiques |
| `tokman git status` | État du dépôt filtré |
| `tokman go test` | Tests Go avec résultats agrégés |
| `tokman docker ps` | Conteneurs Docker compacts |
| `tokman discover` | Trouver les économies manquées |

## Licence

MIT License — voir [LICENSE](LICENSE).
