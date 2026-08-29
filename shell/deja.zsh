# Deja's live command-variant palette for Zsh.
# Source this file from an interactive .zshrc. Deja never binds Enter and never
# invokes accept-line; choosing a candidate only replaces the editable buffer.

[[ -o interactive ]] || return 0
[[ "${_deja_loaded_pid:-}" == "$$" ]] && return 0

typeset -g DEJA_ROOT="${${(%):-%N}:A:h:h}"

# Carry existing pre-rename settings forward. DEJA_* always wins when both are
# present; new installations should use DEJA_* exclusively.
[[ -z "${DEJA_BIN:-}" && -n "${DEZA_BIN:-}" ]] && typeset -gx DEJA_BIN="${DEZA_BIN}"
[[ -z "${DEJA_CONFIG:-}" && -n "${DEZA_CONFIG:-}" ]] && typeset -gx DEJA_CONFIG="${DEZA_CONFIG}"
[[ -z "${DEJA_STORE:-}" && -n "${DEZA_STORE:-}" ]] && typeset -gx DEJA_STORE="${DEZA_STORE}"
[[ -z "${DEJA_LIMIT:-}" && -n "${DEZA_LIMIT:-}" ]] && typeset -gx DEJA_LIMIT="${DEZA_LIMIT}"

if [[ -z "${DEJA_BIN:-}" ]]; then
  if [[ -x "${DEJA_ROOT}/bin/deja" ]]; then
    typeset -g DEJA_BIN="${DEJA_ROOT}/bin/deja"
  elif (( ${+commands[deja]} )); then
    typeset -g DEJA_BIN="${commands[deja]}"
  else
    typeset -g DEJA_BIN="${DEJA_ROOT}/bin/deja"
  fi
else
  typeset -g DEJA_BIN
fi
typeset -g DEJA_LIMIT="${DEJA_LIMIT:-}"
if [[ -z "${DEJA_CONFIG:-}" && -r "${DEJA_ROOT}/deja.json" ]]; then
  typeset -gx DEJA_CONFIG="${DEJA_ROOT}/deja.json"
fi
typeset -g _deja_pending_command=""
typeset -g _deja_pending_cwd=""
typeset -gF _deja_pending_started=0
typeset -g _deja_last_buffer=""
typeset -gi _deja_selected=1
typeset -gi _deja_visible_rows=6
typeset -gi _deja_window_start=1
typeset -gi _deja_window_end=1
typeset -gi _deja_refreshing=0
typeset -gi _deja_suppressed=0
typeset -ga _deja_candidate_lines=()

if [[ ! -x "${DEJA_BIN}" ]]; then
  print -u2 -- "Deja: ${DEJA_BIN} is not executable"
  return 1
fi
if ! "${DEJA_BIN}" config check >/dev/null 2>&1; then
  print -u2 -- "Deja: invalid configuration at ${DEJA_CONFIG:-$("${DEJA_BIN}" config path)}"
  print -u2 -- "Run: ${DEJA_BIN} config check"
  return 1
fi

