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

## 전달 차단 및 잔여 조건

lint 로그에는 고정 golangci-lint 2.12.2가 현재 Go 1.27.3의 export data
version 5를 읽지 못한다는 typecheck 오류가 있다. lint PASS가 아니므로
현재 후보를 통합 가능 상태로 표시하지 않는다. 도구 pin 변경이나 이전 Go로
우회하지 않았으며, 별도 Go 도구 계약 작업의 결과를 확인한 뒤 재검증해야 한다.
이 기록은 해당 후속 작업의 해결을 주장하지 않는다.

로컬 원본 증거는 작업 worktree의 `tmp/issue352`에 보존한다
(`compare.json`, `checks.json`, `test-result.json`, 각 target 로그와 좁은 회귀 진단).
Windows 실제 실행, native 통합, master push, 제품 배포는 미실행이다.
저장소의 CE lifecycle 선언도 없어 `ce task run-doctor`가 BLOCKED를 반환했다.
이번 변경에 저장소 전용 lifecycle 스크립트는 추가하지 않았다.
