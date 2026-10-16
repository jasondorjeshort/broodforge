---
date: 2026-10-16
status: resolved
area: go / windows named pipes
promoted_to: docs/reference/go-dependencies.md
---

# Go named pipes on Windows: three pitfalls in the control transport

## Question

The control socket on Windows is a named pipe (`internal/control/pipe_windows.go`). A connection
reads and writes at once, one goroutine each. Why did the first versions fail under `-race` and
under load, and what shape of code avoids every failure?

## What was tried

Environment: Go 1.27.2, Windows, `-race` run as
`PATH="/c/msys64/ucrt64/bin:$PATH" go test -race ./internal/control/...` (the mingw gcc `-race`
needs). The account is the P2 coder's report and the P2 review of 2026-10-16; the line numbers
are those of the working tree after the review fix, not yet committed.

1. First version: the pipe handle wrapped with `os.NewFile`, then used from a reader and a writer
   goroutine. `go test -race` reported a data race inside the standard library.
2. Second version: overlapped I/O through `golang.org/x/sys/windows`, each `windows.Overlapped`
   a local variable of the calling function. Under load `GetOverlappedResult` returned
   `ERROR_IO_INCOMPLETE` after the completion event had fired.
3. Third version: no `os.NewFile`; each `Overlapped` and its event live in a heap-allocated
   `op` (`pipe_windows.go` lines 50-66), and `ReadFile` and `WriteFile` use the connection's own
   heap buffers (lines 26-27, 310, 331), never the caller's slice. The review then found the
   handle array passed to `WaitForMultipleObjects` still a slice literal on the stack: a
   concurrent-id stress with a `runtime.GC()` loop under `-race` failed 6 of 20 counts with a
   fresh connection's Read returning `ERROR_INVALID_HANDLE` (clients saw `internal: context
   canceled` or `internal: The handle is invalid.`). `TestStressUnderGC`
   (`internal/control/stress_test.go`) reproduced it, 1 of 20 counts.
4. Current version: the array is the op's `hs [2]windows.Handle` field (lines 56, 83-84), and a
   failed wait cancels and drains the pending operation before returning (lines 72-76, 85-88).

## Result

Pitfall 1, `os.NewFile` on a pipe. For a pipe, `internal/poll.FD` updates one shared offset
field from both Read and Write, each under its own lock, so a reader and a writer running
together race on it; `-race` reports it. The pipe therefore does not use `os.NewFile`. It opens
every handle with `FILE_FLAG_OVERLAPPED` and waits on a per-operation event plus a close event
(lines 17-19, 33-35, 82-95), which also lets Close cancel a pending operation. One Read and one
Write may run at once, each with its own `op` and mutex (lines 264-278).

Pitfall 2, a stack `Overlapped`. The kernel writes the completion into the address it was given
at call time. A goroutine stack moves when it grows, so an `Overlapped` on the stack was copied
and the kernel completed into the old copy: the event fired, but `GetOverlappedResult` saw the
new copy still pending and returned `ERROR_IO_INCOMPLETE`. Each `Overlapped` now lives in a heap
`op` that its owner keeps until the operation has finished (the read and write ops at lines
272-276, the connect op at line 103; Close waits for pending operations before freeing them,
lines 353-369). The caller's slice is copied into a connection buffer for the same reason.

Pitfall 3, a stack handle array. `WaitForMultipleObjects([]windows.Handle{...})` with a literal
does not escape (`go build -gcflags=-m`), and x/sys v0.49.0 converts `&handles[0]` to a
`uintptr` and then calls another Go function before the syscall, so a stack move in between
hands the kernel freed stack memory. The array is now a field of the heap `op` (comment, lines
28-31). After the fix `TestStressUnderGC` passed every count under `-race -count=20` and a
further 60.

The durable part is the three rules, recorded under `golang.org/x/sys` in
`docs/reference/go-dependencies.md`: no `os.NewFile` for a pipe used by a reader and a writer
together; no `Overlapped` or I/O buffer on a goroutine stack; no handle array on one either.
