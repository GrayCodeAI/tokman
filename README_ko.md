# TokMan 🌸

**세계 최고의 토큰 절감 시스템** — 14개 연구 기반 압축 파이프라인으로 95-99% 토큰 절감.

## 압축 성능

| 입력 크기 | 원본 | 최종 | 절감 |
|-----------|------|------|------|
| 소규모 (100줄) | 982 토큰 | 44 토큰 | **95.5%** |
| 중간 (1000줄) | 9,737 토큰 | 52 토큰 | **99.5%** |
| 대규모 (5000줄) | 49,437 토큰 | 63 토큰 | **99.9%** |

## 기능

- 🧠 **14개 압축 파이프라인** — 연구 기반 토큰 절감 (95-99%)
- 🔧 **Git 명령어** — `status`, `diff`, `log`, `add`, `commit`, `push` 필터링
- 🐳 **인프라** — Docker, kubectl, AWS CLI의 간결한 출력
- 📦 **패키지 관리자** — npm, pnpm, pip, cargo 압축
- 🧪 **테스트** — Go, pytest, vitest, jest, playwright 결과 집계
- 📊 **토큰 추적** — SQLite 기반 절감 메트릭
- 🔄 **셸 통합** — 훅을 통한 자동 명령어 재작성
- 💰 **경제 분석** — 지출 vs 절감 비교

## 설치

```bash
git clone https://github.com/GrayCodeAI/tokman.git
cd tokman
go build -o tokman ./cmd/tokman
sudo mv tokman /usr/local/bin/
```

## 빠른 시작

```bash
# TokMan 초기화
tokman init

# 토큰 절감 확인
tokman status

# 포괄적 분석
tokman gain

# 래핑된 명령어 사용
tokman git status
tokman ls
tokman go test ./...
```

## 사용 예시

### Git Status (77% 절감)
```bash
$ tokman git status
🌿 main (origin/main)
📝 M internal/filter/pipeline.go
📝 M internal/filter/h2o.go
❓ internal/filter/stream.go
```

### Docker PS (83% 절감)
```bash
$ tokman docker ps
🐳 nginx:latest    → web-server   (2h)  0.0.0.0:80
🐳 redis:alpine    → cache-server (3h)  0.0.0.0:6379
```

## 14개 압축 파이프라인

| 레이어 | 이름 | 연구 | 압축 |
|--------|------|------|------|
| 1 | 엔트로피 필터 | Selective Context (Mila 2023) | 2-3x |
| 2 | 퍼플렉시티 프루닝 | LLMLingua (Microsoft 2023) | 20x |
| 3 | 목표 기반 선택 | SWE-Pruner (Shanghai 2025) | 14.8x |
| 4-9 | 연구 기반 압축 | 여러 논문 | 4-30x |
| 10 | 예산 | 산업 표준 | 보장 |
| 11-14 | 고급 압축 | MemGPT, ProCut, H2O | 30x+ |

## 주요 명령어

| 명령어 | 설명 |
|--------|------|
| `tokman init` | 초기화 및 셸 훅 설치 |
| `tokman status` | 절감 요약 |
| `tokman gain` | 그래프가 포함된 포괄적 분석 |
| `tokman git status` | 필터링된 저장소 상태 |
| `tokman go test` | 결과를 집계한 Go 테스트 |
| `tokman docker ps` | 간결한 Docker 컨테이너 |
| `tokman discover` | 놓친 절감 기회 찾기 |

## 라이선스

MIT License — [LICENSE](LICENSE) 참조.
