# 상태 조회의 관측 범위

AWS 상태 조회는 선택된 profile의 기존 identity를 확인하며 새 session 자격증명을 요청하지 않는다. 자격증명 종류를 관측하지 않은 경우 기존 `aws-credentials` 값을 유지한다. Profile 선택은 기존 `AWS_PROFILE` → 라이브러리의 active profile 파일 → 빈 값 순서이며, 비어 있지 않은 선택을 identity 조회에도 전달한다. Region CLI 조회에도 전달받은 context를 사용해 취소가 해당 프로세스에 전파된다.

Docker context 조회 실패는 현재 context 값을 비우고 기존 `StatusError`와 상세 오류 경로로 전달한다. 정상 종료한 빈 출력과 공개 `DefaultContext` 상수는 유지한다.

GCP health의 `auth list` 성공 문구는 인증 설정을 읽을 수 있었다는 범위만 설명한다. 이 조회만으로 실제 서비스 API 접근이나 자격증명 유효성을 검증했다고 주장하지 않는다. 기존 상태 값과 raw `auth_list` 필드는 유지한다.

Checker는 관측 결과와 실패 정보를 기존 상태·상세·경고 필드에 담는다. 회귀 테스트는 고정 native 실행체의 인자·호출 순서·성공·실패·취소를 관측하며 실제 vendor CLI나 사용자 설정을 사용하지 않는다. AWS region fallback의 의미와 광범위한 credential 정책은 별도 확인 대상이며, 이 수정은 새로운 자격증명 분류 알고리즘이나 전체 상태 조회의 원자성을 도입하지 않는다.
