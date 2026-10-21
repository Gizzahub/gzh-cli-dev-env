# Markdown formatter 적용 기록

## 계약

ISSUE-352의 패밀리 채택 기준에 따라 mdformat 1.0.0,
mdformat-gfm 1.0.0, mdformat-frontmatter 2.1.2를 사용한다.
`make install-mdformat`은 `uv tool install`로 저장소의 `bin/tools`에
설치하고, 정확한 버전·플러그인 집합과
`.make/mdformat-constraints.txt`의 하위 의존성을 검증한다.
전역 mdformat 설치는 변경하지 않는다.

`format-md`, `format-md-check`, `fmt`의 Markdown 단계는 Git이 추적하거나
ignore하지 않은 일반 Markdown 파일을 선택한다. symlink·디렉터리와 ignored
도구 venv는 제외하며, 공백·개행 경로를 NUL 구분으로 전달한다.
`format-md-diff`는 HEAD 대비 추적 파일의 변경만 선택하므로 untracked 파일은
포함하지 않는다. Git 조회와 formatter의 실패는 target 실패로 전파한다.
이 선택기는 Bash와 NUL 입력을 지원하는 xargs를 사용한다.

## 검증 — 2026-10-21

입력 base는 `0a73f802caa7f87d99b9068bd6c9351cc0ae327b`이며,
구현·독립 리뷰 요구 등급은 strong이다.

- 저장소 전용 설치 및 버전·하위 의존성 검사: PASS.
- 기존 6개 문서의 구 묶음과 새 묶음 출력: 모두 byte 동일.
  AGENTS.md, PRODUCT.md, docs/.claude-context/README.md 및 common-tasks.md는
  기존부터 canonical 출력과 달라 동일한 출력으로 정렬했다.
- YAML frontmatter 보존, 표 처리, 재실행 멱등성과 미정렬 입력 거부: PASS.
- 버전 순서 허용, postrelease·누락·추가 플러그인 및 의존성 불일치 거부: PASS.
- 독립 strong 리뷰에서 발견한 버전 명령 실패 누락을 수정했다.
  정상 버전 문구를 출력하고 exit 42로 끝나는 명령도 거부한다
  (`tmp/issue352/version-exit-regression.json`).
- 공백·개행 경로, symlink·디렉터리·ignored venv 제외,
  변경 파일 선택, Git 및 formatter 실패 전파: PASS.

| 실행                                                        | 종료 코드 | 소요 시간 |
| ----------------------------------------------------------- | --------- | --------- |
| make fmt                                                    | 0         | 2.850초   |
| make format-md-check                                        | 0         | 2.448초   |
| make lint                                                   | 2         | 21.130초  |
| make test (clean → library build → 전체 Go test → coverage) | 0         | 20.675초  |

각 실행의 외부 진단 제한은 임시 1800초였으며 프로젝트 timeout·병렬도는
변경하지 않았다. 상세 메트릭 수집 설계는 포함하지 않는다.
`make fmt`가 추가로 정렬한 Go 파일 6개는 작업 범위 밖이므로 되돌렸다.
따라서 최종 Go 파일이 fmt 검사를 통과한다고 주장하지 않는다.

## 이전 기준의 전달 차단 기록

lint 로그에는 고정 golangci-lint 2.12.2가 현재 Go 1.27.3의 export data
version 5를 읽지 못한다는 typecheck 오류가 있었다. 이전 후보는 lint 실패로
전달을 차단했다. 이 작업에서 도구 pin을 변경하거나 이전 Go로 우회하지 않았다.
이 실패 기록은 아래 재기준화 결과와 구분해 보존한다.

로컬 원본 증거는 작업 worktree의 `tmp/issue352`에 보존한다
(`compare.json`, `checks.json`, `test-result.json`, 각 target 로그와 좁은 회귀 진단).
Windows 실제 실행, native 통합, master push, 제품 배포는 미실행이다.
저장소의 CE lifecycle 선언도 없어 `ce task run-doctor`가 BLOCKED를 반환했다.
이번 변경에 저장소 전용 lifecycle 스크립트는 추가하지 않았다.

## 재기준화 검증 — 2026-10-21

별도 Go 도구 작업이 반영된 master `b0c1c3011d46dad63566a1bf775c7c240022ea9a`
위에 충돌 없이 재기준화한 구현 후보는 `258eabc5eb84b07f6cb2f3ecb2bf15709dc196ec`다.
Go 1.27.3·golangci-lint 2.15.0을 보존했고, 독립 strong 리뷰에서 구현 PASS를
확인했다. 이전 기준의 검사를 새 기준의 PASS로 전용하지 않았다.

| 실행                                             | 종료 코드 | 소요 시간 |
| ------------------------------------------------ | --------- | --------- |
| 좁은 Markdown 회귀 진단                          | 0         | 7.164초   |
| make lint (설치 포함)                            | 0         | 24.883초  |
| make fmt                                         | 0         | 2.612초   |
| make format-md-check                             | 0         | 0.978초   |
| make test (clean·library build·Go test·coverage) | 0         | 1.849초   |

이번에도 fmt의 범위 밖 Go 파일 6개 변경을 되돌린 뒤 test를 실행했다.
테스트는 Go 캐시를 허용하는 기존 target 그대로 실행했으며 속도 개선을 주장하지 않는다.
원본 로그와 결과는 `tmp/issue352/rebase-20261021`에 보존한다.

설치된 gz-git 0.11.1로 위 source/target SHA를 고정한 `integrate check`는
exit 0, READY였다. 검증 시간은 13.033초이며 make lint·freshness·merge-tree·clean·push
검사가 통과했다. readiness는 `legacy-make`로 관측됐고 head-owned gate 경고 1건이 있었다.
baseline make check target은 없었고 baseline lint는 측정하지 않았으므로
baseline PASS는 주장하지 않는다. 이것은 check 결과이며 통합·배포 완료가 아니다.
이후 검증 기록만 추가한 커밋의 최종 통합은 source/target을 다시 고정해 실행해야 한다.

Windows 실제 실행과 Python 3.11+ 선택 강제는 검증하지 않았다.
상세 메트릭 설계와 lifecycle·readiness 정책 변경은 이번 적용 범위에 포함하지 않는다.
