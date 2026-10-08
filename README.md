# ct

Windows와 Linux에서 코딩 테스트 풀이 파일을 만들고 VS Code로 여는 Go CLI입니다.

## 설치와 업데이트

지원 대상은 **Windows amd64, Linux amd64, Linux arm64**입니다. 배포 바이너리를
사용하면 Go, Git, 저장소 복제나 수동 빌드가 필요 없습니다. 아래 설치 명령은
첫 GitHub Release가 게시된 뒤 사용할 수 있습니다.

### Windows (PowerShell 5.1 이상)

```powershell
irm https://raw.githubusercontent.com/GIRIBUIN/ct/main/scripts/install.ps1 | iex
ct --version
ct config
ct doctor
ct setup
ct 71A
```

최신 정식 릴리스의 `ct-windows-amd64.exe`를 SHA-256 검증 후
`%LOCALAPPDATA%\Programs\ct\ct.exe`에 설치합니다. User PATH에 그 디렉터리만
추가하고 현재 PowerShell에도 반영합니다. 기존 항목을 보존하고 대소문자·구분자를
정규화해 중복을 피하며, 새 항목에 따옴표를 넣지 않습니다. 다른 터미널은 재시작해야
할 수 있습니다. Machine PATH, 실행 정책은 바꾸지 않으며 관리자 권한도 필요 없습니다.
업데이트는 **같은 설치 명령을 다시 실행**합니다. 검증된 파일을 같은 디렉터리에 준비한
후 교체하며, 실행 중인 ct가 잠겨 있으면 종료하고 다시 시도하세요.

일반 제거(설정 보존):

```powershell
irm https://raw.githubusercontent.com/GIRIBUIN/ct/main/scripts/uninstall.ps1 | iex
```

