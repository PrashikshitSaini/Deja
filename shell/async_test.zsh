#!/usr/bin/env zsh

set -eu
typeset script_dir="${${(%):-%N}:A:h}"
typeset test_root="$(command mktemp -d)"
function cleanup() {
  (( ${+functions[_deja_zshexit]} )) && _deja_zshexit
  command rm -rf -- "${test_root}"
}
trap cleanup EXIT
export TMPDIR="${test_root}"
export DEJA_STORE="${test_root}/history.jsonl"
export DEJA_BIN="${DEJA_TEST_BIN:-${script_dir:h}/bin/deja}"
unset HISTFILE
"${DEJA_BIN}" record --timestamp 100 git status
"${DEJA_BIN}" record --timestamp 200 docker ps
source "${script_dir}/deja.zsh" || exit 1

# Exercise the real pipes/worker, driving the callback deterministically.
# ZLE widget registration happened above; only redraw is stubbed here.
function zle() { return 0 }
zmodload zsh/zselect
function await_result() {
  zselect -r "${_deja_response_fd}" -t 500 || {
    print -u2 -- "query worker did not answer within 5 seconds"
    exit 1
  }
  _deja_query_ready "${_deja_response_fd}"
}
typeset BUFFER="git" COLUMNS=80 CURSOR=0
_deja_refresh_palette
await_result
[[ "${_deja_candidate_lines[*]}" == *git*status* ]]
typeset first_pid="${_deja_worker_pid}"

# A stale response must never be insertable for a changed buffer.
BUFFER="docker"
_deja_refresh_palette
# Ensure the old answer is already waiting before issuing the new request.
zselect -r "${_deja_response_fd}" -t 500
BUFFER="git st"
_deja_refresh_palette
[[ ${#_deja_candidate_lines} == 0 ]]
await_result
[[ ${#_deja_candidate_lines} == 0 ]]
while [[ ${#_deja_candidate_lines} == 0 ]]; do await_result; done
[[ "${_deja_candidate_lines[*]}" == *git*status* ]]
[[ "${_deja_candidate_lines[*]}" != *"docker"* ]]
[[ "${_deja_worker_pid}" == "${first_pid}" ]]
_deja_insert_selection
[[ "${BUFFER}" == "git status" ]]

# A purge invalidates the session index before the next answer.
"${DEJA_BIN}" purge --exact 'git status' --force >/dev/null
BUFFER="git"
_deja_refresh_palette
await_result
[[ ${#_deja_candidate_lines} == 0 ]]

typeset state_dir="${_deja_state_dir}"
_deja_zshexit
[[ ! -e "${state_dir}" ]]
print -r -- "async queries, stale-response rejection, insertion and purge refresh: ok"
