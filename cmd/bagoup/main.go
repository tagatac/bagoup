// bagoup - An export utility for macOS Messages.
// Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"time"

	"github.com/jessevdk/go-flags"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/v2/chatdb"
	"github.com/tagatac/bagoup/v2/internal/bagoup"
	"github.com/tagatac/bagoup/v2/opsys"
	"github.com/tagatac/bagoup/v2/pathtools"
)

const _license = "Copyright (C) 2020-2023  David Tagatac <david@tagatac.net>\nSee the source for usage terms."

var _version string

func main() {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				slog.Error(e.Error())
			}
		}
	}()

	startTime := time.Now()
	var opts struct {
		bagoup.Options
		CPUProfile string `long:"cpuprofile" description:"Write CPU profile to this file"`
		MemProfile string `long:"memprofile" description:"Write memory profile to this file"`
		Trace      string `long:"trace" description:"Write execution trace to this file"`
	}
	_, err := flags.Parse(&opts)
	if err != nil && err.(*flags.Error).Type == flags.ErrHelp {
		return
	}
	panicOnErr(err, "parse flags")
	panicOnErr(bagoup.ValidateOptions(opts.Options), "validate options")
	if opts.PrintVersion {
		fmt.Printf("bagoup version %s\n%s\n", _version, _license)
		return
	}

	ptools, err := pathtools.NewPathTools()
	panicOnErr(err, "create pathtools")
	opts.DBPath = ptools.ReplaceTilde(opts.DBPath)

	s := opsys.NewOS(afero.NewOsFs(), os.Stat, _version)
	db, err := sql.Open("sqlite3", opts.DBPath)
	panicOnErr(err, "open DB file %q", opts.DBPath)
	db.SetMaxOpenConns(max(1, runtime.NumCPU()-1))
	defer db.Close()
	cdb := chatdb.NewChatDB(db, opts.SelfHandle)

	if opts.Trace != "" {
		f, err := os.Create(opts.Trace)
		panicOnErr(err, "create trace file %q", opts.Trace)
		panicOnErr(trace.Start(f), "start trace")
		defer f.Close()
		defer trace.Stop()
	}

	if opts.CPUProfile != "" {
		f, err := os.Create(opts.CPUProfile)
		panicOnErr(err, "create CPU profile %q", opts.CPUProfile)
		panicOnErr(pprof.StartCPUProfile(f), "start CPU profile")
		defer f.Close()
		defer pprof.StopCPUProfile()
	}

	logDir := filepath.Join(opts.ExportPath, ".bagoup")
	cfg, err := bagoup.NewConfiguration(opts.Options, s, cdb, ptools, logDir, startTime, _version)
	panicOnErr(err, "create bagoup configuration")
	panicOnErr(cfg.Run(), "run bagoup")

	if opts.MemProfile != "" {
		f, err := os.Create(opts.MemProfile)
		panicOnErr(err, "create memory profile %q", opts.MemProfile)
		runtime.GC()
		panicOnErr(pprof.WriteHeapProfile(f), "write memory profile")
		panicOnErr(f.Close(), "close memory profile %q", opts.MemProfile)
	}
	panicOnErr(db.Close(), "close DB file %q", opts.DBPath)
	dbf, err := os.Open(opts.DBPath)
	panicOnErr(err, "open DB file %q for copying", opts.DBPath)
	defer dbf.Close()
	dbfNewPath := filepath.Join(logDir, filepath.Base(opts.DBPath))
	dbfNew, err := os.Create(dbfNewPath)
	panicOnErr(err, "create file %q to copy chat DB into", dbfNewPath)
	defer dbfNew.Close()
	_, err = io.Copy(dbfNew, dbf)
	panicOnErr(err, "copy DB file from %q to %q", opts.DBPath, dbfNewPath)
	panicOnErr(dbf.Close(), "close DB file %q after copying", opts.DBPath)
	panicOnErr(dbfNew.Close(), "close DB copy %q", dbfNewPath)
}

func panicOnErr(err error, format string, args ...any) {
	if err != nil {
		panic(fmt.Errorf(format+": %w", append(args, err)...))
	}
}
