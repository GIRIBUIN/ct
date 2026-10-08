#!/usr/bin/env bash
set -euo pipefail
scripts=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
for script in install.sh uninstall.sh test-distribution.sh; do bash -n "$scripts/$script"; done
source "$scripts/install.sh"
source "$scripts/uninstall.sh"
tty_opener=$(declare -f ct_open_onboarding_tty)
# Baseline install tests must never prompt, even when launched from a developer terminal.
ct_open_onboarding_tty() { return 1; }
test_root=$(mktemp -d /tmp/ct-dist-test.XXXXXX)
trap 'case "$test_root" in /tmp/ct-dist-test.*) rm -rf -- "$test_root" ;; *) exit 1 ;; esac' EXIT
export HOME=$test_root/home
export XDG_CONFIG_HOME=$test_root/config
export CT_CONFIG_DIR=$test_root/solutions
mkdir -p "$HOME" "$XDG_CONFIG_HOME/ct" "$CT_CONFIG_DIR" "$test_root/release"
printf '#!/bin/sh\nprintf "ct fixture\\n"\n' > "$test_root/release/binary"
asset=$(ct_asset Linux "$(uname -m)")
hash=$(sha256sum "$test_root/release/binary"); hash=${hash%% *}
printf '%s  %s\n' "$hash" "$asset" > "$test_root/release/checksums.txt"
ct_download() {
    case "$1" in
        https://github.com/GIRIBUIN/ct/releases/latest/download/checksums.txt) cp "$test_root/release/checksums.txt" "$2" ;;
        "https://github.com/GIRIBUIN/ct/releases/latest/download/$asset") cp "$test_root/release/binary" "$2" ;;
        *) return 1 ;;
    esac
}
[[ $(ct_asset Linux x86_64) == ct-linux-amd64 ]]
[[ $(ct_asset Linux aarch64) == ct-linux-arm64 ]]
! ct_asset Darwin x86_64
! ct_asset Linux riscv64
[[ $(ct_canonical_config) == "$XDG_CONFIG_HOME/ct" ]]
[[ $(XDG_CONFIG_HOME='' ct_canonical_config) == "$HOME/.config/ct" ]]
! (XDG_CONFIG_HOME=relative ct_canonical_config)
ct_verify_checksum "$test_root/release/binary" "$test_root/release/checksums.txt" "$asset"
printf '%s  %s.extra\n' "$hash" "$asset" > "$test_root/wrong"
! ct_verify_checksum "$test_root/release/binary" "$test_root/wrong" "$asset"
cat "$test_root/release/checksums.txt" "$test_root/release/checksums.txt" > "$test_root/wrong"
! ct_verify_checksum "$test_root/release/binary" "$test_root/wrong" "$asset"

# Byte-preserving removal includes profiles whose original last line has no newline.
printf '# original\nexport ORIGINAL=yes' > "$HOME/.profile"
cp "$HOME/.profile" "$test_root/original"
(export PATH="$PATH:$HOME/.local/bin"; ct_install; cmp "$HOME/.profile" "$test_root/original")
ct_install
ct_install
[[ $(grep -c '^# >>> ct PATH >>>$' "$HOME/.profile") == 1 ]]
[[ -x $HOME/.local/bin/ct ]]
cp "$HOME/.local/bin/ct" "$test_root/installed"
cp "$HOME/.profile" "$test_root/profile-installed"
printf '%064d  %s\n' 0 "$asset" > "$test_root/release/checksums.txt"
# Run as a standalone shell so set -e is not suppressed by a !/if function call.
export -f ct_asset ct_plain_path ct_verify_checksum ct_path_block ct_add_profile ct_download ct_install ct_onboard ct_next_steps ct_open_onboarding_tty
export test_root asset
if bash -c ct_install; then echo 'Mismatch accepted' >&2; exit 1; fi
cmp "$HOME/.local/bin/ct" "$test_root/installed"
cmp "$HOME/.profile" "$test_root/profile-installed"

