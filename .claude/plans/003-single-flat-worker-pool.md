# Plan: Single flat worker pool at the chunk level

## Context

The two-level parallelism (entity pool in `exportChats` + chunk pool in `writePDFs`) can run up to `(N-1)²` goroutines doing weasyprint/HEIC work simultaneously — more than the CPU count can actually parallelise. Replace with a single pool of `max(1, NumCPU-1)` workers where the unit of work is one output file (a PDF chunk or a txt file). A sequential prepare pass gathers all jobs first, then the flat pool writes them in parallel.

## New call flow

```
exportChats
  for each entity: prepareEntityJobs(ec) → []writeJob   // sequential: DB reads + MkdirAll
  single flat pool of N workers
    for each job: writeChunk(job)
      → handleFileContents(outFile, job.messageIDs, job.attDir)
```

ALL prepare calls finish before ANY job is fed to the pool (jobs channel is buffered with `len(allJobs)`; feeding is non-blocking and completes before `<-results` is awaited). So prepare and write are strictly ordered globally — safe to test as two separate phases.

## `writeJob` struct (new, in `write.go`)

```go
type writeJob struct {
    entityName string
    chatPath   string
    messageIDs []chatdb.DatedMessageID
    attDir     string
}
```

## Function changes

| Old | New | File | Notes |
|---|---|---|---|
| `exportChats` | (same name) | `export.go` | Sequential prepare loop + single flat pool |
| `exportEntityChats` → `([]writeJob, error)` | `prepareEntityJobs` | `export.go` | Returns jobs instead of writing |
| `writeFile` → `([]writeJob, error)` | `prepareFileJobs` | `write.go` | Creates dirs, splits chunks, returns jobs |
| `writePDFs` | removed | `write.go` | Splitting moves into `prepareFileJobs` |
| `writeTxt` | removed | `write.go` | Absorbed into `writeChunk` |
| `writePDFChunk` → `writeChunk` | `writeChunk` | `write.go` | Handles both PDF and txt |

Unchanged: `handleFileContents`, `handleAttachments`, `ensureOpenFilesLimit`, `openFilesLimitMu`, `copyAttachment`, `writeAttachment`, `validateAttachmentPath`.

