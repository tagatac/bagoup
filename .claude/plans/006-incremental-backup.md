# Plan: Incremental backup (issue #33)

## Context

bagoup currently refuses to write to an existing export directory
(`internal/bagoup/bagoup.go:302-303`), so users must export everything from
scratch every time. Issue #33 asks for an incremental mode that appends only
messages received since the last run. The repo owner suggested a "pointer
file" approach in the issue thread and limited it to text output (PDFs can't
be appended).

This plan adds an `--incremental` flag, and **always** writes a state file
`<ExportPath>/.bagoup/state.json` regardless of whether the flag was passed.
That way any normal export automatically becomes eligible for a future
incremental run — the flag is only needed at *re-run* time to opt into
appending. On re-run, bagoup loads state, filters messages by **timestamp**
(Apple-epoch nanoseconds from `message.date`), and appends to the existing
`.txt` files.

### Design decisions (and why)

- **Timestamps, not ROWIDs.** ROWIDs are convenient but reset whenever
  `chat.db` is recreated and don't translate across Macs. Timestamps
  (Apple-epoch nanoseconds via `DatedMessageID.Date`, already normalized for
  modern/legacy in `chatdb/message.go:40-43`) survive both. Caveat: a
  message that syncs late with an older send-date can fall below the
  watermark and be skipped — documented in the README. This is the same
  tradeoff the owner accepted when proposing a pointer file.

- **Per-output-file watermarks, not a single global watermark.** Users can
  filter with `--entity Alice` and then later re-run without the filter,
  expecting Bob's older messages to also flow in. A global watermark would
  silently drop those. Per-file keying matches the unit of output the user
  actually sees and is correct under any combination of `--entity` /
  `--separate-chats`.

- **State written on every run, not just incremental.** First run produces
  the state file as a side effect so a later `--incremental` re-run "just
  works". No magic, no `-w` migration step.

- **Filename stability across runs is silent, not warned.** When new chat
  GUIDs join an entity between runs, the *computed* filename (`guid1;;;guid2`)
  would change. We store the original filename in state and reuse it
  verbatim. No warning — it's the documented, intended behavior.

- **Structural-flag drift is rejected at incremental re-run time.** Some
  flags change the *shape* of the output (separate-chats, copy-attachments,
  preserve-paths, pdf, timezone, self-handle). Letting them drift across
  incremental runs would corrupt a single file (mixed timezones, mixed
  sender labels, half the chats merged vs. split). Re-runs with `--incremental`
  must match the originally-recorded values for these flags or error out
  with a clear message naming the specific mismatch. Non-structural flags
  (db-path, mac-os-version, contacts-path, attachments-path, entity,
  profiling) are free to change.

- **Version differences are tolerated silently.** Per user direction: any
  export with a parseable `state.json` is accepted, regardless of the
  bagoup version that produced it. The structural-flag check above gives
  us enough safety; pinning to a specific version adds friction without
  proportional benefit.

---

## State file schema

`<ExportPath>/.bagoup/state.json`:

```go
type State struct {
    BagoupVersion string                   `json:"bagoup_version"`
    MacOSVersion  string                   `json:"mac_os_version"`
    UpdatedAt     time.Time                `json:"updated_at"`
    Signature     StateSignature           `json:"signature"`
    Files         map[string]FileWatermark `json:"files"`
}
type StateSignature struct {
    SeparateChats   bool   `json:"separate_chats"`
    CopyAttachments bool   `json:"copy_attachments"`
    PreservePaths   bool   `json:"preserve_paths"`
    OutputPDF       bool   `json:"output_pdf"`
    Timezone        string `json:"timezone"`
    SelfHandle      string `json:"self_handle"`
}
type FileWatermark struct {
    Filename       string   `json:"filename"`         // basename incl. ".txt"
    GUIDs          []string `json:"guids"`            // GUIDs known at creation
    LastMessageDate int64   `json:"last_message_date"` // Apple-epoch nanoseconds
}
```

Key composition (in `state.StateKey`):
- merge mode (default): `entityName`
- separate-chats mode: `entityName + "/" + chatGUID`

Path: `filepath.Join(exportPath, ".bagoup", "state.json")`. Save atomically
(write `state.json.tmp`, rename) to survive a crash mid-write.