설정까지 제거하는 purge는 매개변수를 전달할 수 있는 ScriptBlock으로 실행합니다:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/GIRIBUIN/ct/main/scripts/uninstall.ps1))) -Purge
```

제거 스크립트는 설치 위치의 `ct.exe`, 비어 있는 설치 디렉터리, 해당 위치와 정확히
일치하는 User/현재 프로세스 PATH 항목만 제거합니다.

### Linux

```sh
curl -fsSL https://raw.githubusercontent.com/GIRIBUIN/ct/main/scripts/install.sh | bash
# PATH를 추가했다는 안내가 나오면 실행하거나 새 로그인 셸을 시작하세요.
. "$HOME/.profile"
ct --version
ct config
ct doctor
ct setup
ct 71A
```

`x86_64`/`amd64`는 `ct-linux-amd64`, `aarch64`/`arm64`는 `ct-linux-arm64`를
선택해 `~/.local/bin/ct`에 설치합니다. Bash, curl과 `sha256sum` 또는 `shasum`이
필요합니다. root/sudo는 사용하지 않습니다. `~/.local/bin`이 PATH에 없을 때만
`~/.profile`에 표시된 ct PATH 블록을 한 번 추가합니다. 파이프로 실행된 설치 스크립트는
부모 셸의 PATH를 바꿀 수 없으므로 로그인 셸을 새로 열거나 위처럼 파일을 읽어야 합니다.
`~/.bash_profile`만 읽는 셸 등에서는 해당 파일이 `~/.profile`을 읽는지도 확인하세요.
업데이트는 **같은 설치 명령을 다시 실행**합니다.

일반 제거와 설정까지 제거하는 purge:

```sh
curl -fsSL https://raw.githubusercontent.com/GIRIBUIN/ct/main/scripts/uninstall.sh | bash
# 또는:
curl -fsSL https://raw.githubusercontent.com/GIRIBUIN/ct/main/scripts/uninstall.sh | bash -s -- --purge
```

Linux 제거는 `~/.local/bin/ct`와 설치 스크립트가 추가한 정확한 PATH 블록만 제거합니다.
사용자가 수정한 블록과 다른 셸 설정은 보존합니다. `~/.local/bin` 안의 다른 파일은
건드리지 않습니다.

### 검증과 보존 범위

두 설치 스크립트는 HTTPS로 바이너리와 `checksums.txt`를 임시 위치에 내려받고,
정확한 자산 이름에 해당하는 SHA-256 항목이 하나인지 확인합니다. 해시가 다르거나
항목이 없거나 중복이면 기존 설치와 PATH를 변경하지 않고 실패합니다. 파일은 검증 후
실행 권한과 버전을 확인하고 교체합니다. 설치는 ct 자체만 담당하며 `ct setup`을
자동으로 실행하지 않습니다. 설정은 `ct config`, 환경 진단은 `ct doctor`, 코딩 테스트
의존성 준비는 `ct setup`으로 각각 수행합니다.

기본 설정 파일이 없는 **대화형 최초 설치**에서는 설치된 바이너리의 전체 경로로
`ct config`를 실행하고, 성공하면 `ct doctor`를 실행합니다. Windows는 콘솔 입출력이
사용 가능한 경우에만 진행합니다. Linux는 `curl ... | bash`의 파이프 입력을 읽지 않고
`/dev/tty`를 열어 config와 doctor의 입출력에 연결합니다. 터미널을 사용할 수 없으면
온보딩을 건너뛰고 `ct config`, `ct doctor`, `ct setup --dry-run`을 안내합니다.
위 예제의 수동 설정·진단 명령은 자동 온보딩을 건너뛰었거나 다시 실행하려는 경우에
사용하면 됩니다.

최초 설치 여부는 바이너리 유무가 아니라 아래 기본 위치의 `config.json` 유무로
판단합니다. 기존 설정이 있으면 재설치·업데이트 시 위저드와 doctor를 다시 실행하지
않습니다. 자동 온보딩 중에는 `CT_CONFIG_DIR`를 제외하여 기본 설정 위치를 사용하고,
완료 후 호출 환경의 값은 보존합니다. config 취소·실패 시 doctor를 실행하지 않으며
설치된 ct도 유지합니다. doctor가 문제를 보고해도 설치 실패로 처리하지 않고
`ct setup --dry-run`으로 검토한 뒤 `ct setup`을 직접 실행하도록 안내합니다.

기본 제거는 설정·레지스트리·사용자 템플릿을 보존합니다. purge는 **기본 설정 위치의
`config.json`, `registry.json`, ct 소유 템플릿과 비어 있는 디렉터리만** 제거합니다. Windows는 `%APPDATA%\ct`, Linux는
`$XDG_CONFIG_HOME/ct` 또는 `~/.config/ct`입니다. `CT_CONFIG_DIR`는 purge 대상 선택에
사용하지 않으며 별도 경로는 보존한다고 안내합니다. 기본 위치에 다른 파일이 있으면
디렉터리를 보존합니다. 심볼릭 링크·Windows junction을 따라 삭제하지 않습니다.
템플릿 삭제는 `templates/languages/<이름>/<플랫폼>.tmpl` 및
`templates/platforms/<이름>/<언어>.tmpl` 형태의 일반 파일에 한정합니다.
그 밖의 파일과 비어 있지 않은 디렉터리는 보존하며, 레지스트리 JSON의 임의 경로를
삭제 대상으로 사용하지 않습니다.
풀이 루트는 읽거나 삭제하지 않으며 VS Code, 확장, GCC, GDB, MSYS2도 제거하지 않습니다.

체크섬은 전송 손상과 바이너리/목록 불일치를 검출합니다. 스크립트와 체크섬 자체는
이 GitHub 저장소 및 HTTPS를 신뢰하므로 독립된 서명 검증을 제공하지는 않습니다.
설치 중 새 릴리스가 게시되어 파일과 체크섬이 어긋나면 실패하며 재실행하면 됩니다.

## 사용법

```sh
ct 71A
ct 71A -l py
ct 71A --language python
ct 181188 -p pg
ct 181188 --platform programmers
ct 181188 -p pg -l py
ct 71A -l java
ct 71A -l rust
ct 181188 -p pg -l java
ct 181188 -p pg -l rust
ct config
ct doctor
ct setup
ct setup --dry-run
ct setup --yes
ct doctor -l java
ct doctor -l rust
ct setup -l java --dry-run
ct setup --dry-run -l rust
```

VS Code는 PATH에서 탐색하며 Windows에서는 일반 사용자·시스템 설치 위치도 확인합니다.

기본값은 플랫폼 `codeforces`, 언어 `cpp`, 편집기 `code`입니다.
옵션은 문제 ID 앞뒤에 사용할 수 있으며 `--help`로 사용법을 확인합니다.

| 구분 | 입력 | 정규 이름 |
| --- | --- | --- |
| 플랫폼 | `cf`, `codeforces` | `codeforces` |
| 플랫폼 | `pg`, `programmers` | `programmers` |
| 언어 | `cpp`, `c++` | `cpp` |
| 언어 | `py`, `python` | `python` |
| 언어 | `java` | `java` |
| 언어 | `rust`, `rs` | `rust` |

ct는 심사 사이트의 지원 정책에 따라 언어 선택을 제한하지 않습니다. 사용자가 선택한
언어의 로컬 작업 공간을 만듭니다. Codeforces와 Programmers 모두 네 언어를 사용할 수
있으며, 같은 문제 디렉터리에 여러 언어의 풀이를 함께 보관할 수 있습니다.

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
`scripts/dev.ps1`은 개발 설정이 없을 때 별도 프로세스로 `ct config`를 실행하고 그
프로세스의 stdin에만 루트와 기본 응답을 전달합니다. 입력과 프로세스가 완전히 종료된
뒤 요청한 명령은 현재 터미널 입력을 그대로 사용하므로
`.\scripts\dev.ps1 language add`, `platform add`, `config` 및 삭제 확인 질문도
대화형으로 사용할 수 있습니다. `--reset`은 격리된 개발 파일을 지운 뒤 다시 준비하고,
`--clean`은 개발 설정·풀이·실행 파일만 정리합니다.
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
<root>/codeforces/71A/Main.java
<root>/codeforces/71A/main.rs
<root>/programmers/181188/solution.cpp
<root>/programmers/181188/solution.py
<root>/programmers/181188/Solution.java
<root>/programmers/181188/solution.rs
```

