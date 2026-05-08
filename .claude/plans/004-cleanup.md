# Plan: Cleanup — complexity, profiling, errgroup, cancellation

## Context

After the flat-pool refactor, several code-quality issues remain:
- `Run()` and `exportChats()` have high cyclomatic complexity (15 and 12).
- CPU/memory/trace profiling lives in `main()`, which the user doesn't want to profile.
- Workers copy the whole `configuration` struct (`localCfg := *cfg`) just to get isolated counts — counts is the *only* field written by workers; all other fields are read-only during the parallel phase. Passing `*counts` explicitly removes the need for the struct copy.
- Jobs are collected into `[]writeJob` then pushed into a channel — redundant. Streaming from a producer goroutine is simpler and enables cancellation.
- No cancellation: a failing job doesn't stop the others.

## Changes

### 1. Add `golang.org/x/sync` (for `errgroup`)

```sh
go get golang.org/x/sync
```

`errgroup.WithContext` gives first-error-wins + automatic ctx cancellation.

---

### 2. Profiling options in `Options` with a flag group (`internal/bagoup/options.go`)

go-flags v1.6.1 supports nested-struct groups. Add at the end of `Options`:

```go
Profiling struct {
    CPUProfile string `long:"cpuprofile" description:"Write CPU profile to this file"`
    MemProfile string `long:"memprofile" description:"Write memory profile to this file"`
    Trace      string `long:"trace"      description:"Write execution trace to this file"`
} `group:"Profiling options"`
```

Access via `cfg.Options.Profiling.CPUProfile` etc. in `bagoup.go`. `main.go` drops its local profiling fields.

---

### 3. Eliminate `localCfg` — pass `*counts` explicitly (`internal/bagoup/write.go`)

`counts` is the **only** field written by workers. Rather than copying the whole struct to isolate it, pass a `*counts` accumulator explicitly through the call chain:

```
writeChunk(job, c *counts)
  → handleFileContents(outFile, messageIDs, attDir, c)
    → handleAttachments(outFile, msgID, attDir, c)
      → copyAttachment(att, attDir, c)
      → writeAttachment(outFile, att, c)
```

All `cfg.counts.xxx` references in these functions become `c.xxx`. Workers:

```go
c := newCounts()
if err := cfg.writeChunk(job, c); err != nil { return err }
mu.Lock(); cfg.mergeCounts(c); mu.Unlock()
```

No struct copy needed. `cfg` is used directly (read-only for all other fields during parallel phase).

`cfg.counts` (`*counts`) on the main configuration still exists as the final accumulator (and receives `chats++` during the sequential prepare phase). `mergeCounts` takes `*counts`.

`newCounts()` returns `*counts`:
```go
func newCounts() *counts {
    return &counts{attachments: map[string]int{}, attachmentsCopied: map[string]int{}, attachmentsEmbedded: map[string]int{}}
}
```

---

### 4. Reduce `Run()` complexity — extract helpers (`internal/bagoup/bagoup.go`)

Extract four private methods called sequentially from `Run()`:

| Method | Branches absorbed |
|---|---|
| `setupLogging() error` | MkdirAll logDir, Create out.log, set MultiWriter |
| `resolveMacOSVersion() error` | if MacOSVersion != nil / else GetMacOSVersion |
| `loadContactMap() (map[string]*vcard.Card, error)` | if ContactsPath != nil / GetContactMap |
| `setupImgConverter() error` | if OutputPDF / GetTempDir / NewImgConverter |

`Run()` after extraction becomes a flat sequence of ~6 calls — complexity drops from 15 to ~7.

---

### 5. Move profiling into `Run()` using `cfg.OS` (`internal/bagoup/bagoup.go`)

Add `startProfiling() (stop func(), err error)` — uses `cfg.OS.Create()` for consistency:

```go
func (cfg *configuration) startProfiling() (func(), error) {
    var stops []func()
    if cfg.Options.Profiling.Trace != "" {
        f, err := cfg.OS.Create(cfg.Options.Profiling.Trace)
        if err != nil { return nil, fmt.Errorf("create trace file: %w", err) }
        if err := trace.Start(f); err != nil { return nil, fmt.Errorf("start trace: %w", err) }
        stops = append(stops, func() { trace.Stop(); f.Close() })
    }
    if cfg.Options.Profiling.CPUProfile != "" {
        f, err := cfg.OS.Create(cfg.Options.Profiling.CPUProfile)
        if err != nil { return nil, fmt.Errorf("create CPU profile: %w", err) }
        if err := pprof.StartCPUProfile(f); err != nil { return nil, fmt.Errorf("start CPU profile: %w", err) }
        stops = append(stops, func() { pprof.StopCPUProfile(); f.Close() })
    }
    return func() {
        for _, s := range stops { s() }
        if cfg.Options.Profiling.MemProfile != "" {
            f, err := cfg.OS.Create(cfg.Options.Profiling.MemProfile)
            if err != nil { return }
            runtime.GC()
            _ = pprof.WriteHeapProfile(f)
            f.Close()
        }
    }, nil
}
```

