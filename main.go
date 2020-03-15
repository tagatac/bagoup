// bagoup - An export utility for Mac OS Messages.
// Copyright (C) 2020 David Tagatac <david@tagatac.net>

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
	"log"
	"os"
	"os/exec"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	"github.com/jessevdk/go-flags"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/chatdb"
	"github.com/tagatac/bagoup/opsys"
)

type options struct {
	DBPath       string  `short:"d" long:"db-path" description:"Path to the Messages chat database file" default:"~/Library/Messages/chat.db"`
	ContactsPath *string `short:"c" long:"contacts-path" description:"Path to the contacts vCard file"`
	ExportPath   string  `short:"o" long:"export-path" description:"Path to which the Messages will be exported" default:"backup"`
	MacOSVersion *string `short:"v" long:"mac-os-version" description:"Version of Mac OS from which the Messages chat database file was copied"`
	SelfHandle   string  `short:"h" long:"self-handle" description:"Prefix to use for for messages sent by you" default:"Me"`
}

func main() {
	var opts options
	_, err := flags.Parse(&opts)
	if err != nil && err.(*flags.Error).Type == flags.ErrHelp {
		os.Exit(0)
	}
	logFatalOnErr(errors.Wrap(err, "parse flags"))

	s := opsys.NewOS(afero.NewOsFs(), os.Stat, exec.Command)
	db, err := sql.Open("sqlite3", opts.DBPath)
	logFatalOnErr(errors.Wrapf(err, "open DB file %q", opts.DBPath))
	defer db.Close()
	cdb := chatdb.NewChatDB(db, opts.SelfHandle)

	logFatalOnErr(bagoup(opts, s, cdb))
}

func logFatalOnErr(err error) {
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}

func bagoup(opts options, s opsys.OS, cdb chatdb.ChatDB) error {
	if exist, err := s.FileExist(opts.ExportPath); exist {
		return fmt.Errorf("export folder %q already exists - move it or specify a different export path", opts.ExportPath)
	} else if err != nil {
		return errors.Wrapf(err, "check export path %q", opts.ExportPath)
	}

	var macOSVersion *semver.Version
	var err error
	if opts.MacOSVersion != nil {
		macOSVersion, err = semver.NewVersion(*opts.MacOSVersion)
		if err != nil {
			return errors.Wrapf(err, "parse Mac OS version %q", *opts.MacOSVersion)
		}
	} else if macOSVersion, err = s.GetMacOSVersion(); err != nil {
		return errors.Wrap(err, "get Mac OS version - specify the Mac OS version from which chat.db was copied with the --mac-os-version option")
	}

	var contactMap map[string]*vcard.Card
	if opts.ContactsPath != nil {
		contactMap, err = s.GetContactMap(*opts.ContactsPath)
		if err != nil {
			return errors.Wrapf(err, "get contacts from vcard file %q", *opts.ContactsPath)
		}
	}

	handleMap, err := cdb.GetHandleMap(contactMap)
	if err != nil {
		return errors.Wrap(err, "get handle map")
	}

	return errors.Wrap(s.ExportChats(cdb, opts.ExportPath, contactMap, handleMap, macOSVersion), "export chats")
}
