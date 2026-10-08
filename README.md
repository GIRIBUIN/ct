# ct

Windows와 Linux에서 코딩 테스트 풀이 파일을 만들고 VS Code로 여는 Go CLI입니다.

```sh
go build ./cmd/ct
ct 71A
ct 71A -l py
ct 71A --language python
ct 181188 -p pg
ct 181188 --platform programmers
ct 181188 -p pg -l py
ct config
ct doctor
```

빌드된 실행 파일을 PATH에 두면 `ct`로 실행할 수 있습니다. 현재 디렉터리에서
직접 실행할 때는 Windows에서 `.\ct.exe`, Linux에서 `./ct`를 사용합니다.
VS Code의 `code` 명령도 PATH에서 실행 가능해야 합니다.

기본값은 플랫폼 `codeforces`, 언어 `cpp`, 편집기 `code`입니다.
옵션은 문제 ID 앞뒤에 사용할 수 있으며 `--help`로 사용법을 확인합니다.

| 구분 | 입력 | 정규 이름 |
| --- | --- | --- |
| 플랫폼 | `cf`, `codeforces` | `codeforces` |
| 플랫폼 | `pg`, `programmers` | `programmers` |
| 언어 | `cpp`, `c++` | `cpp` |
| 언어 | `py`, `python` | `python` |

설정이 없는 상태로 문제를 열면 `ct initial configuration` 위저드가 루트,
기본 플랫폼, 기본 언어, VS Code 프로필을 묻습니다. 저장 후 요청한 문제 파일을
생성하고 엽니다. 같은 위저드는 `ct config`로 언제든 다시 실행할 수 있습니다.
공백이 포함된 경로도 사용할 수 있습니다.
상대 경로를 입력하면 현재 디렉터리 기준의 절대 경로로 저장합니다.

```text
Coding-test root:
Default platform [codeforces]:
Default language [cpp]:
VS Code profile [Default]:
```

기존 설정이 있으면 현재 값을 대괄호 안에 표시하며 Enter로 유지합니다.
루트는 최초 실행 시 필수이고, 플랫폼·언어의 잘못된 입력은 오류를 알린 뒤 다시
묻습니다. 별칭은 위 표의 정규 이름으로 저장합니다. 프로필이 없는 상태에서
Enter는 Default이며, 기존 이름 있는 프로필을 Default로 바꾸려면 `-`를 입력합니다.
비어 있지 않은 프로필 이름은 앞뒤 공백까지 입력한 그대로 사용합니다.

이름 있는 프로필을 선택하면 새 빈 프로필이 만들어질 수 있음을 안내하고
`Continue? [Y/n]`으로 확인합니다. 거절하면 프로필을 다시 묻습니다. 수락하면
기존 VS Code 탐색과 `--profile <name> --list-extensions`로 사용 가능 여부를
확인합니다. CLI를 찾지 못하거나 조회가 실패하면 저장하지 않고 재입력을 받습니다.
Default는 이 확인 없이 사용할 수 있습니다. 확장은 설치하지 않습니다.

설정은 `os.UserConfigDir()` 아래의 `ct/config.json`에 저장합니다.
`CT_CONFIG_DIR`가 비어 있지 않으면 해당 디렉터리의 `config.json`을 대신 사용합니다.
개발 스크립트는 `.tmp-ct-config`로 설정을 격리하고 `.tmp-coding-test`를 풀이 루트로
사용합니다. OS 사용자 설정 환경변수는 변경하지 않으며, `CT_CONFIG_DIR`는 편집기에
전달하지 않습니다.
Windows에서는 `%AppData%\ct\config.json`, Linux에서는
`$XDG_CONFIG_HOME/ct/config.json` 또는 `$HOME/.config/ct/config.json`입니다.
설정 파일을 포함하는 디렉터리는 풀이 루트로 선택할 수 없습니다.
`ct config`는 루트, 기본 플랫폼·언어, 프로필을 변경하고 저장 결과를 요약합니다.
기존 설정 교체는 같은 디렉터리의 임시 파일을 완전히 쓴 뒤 rename으로 수행하므로
저장이 실패해도 기존 정상 설정을 먼저 잘라내지 않습니다. 기존 설정은 일반 문제
명령이나 doctor 실행 시 다시 쓰지 않습니다.
JSON의 `root`, `platform`, `language`, `editor`를 직접 수정할 수도 있으며,
`editor`에는 실행 파일 이름 또는 경로를 지정합니다(추가 인자는 지원하지 않습니다).
선택적 `editor_profile`에 기존 VS Code 프로필 이름을 지정하면 파일 열기와
doctor 확장 조회에 같은 프로필을 사용합니다. 예: `"editor_profile": "coding test"`.
필드가 없거나 빈 문자열이면 `--profile` 옵션을 생략하며, 기존 설정은 그대로
호환됩니다. 프로필은 첫 실행 위저드 또는 `ct config`에서 지정할 수 있습니다.
CLI 옵션은 해당 실행에만 적용되고 저장된 기본값을 변경하지 않습니다.