# Atomically create unique per-session state, then verify its ownership and
# permissions before allowing a results file to influence buffer insertion.
zmodload zsh/stat 2>/dev/null || {
  print -u2 -- "Deja: zsh/stat is required for private runtime state"
  return 1
}
typeset _deja_new_state_dir
_deja_new_state_dir="$(command mktemp -d "${TMPDIR:-/tmp}/deja-${UID}.XXXXXXXX" 2>/dev/null)" || {
  print -u2 -- "Deja: could not create private runtime state"
  return 1
}
command chmod 700 "${_deja_new_state_dir}" 2>/dev/null || {
  command rmdir "${_deja_new_state_dir}" 2>/dev/null
  print -u2 -- "Deja: could not secure private runtime state"
  return 1
}
typeset -A _deja_state_details
zstat -H _deja_state_details "${_deja_new_state_dir}" 2>/dev/null
if (( _deja_state_details[uid] != EUID || (_deja_state_details[mode] & 8#777) != 8#700 )); then
  command rmdir "${_deja_new_state_dir}" 2>/dev/null
  print -u2 -- "Deja: private runtime state failed ownership or permission checks"
  return 1
fi
unset _deja_state_details
typeset -gr _deja_state_dir="${_deja_new_state_dir}"
unset _deja_new_state_dir
typeset -gr _deja_results_file="${_deja_state_dir}/results.json"

zmodload zsh/datetime 2>/dev/null || true
zmodload zsh/terminfo 2>/dev/null || true
autoload -Uz add-zsh-hook add-zle-hook-widget

function _deja_clear_palette() {
  _deja_candidate_lines=()
  _deja_selected=1
  _deja_window_start=1
  _deja_window_end=1
  zle -M -- ""
}

function _deja_update_window() {
  emulate -L zsh
  local count=${#_deja_candidate_lines}
  local visible=${_deja_visible_rows}
  (( visible < 1 )) && visible=1
  (( visible > count && count > 0 )) && visible=count

  if (( count == 0 )); then
    _deja_window_start=1
    _deja_window_end=0
    return
  fi

  (( _deja_selected < _deja_window_start )) && _deja_window_start=${_deja_selected}
  (( _deja_selected > _deja_window_start + visible - 1 )) && \
    _deja_window_start=$(( _deja_selected - visible + 1 ))

  local max_start=$(( count - visible + 1 ))
  (( _deja_window_start < 1 )) && _deja_window_start=1
  (( _deja_window_start > max_start )) && _deja_window_start=${max_start}
  _deja_window_end=$(( _deja_window_start + visible - 1 ))
}

function _deja_render_palette() {
  emulate -L zsh
  local count=${#_deja_candidate_lines}
  if (( count == 0 )); then
    zle -M -- ""
    return
  fi

  _deja_update_window
  local position="${_deja_window_start}-${_deja_window_end} of ${count} variants"
  (( count <= _deja_visible_rows )) && position="${count} distinct variants"
  local message="Deja · ${position} · ↑/↓ scroll · Tab insert"
  local index prefix
  for (( index = _deja_window_start; index <= _deja_window_end; index++ )); do
    if (( index == _deja_selected )); then
      prefix='› '
    else
      prefix='  '
    fi
    message+=$'\n'"${prefix}${_deja_candidate_lines[index]}"
  done
  zle -M -- "${message}"
}

function _deja_refresh_palette() {
  emulate -L zsh
  setopt localoptions noshwordsplit
  (( _deja_refreshing )) && return
  [[ "${BUFFER}" == "${_deja_last_buffer}" ]] && return

  if (( _deja_suppressed )); then
    _deja_suppressed=0
  fi

  _deja_refreshing=1
  {
    _deja_last_buffer="${BUFFER}"
    _deja_selected=1
    _deja_window_start=1

    if [[ -z "${BUFFER//[[:space:]]/}" ]]; then
      _deja_clear_palette
      return
    fi

    local output
    local row_width=$(( COLUMNS > 8 ? COLUMNS - 4 : 0 ))
    local -a query_arguments
    query_arguments=(
      query --query-stdin --cwd "${PWD}" --results-file "${_deja_results_file}"
      --format zle --color auto --width "${row_width}" --zle-meta
    )
    [[ -n "${DEJA_LIMIT}" ]] && query_arguments+=(--visible-rows "${DEJA_LIMIT}")
    output="$(
      print -rn -- "${BUFFER}" |
        "${DEJA_BIN}" "${query_arguments[@]}" 2>/dev/null
    )"
    if [[ -n "${output}" ]]; then
      local -a output_lines
      output_lines=("${(@f)output}")
      if [[ "${output_lines[1]}" == $'__DEJA_META__\t'<-> ]]; then
        _deja_visible_rows="${output_lines[1]##*$'\t'}"
        _deja_candidate_lines=("${output_lines[@]:1}")
      else
        _deja_candidate_lines=("${output_lines[@]}")
      fi
    else
      _deja_candidate_lines=()
    fi
    _deja_render_palette
  } always {
    _deja_refreshing=0
  }
}

function _deja_line_pre_redraw() {
  _deja_refresh_palette
}

function _deja_line_init() {
  _deja_suppressed=0
  _deja_last_buffer=$'\x1f'
  _deja_refresh_palette
}

function _deja_up_or_history() {
  if (( ${#_deja_candidate_lines} )); then
    (( _deja_selected-- ))
    (( _deja_selected < 1 )) && _deja_selected=${#_deja_candidate_lines}
    _deja_render_palette
  else
    zle .up-line-or-history
    _deja_suppressed=1
    _deja_last_buffer="${BUFFER}"
    _deja_clear_palette
  fi
}

function _deja_down_or_history() {
  if (( ${#_deja_candidate_lines} )); then
    (( _deja_selected++ ))
    (( _deja_selected > ${#_deja_candidate_lines} )) && _deja_selected=1
    _deja_render_palette
  else
    zle .down-line-or-history
    _deja_suppressed=1
    _deja_last_buffer="${BUFFER}"
    _deja_clear_palette
  fi
}

function _deja_insert_selection() {
  if (( ! ${#_deja_candidate_lines} )); then
    zle .expand-or-complete
    return
  fi

  local selection
  selection="$(
    "${DEJA_BIN}" pick --results-file "${_deja_results_file}" \
      --index "${_deja_selected}" 2>/dev/null
  )" || {
    zle beep
    return 1
  }

  BUFFER="${selection}"
  CURSOR=${#BUFFER}
  _deja_suppressed=1
  _deja_last_buffer="${BUFFER}"
  _deja_clear_palette
}

function _deja_preexec() {
  if [[ "$1" == [[:space:]]* ]]; then
    _deja_pending_command=""
    return
  fi
  _deja_pending_command="$1"
  _deja_pending_cwd="${PWD}"
  _deja_pending_started=${EPOCHREALTIME:-0}
}

function _deja_precmd() {
  local command_status=$?
  if [[ -n "${_deja_pending_command}" ]]; then
    local now=${EPOCHREALTIME:-0}
    local occurred_at=${_deja_pending_started:-0}
    local duration=0
    (( _deja_pending_started > 0 && now > 0 )) && duration=$(( now - _deja_pending_started ))
    (( occurred_at <= 0 )) && occurred_at=${now}
    print -rn -- "${_deja_pending_command}" |
      "${DEJA_BIN}" record --stdin --cwd "${_deja_pending_cwd}" \
        --exit-status "${command_status}" --timestamp "${occurred_at}" --duration "${duration}" \
        >/dev/null 2>&1
    _deja_pending_command=""
  fi
}

function _deja_zshexit() {
  local -A details
  zstat -H details "${_deja_state_dir}" 2>/dev/null || return 0
  (( details[uid] == EUID && (details[mode] & 8#777) == 8#700 )) || return 0
  [[ "${_deja_state_dir:t}" == "deja-${UID}."* ]] || return 0
  command rm -f -- "${_deja_state_dir}/results.json" 2>/dev/null
  command rmdir "${_deja_state_dir}" 2>/dev/null
}

zle -N deja-up-or-history _deja_up_or_history
zle -N deja-down-or-history _deja_down_or_history
zle -N deja-insert-selection _deja_insert_selection

for _deja_keymap in main emacs viins; do
  bindkey -M "${_deja_keymap}" '^[[A' deja-up-or-history
  bindkey -M "${_deja_keymap}" '^[OA' deja-up-or-history
  bindkey -M "${_deja_keymap}" '^[[B' deja-down-or-history
  bindkey -M "${_deja_keymap}" '^[OB' deja-down-or-history
  if [[ -n "${terminfo[kcuu1]:-}" ]]; then
    bindkey -M "${_deja_keymap}" "${terminfo[kcuu1]}" deja-up-or-history
  fi
  if [[ -n "${terminfo[kcud1]:-}" ]]; then
    bindkey -M "${_deja_keymap}" "${terminfo[kcud1]}" deja-down-or-history
  fi
  bindkey -M "${_deja_keymap}" '^I' deja-insert-selection
done
unset _deja_keymap

add-zle-hook-widget line-pre-redraw _deja_line_pre_redraw
add-zle-hook-widget line-init _deja_line_init
add-zsh-hook preexec _deja_preexec
add-zsh-hook precmd _deja_precmd
add-zsh-hook zshexit _deja_zshexit
typeset -gr _deja_loaded_pid="$$"

# Capture command status before other precmd hooks can replace it.
precmd_functions=(_deja_precmd ${precmd_functions:#_deja_precmd})

if [[ -n "${HISTFILE:-}" && -r "${HISTFILE}" ]]; then
  "${DEJA_BIN}" import --history-file "${HISTFILE}" >/dev/null 2>&1 &!
fi
