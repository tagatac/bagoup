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
	var opts bagoup.Options
	_, err := flags.Parse(&opts)
	if err != nil && err.(*flags.Error).Type == flags.ErrHelp {
		return
	}
	panicOnErr(fmt.Errorf("parse flags: %w", err))
	panicOnErr(fmt.Errorf("validate options: %w", bagoup.ValidateOptions(opts)))
	if opts.PrintVersion {
		fmt.Printf("bagoup version %s\n%s\n", _version, _license)
		return
	}

	ptools, err := pathtools.NewPathTools()
	panicOnErr(fmt.Errorf("create pathtools: %w", err))
	opts.DBPath = ptools.ReplaceTilde(opts.DBPath)

	s := opsys.NewOS(afero.NewOsFs(), os.Stat, _version)
	db, err := sql.Open("sqlite3", opts.DBPath)
	panicOnErr(fmt.Errorf("open DB file %q: %w", opts.DBPath, err))
	defer db.Close()
	cdb := chatdb.NewChatDB(db, opts.SelfHandle)

	logDir := filepath.Join(opts.ExportPath, ".bagoup")
	cfg, err := bagoup.NewConfiguration(opts, s, cdb, ptools, logDir, startTime, _version)
	panicOnErr(fmt.Errorf("create bagoup configuration: %w", err))
	panicOnErr(cfg.Run())
	panicOnErr(fmt.Errorf("close DB file %q: %w", opts.DBPath, db.Close()))
	dbf, err := os.Open(opts.DBPath)
	panicOnErr(fmt.Errorf("open DB file %q for copying: %w", opts.DBPath, err))
	defer dbf.Close()
	dbfNewPath := filepath.Join(logDir, filepath.Base(opts.DBPath))
	dbfNew, err := os.Create(dbfNewPath)
	panicOnErr(fmt.Errorf("create file %q to copy chat DB into: %w", dbfNewPath, err))
	defer dbfNew.Close()
	_, err = io.Copy(dbfNew, dbf)
	panicOnErr(fmt.Errorf("copy DB file from %q to %q: %w", opts.DBPath, dbfNewPath, err))
	panicOnErr(fmt.Errorf("close DB file %q after copying: %w", opts.DBPath, dbf.Close()))
	panicOnErr(fmt.Errorf("close DB copy %q: %w", dbfNewPath, dbfNew.Close()))
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