내장 템플릿은 `go:embed`로 실행 파일에 포함됩니다. Codeforces는 `solve()` 기반이며,
Programmers는 `solution()` 골격만 제공합니다. Programmers의 반환형과 인자는
문제에 맞게 직접 수정해야 합니다.
Java는 Java 17 문법을 사용하며 Codeforces에는 BufferedReader/StringTokenizer와
StringBuilder를 사용한 입력·출력 골격을 제공합니다. Rust는 외부 crate 없이 Rust 2021의
토큰 입력·버퍼 출력 구조를 사용합니다. Programmers의 Java/Rust는 입출력 main 없이
편집 가능한 `solution` 골격만 제공합니다.

기존 풀이 파일은 덮어쓰거나 잘라내지 않습니다. 파일이 이미 있으면 이를 알리고
그대로 엽니다. VS Code에는 `--reuse-window <root> <target>`을 전달하여
루트 폴더와 풀이 파일을 엽니다. 편집기 실행이 실패해도 생성한 파일은 보존합니다.
프로필이 설정되어 있으면 `--profile <editor_profile> --reuse-window <root> <target>`을
전달합니다.

`ct doctor`와 `ct setup`은 설정의 기본 언어를 사용합니다. `-l`/`--language`로 해당
실행에만 언어를 지정할 수 있으며 설정을 다시 쓰지 않습니다. 선택 언어를 출력하고
OS/아키텍처, VS Code CLI, CPH, 설정·루트·프로필은 공통으로 확인합니다. Windows에서는
기존 PATH 및 환경변수 기반 VS Code 탐색을 그대로 사용하며, 설정된 `editor`를
확인합니다. 설정을 읽을 수 없으면 기본 `code`를 탐색합니다.

