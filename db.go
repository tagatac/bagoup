// Copyright (C) 2020 David Tagatac <david@tagatac.net>
// See the COPYING and LICENSE files for full usage terms.

package main

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

const _githubIssueMsg = "open an issue at https://github.com/tagatac/bagoup/issues"

// Adapted from https://apple.stackexchange.com/a/300997/267331
const _datetimeFormula = "(date/1000000000) + STRFTIME('%s', '2001-01-01 00:00:00'), 'unixepoch', 'localtime'"

// How to label messages sent by yourself.
const _selfHandle = "Me"

// Chat represents a row from the chat table.
type Chat struct {
	ID          int
	GUID        string
	Name        string
	DisplayName string
}

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
		handleMap map[int]string
	}
)

// NewChatDB returns a ChatDB interface with a populated handle map.
func NewChatDB(db *sql.DB) (ChatDB, error) {
	handleMap, err := getHandles(db)
	return &chatDB{DB: db, handleMap: handleMap}, errors.Wrap(err, "get handles")
}

func getHandles(db *sql.DB) (map[int]string, error) {
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
		handleMap[handleID] = handle
	}
	return handleMap, nil
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
		chats = append(chats, Chat{
			ID:          id,
			GUID:        guid,
			Name:        name,
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
	messages, err := d.DB.Query(fmt.Sprintf("SELECT is_from_me, handle_id, text, DATETIME(%s) FROM message WHERE ROWID=%d", _datetimeFormula, messageID))
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
