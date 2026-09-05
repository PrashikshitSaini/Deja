#!/usr/bin/env zsh

set -u

typeset test_root
typeset script_dir="${${(%):-%N}:A:h}"
test_root="$(command mktemp -d)"
function cleanup() {
  command rm -rf -- "${test_root}" 2>/dev/null
}
trap cleanup EXIT HUP INT TERM

export TMPDIR="${test_root}"
export DEJA_BIN="${commands[true]}"
unset HISTFILE

command mkdir "${test_root}/deja-${UID}"
print -r -- untouched > "${test_root}/deja-${UID}/marker"
source "${script_dir}/deja.zsh" || exit 1

[[ "${_deja_state_dir}" == "${test_root}/deja-${UID}."* ]] || exit 1
[[ "${_deja_state_dir}" != "${test_root}/deja-${UID}" ]] || exit 1
[[ "${_deja_results_file:h}" == "${_deja_state_dir}" ]] || exit 1

typeset -A details
zstat -H details "${_deja_state_dir}" || exit 1
(( details[uid] == EUID )) || exit 1
(( (details[mode] & 8#777) == 8#700 )) || exit 1

typeset state_dir="${_deja_state_dir}"
_deja_zshexit
[[ ! -e "${state_dir}" ]] || exit 1
[[ "$(<"${test_root}/deja-${UID}/marker")" == untouched ]] || exit 1

print -r -- "private runtime state: ok"
