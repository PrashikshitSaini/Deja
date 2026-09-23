# Issue #1: session query worker

Issue: https://github.com/PrashikshitSaini/Deja/issues/1

Version 0.3.2 already escapes terminal control/bidi characters in palette rows
and creates an ownership-verified, mode-0700 runtime directory using `mktemp`.
The remaining performance problem is synchronous full-store parsing during
palette refreshes.

## Proposed behavior

The first nonempty query starts one Go worker for that shell. Zsh writes query
text, current directory, terminal width and viewport size into a private request
file, then sends its numeric ID over a pipe. The worker waits for a 50ms pause
in typing and coalesces queued requests. Searches and history reloads run
outside the line editor. Large pasted commands do not travel through the pipe.

The worker retains a family index in memory between queries. Each search checks
the store identity, size and modification time under its shared lock. Appends,
atomic replacements and purges trigger a reload; missing history clears the
cache, and parse errors return no stale candidates. Configuration is reloaded
for every answered query, preserving redaction and visibility rules.

Completed rows and insertion results are stored under a unique request ID. ZLE
receives a small completion notification through `zle -F`; it accepts results
only for the current request and buffer. A changed buffer immediately clears
the old palette. Tab still only inserts text. Exit closes the pipes, stops and
waits for the worker, and removes the session's transport files.

## Tradeoffs for review

- The index persists for a shell session, not across shell restarts. There is
  no database migration, new dependency, shared daemon or additional durable
  copy of command history.
- Cold queries and queries after history changes still read the complete
  JSONL store. Those reads now happen asynchronously. Index memory scales with
  event count and is allocated separately for each active shell worker.
- Warm searches still aggregate events in matching families. A family with
  many events will cost more than one with few events.
- The standalone `deja query` command retains its existing behavior. Recording
  and importing also retain their existing full-store deduplication behavior.
- The 50ms debounce is a fixed initial choice. The query worker is an internal
  shell transport, not a supported public protocol.

This is a concrete implementation for the design discussion requested by the
maintainer; a durable disk index or incremental append processing can be
evaluated separately if cold-query cost or per-shell memory warrants it.

## Validation

Go tests compare session and uncached results, including family matching,
directory/usage metadata, filtering, redaction, appends, purges, missing stores,
same-size atomic replacements and malformed history. Worker tests cover
coalescing, escaped display, unchanged insertion text, terminal width, viewport
metadata and private-directory validation. Shell tests exercise real worker
pipes, stale responses, Tab insertion, purge refresh and cleanup.
The Python standard-library PTY test checks actual ZLE callbacks: results must
appear without another keystroke, and Tab must insert the displayed command.

Warm-search benchmark on Linux amd64, Intel i7-11800H (one local run):

| Search | Events | Time/query | Allocated/query |
| --- | ---: | ---: | ---: |
| Existing standalone search | 20,000 | 46.59 ms | 25.06 MB |
| Warm session search | 20,000 | 1.25 ms | 0.58 MB |
| Warm session search | 100,000 | 5.78 ms | 2.50 MB |

These synthetic benchmarks use 250 distinct variants in one family. They
measure search only, excluding startup, debounce, configuration preparation,
rendering and transport. They are measurements, not latency guarantees.

Reproduce with the repository-local Go cache settings from CONTRIBUTING.md:

```sh
go test ./internal/store -run '^$' \
  -bench 'Benchmark(SessionSearch|SearchTwentyThousand)' -benchmem
```
