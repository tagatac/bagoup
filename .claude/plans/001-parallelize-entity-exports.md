# Plan: Parallelize entity exports with a worker pool

## Context

Profiling showed the 8m26s export is entirely subprocess time: weasyprint `Flush()` calls (~67%) and ImageMagick `ConvertHEIC()` calls (~32%). All HEIC conversions for a given PDF happen in the message loop inside `handleFileContents` before `Stage()` and `Flush()` — they are not a separate phase.

The sequential bottleneck is the `exportChats` loop: each entity (contact) is processed one at a time. Parallelizing at the `writePDFs` level would only help for chats split across multiple PDF files (>3072 messages), which is rare. Parallelizing at the `exportChats` level makes all entity exports concurrent, giving parallel weasyprint and HEIC subprocess calls across contacts regardless of chat size.

**Worker count:** `max(1, runtime.NumCPU()-1)` — leave one CPU for the OS and Go runtime.

## Design

Use the classic Go goroutine pool: N worker goroutines each reading from a jobs channel. Each worker gets a shallow copy of `configuration` with a fresh `counts` struct — no mutex needed for counts. A results channel carries each worker's local counts and error back to the main goroutine for aggregation.

```
exportChats:
  jobs    := make(chan EntityChats, len(chats))   // buffered: non-blocking feed
  results := make(chan result, len(chats))         // buffered: non-blocking workers

  for i := 0; i < workers; i++ {
      go func() {
          for ec := range jobs {
              localCfg := *cfg                      // shallow copy: shares OS, ChatDB, etc.
              localCfg.counts = newCounts()         // own counts; no shared mutable state
              err := localCfg.exportEntityChats(ec)
              results <- result{localCfg.counts, err}
          }
      }()
  }
  for _, ec := range chats { jobs <- ec }          // feed, then close to signal workers
  close(jobs)

  for range chats {                                // drain results
      r := <-results
      cfg.mergeCounts(r.counts)                   // single-threaded: no lock needed
      bar.RenderPBar(...)
      if r.err != nil && firstErr == nil { firstErr = r.err }
  }
  return firstErr
```

The `localCfg := *cfg` shallow copy is safe:
- `OS`, `ChatDB`, `ImgConverter`, `PathTools` — shared interfaces, all goroutine-safe
- `handleMap`, `attachmentPaths` — maps populated before parallelism, read-only during export
- `counts` — reset to a fresh value per goroutine; merged single-threadedly at the end

## HEIC filename collision fix

`ConvertHEIC` currently uses `filepath.Base(src)` for the temp filename. Two entities with attachments sharing the same basename (e.g. `photo.heic`) would race to write the same temp file. Fix by prepending an FNV-64a hash of the full source path — stdlib, no new dependency:

```go
import "hash/fnv"

h := fnv.New64a()
h.Write([]byte(src))
base := strings.TrimRight(filepath.Base(src), "HEICheic")
jpgFilename := fmt.Sprintf("%016x_%s.jpeg", h.Sum64(), base)
```

## SQLite thread safety

`database/sql.DB` is goroutine-safe by design. `go-sqlite3` supports concurrent reads (SQLite uses shared locks; multiple readers don't block each other). All ChatDB calls during export are read-only (`GetMessage`, `GetMessageIDs`). However, without `SetMaxOpenConns`, `database/sql` may open many connections to the SQLite file under concurrent load, potentially causing "database is locked" errors on some systems.

**Fix:** add `db.SetMaxOpenConns(max(1, runtime.NumCPU()-1))` in `cmd/bagoup/main.go` after `sql.Open` — cap the pool to the worker count. Each worker may need a connection simultaneously; allowing more than that has no benefit. Since all access is reads, multiple concurrent connections are safe.

## Files to modify

| File | Change |
|---|---|
| `cmd/bagoup/main.go` | Add `db.SetMaxOpenConns(1)` after `sql.Open` |
| `internal/bagoup/export.go` | Replace sequential `exportChats` loop with N-worker goroutine pool |
| `internal/bagoup/bagoup.go` | Add `mergeCounts(counts)` method and `newCounts()` helper |
| `imgconv/imgconv.go` | FNV hash prefix in `ConvertHEIC` temp filename |
| `internal/bagoup/export_test.go` (if exists) | Relax cross-entity call ordering in mocks |
| `imgconv/convert_test.go` | Update expected destination filename to match new format |

No changes needed to `write.go`, `outfile.go`, or any opsys code.

## Verification

```sh
go test ./...
go build -o bagoup ./cmd/bagoup
./bagoup --pdf --trace trace2.out -i /path/to/chat.db -o /tmp/export2
go tool trace trace2.out   # flame chart should show multiple weasyprint goroutines overlapping
```

Expected: wall time drops roughly proportional to `min(entities, workers)`; trace shows weasyprint and HEIC goroutines running in parallel across entities.