생성되는 경로는 다음과 같습니다. 언어별 하위 디렉터리는 만들지 않습니다.

```text
<root>/codeforces/71A/main.cpp
<root>/codeforces/71A/main.py
<root>/programmers/181188/solution.cpp
<root>/programmers/181188/solution.py
```

템플릿은 `go:embed`로 실행 파일에 포함됩니다. Codeforces는 `solve()` 기반이며,
Programmers는 `solution()` 골격만 제공합니다. Programmers의 반환형과 인자는
문제에 맞게 직접 수정해야 합니다.

기존 풀이 파일은 덮어쓰거나 잘라내지 않습니다. 파일이 이미 있으면 이를 알리고
그대로 엽니다. VS Code에는 `--reuse-window <root> <target>`을 전달하여
루트 폴더와 풀이 파일을 엽니다. 편집기 실행이 실패해도 생성한 파일은 보존합니다.
프로필이 설정되어 있으면 `--profile <editor_profile> --reuse-window <root> <target>`을
전달합니다.

`ct doctor`는 OS/아키텍처, VS Code CLI, g++/GDB, C++20 및 `bits/stdc++.h`
컴파일, C/C++·CPH 확장, 설정과 풀이 루트 디렉터리를 진단합니다. Windows에서는
기존 PATH 및 환경변수 기반 VS Code 탐색을 그대로 사용하며, 설정된 `editor`를
확인합니다. 설정을 읽을 수 없으면 기본 `code`를 탐색합니다.

각 `[FAIL]`은 필수 검사 실패이며, 모든 검사를 마친 뒤 하나 이상이면 종료 코드 1,
모두 통과하면 0을 반환합니다. 누락된 도구 때문에 실행할 수 없는 후속 검사는
`[SKIP]`으로 표시하고 실패 수에 중복 집계하지 않습니다. g++/GDB의 버전 문자열은
참고 정보이며 조회 실패나 최신 버전 여부는 성공 판정에 영향을 주지 않습니다.
C++ 중심 환경 진단이므로 기본 언어가 Python이어도 같은 검사를 수행합니다.

설정이 없으면 실패를 보고하며 최초 설정을 시작하지 않습니다. `CT_CONFIG_DIR`를
존중하고 설정·풀이 루트를 생성하거나 수정하지 않습니다. 컴파일 검사는 OS 임시
디렉터리에서 실행하고 결과물을 정리하며, 생성한 실행 파일은 실행하지 않습니다.
외부 명령은 각각 최대 20초로 제한합니다. 확장 조회는 프로필 설정 시
`--profile <editor_profile> --list-extensions`, 미설정 시 `--list-extensions`를 사용합니다.
doctor에는 선택한 프로필 이름 또는 `default`가 표시됩니다. Windows 배치 호출에서
안전하게 전달할 수 없는 큰따옴표·줄바꿈·NUL을 포함한 프로필 이름은 오류로 처리합니다.

setup, 설치 도구, 릴리스·업데이트 및 제출 자동화는 지원하지 않습니다.

검증:

```sh
go test ./...
go vet ./...
go build ./cmd/ct
```

테스트는 임시 디렉터리와 가짜 편집기를 사용하며 실제 사용자 설정이나 풀이
디렉터리를 변경하지 않습니다.
