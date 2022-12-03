// bagoup - An export utility for Mac OS Messages.
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
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jessevdk/go-flags"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/chatdb"
	"github.com/tagatac/bagoup/internal/bagoup"
	"github.com/tagatac/bagoup/opsys"
	"github.com/tagatac/bagoup/opsys/scall"
	"github.com/tagatac/bagoup/pathtools"
)

const _license = "Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>\nSee the source for usage terms."

var _version string

func main() {
	startTime := time.Now()
	var opts bagoup.Options
	_, err := flags.Parse(&opts)
	if err != nil && err.(*flags.Error).Type == flags.ErrHelp {
		os.Exit(0)
	}
	logFatalOnErr(errors.Wrap(err, "parse flags"))
	if opts.PrintVersion {
		fmt.Printf("bagoup version %s\n%s\n", _version, _license)
		return
	}

	ptools, err := pathtools.NewPathTools()
	logFatalOnErr(errors.Wrap(err, "create pathtools"))
	opts.DBPath = ptools.ReplaceTilde(opts.DBPath)

	s, err := opsys.NewOS(afero.NewOsFs(), os.Stat, scall.NewSyscall())
	logFatalOnErr(errors.Wrap(err, "instantiate OS"))
	db, err := sql.Open("sqlite3", opts.DBPath)
	logFatalOnErr(errors.Wrapf(err, "open DB file %q", opts.DBPath))
	defer db.Close()
	cdb := chatdb.NewChatDB(db, opts.SelfHandle)

	logDir := filepath.Join(opts.ExportPath, ".bagoup")
	cfg, err := bagoup.NewConfiguration(opts, s, cdb, ptools, logDir, startTime, _version)
	logFatalOnErr(errors.Wrap(err, "create bagoup configuration"))
	logFatalOnErr(cfg.Run())
	logFatalOnErr(errors.Wrapf(db.Close(), "close DB file %q", opts.DBPath))
	dbf, err := os.Open(opts.DBPath)
	logFatalOnErr(errors.Wrapf(err, "open DB file %q for copying", opts.DBPath))
	defer dbf.Close()
	dbfNewPath := filepath.Join(logDir, filepath.Base(opts.DBPath))
	dbfNew, err := os.Create(dbfNewPath)
	logFatalOnErr(errors.Wrapf(err, "create file %q to copy chat DB into", dbfNewPath))
	defer dbfNew.Close()
	_, err = io.Copy(dbfNew, dbf)
	logFatalOnErr(errors.Wrapf(err, "copy DB file from %q to %q", opts.DBPath, dbfNewPath))
	logFatalOnErr(errors.Wrapf(dbf.Close(), "close DB file %q after copying", opts.DBPath))
	logFatalOnErr(errors.Wrapf(dbfNew.Close(), "close DB copy %q", dbfNewPath))
}

func logFatalOnErr(err error) {
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}
