# AWS 전환과 대상별 복구

AWS 선택·대상 해석과 복구는 이 라이브러리가 소유한다. 요청 DTO `AWSConfig`와
`ServiceSwitcher`는 유지하며, AWS 구현은 선택적 `StateCapturer`·`StateApplier`·
`RecoveryValidator`를 제공한다. manager는 표시용 조회 대신 요청의 변경 대상을 캡처하고,
그 상태를 journal에 기록한 뒤 같은 계획을 적용한다.

## 요청과 환경변수

- 부모 프로세스의 환경변수를 바꾸지 않는다. nonempty `AWS_PROFILE` → 도구의 선택 파일 →
  `default` 순서로 profile을 해석한다. 빈 환경변수는 선택 override로 사용하지 않는다.
- 명시 Profile과 nonempty `AWS_PROFILE`이 다르면 해당 AWS 변경 전에 오류를 반환한다.
- Region과 nonempty `AWS_REGION`이 충돌하면 오류다. `AWS_REGION`이 비면
  `AWS_DEFAULT_REGION`을 같은 규칙으로 사용한다. 이는 config-file region 변경의 충돌
  검사이며 모든 AWS CLI/SDK의 유효 설정을 재현한다는 뜻은 아니다.
- nonempty AccountID는 명시 미지원이다. 완전히 빈 요청은 no-op다.
- Profile 없는 Region 요청도 해석된 profile을 명시하여 쓴다. 선택 파일 B가 있는데
  ambient/default에 쓰던 이전 동작과 다른 계약이다.

## 변경 대상과 복원

표시용 `GetCurrentState`의 Profile/Region 값은 정확한 복구 상태가 아니다.
`CaptureState`는 선택 파일의 존재·검증된 profile 원문·mode, 실제 config 경로와 대상
profile, region key의 존재·값 및 예상 변경 결과를 따로 저장한다. 전체 AWS config나
credential 원문은 journal에 저장하지 않는다.

`ApplyState`는 journal에 기록된 계획을 적용하며, 대상 상태가 캡처 이후 달라졌으면
재해석하여 덮어쓰지 않고 오류를 반환한다. 직접 `Switch` 호출도 캡처·적용 경로를 사용한다.
Region 쓰기는 명시 profile과 config 경로를 사용하는 AWS CLI를 통해 수행한다.

복원은 현재 값이 캡처 전 또는 예상 변경 결과일 때만 수행한다. 다른 writer의 변경은
충돌로 반환한다. 선택 파일 부재·빈 원문·mode와 대상 region key 부재를 구분해 복원한다.
다른 config key는 보존한다. 원래 없던 config 파일에 다른 설정이 추가됐다면 파일을
삭제하지 않고 불완전 복원 오류를 남긴다. 전체 config 파일의 byte-identical 복원이나
서로 다른 프로세스 사이의 원자적 CAS를 보장하지 않는다.

mode는 Go가 해당 플랫폼에서 관측하는 파일 mode다. 새 선택 파일은 계속 `0600`을 요청해
생성한다. POSIX에서는 예상 mode도 `0600`이지만, Windows의 Go 파일 정보는 쓰기 가능한
일반 파일을 `0666`, readonly 파일을 `0444`로 표현한다. Windows의 `Chmod`는 Unix 권한이나
ACL을 구현하지 않는다. 이 복구 상태는 Windows ACL/security descriptor 복원을 보장하지 않는다.

기존 v0.2.0 journal에서 선택 파일이 원래 부재이고, nonempty Profile 요청과 경로·존재·원문이
정확히 일치하며 예상 mode만 `0600`인 경우에는 Windows에서 그 쓰기 의도를 관측 mode
`0666`으로 해석한다. 기존 원문의 before mode나 현재 파일을 일괄 정규화하지 않고, readonly
변경·다른 mode·원문·경로 변경은 계속 거절한다. 저장된 journal은 임의 rewrite하지 않는다.
이것은 생성 OS를 인증하는 기능이 아니며 플랫폼별 Go 쓰기 의미의 좁은 호환 처리다.
새 Windows 예상 mode를 포함한 journal은 기존 v0.2.0 reader에서 검증 오류로 거절될 수 있다.

모호한 INI 형식·중복 key/section·비일반 파일은 추측하지 않고 오류로 반환한다.
외부 writer와의 검사-쓰기 사이 경쟁은 남아 있으며 임의 파일시스템 경로에 대한
완전한 격리 보장이 아니다.

## journal 호환성과 복구

AWS target 상태가 포함된 journal의 wire version은 문자열 `aws-target-v2`다.
공개 `RollbackJournal.Version`은 정수 2지만 JSON 재직렬화도 이 문자열을 유지한다.
v0.1.1 reader는 정수 version을 요구하므로 같은 경로의 새 journal을 읽는 단계에서
거절한다. 숫자만 2로 바꾸는 방식은 사용하지 않는다.

새 reader는 legacy 숫자 1의 non-AWS 상태를 유지한다. legacy AWSConfig는 변경 대상·
존재 정보가 없어 journal 복구에서 거절한다. 모든 entry의 형식·서비스 대응·등록 여부와
필요 capability를 확인한 뒤에만 첫 Rollback을 호출한다. 실패하면 journal을 보존한다.
등록형 사용자 정의 non-AWS 서비스의 기존 직렬화·복원 경로는 유지한다.

이는 잘못된 복구 metadata 때문에 일부 서비스부터 복원하는 일을 막는 경계다.
유효한 복구 실행 중 I/O 실패·취소가 발생했을 때 여러 서비스의 원자적 복원을 뜻하지
않는다. 취소된 context의 자동 교체·복구 시간 한도는 이 변경에 포함하지 않는다.
hook과 먼저 실행된 다른 서비스의 무변경도 보장하지 않는다.

## 소비자 연결

표시용 snapshot만 새 owner로 바꾸지 않는다. owner 검증·공개 모듈 검증 이후 main의
전환·조회·복원을 함께 연결한다. wrapper가 있다면 `StateCapturer`·`StateApplier`·
`RecoveryValidator`도 전달해야 한다. 세 기본 메서드만 위임해 표시 DTO를 다시 journal에
저장하는 연결은 이 계약을 충족하지 않는다. 새 module tag·게시와 main pin 변경은 별도 출시 단계다.
