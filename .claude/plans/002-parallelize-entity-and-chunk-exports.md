# Plan: Parallelize entity and chunk exports with worker pools

## Context

Profiling showed the 8m26s export is entirely subprocess time: weasyprint `Flush()` calls (~67%) and ImageMagick `ConvertHEIC()` calls (~32%). All HEIC conversions for a given PDF happen in the message loop inside `handleFileContents` before `Stage()` and `Flush()` — they are not a separate phase.

**Entity-level parallelism (DONE):** `exportChats` now uses a worker pool of `max(1, runtime.NumCPU()-1)` goroutines, each processing one entity (contact) concurrently. Each worker gets a shallow copy of `configuration` with a fresh `counts` struct. Results collected into a slice, `wg.Wait()` ensures workers exit, then counts merged single-threadedly.

**Chunk-level parallelism (TODO):** In production there is often a long tail waiting for the PDF chunks for a single frequent contact. `writePDFs` splits contacts with >3072 messages into multiple PDF chunks. These chunks are currently processed sequentially — the second bottleneck to fix.

## Chunk-level design

Apply the same worker pool + collect-then-merge pattern to `writePDFs`.

### 1. Extract `writePDFChunk` helper

Extract the per-chunk body from the `writePDFs` for-loop into a method:

```go
func (cfg *configuration) writePDFChunk(entityName, chatPath string, messageIDs []chatdb.DatedMessageID, attDir string) error {
    chatFile, err := cfg.OS.Create(chatPath)
    if err != nil {
        return fmt.Errorf("create file %q: %w", chatPath, err)
    }
    defer chatFile.Close()
    var outFile opsys.OutFile
    if cfg.Options.UseWkhtmltopdf {
        pdfg, err := pdfgen.NewPDFGenerator(chatFile)
        if err != nil {
            return fmt.Errorf("create PDF generator: %w", err)
        }
        outFile = cfg.OS.NewWkhtmltopdfFile(entityName, chatFile, pdfg, cfg.Options.IncludePPA)
    } else {
        outFile = cfg.OS.NewWeasyPrintFile(entityName, chatFile, cfg.Options.IncludePPA)
    }
    return cfg.handleFileContents(outFile, messageIDs, attDir)
}
```

### 2. Replace `writePDFs` for-loop with worker pool

`messageIDsAndChatPath` stays as a local type inside `writePDFs`. The jobs channel carries it; results carry counts + error:

```go
workers := max(1, runtime.NumCPU()-1)
jobs := make(chan messageIDsAndChatPath, len(idsAndPaths))
type result struct { counts counts; err error }
results := make(chan result, len(idsAndPaths))
var wg sync.WaitGroup
for range workers {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for iap := range jobs {
            localCfg := *cfg
            localCfg.counts = newCounts()
            err := localCfg.writePDFChunk(entityName, iap.chatPath, iap.messageIDs, attDir)
            results <- result{localCfg.counts, err}
        }
    }()
}
for _, iap := range idsAndPaths { jobs <- iap }
close(jobs)
collected := make([]result, 0, len(idsAndPaths))
for range idsAndPaths { collected = append(collected, <-results) }
wg.Wait()
var firstErr error
for _, r := range collected {
    cfg.mergeCounts(r.counts)
    if r.err != nil && firstErr == nil { firstErr = r.err }
}
return firstErr
```

### 3. Protect the rlimit TOCTOU with a package-level mutex

`GetOpenFilesLimit` / `SetOpenFilesLimit` in `handleFileContents` is a check-then-set with no atomicity — concurrent goroutines can both observe a limit below `imgCount*2` and both try to raise it. Extract to a helper protected by a package-level mutex in `write.go`:

```go
var openFilesLimitMu sync.Mutex

func (cfg *configuration) ensureOpenFilesLimit(imgCount int, outFileName string) error {
    openFilesLimitMu.Lock()
    defer openFilesLimitMu.Unlock()
    openFilesLimit, err := cfg.OS.GetOpenFilesLimit()
    if err != nil {
        return err
    }
