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
- 현재 **둘 다 실패**. `.make/build.mk`가 (1) Go에서 제거된
  `GOEXPERIMENT=rangefunc`를 설정하고 (2) 본 리포에 없는 `./cmd/gzh-git`를
  빌드한다. `make test`는 `build`에 의존하므로 함께 실패하고, **CI가 red다**.
  `go test ./...` 직접 실행은 통과한다. **최우선 과제다**

G3. **Concurrency safety**

- Target: `go test -race ./...` 통과
- 현재 **실패 (재현되는 실제 데이터 레이스)**. `switchServicesParallel`의 goroutine들이
  뮤텍스 없이 공유 `previousStates` 맵에 쓰고 result 슬라이스에 append한다
  (`es.mu`는 레지스트리만 보호). **`switch-all --parallel`은 현재 안전하지 않다**

G4. **Dry-run honesty**

- Target: `--dry-run`은 어떤 부작용도 만들지 않는다
- 현재 **미충족** — pre/post 셸 훅이 dry-run 가드 없이 실제 실행된다. CLI는
  "DRY-RUN MODE: No changes will be made"를 출력하지만 훅이 있으면 그 문구는 거짓이다

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
- **미비**: dry-run이 훅을 실행한다 (G4); 롤백은 **인메모리 전용**이라 전환 중
  크래시 시 복구 아티팩트가 없다; 훅 차단목록은 `sh -c` 위의 블록리스트라 우회
  가능하다; `SwitchOptions.Force`는 라이브러리에서 읽히지 않는 죽은 필드다
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

**Baseline (진행 중)**

- `LICENSE` 미보유 (GUIDELINES §4 격차). 전 소스에 `SPDX-License-Identifier: MIT`
  헤더가, README에 MIT 주장이 있으나 라이선스 원문이 없다 — `go get`으로 소비되는
  공개 라이브러리로서 실질적 결함이다

______________________________________________________________________

## Quality Gates (Release Readiness)

**Build and Lint**

- `make build` · `make test` 통과 (G2 — 현재 미충족)

**Concurrency**

- `go test -race ./...` 통과 (G3 — 현재 미충족)

**Testing**

- 커버리지 >= 80% (현재 60.9%)

**Docs**

- 도움말·컨텍스트 문서가 실제 등록된 명령과 일치한다 (현재 미충족: 루트 도움말이
  `kubeconfig save`·`aws-profile list` 등 미등록 명령을 광고한다)

______________________________________________________________________

## Decision Rules

- **새 서비스는 checker + switcher + Rollback 실구현을 함께 포함해야 한다** —
  SSH 같은 no-op 스텁을 반복하지 않는다 (G1)
- **동시성 코드는 `-race` 통과 없이 머지될 수 없다** (G3)
- `--dry-run`을 광고하는 실행 경로는 부작용 0을 증명해야 한다 (G4)
- 벤더 SDK 도입은 "감싸되 대체하지 않는다"(SOUL 신념 1)에 반하므로 오너 승인을 요구한다
- 새 기능은 SOUL.md 4-게이트(틈 · 라이브러리 · 대량/전환 · 날카로움)를 통과해야 한다
- Quality Gates 미충족 시 릴리스는 차단된다

______________________________________________________________________

**End of Document**
