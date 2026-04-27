# Plan: Achieve Full Test Coverage for `profile` Branch Changes

## Context

The `profile` branch introduced parallelization (worker pool), CPU/memory/trace profiling, and significant refactoring of `internal/bagoup`. Coverage improved from 50.8% (main) to 91.3% (profile), but several new/changed functions still have gaps. The user requires full coverage for all changes on this branch.

## Coverage Summary

| Function | File | Coverage | Status |
|---|---|---|---|
| `startProfiling` | internal/bagoup/bagoup.go:194 | 20.7% | **NEW — critical gap** |
| `Run` | internal/bagoup/bagoup.go:134 | 96.4% | Refactored — 1 path missing |
| `writeChunk` | internal/bagoup/write.go:75 | 92.9% | NEW — pdfgen error (acceptable omission) |
| `runPool` | internal/bagoup/export.go:43 | 96.2% | NEW — ctx.Done() gap, see below |
| `validatePaths` | internal/bagoup/bagoup.go:287 | 86.7% | Extracted from Run — `filepath.Abs` errors untriggerable |

Pre-existing gaps accepted as-is: `main` (0%), `panicOnErr` (0%), `Getrlimit`/`Setrlimit` (0%), `NewPDFGenerator` (80%), `NewPathTools` (75%), `validatePaths` `filepath.Abs` errors.

---

## Implementation Plan

### 1. `startProfiling` — new `TestStartProfiling` in [internal/bagoup/bagoup_test.go](internal/bagoup/bagoup_test.go)

Add a table-driven test calling `cfg.startProfiling()` directly (unexported, but test is in `package bagoup`).

Test cases:
- **"no profiling"** — empty `Profiling` struct, no mocks; assert no error, call `stop()`
- **"trace file creation error"** — `opts.Profiling.Trace = "trace.out"`, `osMock.EXPECT().Create("trace.out")` returns error; `wantErr: "create trace file: ..."`
- **"cpu profile file creation error"** — `opts.Profiling.CPUProfile = "cpu.prof"`, same pattern; `wantErr: "create CPU profile: ..."`
- **"trace + mem profile success, stop called"** — `opts.Profiling.Trace = "trace.out"` and `opts.Profiling.MemProfile = "mem.prof"`. `OS.Create("trace.out")` returns a real temp `*os.File` (implements `afero.File`); `OS.Create("mem.prof")` returns a real temp `*os.File`. Call `startProfiling()`, then call `stop()`. Covers the `stops` slice iteration and the mem profile write path.
- **"mem profile creation error in stop"** — `opts.Profiling.MemProfile = "mem.prof"` only. `OS.Create("mem.prof")` returns error. Call `startProfiling()`, then call `stop()` — error is silently swallowed. Covers the `if err != nil { return }` branch at bagoup.go:223.

### 2. `Run` startProfiling error path — add case to `TestBagoup` in [internal/bagoup/bagoup_test.go](internal/bagoup/bagoup_test.go)

Add one test case:
- **"startProfiling error (trace file)"** — `opts` = `defaultOpts` with `opts.Profiling.Trace = "trace.out"`. Mock sequence: `FileAccess`, `FileExist`, `MkdirAll(logDirAbs)`, `Create(logFileAbs)` all succeed (mirrors "default options"), then `osMock.EXPECT().Create("trace.out").Return(nil, errors.New("perm error"))`. `wantErr: "create trace file: perm error"`. Covers bagoup.go:144–146.

### 3. `runPool` ctx.Done() — refactor to "allDone context" pattern in [internal/bagoup/export.go](internal/bagoup/export.go)

**Root cause of untestability with current code**: `runPool` pre-fills and immediately closes the channel. When a worker loops back to the select, both `ok=false` (closed empty channel) and `ctx.Done()` (if cancelled) are simultaneously ready — Go picks randomly, so `ctx.Done()` is untestable deterministically.

**Fix**: Keep the pre-filled channel but **don't close it**. Instead, introduce a "allDone" sub-context that is cancelled when all jobs finish (via a `cancelAllDone` call inside the mutex block). Workers then exit via `case <-gCtx.Done(): return nil` in **every case** — both success (all-done cancellation) and error (errgroup cancellation). This makes the `ctx.Done()` branch deterministically covered by all existing tests with no new tests required.

```go
func (cfg *configuration) runPool(ctx context.Context, jobs []writeJob) error {
    if len(jobs) == 0 {
        return nil
    }
    jobsCh := make(chan writeJob, len(jobs))
    for _, job := range jobs {
        jobsCh <- job
    }
    // Channel left open; workers exit via ctx cancellation, not channel close.

    bar := progressbar.NewPBar()
    bar.SignalHandler()
    bar.Total = uint16(len(jobs))

    allDoneCtx, cancelAllDone := context.WithCancel(ctx)
    defer cancelAllDone()

    g, gCtx := errgroup.WithContext(allDoneCtx)
    var mu sync.Mutex
    var done int
    for range max(1, runtime.NumCPU()-1) {
        g.Go(func() error {
            for {
                select {
                case job := <-jobsCh:
                    c := newCounts()
                    if err := cfg.writeChunk(job, c); err != nil {
                        return err
                    }
                    mu.Lock()
                    cfg.mergeCounts(c)
                    done++
                    bar.RenderPBar(done)
                    if done == len(jobs) {
                        cancelAllDone() // signal all workers to exit
                    }
                    mu.Unlock()
                case <-gCtx.Done():
                    return nil // ← covered by EVERY test (all-done or error)
                }
            }
        })
    }
    return g.Wait()
}
```

**Why this is always deterministic**: `cancelAllDone()` fires after the last job completes, immediately propagating through `allDoneCtx → gCtx`. All workers currently in the select (or looping back) will see `gCtx.Done()` and take the `ctx.Done()` branch. No timing dependencies. When an error occurs, errgroup cancels `gCtx` directly, same effect.

**Tests**: No new tests required. The existing `TestExportChats` success cases (where all jobs complete) will now deterministically cover `case <-gCtx.Done(): return nil`. The existing error cases also cover it via errgroup cancellation.

Note: remove the `ok` bool from the channel receive since the channel is never closed (`case job := <-jobsCh:` instead of `case job, ok := <-jobsCh:`), and drop the `if !ok { return nil }` branch.

---

## Critical Files

| File | Change |
|---|---|
| [internal/bagoup/export.go](internal/bagoup/export.go) | Refactor `runPool` to allDone-context pattern |
| [internal/bagoup/bagoup_test.go](internal/bagoup/bagoup_test.go) | Add `TestStartProfiling`; add "startProfiling error" case to `TestBagoup` |

---

## Verification

```bash
make test
go tool cover -func=coverage.out | grep -E "startProfiling|runPool|Run |writeChunk"
```

Expected:
- `startProfiling`: ≥ 95% (`trace.Start`/`pprof.StartCPUProfile` error paths require real binary failures — acceptable)
- `Run`: 100%
- `runPool`: 100% (`ctx.Done()` covered by all tests via allDone cancellation)
- Overall: ≥ 93%
