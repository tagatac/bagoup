// Copyright (C) 2020 David Tagatac <david@tagatac.net>
// See the COPYING and LICENSE files for full usage terms.

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/Masterminds/semver"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

const (
	_dbFileName   = "chat.db"
	_backupFolder = "backup"
)

var _reqOSVersion = semver.MustParse("10.13")

func main() {
	ok, err := validateOSVersion()
	if err != nil {
		log.Fatal(errors.Wrap(err, "validate OS version"))
	}
	if !ok {
		log.Fatalf("invalid OS version; update to Mac OS %s or newer", _reqOSVersion)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "get working directory"))
	}
	dbPath := path.Join(wd, _dbFileName)
	backupPath := path.Join(wd, _backupFolder)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "open DB file %q", dbPath))
	}
	defer db.Close()
	cdb, err := NewChatDB(db)
	if err != nil {
		log.Fatal(errors.Wrap(err, "create ChatDB"))
	}

	chats, err := cdb.GetChats()
	if err != nil {
		log.Fatal(errors.Wrap(err, "get chats"))
	}
	for _, chat := range chats {
		chatDirPath := path.Join(backupPath, chat.DisplayName)
		if err := os.MkdirAll(chatDirPath, os.ModePerm); err != nil {
			log.Fatal(errors.Wrapf(err, "create directory %q", chatDirPath))
		}
		chatPath := path.Join(chatDirPath, fmt.Sprintf("%s.txt", chat.GUID))
		chatFile, err := os.OpenFile(chatPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(errors.Wrapf(err, "open/create file %s", chatPath))
		}
		defer chatFile.Close()

		messageIDs, err := cdb.GetMessageIDs(chat.ID)
		if err != nil {
			log.Fatal(errors.Wrapf(err, "get message IDs for chat ID %d", chat.ID))
		}
		for _, messageID := range messageIDs {
			msg, err := cdb.GetMessage(messageID)
			if err != nil {
				log.Fatal(errors.Wrapf(err, "get message with ID %d", messageID))
			}
			if _, err := chatFile.WriteString(msg); err != nil {
				log.Fatal(errors.Wrapf(err, "write message %q to file %q", msg, chatFile.Name()))
			}
		}
	}
}

func validateOSVersion() (bool, error) {
	cmd := exec.Command("sw_vers", "-productVersion")
	o, err := cmd.Output()
	if err != nil {
		return false, errors.Wrap(err, "call sw_vers")
	}
	vstr := strings.TrimSuffix(string(o), "\n")
	v, err := semver.NewVersion(vstr)
	if err != nil {
		return false, errors.Wrapf(err, "parse semantic version %q", vstr)
	}
	return !v.LessThan(_reqOSVersion), nil
}
