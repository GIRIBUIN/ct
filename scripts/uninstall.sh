#!/usr/bin/env bash

ct_remove_plain_path() {
    local cursor=$1
    [[ $cursor == /* ]] || { echo "Expected absolute path: $cursor" >&2; return 1; }
    while [[ $cursor != / ]]; do
        [[ ! -L $cursor ]] || { echo "Refusing linked path: $cursor" >&2; return 1; }
        cursor=$(dirname -- "$cursor")
    done
}

ct_canonical_config() {
    local base=${XDG_CONFIG_HOME:-${HOME:?}/.config}
    [[ $base == /* ]] || { echo 'Config base must be an absolute path.' >&2; return 1; }
    printf '%s/ct\n' "${base%/}"
}

ct_remove_profile() {
    local profile=$1 content block updated temp
    ct_remove_plain_path "$profile" || return 1
    [[ -e $profile ]] || return 0
    [[ -f $profile ]] || { echo "Not a regular profile: $profile" >&2; return 1; }
    content=$(cat -- "$profile"; printf '.'); content=${content%.}
    block=$'\n# >>> ct PATH >>>\ncase ":$PATH:" in\n    *":$HOME/.local/bin:"*) ;;\n    *) export PATH="$PATH:$HOME/.local/bin" ;;\nesac\n# <<< ct PATH <<<\n'
    if [[ $content != *"$block"* ]]; then
        if [[ $content == *'# >>> ct PATH >>>'* || $content == *'# <<< ct PATH <<<'* ]]; then
            echo 'Edited ct PATH block left untouched; remove it manually if desired.'
        fi
        return 0
    fi
    updated=${content/"$block"/}
    temp=$(mktemp "${profile}.ct-remove.XXXXXX") || return 1
    if ! { printf '%s' "$updated" > "$temp" && chmod --reference="$profile" "$temp" && mv -fT -- "$temp" "$profile"; }; then
        rm -f -- "$temp"
        return 1
    fi
}

ct_uninstall() (
    set -euo pipefail
    local purge=false config target
    case "$#:${1:-}" in
        0:) ;;
        1:--purge) purge=true ;;
        *) echo 'Usage: uninstall.sh [--purge]' >&2; exit 1 ;;
    esac
    [[ $(uname -s) == Linux ]] || { echo 'This uninstaller requires Linux.' >&2; exit 1; }
    [[ ${HOME:-} == /* && $HOME != / ]] || { echo 'HOME must be a non-root absolute path.' >&2; exit 1; }
    target=$HOME/.local/bin/ct
    ct_remove_plain_path "$target"
    [[ ! -e $target || -f $target ]] || { echo "Not a regular file: $target" >&2; exit 1; }
    rm -f -- "$target"
    ct_remove_profile "$HOME/.profile"
    if $purge; then
        if [[ -n ${CT_CONFIG_DIR:-} ]]; then echo 'CT_CONFIG_DIR is not a purge target; only canonical config.json is eligible.'; fi
        config=$(ct_canonical_config)
        ct_remove_plain_path "$config/config.json"
        [[ ! -e $config/config.json || -f $config/config.json ]] || { echo 'config.json is not a regular file; preserved.' >&2; exit 1; }
        rm -f -- "$config/config.json"
        if [[ -d $config ]]; then
            rmdir -- "$config" 2>/dev/null || echo "Preserved nonempty configuration directory: $config"
        fi
    fi
    echo 'ct removed. Solution files and development tools were preserved.'
    if ! $purge; then echo 'ct configuration was preserved (use --purge to remove canonical config.json).'; fi
    echo 'Start a new shell to refresh PATH; other entries in ~/.local/bin are unchanged.'
)

if [[ ${BASH_SOURCE[0]:-} == "$0" || -z ${BASH_SOURCE[0]:-} ]]; then ct_uninstall "$@"; fi
