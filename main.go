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

const _dbFileName = "chat.db"

// Adapted from https://apple.stackexchange.com/a/300997/267331
const _datetimeFormula = "(date/1000000000) + STRFTIME('%s', '2001-01-01 00:00:00'), 'unixepoch', 'localtime'"

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
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "open DB file %q", dbPath))
	}
	defer db.Close()

	handleMap, err := getHandles(db)
	if err != nil {
		log.Fatal(errors.Wrap(err, "get handles"))
	}

	chats, err := db.Query("SELECT ROWID, guid, chat_identifier, display_name FROM chat")
	if err != nil {
		log.Fatal(errors.Wrap(err, "get chats"))
	}
	defer chats.Close()
	skip := true
	for chats.Next() {
		if skip {
			skip = false
			continue
		}
		var chatID int
		var guid, chatName, displayName string
		if err := chats.Scan(&chatID, &guid, &chatName, &displayName); err != nil {
			log.Fatal(errors.Wrap(err, "read chat"))
		}
		if displayName == "" {
			displayName = chatName
		}
		fmt.Printf("CHATID: %d\nGUID: %s\nDISPLAYNAME: %s\n", chatID, guid, displayName)

		messageIDs, err := db.Query(fmt.Sprintf("SELECT message_id FROM chat_message_join WHERE chat_id=%d", chatID))
		if err != nil {
			log.Fatal(errors.Wrapf(err, "get message IDs for chat ID %d", chatID))
		}
		defer messageIDs.Close()
		for messageIDs.Next() {
			var messageID int
			if err := messageIDs.Scan(&messageID); err != nil {
				log.Fatal(errors.Wrapf(err, "read message ID for chat ID %d", chatID))
			}
			messages, err := db.Query(fmt.Sprintf("SELECT is_from_me, handle_id, text, DATETIME(%s) FROM message WHERE ROWID=%d", _datetimeFormula, messageID))
			if err != nil {
				log.Fatal(errors.Wrapf(err, "get message with ID %d", messageID))
			}
			defer messages.Close()
			messages.Next()
			var fromMe, handleID int
			var text, date string
			if err := messages.Scan(&fromMe, &handleID, &text, &date); err != nil {
				log.Fatal(errors.Wrapf(err, "read data for message ID %d", messageID))
			}
			fmt.Printf("FROMME: %d\nHANDLEID: %d\nHANDLE: %s\nTEXT: %s\ndate: %s\n", fromMe, handleID, handleMap[handleID], text, date)
			//fmt.Printf("TEXT: %s\n", text)
			if messages.Next() {
				log.Fatalf("multiple messages with the same ID: %d - message ID uniqeness assumption violated - open an issue at https://github.com/tagatac/bagoup/issues", messageID)
			}
			return
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

func getHandles(db *sql.DB) (map[int]string, error) {
	handleMap := make(map[int]string)
	handles, err := db.Query("SELECT ROWID, id FROM handle")
	if err != nil {
		return nil, errors.Wrap(err, "get handles")
	}
	defer handles.Close()
	for handles.Next() {
		var handleID int
		var handle string
		if err := handles.Scan(&handleID, &handle); err != nil {
			return nil, errors.Wrap(err, "read handle")
		}
		if _, ok := handleMap[handleID]; ok {
			return nil, fmt.Errorf("multiple handles with the same ID: %d - handle uniqueness assumption violated - open an issue at https://github.com/tagatac/bagoup/issues", handleID)
		}
		handleMap[handleID] = handle
	}
	return handleMap, nil
}
