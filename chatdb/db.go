// Copyright (C) 2020 David Tagatac <david@tagatac.net>
// See the COPYING and LICENSE files for full usage terms.

// Package chatdb provides an interface ChatDB for interacting with the Mac OS
// Messages database typically located at $HOME/Library/Messages/chat.db. See
// [this Medium post](https://towardsdatascience.com/heres-how-you-can-access-your-entire-imessage-history-on-your-mac-f8878276c6e9)
// for a decent primer on navigating the database. Specifically, this package is
// tailored to supporting the exporting of all messages in the database to
// readable, searchable text files.
package chatdb

import (
	"database/sql"
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	"github.com/pkg/errors"
)

const _githubIssueMsg = "open an issue at https://github.com/tagatac/bagoup/issues"

// Adapted from https://apple.stackexchange.com/a/300997/267331
const (
	_datetimeFormulaLegacy = "date + STRFTIME('%s', '2001-01-01 00:00:00'), 'unixepoch', 'localtime'"
	_datetimeFormula       = "(date/1000000000) + STRFTIME('%s', '2001-01-01 00:00:00'), 'unixepoch', 'localtime'"
)

// How to label messages sent by yourself.
const _selfHandle = "Me"

// Chat represents a row from the chat table.
type Chat struct {
	ID          int
	GUID        string
	DisplayName string
}

//go:generate mockgen -source=db.go -destination=../mocks/mock_chatdb/mock_db.go -copyright_file=../COPYING

type (
	// ChatDB extracts data from a Mac OS Messages database on disk.
	ChatDB interface {
		// GetChats returns a slice of Chat, effectively a table scan of the chat
		// table.
		GetChats() ([]Chat, error)
		// GetMessageIDs returns a slice of message IDs corresponding to a given
		// chat ID, in the order that the messages are timestamped.
		GetMessageIDs(chatID int) ([]int, error)
		// GetMessage returns a message retrieved from the database formatted for
		// writing to a chat file.
		GetMessage(messageID int) (string, error)
	}

	chatDB struct {
		*sql.DB
		handleMap       map[int]string
		contactMap      map[string]*vcard.Card
		datetimeFormula string
	}
)

// NewChatDB returns a ChatDB interface with a populated handle map.
func NewChatDB(db *sql.DB, contactMap map[string]*vcard.Card, macOSVersion *semver.Version) (ChatDB, error) {
	handleMap, err := getHandleMap(db, contactMap)
	if err != nil {
		return nil, errors.Wrap(err, "get handles")
	}
	return &chatDB{
		DB:              db,
		handleMap:       handleMap,
		contactMap:      contactMap,
		datetimeFormula: getDatetimeFormula(macOSVersion),
	}, nil
}

func getHandleMap(db *sql.DB, contactMap map[string]*vcard.Card) (map[int]string, error) {
	handleMap := make(map[int]string)
	handles, err := db.Query("SELECT ROWID, id FROM handle")
	if err != nil {
		return nil, errors.Wrap(err, "get handles from DB")
	}
	defer handles.Close()
	for handles.Next() {
		var handleID int
		var handle string
		if err := handles.Scan(&handleID, &handle); err != nil {
			return nil, errors.Wrap(err, "read handle")
		}
		if _, ok := handleMap[handleID]; ok {
			return nil, fmt.Errorf("multiple handles with the same ID: %d - handle ID uniqueness assumption violated - %s", handleID, _githubIssueMsg)
		}
		if card, ok := contactMap[handle]; ok {
			name := card.Name()
			if name != nil && name.GivenName != "" {
				handle = name.GivenName
			}
		}
		handleMap[handleID] = handle
	}
	return handleMap, nil
}

func getDatetimeFormula(macOSVersion *semver.Version) string {
	if macOSVersion != nil && macOSVersion.LessThan(semver.MustParse("10.13")) {
		return _datetimeFormulaLegacy
	}
	return _datetimeFormula
}

func (d chatDB) GetChats() ([]Chat, error) {
	chatRows, err := d.DB.Query("SELECT ROWID, guid, chat_identifier, display_name FROM chat")
	if err != nil {
		return nil, errors.Wrap(err, "query chats table")
	}
	defer chatRows.Close()
	chats := []Chat{}
	for chatRows.Next() {
		var id int
		var guid, name, displayName string
		if err := chatRows.Scan(&id, &guid, &name, &displayName); err != nil {
			return nil, errors.Wrap(err, "read chat")
		}
		if displayName == "" {
			displayName = name
		}
		if card, ok := d.contactMap[displayName]; ok {
			contactName := card.PreferredValue(vcard.FieldFormattedName)
			if contactName != "" {
				displayName = contactName
			}
		}
		chats = append(chats, Chat{
			ID:          id,
			GUID:        guid,
			DisplayName: displayName,
		})
	}
	return chats, nil
}

func (d chatDB) GetMessageIDs(chatID int) ([]int, error) {
	rows, err := d.DB.Query(fmt.Sprintf("SELECT message_id FROM chat_message_join WHERE chat_id=%d", chatID))
	if err != nil {
		return nil, errors.Wrapf(err, "query chat_message_join table for chat ID %d", chatID)
	}
	defer rows.Close()
	messageIDs := []int{}
	for rows.Next() {
		var messageID int
		if err := rows.Scan(&messageID); err != nil {
			return nil, errors.Wrapf(err, "read message ID for chat ID %d", chatID)
		}
		messageIDs = append(messageIDs, messageID)
	}
	return messageIDs, nil
}

func (d chatDB) GetMessage(messageID int) (string, error) {
	messages, err := d.DB.Query(fmt.Sprintf("SELECT is_from_me, handle_id, COALESCE(text, ''), DATETIME(%s) FROM message WHERE ROWID=%d", d.datetimeFormula, messageID))
	if err != nil {
		return "", errors.Wrapf(err, "query message table for ID %d", messageID)
	}
	defer messages.Close()
	messages.Next()
	var fromMe, handleID int
	var text, date string
	if err := messages.Scan(&fromMe, &handleID, &text, &date); err != nil {
		return "", errors.Wrapf(err, "read data for message ID %d", messageID)
	}
	if messages.Next() {
		return "", fmt.Errorf("multiple messages with the same ID: %d - message ID uniqeness assumption violated - %s", messageID, _githubIssueMsg)
	}
	handle := d.handleMap[handleID]
	if fromMe == 1 {
		handle = _selfHandle
	}
	return fmt.Sprintf("[%s] %s: %s\n", date, handle, text), nil
}
