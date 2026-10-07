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

최초 실행에서는 코딩 테스트 루트 디렉터리를 입력받고 설정을 저장한 뒤,
요청한 문제 파일을 바로 생성합니다. 공백이 포함된 경로도 사용할 수 있습니다.
상대 경로를 입력하면 현재 디렉터리 기준의 절대 경로로 저장합니다.

설정은 `os.UserConfigDir()` 아래의 `ct/config.json`에 저장합니다.
`CT_CONFIG_DIR`가 비어 있지 않으면 해당 디렉터리의 `config.json`을 대신 사용합니다.
개발 스크립트는 `.tmp-ct-config`로 설정을 격리하고 `.tmp-coding-test`를 풀이 루트로
사용합니다. OS 사용자 설정 환경변수는 변경하지 않으며, `CT_CONFIG_DIR`는 편집기에
전달하지 않습니다.
Windows에서는 `%AppData%\ct\config.json`, Linux에서는
`$XDG_CONFIG_HOME/ct/config.json` 또는 `$HOME/.config/ct/config.json`입니다.
설정 파일을 포함하는 디렉터리는 풀이 루트로 선택할 수 없습니다.
JSON의 `root`, `platform`, `language`, `editor`를 직접 수정할 수 있으며,
`editor`에는 실행 파일 이름 또는 경로를 지정합니다(추가 인자는 지원하지 않습니다).
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
그대로 엽니다. VS Code에는 `--reuse-window <root> --goto <target>`을 전달하여
루트 폴더와 풀이 파일을 엽니다. 편집기 실행이 실패해도 생성한 파일은 보존합니다.

v0.1에는 doctor/setup, 설치 도구, 릴리스·업데이트 및 제출 자동화가 없습니다.

검증:

```sh
go test ./...
go vet ./...
go build ./cmd/ct
```

테스트는 임시 디렉터리와 가짜 편집기를 사용하며 실제 사용자 설정이나 풀이
디렉터리를 변경하지 않습니다.
