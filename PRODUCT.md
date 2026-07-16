# Product Goals (No-PRD)

**Project**: gzh-cli-dev-env (library — `NewRootCmd()`, 바이너리 없음)
**Doc Type**: Goals + Constraints + Quality Gates
**Status**: Active — 추출 진행 중 (gzh-cli 기본 경로 아님)
**Last Updated**: 2026-07-16

______________________________________________________________________

## Product Intent

gzh-cli-dev-env **switches a developer's active cloud/container context** across
AWS · GCP · Azure · Docker · Kubernetes · SSH. It:

- reads status and switches context by shelling out to the vendor CLIs rather than
  embedding their SDKs (SOUL 신념 1),
- applies a named YAML environment (e.g. "production") across services in
  dependency order with rollback-on-error,
- and is an **in-progress extraction** from gzh-cli — not yet the default code path.

This is a feature-library project — a single PRODUCT.md is sufficient. It
replaces a PRD.

| 제공하는 것 (Is)                              | 되지 않을 것 (Is Not)                       |
| --------------------------------------------- | ------------------------------------------- |
| 6개 서비스의 활성 컨텍스트 조회·전환          | 리소스 프로비저닝 (생성·삭제)               |
| 의존 순서 기반 환경 일괄 전환·롤백            | 자격증명 저장·시크릿 관리                   |
| 벤더 CLI 셸아웃 (SDK 미도입)                  | 클라우드 SDK 래퍼                           |
| gzh-cli wrapper가 마운트하는 라이브러리       | 독립 실행 바이너리                          |

______________________________________________________________________

## Goals (Measurable Targets)

G1. **Service parity (조회 = 전환)**

- Target: 6개 서비스(aws·gcp·azure·docker·kubernetes·ssh) 모두 checker + switcher
  실동작
- 현재 checker **6/6**, switcher **5/6** — SSH switcher는 설정을 버리는 no-op
  스텁이며 `GetCurrentState`가 `"default"`를 하드코딩 반환한다

G2. **Buildable and testable**

- Target: `make build` · `make test` 통과
- 현재 **충족** (2026-07-16). 제거된 `GOEXPERIMENT=rangefunc` 설정과 본 리포에
  없는 `./cmd/gzh-git` 빌드 대상이 원인이었다. 라이브러리이므로 `go build ./...`로
  전환했다. CI test 잡 green
- `make lint`도 **충족** (2026-07-16). 같은 설정 오류가 golangci-lint의 패키지
  로딩까지 깨뜨려 린터가 0건을 보고하고 있었다 — 침묵이 통과로 오독되던 상태다.
  린터를 되살리자 드러난 111건을 해소했다 (오탐은 설정 정렬, 나머지는 수정).
  **CI 3개 잡(test·lint·vulncheck) 전부 green**

G3. **Concurrency safety**

- Target: `go test -race ./...` 통과
- 현재 **충족** (2026-07-16). `switchServicesParallel`의 goroutine들이 공유
  `previousStates` 맵과 result 슬라이스에 무방비로 쓰던 레이스를 `stateMu`로
  보호했다 (`es.mu`는 레지스트리 전용). `-race` 전체 통과, 병렬 테스트 `-count=50` 반복 통과

G4. **Dry-run honesty**

- Target: `--dry-run`은 어떤 부작용도 만들지 않는다
- 현재 **충족** (2026-07-16). 부작용 경로가 둘이었고 둘 다 막았다:
  1. **훅** — pre/post 셸 훅이 가드 없이 실행되어 "No changes will be made"가
     거짓이었다. `runHooks`로 모아 건너뛰고, 실행됐을 훅은 "would run"으로 보고한다
  2. **롤백** — `previousStates`는 dry-run에서도 채워지므로(가드가 `Switch()`에만
     있었다), 어떤 서비스가 실패하면 `RollbackOnError` 경로가 전환된 적 없는
     서비스에 실제 `Rollback()`을 걸었다. `GetCurrentState`가 벤더 CLI 셸아웃이라
     도달 가능한 경로였다. 가드를 `rollbackServices` 안으로 넣었다
- 두 경로 모두 회귀 테스트가 부작용을 직접 관측한다 (훅은 마커 파일, 롤백은
  Rollback 호출). 가드 없는 코드에서 실제로 FAIL함을 확인했다
- 설계 규칙: dry-run 가드는 **부작용을 내는 함수 자신이 소유한다**. 호출부에
  흩뿌리면 새 호출 지점에서 빠지고, 실제로 훅 수정 때 롤백 경로를 놓쳤다

G5. **Test reliability**

- Target: 커버리지 >= 80% (리포 CLAUDE.md의 자체 목표)
- 현재 **60.9%** — status 98.0% · environment 82.4%만 목표 충족;
  azure 40.0% · kubernetes 41.9% · ssh 48.9%, cmd/devenv 0%

______________________________________________________________________