각 `[FAIL]`은 필수 검사 실패이며, 모든 검사를 마친 뒤 하나 이상이면 종료 코드 1,
모두 통과하면 0을 반환합니다. 누락된 도구 때문에 실행할 수 없는 후속 검사는
`[SKIP]`으로 표시하고 실패 수에 중복 집계하지 않습니다. g++/GDB의 버전 문자열은
참고 정보이며 조회 실패나 최신 버전 여부는 성공 판정에 영향을 주지 않습니다.

| 선택 언어 | 필수 도구·검사 | 언어 확장 |
| --- | --- | --- |
| C++ | g++, GDB, C++20 및 `bits/stdc++.h` 컴파일 | `ms-vscode.cpptools` |
| Python | 실행 가능한 Python 3의 경로·버전 (`python3`, `python`, `py -3` 순서) | `ms-python.python` |
| Java | java·javac 버전 17+, `javac --release 17` 임시 컴파일 | `redhat.java` |
| Rust | rustc 버전·`--edition=2021` 임시 컴파일 | `rust-lang.rust-analyzer` |

Java는 JRE만으로 통과하지 않습니다. Rust는 Cargo를 요구하지 않으며 rustup의 자동
도구체인 다운로드를 비활성화한 상태로 검사합니다. 다른 언어를 진단할 때 C++ 도구는
필수 검사항목으로 실행하지 않습니다. 설정을 읽을 수 없으면 기본 C++로 진단하며
설정 오류를 함께 보고합니다.

설정이 없으면 실패를 보고하며 최초 설정을 시작하지 않습니다. `CT_CONFIG_DIR`를
존중하고 설정·풀이 루트를 생성하거나 수정하지 않습니다. 컴파일 검사는 OS 임시
디렉터리에서 실행하고 결과물을 정리하며, 생성한 실행 파일은 실행하지 않습니다.
외부 명령은 각각 최대 20초로 제한합니다. 확장 조회는 프로필 설정 시
`--profile <editor_profile> --list-extensions`, 미설정 시 `--list-extensions`를 사용합니다.
doctor에는 선택한 프로필 이름 또는 `default`가 표시됩니다. Windows 배치 호출에서
안전하게 전달할 수 없는 큰따옴표·줄바꿈·NUL을 포함한 프로필 이름은 오류로 처리합니다.

`ct setup`은 doctor의 검사 결과를 재사용하여 누락된 구성 요소와 실행할 계획을
먼저 표시합니다. `Continue? [Y/n]`에서 승인한 뒤에만 설치나 PATH 변경을 수행합니다.
`--yes`는 표시된 종류의 작업을 사전 승인하며 확인 질문을 생략합니다.
`--dry-run`은 질문이나 설치 없이 계획만 출력합니다. 두 옵션을 함께 사용해도
dry-run이 우선합니다. 계획을 만들 수 있으면 dry-run은 미지원 항목이 있어도 0으로
종료합니다. 환경이 이미 준비되어 있으면 설치할 것이 없음을 알리고 종료합니다.

설치 대상은 doctor가 누락으로 보고한 항목뿐이며, 정상 도구나 확장을 다시
설치하지 않습니다. 확장 조회 자체가 실패하면 설치 여부를 추측하지 않습니다.
설정이 없거나 잘못되었다면 `ct config`로 먼저 해결해야 합니다. setup은 설정이나
풀이 디렉터리를 생성·변경하지 않습니다.

