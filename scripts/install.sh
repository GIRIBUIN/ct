#!/usr/bin/env bash

ct_asset() {
    [[ $1 == Linux ]] || { echo 'ct requires Linux.' >&2; return 1; }
    case "$2" in
        x86_64|amd64) printf '%s\n' ct-linux-amd64 ;;
        aarch64|arm64) printf '%s\n' ct-linux-arm64 ;;
        *) echo "Unsupported Linux architecture: $2" >&2; return 1 ;;
    esac
}

ct_plain_path() {
    local cursor=$1
    [[ $cursor == /* ]] || { echo "Expected absolute path: $cursor" >&2; return 1; }
    while [[ $cursor != / ]]; do
        [[ ! -L $cursor ]] || { echo "Refusing linked path: $cursor" >&2; return 1; }
        cursor=$(dirname -- "$cursor")
    done
}

ct_verify_checksum() {
    local binary=$1 manifest=$2 asset=$3 line hash='' count=0 actual
    while IFS= read -r line || [[ -n $line ]]; do
        line=${line%$'\r'}
        if [[ $line =~ ^([[:xdigit:]]{64})\ [\ \*](.+)$ && ${BASH_REMATCH[2]} == "$asset" ]]; then
            hash=${BASH_REMATCH[1],,}
            count=$((count + 1))
        fi
    done < "$manifest"
    [[ $count == 1 ]] || { echo "Expected exactly one checksum for $asset." >&2; return 1; }
    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum -- "$binary") || return 1
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 -- "$binary") || return 1
    else
        echo 'Install sha256sum or shasum to verify this download.' >&2; return 1
    fi
    actual=${actual%% *}
    [[ ${actual,,} == "$hash" ]] || { echo "SHA-256 mismatch for $asset; installation unchanged." >&2; return 1; }
}

ct_path_block() {
    printf '\n# >>> ct PATH >>>\ncase ":$PATH:" in\n    *":$HOME/.local/bin:"*) ;;\n    *) export PATH="$PATH:$HOME/.local/bin" ;;\nesac\n# <<< ct PATH <<<\n'
}

ct_add_profile() {
    local profile=$1 content='' block
    ct_plain_path "$profile" || return 1
    [[ ! -e $profile || -f $profile ]] || { echo "Not a regular profile: $profile" >&2; return 1; }
    if [[ -f $profile ]]; then content=$(cat -- "$profile"; printf '.'); content=${content%.}; fi
    block=$(ct_path_block; printf '.'); block=${block%.}
    if [[ $content == *'# >>> ct PATH >>>'* || $content == *'# <<< ct PATH <<<'* ]]; then
        [[ $content == *"$block"* ]] || { echo 'Existing ct PATH marker was edited; update ~/.profile manually.' >&2; return 1; }
        return 0
    fi
    printf '%s' "$block" >> "$profile"
}

ct_download() {
    curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' "$1" --output "$2"
}

ct_next_steps() {
    printf 'Next:\n  ct config\n  ct doctor\n  ct setup --dry-run\n'
}

ct_open_onboarding_tty() {
    # stdin may contain the installer itself (curl | bash). Open the controlling tty instead.
    { exec 3<>/dev/tty; } 2>/dev/null
}

ct_onboard() (
    local target=$1 base=${XDG_CONFIG_HOME:-$HOME/.config}
    if [[ $base != /* ]]; then
        echo 'ct is installed; canonical configuration location is unavailable.'
        ct_next_steps
        return 0
    fi
    if [[ -e $base/ct/config.json || -L $base/ct/config.json ]]; then return 0; fi
    if ! ct_open_onboarding_tty; then
        ct_next_steps
        return 0
    fi
    # Subshell isolation preserves the caller's override and closes descriptor 3 on return.
    unset CT_CONFIG_DIR
    if ! "$target" config <&3 >&3; then
        echo 'ct remains installed. Configuration was cancelled or failed.'
        ct_next_steps
        return 0
    fi
    if ! "$target" doctor <&3 >&3; then
        printf 'ct is installed, but some coding-test environment checks failed.\nReview:\n  ct setup --dry-run\nInstall:\n  ct setup\n'
    fi
    return 0
)

ct_install() (
    set -euo pipefail
    local asset temp stage='' bin target
    asset=$(ct_asset "$(uname -s)" "$(uname -m)")
    [[ ${HOME:-} == /* && $HOME != / && $HOME != *:* && $HOME != *$'\n'* ]] || { echo 'HOME must be a non-root absolute path without colons/newlines.' >&2; exit 1; }
    command -v curl >/dev/null || { echo 'curl is required.' >&2; exit 1; }
    bin=$HOME/.local/bin
    target=$bin/ct
    ct_plain_path "$target"
    [[ ! -e $target || -f $target ]] || { echo "Not a regular file: $target" >&2; exit 1; }
    temp=$(mktemp -d)
    # Exact files only; never recursively delete a user-controlled directory.
    trap 'rm -f -- "$temp/binary" "$temp/checksums.txt"; rmdir -- "$temp"; if [[ -n $stage ]]; then rm -f -- "$stage"; fi' EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM
    local base=https://github.com/GIRIBUIN/ct/releases/latest/download
    ct_download "$base/$asset" "$temp/binary"
    ct_download "$base/checksums.txt" "$temp/checksums.txt"
    ct_verify_checksum "$temp/binary" "$temp/checksums.txt" "$asset"
    mkdir -p -- "$bin"
    ct_plain_path "$target"
    stage=$(mktemp "$bin/.ct-install.XXXXXX")
    cp -- "$temp/binary" "$stage"
    chmod 755 "$stage"
    "$stage" --version
    mv -fT -- "$stage" "$target"
    stage=''
    case ":$PATH:" in
        *":$bin:"*) ;;
        *) ct_add_profile "$HOME/.profile"
           echo 'PATH entry added to ~/.profile. Run: . "$HOME/.profile" (or start a new login shell).' ;;
    esac
    "$target" --version
    printf 'Installed: %s\n' "$target"
    ct_onboard "$target"
    echo 'Rerun this installer to update ct.'
)

# Sourcing defines helpers for isolated tests. A piped script has no BASH_SOURCE.
if [[ ${BASH_SOURCE[0]:-} == "$0" || -z ${BASH_SOURCE[0]:-} ]]; then ct_install; fi