---

## Files to change

### New: `internal/bagoup/state.go`

Pure data + I/O. Only depends on `opsys.OS` and the standard library.

```go
func LoadState(s opsys.OS, exportPath string) (*State, bool, error)
func (st *State) Save(s opsys.OS, exportPath string) error
func StateKey(entityName string, chatGUID string, mergeChats bool) string
func SignatureFromOptions(opts Options) StateSignature
func (a StateSignature) Diff(b StateSignature) []string // empty when equal
```

### `internal/bagoup/options.go`

- Add field: `Incremental bool \`long:"incremental" description:"Append only messages received since the last export (requires an existing export directory containing .bagoup/state.json)"\``.
- In `ValidateOptions`, reject `Incremental && OutputPDF` ("--incremental
  requires text output; remove --pdf"). The remaining structural-flag
  checks happen against `state.json` in `validatePaths`, not here, because
  the comparison is against persisted values.

### `internal/bagoup/bagoup.go`

- Add fields to `configuration`: `state *State`, `stateMu sync.Mutex`.
- `validatePaths` (line 291): branch on `cfg.Options.Incremental`:
  - **incremental + dir does not exist** → error ("--incremental requires
    an existing export directory at %q with .bagoup/state.json").
  - **incremental + dir exists + state.json present** → `LoadState`; call
    `state.Signature.Diff(SignatureFromOptions(cfg.Options))`; if the diff
    is non-empty, error listing the specific mismatched fields and the
    expected values. Set `cfg.state`.
  - **incremental + dir exists + no state.json** → error ("no incremental
    state at %q — was this directory produced by an earlier bagoup run?").
  - **non-incremental + dir exists** → keep the existing rejection at
    line 302-303 (unchanged).
  - **non-incremental + dir does not exist** → fresh export; initialize an
    empty `cfg.state` so the run records state at the end.
- `setupLogging` (line 177): when `cfg.state != nil && cfg.Options.Incremental`,
  open `.bagoup/out.log` with `cfg.OS.OpenAppend` instead of `Create` so
  previous-run logs are preserved.
- `Run` (line 135): after `exportChats` returns nil, populate `cfg.state`'s
  `BagoupVersion` / `MacOSVersion` / `UpdatedAt` / `Signature` and call
  `cfg.state.Save`. This happens for every run, not just incremental.

### `opsys/opsys.go` + `opsys/mock_opsys/`

- Add to the `OS` interface:
  ```go
  OpenAppend(fp string) (afero.File, error)
  ```
- Implementation: `s.Fs.OpenFile(fp, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)`.
- Regenerate mocks via the existing `go:generate mockgen` directive.

### `internal/bagoup/export.go`

- `prepareEntityJobs` (line 129):
  - For each chat, compute `key := state.StateKey(entityChats.Name,
    chat.GUID, mergeChats)`.
  - If `cfg.state != nil` and `cfg.state.Files[key]` exists, filter
    `messageIDs` to those with `id.Date > watermark.LastMessageDate`.
  - In merge mode, look up the watermark by `entityName`; in
    separate-chats mode, by `entityName + "/" + chatGUID`.
  - When merging and an existing watermark is found, pass its
    `Filename` and `GUIDs` through into `prepareFileJobs` so the same
    output file is reused.
  - Skip producing a write job entirely when no new messages remain.

### `internal/bagoup/write.go`

- Extend `writeJob` with `stateKey string`, `appendMode bool`,
  `storedFilename string`, `guids []string`.
- `prepareFileJobs` (line 36): when `storedFilename` is non-empty, use it
  verbatim instead of recomputing from GUIDs / `;;;` / 251-char truncation.
  Guarantees the same file is appended to even if the entity's GUID set
  has changed.
- `writeChunk` (line 75): if `job.appendMode`, call `cfg.OS.OpenAppend`;
  else `cfg.OS.Create`. PDF branch is unreachable here when incremental
  (gated off by `ValidateOptions`).
- `handleFileContents` (line 98): track `maxDate := max(messageID.Date, ...)`
  while iterating. After `outFile.Flush` succeeds:
  ```go
  cfg.stateMu.Lock()
  cur := cfg.state.Files[job.stateKey]
  if int64(maxDate) > cur.LastMessageDate {
      cur.LastMessageDate = int64(maxDate)
  }
  if cur.Filename == "" {
      cur.Filename = filepath.Base(job.chatPath)
      cur.GUIDs = job.guids
  }
  cfg.state.Files[job.stateKey] = cur
  cfg.state.Save(cfg.OS, cfg.Options.ExportPath) // per-file save for crash safety
  cfg.stateMu.Unlock()
  ```

### `chatdb/message.go`

No code changes. `DatedMessageID.Date` is already normalized to Apple-epoch
nanoseconds on both the modern (line 40-43) and legacy (line 65-80) paths,
so the comparison is uniform.

### `README.md`

Add an "Incremental backups" subsection under Usage:

- First run (any `bagoup` invocation): writes `messages-export/.bagoup/state.json`.
- Subsequent run: `bagoup --incremental -o messages-export` appends new
  messages to the existing per-entity `.txt` files.
- Caveats:
  - Don't delete `.bagoup/state.json`; without it `--incremental` will
    error.
  - The following flags must match between the original run and the
    `--incremental` re-run, or bagoup will refuse: `--separate-chats`,
    `--copy-attachments`, `--preserve-paths`, `--pdf` (must remain off),
    `--timezone`, `--self-handle`.
  - Late-syncing messages with send-dates older than the watermark are
    skipped. This is rare for normal usage; if you need to re-capture
    them, start a fresh export directory.
  - `--incremental` only works with text output (not PDF).

---

## Reused existing infrastructure

- `opsys.OS.CopyFile(src, dst, unique=true)` (`opsys/opsys.go`) — already
  deduplicates attachment copies safely; no changes needed for re-runs.
- `pathtools.PathTools` — existing re-run support via `--attachments-path`
  + `.tildeexpansion` is the precedent for sentinel/state files at the
  export root.
- `afero.NewMemMapFs` — for unit tests of state I/O.
- `go-sqlmock` + `gomock` — existing patterns for chatdb / write tests.

---

## Implementation order (small, testable steps)

1. Add `OpenAppend` to `opsys.OS`; regenerate mock; unit test.
2. Add `internal/bagoup/state.go` + `state_test.go` (round-trip,
   missing file, corrupted JSON, atomic-rename, `StateSignature.Diff`).
3. Add `--incremental` flag + `ValidateOptions` rule + `options_test.go`
   cases.
4. Wire `validatePaths` and `setupLogging` paths in `bagoup.go`; tests in
   `bagoup_test.go` (non-existent dir + incremental → error; existing dir
   + state + matching signature → loads; existing dir + state + mismatched
   signature → error names the bad field; existing dir + no state +
   incremental → error; non-incremental fresh run → still writes state at
   end; log append).
5. Always-save: wire `state.Save` into `Run` for non-incremental runs;
   verify state.json appears in golden export tests.
6. Thread state into `prepareEntityJobs` / `prepareFileJobs` / `writeJob`;
   add date filtering and stored-filename reuse; tests in `export_test.go`
   (watermark filter; no-new-messages skip; new GUID silently reuses
   filename; separate-chats keying).
7. Add append routing in `writeChunk` and per-file watermark recording in
   `handleFileContents`; tests in `write_test.go` (writes appended, not
   truncated; watermark equals max date written; per-file save called
   after Flush).
8. README update.

---

## Verification

End-to-end manual test:

```bash
# First run — produces export and state (no flag needed)
go run ./cmd/bagoup -i testdata/chat.db -o /tmp/messages-export
jq . /tmp/messages-export/.bagoup/state.json

# Add new messages (or swap in a newer chat.db snapshot), re-run with flag
go run ./cmd/bagoup --incremental -i testdata/chat.db.v2 -o /tmp/messages-export

# Confirm:
# - existing .txt files grew (tail shows the new lines)
# - state.json's last_message_date increased
# - out.log was appended, not truncated
# - no duplicate lines anywhere
# - same command with --separate-chats now errors with a signature-mismatch
#   message naming "separate_chats"
```

Automated: `go test ./... && go vet ./...`

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
