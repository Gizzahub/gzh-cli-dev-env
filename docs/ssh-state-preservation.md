# SSH 설정 전환과 복구

SSH switcher는 `~/.ssh/config`를 관리한다. 일반 파일은 요청한 설정과 같은 파일일 때만 그대로 사용하며, 다른 일반 파일이나 디렉터리를 덮어쓰지 않는다. 기존 계약대로 활성 경로의 symlink는 관리형 레이아웃으로 취급한다. 링크 대상 설정 파일의 내용은 수정하거나 삭제하지 않는다.

## 실패 경계

새 symlink를 활성 경로와 같은 파일시스템의 전용 staging 디렉터리에 먼저 만든다. 활성 경로가 여전히 관리형인지와 context 취소 여부를 확인한 뒤 rename으로 반영한다. staging 디렉터리·링크 생성 또는 rename이 실패하면 이전 활성 링크를 먼저 지우지 않는다. 정리는 이 호출이 만든 staging 경로만 대상으로 하며 실패를 반환 오류에 포함한다.

rename이 성공한 뒤 staging 디렉터리 정리가 실패하면 활성 설정은 이미 바뀌었을 수 있다. 이 경우에도 정리 실패를 숨기지 않는다. 반환 오류가 언제나 “아무 변화 없음”을 의미하지는 않는다. 다른 프로세스와의 동시 변경에 대한 조건부 갱신이나 모든 OS에서의 rename 원자성을 보장하지 않는다.

`Switch`, `GetCurrentState`, `Rollback`은 진입 시 취소된 context를 거부한다. 활성 경로를 변경하기 직전에도 취소를 확인한다. commit 후 도착한 취소가 이미 끝난 파일 작업을 되돌린다는 보장은 없다. 기존 nil context 호출은 제한 없는 context와 같은 취급을 유지한다. 잘못된 타입과 typed nil 설정은 panic 대신 오류를 반환한다.

## 없는 설정의 복원

현재 설정이 없으면 `GetCurrentState`는 기존처럼 `SSHConfig{Config: ""}`를 반환한다. 일반 `Switch`는 여전히 빈 설정 경로를 거부한다. `Rollback`은 이 snapshot을 부재 상태로 해석해 활성 symlink만 제거한다. 이미 없으면 성공하고, 일반 파일·디렉터리는 보존하며 오류를 반환한다. 링크 대상 파일이나 `.ssh` 디렉터리를 삭제하지 않는다.

Switcher는 stateless이며 공개 인터페이스와 snapshot 형식은 유지한다. 따라서 새 switcher 인스턴스와 기존 디스크 journal로도 부재 상태를 복원할 수 있다. journal 복구가 실패하면 journal을 남겨 후속 복구에 사용할 수 있게 한다.

## 검증 범위

회귀는 임시 HOME과 실제 파일·symlink를 사용한다. staging·rename 실패와 commit 전 취소에서 이전 inode·대상·내용 및 staging 정리를 확인하고, 정리 실패 자체도 관측한다. 새 인스턴스의 복원과 실제 `WriteJournal` → `RecoverFromJournal` 경로, 일반 파일 충돌 시 사용자 bytes와 journal 보존을 검증한다. 사용자 SSH 설정이나 실제 SSH 연결은 테스트 대상으로 사용하지 않는다.
