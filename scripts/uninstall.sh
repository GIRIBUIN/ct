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

# Only the fixed ct template layout is eligible. Unknown files and linked paths
# are preserved; no registry value is interpreted as a deletion path.
ct_remove_templates() {
    local base=$1/templates kind owner file name
    ct_remove_plain_path "$base" || return 1
    [[ -d $base ]] || return 0
    for kind in languages platforms; do
        ct_remove_plain_path "$base/$kind" || return 1
        [[ -d $base/$kind ]] || continue
        for owner in "$base/$kind"/*; do
            [[ -e $owner || -L $owner ]] || continue
            name=${owner##*/}
            [[ $name =~ ^[a-z][a-z0-9_-]*$ ]] || continue
            ct_remove_plain_path "$owner" || return 1
            [[ -d $owner ]] || continue
            for file in "$owner"/*.tmpl; do
                [[ -e $file || -L $file ]] || continue
                name=${file##*/}
                [[ $name =~ ^[a-z][a-z0-9_-]*\.tmpl$ ]] || continue
                ct_remove_plain_path "$file" || return 1
                [[ -f $file ]] || continue
                rm -f -- "$file" || return 1
            done
            rmdir -- "$owner" 2>/dev/null || true
        done
        rmdir -- "$base/$kind" 2>/dev/null || true
    done
    rmdir -- "$base" 2>/dev/null || true
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
        if [[ -n ${CT_CONFIG_DIR:-} ]]; then echo 'CT_CONFIG_DIR is not a purge target; only canonical ct configuration/registry/templates are eligible.'; fi
        config=$(ct_canonical_config)
        for target in config.json registry.json; do
            ct_remove_plain_path "$config/$target"
            [[ ! -e $config/$target || -f $config/$target ]] || { echo "$target is not a regular file; preserved." >&2; exit 1; }
        done
        ct_remove_templates "$config"
        rm -f -- "$config/config.json" "$config/registry.json"
        if [[ -d $config ]]; then
            rmdir -- "$config" 2>/dev/null || echo "Preserved nonempty configuration directory: $config"
        fi
    fi
    echo 'ct removed. Solution files and development tools were preserved.'
    if ! $purge; then echo 'ct configuration, registry and templates were preserved (use --purge to remove canonical ct data).'; fi
    echo 'Start a new shell, or run: hash -r. Other entries in ~/.local/bin are unchanged.'
)

if [[ ${BASH_SOURCE[0]:-} == "$0" || -z ${BASH_SOURCE[0]:-} ]]; then ct_uninstall "$@"; fi