# Onboarding fixtures log only config/doctor; they never run the real ct or development tools.
cat > "$test_root/release/binary" <<'FIXTURE'
#!/usr/bin/env bash
if [[ $1 == --version ]]; then echo 'ct fixture'; exit 0; fi
[[ $0 == "$HOME/.local/bin/ct" && -z ${CT_CONFIG_DIR:-} ]] || exit 90
if [[ ${CT_TEST_REQUIRE_TTY:-0} == 1 ]]; then [[ -t 0 && -t 1 ]] || exit 91; fi
printf '%s\n' "$1" >> "$CT_TEST_CALLS"
case "$1" in
    config) exit "${CT_TEST_CONFIG_EXIT:-0}" ;;
    doctor) exit "${CT_TEST_DOCTOR_EXIT:-0}" ;;
    *) exit 92 ;;
esac
FIXTURE
hash=$(sha256sum "$test_root/release/binary"); hash=${hash%% *}
printf '%s  %s\n' "$hash" "$asset" > "$test_root/release/checksums.txt"
export CT_TEST_CALLS=$test_root/onboarding-calls
export CT_TEST_TARGET=$HOME/.local/bin/ct
printf 'override preserved' > "$CT_CONFIG_DIR/config.json"
ct_install > "$test_root/noninteractive-output"
[[ ! -e $CT_TEST_CALLS ]]
grep -q 'ct setup --dry-run' "$test_root/noninteractive-output"
ct_open_onboarding_tty() { exec 3<>"$test_root/fake-tty"; }
for scenario in fresh config-failed doctor-failed existing; do
    export CT_TEST_CONFIG_EXIT=0 CT_TEST_DOCTOR_EXIT=0
    : > "$CT_TEST_CALLS"
    case "$scenario" in
        config-failed) export CT_TEST_CONFIG_EXIT=1 ;;
        doctor-failed) export CT_TEST_DOCTOR_EXIT=1 ;;
        existing) printf 'existing preserved' > "$XDG_CONFIG_HOME/ct/config.json" ;;
    esac
    ct_install > "$test_root/onboarding-output"
    expected=$'config\ndoctor'
    case "$scenario" in config-failed) expected=config ;; existing) expected='' ;; esac
    [[ $(cat "$CT_TEST_CALLS") == "$expected" && -x $CT_TEST_TARGET ]]
    [[ $CT_CONFIG_DIR == "$test_root/solutions" ]]
    if [[ $scenario == doctor-failed ]]; then grep -q 'ct setup --dry-run' "$test_root/onboarding-output"; fi
done
[[ $(cat "$XDG_CONFIG_HOME/ct/config.json") == 'existing preserved' ]]
[[ $(cat "$CT_CONFIG_DIR/config.json") == 'override preserved' ]]
rm -- "$XDG_CONFIG_HOME/ct/config.json"

# A real controlling PTY with piped stdin verifies /dev/tty routing (util-linux on CI).
eval "$tty_opener"
export -f ct_open_onboarding_tty
export CT_TEST_REQUIRE_TTY=1
: > "$CT_TEST_CALLS"
printf '%s\n' 'printf '\''ct_onboard "$CT_TEST_TARGET"\n'\'' | bash' > "$test_root/tty-driver"
script -q -e -c "bash $test_root/tty-driver" /dev/null < /dev/null > "$test_root/tty-output"
[[ $(cat "$CT_TEST_CALLS") == $'config\ndoctor' ]]
printf '{}' > "$XDG_CONFIG_HOME/ct/config.json"
printf 'solution' > "$CT_CONFIG_DIR/main.cpp"
printf 'sibling solution' > "$XDG_CONFIG_HOME/ct/main.cpp"
ct_uninstall
[[ ! -e $HOME/.local/bin/ct && -f $XDG_CONFIG_HOME/ct/config.json ]]
cmp "$HOME/.profile" "$test_root/original"
ct_uninstall --purge
[[ ! -e $XDG_CONFIG_HOME/ct/config.json && -f $XDG_CONFIG_HOME/ct/main.cpp && -f $CT_CONFIG_DIR/main.cpp ]]
printf '# >>> ct PATH >>>\n# user edited\n# <<< ct PATH <<<\n' >> "$HOME/.profile"
cp "$HOME/.profile" "$test_root/edited"
ct_remove_profile "$HOME/.profile"
cmp "$HOME/.profile" "$test_root/edited"
ln -s "$CT_CONFIG_DIR" "$test_root/link"
! ct_remove_plain_path "$test_root/link/main.cpp"
echo 'Linux distribution tests passed (temporary HOME, mocked downloads).'