## Non-Goals (Explicitly Out of Scope)

- No 독립 실행 바이너리 — `main` 패키지 없음, `NewRootCmd()`로 마운트된다
- No 자격증명 저장·시크릿 관리 — 상태를 읽을 뿐 자격증명을 보유하지 않는다
- No 프로비저닝 — 리소스를 만들지 않는다. *어떤* 기존 컨텍스트가 활성인지만 바꾼다
- No 벤더 SDK 도입 — `aws`/`gcloud`/`az`/`docker`/`kubectl` 셸아웃을 유지한다
- No `pkg/`에 CLI 전용 로직

______________________________________________________________________

## Guardrails and Technical Constraints

**Architecture**

- `ServiceChecker`(조회) / `ServiceSwitcher`(변경 + 롤백) 인터페이스로 서비스를 추상화
- 환경 전환은 의존 순서로 그룹을 나눠 그룹 내 병렬 실행

**Dependency Boundaries**

- `gzh-cli-core`만 의존 가능; 다른 feature 라이브러리 의존 금지 (GUIDELINES §2)
- 현재 core 미사용 (cobra·bubbletea 계열·yaml.v3만) — 클라우드 SDK 0건은 유지한다

**Compatibility**

- Go 1.25+ (`go.mod` go 1.25.7; devbox 툴체인 1.26)

**Safety**

- `--dry-run`, `[y/N]` 확인(`--force`로 생략), `RollbackOnError` 기본 활성
- 훅은 차단목록 + 30초 타임아웃 + 길이 제한으로 검증한다
- **미비**: 롤백은 **인메모리 전용**이라 전환 중 크래시 시 복구 아티팩트가 없다;
  훅 차단목록은 `sh -c` 위의 블록리스트라 우회 가능하다; `SwitchOptions.Force`는
  라이브러리에서 읽히지 않는 죽은 필드다
- **AWS 전환은 의미상 오작동 가능성이 높다** — `aws configure set profile X`는
  기본 프로필 블록에 `profile` 키를 쓸 뿐 활성 프로필을 바꾸지 않는다
  (활성 전환은 `AWS_PROFILE`/`--profile` 소관)

**Integration Status**

- **본 라이브러리는 gzh-cli의 기본 경로가 아니다.** `devenv_external` 빌드 태그가
  기본 off이며, 기본 `gz dev-env`는 gzh-cli 자체 `cmd/dev-env`가 서비스한다.
  본 리포는 3개 명령(status·tui·switch-all)만 제공하고 나머지(kubeconfig·aws-profile
  등)는 아직 gzh-cli에 남아 있다. wrapper 메타의
  `LifecycleStable` / `Version 1.0.0` 선언은 실제 상태(v0.1.0, 빌드 실패,
  태그 없음)와 모순되므로 신뢰 근거로 쓸 수 없다

**Baseline**

- GUIDELINES §3 베이스라인 충족 — `Makefile`·`.golangci.yml`(v2)·CI·`LICENSE`(MIT,
  소스의 SPDX 헤더 및 README 주장과 일치)·문서·본 PRODUCT.md 보유.
  CI 3개 잡(test·lint·vulncheck) 전부 green (G2)

______________________________________________________________________

## Quality Gates (Release Readiness)

**Build and Lint**

- `make build` · `make test` · `make lint` 통과 (G2 — 현재 충족)

**Concurrency**

- `go test -race ./...` 통과 (G3 — 현재 충족)

**Testing**

- 커버리지 >= 80% (현재 60.9%)

**Docs**

- 도움말·컨텍스트 문서가 실제 등록된 명령과 일치한다 (현재 충족: `status`·`tui`·`switch-all` only)

______________________________________________________________________

## Decision Rules

- **새 서비스는 checker + switcher + Rollback 실구현을 함께 포함해야 한다** —
  SSH 같은 no-op 스텁을 반복하지 않는다 (G1)
- **동시성 코드는 `-race` 통과 없이 머지될 수 없다** (G3)
- `--dry-run`을 광고하는 실행 경로는 부작용 0을 증명해야 한다 (G4).
  증명은 관측으로 한다 — 부작용을 실제로 관측하는 테스트가 가드 없는 코드에서
  FAIL해야 한다. 통과만 하는 테스트는 아무것도 증명하지 않는다
- **부작용을 내는 함수가 dry-run 가드를 소유한다** — 호출부에 흩뿌리지 않는다.
  훅(`runHooks`)·롤백(`rollbackServices`)이 그 예다
- 벤더 SDK 도입은 "감싸되 대체하지 않는다"(SOUL 신념 1)에 반하므로 오너 승인을 요구한다
- 새 기능은 SOUL.md 4-게이트(틈 · 라이브러리 · 대량/전환 · 날카로움)를 통과해야 한다
- Quality Gates 미충족 시 릴리스는 차단된다

______________________________________________________________________

**End of Document**