- C++ / Windows amd64: 환경변수 `MSYS2_ROOT`/`MSYSTEM_PREFIX`, 실행 파일과 PATH,
  시스템 드라이브의 일반 설치 위치, 사용자 설치 위치 및 MSYS2 제거 등록 정보를
  이용해 기존 설치를 찾습니다. 사용할 수 있는 설치가 없을 때만 `winget`의
  `MSYS2.MSYS2`를 `%LOCALAPPDATA%\Programs\ct-msys2`에 사용자 범위로 설치합니다.
  winget 소스·패키지 약관 수락도 계획에 표시합니다. 등록된 설치가 손상되었으면
  두 번째 설치를 만들지 않고 수동 복구를 안내합니다.
- Windows 패키지: 기존 MSYS2의 `pacman -S --needed --noconfirm`으로 필요한
  `mingw-w64-ucrt-x86_64-gcc` 및 `mingw-w64-ucrt-x86_64-gdb`와 의존성을 설치합니다.
  실행 파일이 이미 UCRT64 bin에 있고 PATH에서만 누락되었다면 재설치하지 않습니다.
  전체 MSYS2 업그레이드는 수행하지 않으므로 패키지 DB가 오래된 경우
  [MSYS2 공식 안내](https://www.msys2.org/docs/package-management/)에 따라 수동 업데이트해야 합니다.
- Windows PATH: 필요한 UCRT64 bin 추가를 계획에 명시하고 **User PATH만** 수정합니다.
  기존 문자열을 보존하고 대소문자·구분자·끝 슬래시를 정규화하여 중복을 피하며,
  새 항목에는 따옴표를 넣지 않습니다. 쓰기 직전에 User PATH를 다시 읽습니다.
  현재 ct 프로세스에도 추가하여 마지막 doctor 검사에 사용합니다. 다른 터미널은
  재시작해야 반영됩니다. Machine PATH나 PowerShell 실행 정책은 변경하지 않습니다.
- C++ / Linux Debian/Ubuntu 계열: 필요한 `build-essential`, `gdb`를 apt-get으로 설치합니다.
  패키지 목록 갱신과 설치 명령을 모두 계획에 표시하며, root가 아니면 승인 후에만
  sudo를 실행합니다. 인증 질문은 별도로 나타날 수 있습니다. 다른 배포판에서는
  자동 컴파일러 설치를 시도하지 않습니다.
- Java: Windows는 [Microsoft OpenJDK 21](https://learn.microsoft.com/en-us/windows/dev-environment/java)의
  `Microsoft.OpenJDK.21` winget 패키지를 계획에 표시하고 승인 후 설치합니다. 공급자
  설치 프로그램이 권한 상승과 환경 등록을 수행할 수 있으며, ct는 최종 검사를 위해
  현재 프로세스 PATH만 새로 읽습니다. ct가 JAVA_HOME을 직접 수정하지 않습니다.
  Debian/Ubuntu는 `openjdk-21-jdk`를 apt로 설치합니다. 저장소에 패키지가 없으면 오류를
  보고하며 저장소를 임의로 추가하지 않습니다. 다른 JDK가 PATH에서 우선하면 사용자가
  JDK 선택을 바로잡아야 합니다.
- Rust: rust-analyzer 확장은 자동 설치할 수 있지만, 도구체인은 [공식 rustup 안내](https://rustup.rs/)에
  따라 수동 설치합니다. OS별 링커 준비도 사용자가 수행합니다.
- Python: Python 확장은 자동 설치할 수 있지만 인터프리터 자체는 설치하지 않습니다.
- VS Code 확장: 설정된 프로필에는 `code --profile <profile> --install-extension <id>`,
  Default에는 `code --install-extension <id>`를 사용합니다. 대상은 선택한 언어의 확장과 CPH이며
  기존 VS Code 탐색·Windows 인자 처리를 재사용합니다. `--force`는 사용하지 않습니다.

작업이 실패하면 그 작업에 의존하는 후속 작업은 건너뛰고, 독립적인 작업은 계속합니다.
작업 후 doctor를 다시 실행하여 같은 프로필의 확장과 선택 언어의 컴파일/실행 능력까지
검증합니다. 환경이 준비되지 않았거나 작업이 실패하면 0이 아닌 종료 코드를 반환합니다.
doctor는 계속 읽기 전용 진단이며 설치나 PATH 수정을 수행하지 않습니다.

VS Code 자체 설치, Python 관리, 기존 비호환 컴파일러 교체, CLI 자체 업데이트 및
제출 자동화는 지원하지 않습니다. MSYS2의 임의 비등록 설치는 `MSYS2_ROOT`로 알려줄 수
있습니다. 자동 설치 경로에 공백이 있거나 필요한 패키지 관리자가 없으면 수동 설치를
안내합니다. 실제 설치에는 네트워크와 각 설치 도구의 권한이 필요합니다.

## Built-in languages

내장 언어 정의와 템플릿은 실행 파일에 포함됩니다. `registry.json`이 없어도 기존
동작을 그대로 사용할 수 있습니다.

| 이름 | 별칭 | 확장자 | ENV |
| --- | --- | --- | --- |
| cpp | c++ | cpp | full |
| python | py | py | detect |
| java | — | java | full |
| rust | rs | rs | partial |

`full`은 언어 환경 진단과 지원 OS의 자동 설치, `detect`는 런타임 진단,
`partial`은 일부 수동 준비가 필요한 환경 지원, `none`은 파일 생성만 지원함을
뜻합니다. 기존 언어별 OS·설치 제한은 위의 doctor/setup 설명과 같습니다.

```sh
ct language list
ct language show rs
ct language disable python
ct language enable py
```

목록에는 비활성 항목도 표시하며 `list --all`은 동일하게 동작합니다.
내장 항목은 삭제할 수 없지만 비활성화·재활성화할 수 있습니다.

## Built-in platforms

내장 플랫폼은 `codeforces`(`cf`), `programmers`(`pg`)입니다.
Codeforces 문제 ID 규칙과 Programmers 숫자 ID 규칙은 유지됩니다.

```sh
ct platform list
ct platform show pg
ct platform disable programmers
ct platform enable pg
```

`show`에는 정규 이름, 별칭, 출처, 활성 상태, 언어별 파일명과 템플릿 위치를 표시합니다.
언어의 `show`에는 확장자, 환경 지원 수준과 해당 내장 VS Code 확장도 표시합니다.

## Custom languages

```sh
ct language add
# Name: kotlin
# Aliases: kt
# File extension: kt
```

추가는 대화형입니다. 현재 활성 플랫폼마다 파일명을 묻습니다. Codeforces는
`main.kt`, Programmers는 `solution.kt`가 기본값이며 `Main.kt`, `Solution.kt` 등으로
변경할 수 있습니다. 각 조합의 템플릿 원본 파일 경로를 입력하면 내용을 복사하고,
Enter를 누르면 빈 템플릿을 만듭니다. 완료 시 출력하는 경로를 편집기로 열어 수정하세요.
빈 템플릿도 정상적으로 사용할 수 있으며, 원본 파일은 변경하거나 삭제하지 않습니다.

이름·별칭은 소문자로 정규화합니다. 정규 이름에는 영문자로 시작하는 영문 소문자,
숫자, `-`, `_`를 사용할 수 있고 별칭에는 `+`도 허용합니다. 빈 값, 중복, 내장·사용자
항목과의 충돌, 경로 구분자 및 Windows 예약 이름은 거부합니다. 확장자는 점 없이
영문자·숫자로 입력합니다. 파일명은 하위 경로가 아닌 단일 파일명이어야 합니다.

```sh
ct language show kt
ct 71A -l kt
ct language disable kotlin
ct language enable kotlin
ct language remove kotlin
```

사용자 언어는 로컬 파일 생성용이며 자동 컴파일러 설치 기능을 얻지 않습니다.
`ct doctor -l kotlin`은 언어별 컴파일러 검사를 실행하지 않고 제공자 부재를 `[SKIP]`으로
알립니다. VS Code CLI·CPH·설정·루트·프로필 공통 검사는 계속 수행합니다. 제공자 부재
자체는 정보이며 종료 코드 실패 사유가 아닙니다. 공통 검사를 통과하면 언어 환경
진단을 제공하지 않는다는 요약을 표시합니다. `ct setup -l kotlin`은 자동 setup이
지원되지 않는다고 알리고 설치 작업 없이 종료합니다. 사용자 명령·셸 후크는 실행하지 않습니다.

## Custom platforms

```sh
ct language add
# kotlin / kt, 확장자 kt
ct platform add
# baekjoon / boj
# kotlin 파일명은 Main.kt로 지정하고 원하는 템플릿 원본을 선택
ct 1000 -p boj -l kt
```

플랫폼 추가 시 현재 활성 언어별 파일명·템플릿을 묻습니다. 일반 기본 파일명은 언어
메타데이터에서 가져오며 C++ `main.cpp`, Python `main.py`, Java `Main.java`, Rust
`main.rs`, 사용자 언어 `main.<확장자>`입니다. 위 예는
`<root>/baekjoon/1000/Main.kt`를 만듭니다. 사용자 플랫폼 문제 ID에는 영문자·숫자로
시작하는 영문자, 숫자, `-`, `_`를 허용합니다. 기존 파일은 덮어쓰지 않고 엽니다.

ct는 심사 사이트의 언어 지원 여부를 제한하거나 추측하지 않습니다. 사용자 플랫폼은
언어별 doctor/setup 검사에 별도 시스템 설치 동작을 추가하지 않습니다.

```sh
ct platform show boj
ct platform disable baekjoon
ct platform enable boj
ct platform remove baekjoon
```

언어·플랫폼 모두 비활성 상태에서는 새 생성과 새 기본값 선택에 사용할 수 없습니다.
저장된 기본값을 비활성화·삭제해도 설정을 자동으로 바꾸지 않습니다. 문제 생성 시
`-p`/`-l`로 활성 항목을 지정하거나, 해당 항목을 재활성화하거나 `ct config`에서
기본값을 바꾸세요. `ct config`는 유효하지 않은 현재 기본값을 안내하고 잘못된
선택을 다시 묻습니다. `boj`, `kt` 같은 별칭은 정규 이름으로 저장합니다.

사용자 항목 삭제는 제거할 레지스트리 항목과 ct 소유 템플릿의 범위를 보여준 뒤
`Continue? [y/N]`으로 확인합니다. Enter/EOF는 취소입니다. 삭제·비활성화는
**기존 풀이 파일, 컴파일러, VS Code 확장을 제거하지 않습니다.** 언어·플랫폼 조합에
연결된 검증된 템플릿 파일과 빈 디렉터리만 삭제하며 임의 경로를 재귀 삭제하지 않습니다.

### 레지스트리 저장과 조합

레지스트리는 `config.json`과 같은 디렉터리의 `registry.json`에 저장합니다.
Windows `%APPDATA%\ct`, Linux `${XDG_CONFIG_HOME:-$HOME/.config}/ct`가 기본이며,
비어 있지 않은 `CT_CONFIG_DIR`가 우선합니다. 기존 설정만 있는 사용자는 이 파일을
미리 만들 필요가 없습니다. 조회는 파일을 만들지 않고 최초 변경 명령에서 저장합니다.

```text
<ct-config-dir>/
  config.json
  registry.json
  templates/languages/kotlin/codeforces.tmpl
  templates/languages/kotlin/programmers.tmpl
  templates/platforms/baekjoon/kotlin.tmpl
```

스키마 버전은 `1`이며 `languages`, `platforms`, `disabled_languages`,
`disabled_platforms`, `bindings`를 저장합니다. 내장 정의 자체는 JSON에 복제하지 않습니다.
각 binding은 `platform`, `language`, `filename`, 설정 디렉터리 기준 상대 `template`
경로를 연결합니다. 예를 들어 사용자 플랫폼의 Kotlin 바인딩은 다음과 같습니다.

```json
{
  "platform": "baekjoon",
  "language": "kotlin",
  "filename": "Main.kt",
  "template": "templates/platforms/baekjoon/kotlin.tmpl"
}
```

추가할 당시 비활성이어서 바인딩이 없던 조합은 재활성화 후 기본 파일명과 빈 내용으로
생성할 수 있습니다. 사용자 바인딩의 파일명은 `registry.json`에서, 템플릿 내용은
표시된 `.tmpl` 파일에서 수정할 수 있습니다. 내장 조합의 정의·별칭을 사용자 항목으로
덮어쓸 수 없습니다. 템플릿 경로는 위의 ct 소유 구조만 허용합니다.

레지스트리는 동일 디렉터리의 임시 파일을 쓰고 flush·close한 뒤 교체합니다.
동시 갱신은 잠금과 원본 비교로 충돌을 알리며 이전 데이터를 조용히 덮어쓰지 않습니다.
손상된 JSON은 경로와 복구 안내를 출력하고 보존합니다. 강제 종료로 `.registry.lock`이
남았다면 다른 ct 갱신이 실행 중이지 않은지 확인한 뒤 해당 잠금 파일만 제거하세요.
머신별 로컬 데이터이며 import/export, 동기화, 원격 레지스트리와 플러그인 실행은 지원하지 않습니다.

## 개발 및 릴리스

소스에서 개발할 때만 `go.mod`에 명시된 Go가 필요합니다. 일반 개발 빌드의
`ct --version`은 `ct dev`를 출력합니다. 릴리스는 소스를 수정하지 않고
`-ldflags "-X github.com/GIRIBUIN/ct/internal/version.Version=<tag>"`로 버전을 주입합니다.

유지관리자가 검증한 커밋에 태그를 만들어 푸시하면 됩니다:

```sh
git tag v0.1.0
git push origin v0.1.0
```

`.github/workflows/release.yml`은 `v*` 태그에만 실행됩니다. 태그의 정확한 소스를
체크아웃하고 `go.mod` 버전으로 Windows/Linux 테스트와 vet를 실행합니다. 모두 성공하면
`CGO_ENABLED=0`으로 세 바이너리를 만들고 SHA-256 목록과 함께 같은 태그의 Release에
게시합니다. 프리릴리스 태그는 prerelease로 게시하여 최신 정식 설치 대상에서 제외합니다.
테스트·빌드가 실패하면 게시하지 않습니다. 게시에는 공식 Actions와
[GitHub CLI의 `gh release create --verify-tag`](https://cli.github.com/manual/gh_release_create)를 사용합니다.

자산 이름은 고정입니다:

```text
ct-windows-amd64.exe
ct-linux-amd64
ct-linux-arm64
checksums.txt
```

일반 브랜치 push/PR은 별도의 `ci.yml`에서 Windows/Linux 테스트, vet, 빌드와
배포 스크립트 테스트만 수행합니다. 배포하지 않습니다. `ct update`, `ct uninstall`
명령은 제공하지 않으며 업데이트·제거는 외부 스크립트를 사용합니다.

로컬 검증:

```sh
go test ./...
go vet ./...
go build ./cmd/ct
git diff --check
```

PowerShell 스크립트 테스트는 `scripts/test-distribution.ps1`, Linux 테스트는
`bash scripts/test-distribution.sh`입니다. 테스트는 임시 디렉터리, 모의 다운로드,
모의 User PATH를 사용하며 실제 사용자 설정이나 풀이 디렉터리를 변경하지 않습니다.
스크립트 테스트에는 테스트용 실행 파일을 빌드하기 위한 Go가 필요합니다.

`CT_TEMPLATE_SMOKE=1 go test ./internal/template -v`는 이미 설치된 javac/rustc로
Java 17·Rust 2021 템플릿을 컴파일합니다(PowerShell은 환경변수를 먼저 설정).
일반 CI는 이 선택적 검사를 요구하지 않으며 도구체인을 설치하지 않습니다.
