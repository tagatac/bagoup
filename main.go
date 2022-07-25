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
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/Masterminds/semver"
	"github.com/jessevdk/go-flags"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/chatdb"
	"github.com/tagatac/bagoup/opsys"
	"github.com/tagatac/bagoup/pathtools"
)

const _license = "Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>\nSee the source for usage terms."
const _readmeURL = "https://github.com/tagatac/bagoup/blob/master/README.md#protected-file-access"

var _version string

type (
	options struct {
		DBPath          string  `short:"i" long:"db-path" description:"Path to the Messages chat database file" default:"~/Library/Messages/chat.db"`
		ExportPath      string  `short:"o" long:"export-path" description:"Path to which the Messages will be exported" default:"messages-export"`
		MacOSVersion    *string `short:"m" long:"mac-os-version" description:"Version of Mac OS, e.g. '10.15', from which the Messages chat database file was copied (not needed if bagoup is running on the same Mac)"`
		ContactsPath    *string `short:"c" long:"contacts-path" description:"Path to the contacts vCard file"`
		SelfHandle      string  `short:"s" long:"self-handle" description:"Prefix to use for for messages sent by you" default:"Me"`
		SeparateChats   bool    `long:"separate-chats" description:"Do not merge chats with the same contact (e.g. iMessage and SMS) into a single file"`
		OutputPDF       bool    `short:"p" long:"pdf" description:"Export text and images to PDF files (requires full disk access)"`
		IncludePPA      bool    `long:"include-ppa" description:"Include plugin payload attachments (e.g. link previews) in generated PDFs"`
		CopyAttachments bool    `short:"a" long:"copy-attachments" description:"Copy attachments to the same folder as the chat which included them (requires full disk access)"`
		PreservePaths   bool    `short:"r" long:"preserve-paths" description:"When copying attachments, preserve the full path instead of co-locating them with the chats which included them"`
		PrintVersion    bool    `short:"v" long:"version" description:"Show the version of bagoup"`
	}
	configuration struct {
		Options options
		opsys.OS
		chatdb.ChatDB
		MacOSVersion    *semver.Version
		HandleMap       map[int]string
		AttachmentPaths map[int][]chatdb.Attachment
		Counts          counts
		StartTime       time.Time
	}
	counts struct {
		files               int
		chats               int
		messages            int
		attachments         map[string]int
		attachmentsEmbedded map[string]int
		attachmentsMissing  int
		conversions         int
		conversionsFailed   int
	}
)

func main() {
	startTime := time.Now()
	var opts options
	_, err := flags.Parse(&opts)
	if err != nil && err.(*flags.Error).Type == flags.ErrHelp {
		os.Exit(0)
	}
	logFatalOnErr(errors.Wrap(err, "parse flags"))
	if opts.PrintVersion {
		fmt.Printf("bagoup version %s\n%s\n", _version, _license)
		return
	}

	dbPath, err := pathtools.ReplaceTilde(opts.DBPath)
	logFatalOnErr(errors.Wrap(err, "replace tilde"))
	opts.DBPath = dbPath

	s, err := opsys.NewOS(afero.NewOsFs(), os.Stat, exec.Command)
	logFatalOnErr(errors.Wrap(err, "instantiate OS"))
	db, err := sql.Open("sqlite3", opts.DBPath)
	logFatalOnErr(errors.Wrapf(err, "open DB file %q", opts.DBPath))
	defer db.Close()
	cdb := chatdb.NewChatDB(db, opts.SelfHandle)

	cfg := configuration{
		OS:        s,
		ChatDB:    cdb,
		Options:   opts,
		Counts:    counts{attachments: map[string]int{}, attachmentsEmbedded: map[string]int{}},
		StartTime: startTime,
	}
	logFatalOnErr(cfg.bagoup())
}

func logFatalOnErr(err error) {
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}
