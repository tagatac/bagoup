# Plan: Incremental backup (issue #33)

## Context

bagoup currently refuses to write to an existing export directory
(`internal/bagoup/bagoup.go:302-303`), so users must export everything from
scratch every time. Issue #33 asks for an incremental mode that appends only
messages received since the last run. The repo owner suggested a "pointer
file" approach in the issue thread and limited it to text output (PDFs can't
be appended).

This plan adds an `--incremental` flag that:
- Records per-output-file watermarks in `<ExportPath>/.bagoup/state.json`.
- On re-run, loads the state file, filters messages by **ROWID** (`message.ROWID`),
  and **appends** to the existing `.txt` files instead of recreating them.
- Reuses the stored filename so a changed set of chat GUIDs for an entity
  doesn't fragment the thread.
- Is permissive about bagoup version: any export with a valid `state.json` is
  accepted; a version difference is logged as a warning, not an error.

ROWID is preferred over `message.date` as the watermark because ROWIDs are
monotonically assigned by SQLite at insertion time, so late-syncing messages
still get a fresh higher ROWID and won't be missed. ROWID also sidesteps the
Apple-epoch / legacy date-divisor complexity entirely.

---

## Approach

Add `--incremental` flag. On first run with the flag against a non-existent
directory, behave like a normal export but additionally write
`<ExportPath>/.bagoup/state.json` recording, per output file, the largest
ROWID written. On subsequent runs against an existing directory:

1. Load `state.json`; warn (don't fail) on version mismatch.
2. For each chat, filter `GetMessageIDs` results to `ID > watermark.LastRowID`.
3. Skip producing a write job when no new messages remain for that file.
4. Open the existing `.txt` file with `O_APPEND` instead of `Create`.
5. After each file flushes, update its state entry; save `state.json` on
   completion.

Restricted to text mode: `--incremental` + `--pdf` is rejected in
`ValidateOptions`.

---

## Files to change

### New: `internal/bagoup/state.go`
Pure data + I/O layer. No coupling to other bagoup internals beyond `opsys.OS`.
```go
type State struct {
    BagoupVersion string                    `json:"bagoup_version"`
    MacOSVersion  string                    `json:"mac_os_version"`
    Timezone      string                    `json:"timezone"`
    UpdatedAt     time.Time                 `json:"updated_at"`
    Files         map[string]FileWatermark  `json:"files"`
}
type FileWatermark struct {
    Filename   string   `json:"filename"`    // basename incl. ".txt", relative to entity dir
    GUIDs      []string `json:"guids"`       // GUIDs known at creation time
    LastRowID  int      `json:"last_row_id"` // largest message.ROWID written so far
}

func LoadState(s opsys.OS, exportPath string) (*State, bool, error)
func (st *State) Save(s opsys.OS, exportPath string) error
func StateKey(entityName string, guids []string, mergeChats bool) string
```
- Path: `filepath.Join(exportPath, ".bagoup", "state.json")`.
- `StateKey`: `entityName` in merge mode; `entityName + "/" + guid` in
  separate-chats mode.
- Write atomically: write to `state.json.tmp`, then rename — guards against
  crash mid-write.

### `internal/bagoup/options.go`
- Add field: `Incremental bool \`long:"incremental" description:"Append messages received since the last export (text only, requires an existing export with state)"\``.
- In `ValidateOptions`, reject:
  - `Incremental && OutputPDF` ("--incremental requires text output; remove --pdf")
  - `Incremental && (Profiling.CPUProfile|MemProfile|Trace != "")` (profile files would overwrite each other across runs)

### `internal/bagoup/bagoup.go`
- Add fields to `configuration`: `state *State`, `stateMu sync.Mutex`.
- `validatePaths` (line 291): branch on `cfg.Options.Incremental`:
  - dir does not exist → proceed as fresh export (state will be created at
    end of run).
  - dir exists, `state.json` exists → load it; if `state.BagoupVersion !=
    cfg.version`, `slog.Warn` but continue; if `state.Timezone !=
    cfg.Options.Timezone`, `slog.Warn`. Set `cfg.state`.
  - dir exists, no `state.json` → error ("no incremental state found at
    %q — was this export produced by bagoup --incremental?").
  - non-incremental path keeps the existing rejection at line 302-303.
- `setupLogging` (line 177): when `cfg.state != nil`, open the log file with
  `cfg.OS.OpenAppend` instead of `Create` so previous run logs are preserved.
- `Run` (line 135): after `exportChats` returns nil, ensure `cfg.state` is
  non-nil (create empty one if first run), populate
  `BagoupVersion`/`MacOSVersion`/`Timezone`/`UpdatedAt`, and call `Save`.

### `opsys/opsys.go` + `opsys/mock_opsys/`
- Add method to the `OS` interface:
  ```go
  OpenAppend(fp string) (afero.File, error)
  ```
- Implementation: `s.Fs.OpenFile(fp, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)`.
- Regenerate mocks via existing `go:generate mockgen` directive.

### `internal/bagoup/export.go`
- `prepareEntityJobs` (line 129): for each chat, after `GetMessageIDs`:
  - Compute `key := StateKey(entityChats.Name, guidsForKey, mergeChats)`.
  - If `cfg.state != nil` and `cfg.state.Files[key]` exists, filter
    `messageIDs` to `id.ID > watermark.LastRowID`. Also detect any GUIDs
    in `entityChats.Chats` not in `watermark.GUIDs` and `slog.Warn`
    ("new chats joined %q since last run; appending to existing file %q").
  - Skip the job when no new messages remain (don't call `prepareFileJobs`).
- Pass the stored filename (when present) and the state key down into
  `prepareFileJobs` / `writeJob`.

### `internal/bagoup/write.go`
- Extend `writeJob` with `stateKey string`, `appendMode bool`,
  `storedFilename string`.
- `prepareFileJobs` (line 36): when `storedFilename` is non-empty, use it
  verbatim for `chatPathNoExt` instead of computing from GUIDs/`;;;`/251-char
  truncation — guarantees filename stability across runs.
- `writeChunk` (line 75): if `job.appendMode`, call `cfg.OS.OpenAppend`;
  else `cfg.OS.Create`. PDF branch unchanged (gated off by `ValidateOptions`).
- `handleFileContents` (line 98): track `maxRowID := max(messageID.ID, ...)`
  while iterating. After successful `Flush`:
  ```go
  cfg.stateMu.Lock()
  cur := cfg.state.Files[job.stateKey]
  if maxRowID > cur.LastRowID {
      cur.LastRowID = maxRowID
  }
  if cur.Filename == "" {
      cur.Filename = filepath.Base(job.chatPath)
      cur.GUIDs = job.guids // thread through writeJob
  }
  cfg.state.Files[job.stateKey] = cur
  cfg.stateMu.Unlock()
  ```
  Persist state per-file (call `state.Save`) so a crash mid-run still leaves
  a consistent watermark for already-flushed files.

### `chatdb/message.go`
No code changes. `DatedMessageID.ID` is already the SQLite ROWID
(message.go:43), which is what the watermark now keys on.

### `README.md`
Add a short "Incremental backups" section under Usage:
- Flag: `--incremental`, text mode only.
- First run: `bagoup --incremental -o messages-export` creates the directory
  and writes `messages-export/.bagoup/state.json`.
- Subsequent runs: same command; bagoup reads the state file and appends new
  messages to the existing per-entity `.txt` files.
- Caveats:
  - Don't delete `.bagoup/state.json`.
  - If you switch source Macs (different `chat.db` instance), ROWIDs reset
    and the watermark becomes meaningless — start a fresh export directory.
  - Attachments: same `--copy-attachments` / `--preserve-paths` semantics as
    a normal run; existing files are not re-copied (`opsys.CopyFile`
    already deduplicates).

---

## Reused existing infrastructure

- `opsys.OS.CopyFile(src, dst, unique=true)` (opsys.go) — already
  deduplicates attachment copies safely, no changes needed.
- `pathtools.PathTools` — re-run tilde-expansion handling already supported
  via `--attachments-path` + `.tildeexpansion` (precedent for state files at
  the export root).
- `slog.Warn` — used throughout for non-fatal issues.
- `afero.NewMemMapFs` — for unit tests.
- `go-sqlmock` + `gomock` — existing patterns for chatdb/write tests.

---

## Implementation order (small, testable steps)

1. Add `OpenAppend` to `opsys.OS` interface; regenerate mock; unit test.
2. Add `internal/bagoup/state.go` + `state_test.go` (round-trip, missing
   file, corrupted JSON, atomic-rename behavior).
3. Add `--incremental` flag + `ValidateOptions` rules + `options_test.go`
   cases.
4. Wire `validatePaths` and `setupLogging` paths in `bagoup.go`; tests in
   `bagoup_test.go` (non-existent dir, existing dir with state, existing
   dir without state, version mismatch warn, log append).
5. Thread state into `prepareEntityJobs` / `prepareFileJobs` / `writeJob`;
   add filtering and stored-filename reuse; tests in `export_test.go`
   (watermark filter, no-new-messages skip, new GUID warning, separate-chats
   keying).
6. Add append routing in `writeChunk` and per-file watermark recording in
   `handleFileContents`; tests in `write_test.go`.
7. Per-file `state.Save` after `Flush` + final `Save` in `Run`; test crash
   recovery (state matches last-flushed file).
8. README update.

---

## Verification

End-to-end manual test against a synthetic chat.db (or the test fixtures
already in `chatdb/testdata/`):

```bash
# First run — produces export and state
go run ./cmd/bagoup --incremental -i testdata/chat.db -o /tmp/messages-export

# Inspect state
jq . /tmp/messages-export/.bagoup/state.json

# Add a new message to chat.db (or use a newer snapshot), re-run
go run ./cmd/bagoup --incremental -i testdata/chat.db.v2 -o /tmp/messages-export

# Confirm: existing .txt files grew (tail -n 5 of an entity file shows the
# new message); state.json's last_row_id increased; out.log was appended,
# not truncated; no duplicate lines in any .txt file.
```

Automated:
```bash
go test ./...
go vet ./...
```
Add explicit coverage for: filter drops messages with `ID <= LastRowID`;
append mode produces concatenated output (not truncated); state.json round-
trips correctly across runs; `--incremental --pdf` rejected by
`ValidateOptions`.

---

## Critical files

- `internal/bagoup/state.go` (new)
- `internal/bagoup/bagoup.go` — `validatePaths`, `setupLogging`, `Run`,
  `configuration` struct
- `internal/bagoup/options.go` — `Options`, `ValidateOptions`
- `internal/bagoup/export.go` — `prepareEntityJobs`
- `internal/bagoup/write.go` — `writeJob`, `prepareFileJobs`, `writeChunk`,
  `handleFileContents`
- `opsys/opsys.go` (+ `opsys/mock_opsys/mock_opsys.go`)
- `README.md`
