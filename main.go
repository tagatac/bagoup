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
	"path"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	"github.com/jessevdk/go-flags"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/chatdb"
	"github.com/tagatac/bagoup/opsys"
)

const _readmeURL = "https://github.com/tagatac/bagoup/blob/master/README.md#chatdb-access"
const _defaultDBPath = "~/Library/Messages/chat.db"

type options struct {
	DBPath        string  `short:"i" long:"db-path" description:"Path to the Messages chat database file" default:"~/Library/Messages/chat.db"`
	ExportPath    string  `short:"o" long:"export-path" description:"Path to which the Messages will be exported" default:"backup"`
	MacOSVersion  *string `short:"m" long:"mac-os-version" description:"Version of Mac OS, e.g. '10.15', from which the Messages chat database file was copied (not needed if bagoup is running on the same Mac)"`
	ContactsPath  *string `short:"c" long:"contacts-path" description:"Path to the contacts vCard file"`
	SelfHandle    string  `short:"s" long:"self-handle" description:"Prefix to use for for messages sent by you" default:"Me"`
	SeparateChats bool    `long:"separate-chats" description:"Do not merge chats with the same contact into a single file, e.g. iMessage and SMS"`
}

type configuration struct {
	opsys.OS
	chatdb.ChatDB
	ExportPath   string
	MacOSVersion *semver.Version
	HandleMap    map[int]string
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

	config := configuration{
		OS:         s,
		ChatDB:     cdb,
		ExportPath: opts.ExportPath,
	}
	logFatalOnErr(bagoup(opts, config))
}

func logFatalOnErr(err error) {
	if err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}

func bagoup(opts options, config configuration) error {
	if err := validatePaths(config.OS, opts); err != nil {
		return err
	}

	var err error
	if opts.MacOSVersion != nil {
		config.MacOSVersion, err = semver.NewVersion(*opts.MacOSVersion)
		if err != nil {
			return errors.Wrapf(err, "parse Mac OS version %q", *opts.MacOSVersion)
		}
	} else if config.MacOSVersion, err = config.OS.GetMacOSVersion(); err != nil {
		return errors.Wrap(err, "get Mac OS version - FIX: specify the Mac OS version from which chat.db was copied with the --mac-os-version option")
	}

	var contactMap map[string]*vcard.Card
	if opts.ContactsPath != nil {
		contactMap, err = config.OS.GetContactMap(*opts.ContactsPath)
		if err != nil {
			return errors.Wrapf(err, "get contacts from vcard file %q", *opts.ContactsPath)
		}
	}

	config.HandleMap, err = config.ChatDB.GetHandleMap(contactMap)
	if err != nil {
		return errors.Wrap(err, "get handle map")
	}

	count, err := exportChats(config, contactMap, !opts.SeparateChats)
	fmt.Printf("%d messages successfully exported to folder %q\n", count, opts.ExportPath)
	if err != nil {
		return errors.Wrap(err, "export chats")
	}
	return nil
}

func validatePaths(s opsys.OS, opts options) error {
	if opts.DBPath == _defaultDBPath {
		f, err := s.Open(opts.DBPath)
		if err != nil {
			return errors.Wrapf(err, "test DB file %q - FIX: %s", opts.DBPath, _readmeURL)
		}
		f.Close()
	}

	if exist, err := s.FileExist(opts.ExportPath); exist {
		return fmt.Errorf("export folder %q already exists - FIX: move it or specify a different export path with the --export-path option", opts.ExportPath)
	} else if err != nil {
		return errors.Wrapf(err, "check export path %q", opts.ExportPath)
	}
	return nil
}

func exportChats(config configuration, contactMap map[string]*vcard.Card, mergeChats bool) (int, error) {
	count := 0
	chats, err := config.ChatDB.GetChats(contactMap)
	if err != nil {
		return count, errors.Wrap(err, "get chats")
	}
	for _, entityChats := range chats {
		var guids []string
		var entityMessageIDs []chatdb.DatedMessageID
		for _, chat := range entityChats.Chats {
			messageIDs, err := config.ChatDB.GetMessageIDs(chat.ID)
			if err != nil {
				return count, errors.Wrapf(err, "get message IDs for chat ID %d", chat.ID)
			}
			if mergeChats {
				guids = append(guids, chat.GUID)
				entityMessageIDs = append(entityMessageIDs, messageIDs...)
			} else {
				thisCount, err := writeFile(config, entityChats.Name, []string{chat.GUID}, messageIDs)
				if err != nil {
					return count + thisCount, err
				}
				count += thisCount
			}
		}
		if mergeChats {
			thisCount, err := writeFile(config, entityChats.Name, guids, entityMessageIDs)
			if err != nil {
				return count + thisCount, err
			}
			count += thisCount
		}
	}
	return count, nil
}

func writeFile(config configuration, entityName string, guids []string, messageIDs []chatdb.DatedMessageID) (int, error) {
	chatDirPath := path.Join(config.ExportPath, entityName)
	if err := config.OS.MkdirAll(chatDirPath, os.ModePerm); err != nil {
		return 0, errors.Wrapf(err, "create directory %q", chatDirPath)
	}
	fileName := strings.Join(guids, ";;;")
	chatPath := path.Join(chatDirPath, fmt.Sprintf("%s.txt", fileName))
	chatFile, err := config.OS.OpenFile(chatPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, errors.Wrapf(err, "open/create file %s", chatPath)
	}
	defer chatFile.Close()

	sort.SliceStable(messageIDs, func(i, j int) bool { return messageIDs[i].Date < messageIDs[j].Date })
	var count int
	for _, messageID := range messageIDs {
		msg, err := config.ChatDB.GetMessage(messageID.ID, config.HandleMap, config.MacOSVersion)
		if err != nil {
			return count, errors.Wrapf(err, "get message with ID %d", messageID)
		}
		if _, err := chatFile.WriteString(msg); err != nil {
			return count, errors.Wrapf(err, "write message %q to file %q", msg, chatFile.Name())
		}
		count++
	}
	chatFile.Close()
	return count, nil
}