`Run()` calls this right after `setupLogging()`:
```go
stopProfiling, err := cfg.startProfiling()
if err != nil { return err }
defer stopProfiling()
```

Tests are unaffected: all three profile path fields are empty strings by default → `startProfiling` is a no-op, no mock calls needed.

---

### 6. Replace `var allJobs []writeJob` + buffered channel with `errgroup` + cancellation (`internal/bagoup/export.go`)

The sequential prepare loop is unchanged. Its result (`allJobs`) is used to:
- size the buffered channel (capacity = `len(allJobs)`) so all sends are non-blocking
- set `bar.Total` once before any worker starts
No producer goroutine is needed. Workers use a `select` for cancellation.

```go
func (cfg *configuration) exportChats(contactMap map[string]*vcard.Card) error {
    if err := getAttachmentPaths(cfg); err != nil { return err }
    chats, err := cfg.ChatDB.GetChats(contactMap)
    if err != nil { return fmt.Errorf("get chats: %w", err) }
    chats = filterEntities(cfg.Options.Entities, chats)

    var allJobs []writeJob
    for _, ec := range chats {
        jobs, err := cfg.prepareEntityJobs(ec)
        if err != nil { return err }
        allJobs = append(allJobs, jobs...)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    return cfg.runPool(ctx, allJobs)
}

func (cfg *configuration) runPool(ctx context.Context, jobs []writeJob) error {
    jobsCh := make(chan writeJob, len(jobs))
    for _, job := range jobs { jobsCh <- job }
    close(jobsCh)

    bar := progressbar.NewPBar()
    bar.SignalHandler()
    bar.Total = uint16(len(jobs))  // set once, before workers start

    g, ctx := errgroup.WithContext(ctx)
    var mu sync.Mutex
    var done int
    for range max(1, runtime.NumCPU()-1) {
        g.Go(func() error {
            for {
                select {
                case job, ok := <-jobsCh:
                    if !ok { return nil }
                    c := newCounts()
                    if err := cfg.writeChunk(job, c); err != nil { return err }
                    mu.Lock()
                    cfg.mergeCounts(c)
                    done++
                    bar.RenderPBar(done)
                    mu.Unlock()
                case <-ctx.Done():
                    return nil
                }
            }
        })
    }
    return g.Wait()
}
```

**Cancellation**: when any worker returns an error, errgroup cancels `ctx`. Other workers see `ctx.Done()` in their `select` at the next job pickup and return nil — abandoning the remaining jobs in the channel. Workers finish their current `writeChunk` call (no mid-job interruption). `g.Wait()` returns the first error.

**`exportChats` complexity** drops from 12 to ~5. `runPool` has ~5 (no longer has the old collect-drain-merge loops).

---

### 7. Simplify `main.go`

Remove the three profiling `if` blocks and the local profiling fields. `cfg.Run()` remains the single call site. The DB copy and close logic stays in `main`.

---

## Files to modify

| File | Change |
|---|---|
| `internal/bagoup/options.go` | Add `Profiling` nested struct with group tag |
| `internal/bagoup/bagoup.go` | `counts` → `*counts`; extract 4 helpers from `Run()`; add `startProfiling()` |
| `internal/bagoup/export.go` | `exportChats` with context; new `runPool` using `errgroup` + producer |
| `internal/bagoup/write.go` | `writeChunk`, `handleFileContents`, `handleAttachments`, `copyAttachment`, `writeAttachment` gain `*counts` param |
| `cmd/bagoup/main.go` | Remove profiling blocks and local profiling fields |
| `go.mod` / `go.sum` | Add `golang.org/x/sync` |
| `internal/bagoup/bagoup_test.go` | Update for extracted methods, `*counts` field, profiling helpers |
| `internal/bagoup/export_test.go` | Update for `runPool`-based `exportChats` |
| `internal/bagoup/write_test.go` | Add `*counts` param to `writeChunk` calls in `TestWriteChunk` |

## Verification

```sh
go test ./... -count=1 -race
go build -o bagoup ./cmd/bagoup
./bagoup --cpuprofile cpu.out -i /path/to/chat.db -o /tmp/export
go tool pprof cpu.out
```
