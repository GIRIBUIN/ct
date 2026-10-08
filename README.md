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

기본 제거는 설정을 보존합니다. purge도 **기본 설정 위치의 `config.json`과 비어 있는
ct 디렉터리만** 제거합니다. Windows는 `%APPDATA%\ct`, Linux는
`$XDG_CONFIG_HOME/ct` 또는 `~/.config/ct`입니다. `CT_CONFIG_DIR`는 purge 대상 선택에
사용하지 않으며 별도 경로는 보존한다고 안내합니다. 기본 위치에 다른 파일이 있으면
디렉터리를 보존합니다. 심볼릭 링크·Windows junction을 따라 삭제하지 않습니다.
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

템플릿은 `go:embed`로 실행 파일에 포함됩니다. Codeforces는 `solve()` 기반이며,
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