## `exportChats` implementation

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

    workers := max(1, runtime.NumCPU()-1)
    jobsCh := make(chan writeJob, len(allJobs))
    type result struct { counts counts; err error }
    resultsCh := make(chan result, len(allJobs))
    var wg sync.WaitGroup
    for range workers {
        wg.Go(func() {
            for job := range jobsCh {
                localCfg := *cfg
                localCfg.counts = newCounts()
                err := localCfg.writeChunk(job)
                resultsCh <- result{localCfg.counts, err}
            }
        })
    }
    bar := progressbar.NewPBar()
    bar.SignalHandler()
    bar.Total = uint16(len(allJobs))
    for _, job := range allJobs { jobsCh <- job }
    close(jobsCh)
    collected := make([]result, 0, len(allJobs))
    for i := range allJobs {
        r := <-resultsCh
        bar.RenderPBar(i)
        collected = append(collected, r)
    }
    wg.Wait()
    var firstErr error
    for _, r := range collected {
        cfg.mergeCounts(r.counts)
        if r.err != nil && firstErr == nil { firstErr = r.err }
    }
    return firstErr
}
```

Edge case: `len(allJobs) == 0` (no chats after filtering). `make(chan writeJob, 0)` creates an unbuffered channel; the feed loop doesn't run; `close(jobsCh)` causes workers to exit immediately; `wg.Wait()` returns. No deadlock.

## `prepareEntityJobs` (replaces `exportEntityChats`)

Same logic as current `exportEntityChats` but calls `cfg.prepareFileJobs` instead of `cfg.writeFile`, and returns `[]writeJob`. Increments `cfg.counts.chats` as before (prepare pass runs on the main goroutine, so no race).

## `prepareFileJobs` (replaces `writeFile`)

Same setup logic as current `writeFile` (MkdirAll, filename truncation, sort, attDir). Instead of calling `writePDFs`/`writeTxt`, builds and returns `[]writeJob`:
- Non-PDF: single job with path `chatPathNoExt + ".txt"`
- PDF: use the same chunk-splitting loop from `writePDFs`; one job per chunk

## `writeChunk` (replaces `writePDFChunk` + `writeTxt`)

```go
func (cfg *configuration) writeChunk(job writeJob) error {
    chatFile, err := cfg.OS.Create(job.chatPath)
    if err != nil { return fmt.Errorf("create file %q: %w", job.chatPath, err) }
    defer chatFile.Close()
    var outFile opsys.OutFile
    if cfg.Options.OutputPDF {
        if cfg.Options.UseWkhtmltopdf {
            pdfg, err := pdfgen.NewPDFGenerator(chatFile)
            if err != nil { return fmt.Errorf("create PDF generator: %w", err) }
            outFile = cfg.OS.NewWkhtmltopdfFile(job.entityName, chatFile, pdfg, cfg.Options.IncludePPA)
        } else {
            outFile = cfg.OS.NewWeasyPrintFile(job.entityName, chatFile, cfg.Options.IncludePPA)
        }
    } else {
        outFile = cfg.OS.NewTxtOutFile(chatFile)
    }
    return cfg.handleFileContents(outFile, job.messageIDs, job.attDir)
}
```

## Import changes

- `export.go`: keep `"runtime"`, `"sync"` (pool moves here from write.go)
- `write.go`: remove `"runtime"` (no pool); keep `"sync"` (for `openFilesLimitMu`)

## Test changes

### `internal/bagoup/export_test.go`

Since the prepare pass is now fully sequential, all prepare calls (`GetMessageIDs`, `MkdirAll`) can go in one `gomock.InOrder` with `GetAttachmentPaths`/`GetChats`. Write calls (per job) go in separate `gomock.InOrder` groups.

For each test case:

| Test | Change |
|---|---|
| "two chats for one display name, one for another" | Single prepare InOrder (GetAttachmentPaths+GetChats+GetMessageIDs×3+MkdirAll×2); two write InOrders (one per entity) |
| "filter one entity" | Single InOrder (everything sequential since 1 job) |
| "specify both entities, so don't filter any" | Single prepare InOrder; two write InOrders |
| "separate chats" | Single prepare InOrder (GetAttachmentPaths+GetChats+GetMessageIDs×2+MkdirAll×2); two write InOrders (one per chat) |
| "pdf", "pdf without attachments", "copy attachments" | Single InOrder (1 job, sequential) |
| Error cases (GetMessageIDs error, writeFile error, separate chats - writeFile error) | Single InOrder unchanged (error occurs during prepare, returns immediately) |

### `internal/bagoup/write_test.go`

Split `TestWriteFile` into `TestPrepareFileJobs` + `TestWriteChunk`.

**`TestPrepareFileJobs`** — only mocks `MkdirAll`; verifies returned `[]writeJob`:
- `chat directory creation error` → MkdirAll fails, error returned
- `error creating attachments folder` → second MkdirAll fails, error returned
- `long email address` → truncated filename in returned job
- `multiple PDF chunks` (replaces "multiple PDF files") → 4000 messages → 2 writeJobs with correct paths and message ID slices; only MkdirAll mocked

**`TestWriteChunk`** — no `MkdirAll` mock; mocks `Create`, outFile, DB, imgconv:
- All current write-path test cases: `text export`, `WeasyPrint pdf export`, `wkhtmltopdf export`, `pdf export needs open files limit increase`, `copy attachments`, `copy attachments preserving paths`, `copy attachments and pdf export`, `pdf chat file creation error`, `chat file creation error`, `GetMessage error`, `WriteMessage error`, `Staging error`, `get open files limit fails`, `open files limit increase fails`, `Flush error`, `attachment file does not exist`, `error referencing attachment`, `file existence check fails`, `error creating preserved path`, `CopyFile error`, `HEIC conversion error`, `WriteAttachment error`, `1 message invalid`
- Input is a `writeJob` struct (pre-built, no prepare needed)
- Counts verified on `cfg.counts` after the call (same as now, no pool involved)

## Files to modify

| File | Change |
|---|---|
| `internal/bagoup/export.go` | Sequential prepare loop + flat pool in `exportChats`; `exportEntityChats` → `prepareEntityJobs` |
| `internal/bagoup/write.go` | Add `writeJob`; `writeFile` → `prepareFileJobs`; `writePDFChunk`+`writeTxt` → `writeChunk`; remove `writePDFs`; remove `"runtime"` import |
| `internal/bagoup/export_test.go` | Single prepare InOrder + per-job write InOrders for multi-entity/multi-job tests |
| `internal/bagoup/write_test.go` | `TestWriteFile` → `TestPrepareFileJobs` + `TestWriteChunk` |

## Verification

```sh
go test ./... -count=1 -race
go build -o bagoup ./cmd/bagoup
./bagoup --pdf --trace trace3.out -i /path/to/chat.db -o /tmp/export3
go tool trace trace3.out   # single goroutine pool; chunks from different contacts interleave
```
