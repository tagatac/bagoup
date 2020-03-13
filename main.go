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
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"unicode"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	"github.com/jessevdk/go-flags"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/chatdb"
)

func main() {
	var opts struct {
		DBPath       string  `short:"d" long:"db-path" description:"Path to the Messages chat database file" default:"~/Library/Messages/chat.db"`
		ContactsPath string  `short:"c" long:"contacts-path" description:"Path to the contacts vCard file" default:"contacts.vcf"`
		ExportPath   string  `short:"o" long:"export-path" description:"Path to which the Messages will be exported" default:"backup"`
		MacOSVersion *string `short:"v" long:"mac-os-version" description:"Version of Mac OS from which the Messages chat database file was copied"`
		SelfHandle   string  `short:"h" long:"self-handle" description:"Prefix to use for for messages sent by you" default:"Me"`
	}
	_, err := flags.Parse(&opts)
	if err != nil && err.(*flags.Error).Type == flags.ErrHelp {
		os.Exit(0)
	}
	exitOnError("parse flags", err)

	if _, err := os.Stat(opts.ExportPath); !os.IsNotExist(err) {
		exitOnError(fmt.Sprintf("check export path %q", opts.ExportPath), err)
		log.Fatalf("ERROR: export folder %q already exists - move it or specify a different export path", opts.ExportPath)
	}

	var macOSVersion *semver.Version
	if opts.MacOSVersion != nil {
		macOSVersion, err = semver.NewVersion(*opts.MacOSVersion)
		exitOnError(fmt.Sprintf("parse Mac OS version %q", *opts.MacOSVersion), err)
	} else {
		macOSVersion, err = getMacOSVersion(exec.Command)
		exitOnError("get Mac OS version - see bagoup --help about the mac-os-version option", err)
	}

	db, err := sql.Open("sqlite3", opts.DBPath)
	exitOnError(fmt.Sprintf("open DB file %q", opts.DBPath), err)
	defer db.Close()
	contactMap, err := getContactMap(opts.ContactsPath, afero.NewOsFs())
	exitOnError(fmt.Sprintf("get contacts from vcard file %q", opts.ContactsPath), err)
	cdb, err := chatdb.NewChatDB(db, contactMap, macOSVersion, opts.SelfHandle)
	exitOnError("create ChatDB", err)

	exitOnError("export chats", exportChats(cdb, opts.ExportPath, afero.NewOsFs()))
}

func exitOnError(activity string, err error) {
	if err != nil {
		log.Fatalf("ERROR: %s", errors.Wrap(err, activity))
	}
}

func getMacOSVersion(execCommand func(string, ...string) *exec.Cmd) (*semver.Version, error) {
	cmd := execCommand("sw_vers", "-productVersion")
	o, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "call sw_vers")
	}
	vstr := strings.TrimSuffix(string(o), "\n")
	v, err := semver.NewVersion(vstr)
	if err != nil {
		return nil, errors.Wrapf(err, "parse semantic version %q", vstr)
	}
	return v, nil
}

func getContactMap(contactsFilePath string, fs afero.Fs) (map[string]*vcard.Card, error) {
	f, err := fs.Open(contactsFilePath)
	if os.IsNotExist(err) {
		log.Print(errors.Wrapf(err, "open file %q - continuing without contacts", contactsFilePath))
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "open file %q", contactsFilePath)
	}
	defer f.Close()

	dec := vcard.NewDecoder(f)
	contactMap := map[string]*vcard.Card{}
	for {
		card, err := dec.Decode()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrapf(err, "decode vcard")
		}
		phones := card.Values(vcard.FieldTelephone)
		for i, phone := range phones {
			phones[i] = sanitizePhone(phone)
		}
		phonesAndEmails := append(phones, card.Values(vcard.FieldEmail)...)
		for _, phoneOrEmail := range phonesAndEmails {
			if c, ok := contactMap[phoneOrEmail]; ok {
				log.Printf("multiple contacts %q and %q share the same phone or email %q", c.PreferredValue(vcard.FieldFormattedName), card.PreferredValue(vcard.FieldFormattedName), phoneOrEmail)
			}
			contactMap[phoneOrEmail] = &card
		}
	}
	return contactMap, nil
}

// Adapted from https://stackoverflow.com/a/44009184/5403337
func sanitizePhone(dirty string) string {
	return strings.Map(
		func(r rune) rune {
			if strings.ContainsRune("()-", r) || unicode.IsSpace(r) {
				return -1
			}
			return r
		},
		dirty,
	)
}

func exportChats(cdb chatdb.ChatDB, exportPath string, fs afero.Fs) error {
	chats, err := cdb.GetChats()
	if err != nil {
		return errors.Wrap(err, "get chats")
	}
	for _, chat := range chats {
		chatDirPath := path.Join(exportPath, chat.DisplayName)
		if err := fs.MkdirAll(chatDirPath, os.ModePerm); err != nil {
			return errors.Wrapf(err, "create directory %q", chatDirPath)
		}
		chatPath := path.Join(chatDirPath, fmt.Sprintf("%s.txt", chat.GUID))
		chatFile, err := fs.OpenFile(chatPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return errors.Wrapf(err, "open/create file %s", chatPath)
		}
		defer chatFile.Close()

		messageIDs, err := cdb.GetMessageIDs(chat.ID)
		if err != nil {
			return errors.Wrapf(err, "get message IDs for chat ID %d", chat.ID)
		}
		for _, messageID := range messageIDs {
			msg, err := cdb.GetMessage(messageID)
			if err != nil {
				return errors.Wrapf(err, "get message with ID %d", messageID)
			}
			if _, err := chatFile.WriteString(msg); err != nil {
				return errors.Wrapf(err, "write message %q to file %q", msg, chatFile.Name())
			}
		}
		chatFile.Close()
	}
	return nil
}
