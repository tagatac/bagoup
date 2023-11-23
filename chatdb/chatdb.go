// Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

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
	"os/exec"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	"github.com/pkg/errors"
	"github.com/tagatac/bagoup/pathtools"
)

const _githubIssueMsg = "open an issue at https://github.com/tagatac/bagoup/issues"

// The modern version of Mac OS as it pertains to date representation in chat.db
var _modernVersion = semver.MustParse("10.13")

const _modernVersionDateDivisor = 1_000_000_000

//go:generate mockgen -destination=mock_chatdb/mock_chatdb.go github.com/tagatac/bagoup/chatdb ChatDB

type (
	// ChatDB extracts data from a Mac OS Messages database on disk.
	ChatDB interface {
		// Init determines the version of the database, preparing it to make the
		// appropriate queries.
		Init(macOSVersion *semver.Version) error
		// GetHandleMap returns a mapping from handle ID to phone number or email
		// address. If a contact map is supplied, it will attempt to resolve these
		// handles to formatted names.
		GetHandleMap(contactMap map[string]*vcard.Card) (map[int]string, error)
		// GetChats returns a slice of EntityChats, effectively a table scan of
		// the chat table.
		GetChats(contactMap map[string]*vcard.Card) ([]EntityChats, error)
		// GetMessageIDs returns a slice of DatedMessageIDs corresponding to a
		// given chat ID.
		GetMessageIDs(chatID int) ([]DatedMessageID, error)
		// GetMessage returns a message retrieved from the database formatted for
		// writing to a chat file, as well as flag indicating the validity of the
		// text in the message.
		GetMessage(messageID int, handleMap map[int]string) (string, bool, error)
		// GetAttachmentPaths returns a list of attachment filepaths associated with
		// each message ID.
		GetAttachmentPaths(ptools pathtools.PathTools) (map[int][]Attachment, error)
	}

	chatDB struct {
		*sql.DB
		selfHandle     string
		dateDivisor    int
		cmJoinHasDates bool
		execCommand    func(string, ...string) *exec.Cmd
	}
)

// NewChatDB returns a ChatDB interface using the given DB. Init must be called
// on it before use.
func NewChatDB(db *sql.DB, selfHandle string) ChatDB {
	return &chatDB{
		DB:          db,
		selfHandle:  selfHandle,
		execCommand: exec.Command,
	}
}

func (d *chatDB) Init(macOSVersion *semver.Version) error {
	// Set the datetime divisor. Adapted from
	// https://apple.stackexchange.com/a/300997/267331
	d.dateDivisor = _modernVersionDateDivisor
	if macOSVersion != nil && macOSVersion.LessThan(_modernVersion) {
		d.dateDivisor = 1
	}

	// Check if the chat_message_join table has a message_date column. See
	// https://github.com/tagatac/bagoup/issues/24.
	columns, err := d.DB.Query("PRAGMA table_info(chat_message_join)")
	if err != nil {
		return errors.Wrap(err, "get chat_message_join table info")
	}
	defer columns.Close()
	for columns.Next() {
		var cid, notnull, pk int
		var name, typ, dflt_value sql.NullString
		if err := columns.Scan(&cid, &name, &typ, &notnull, &dflt_value, &pk); err != nil {
			return errors.Wrap(err, "read chat_message_join column info")
		}
		if name.String == "message_date" {
			d.cmJoinHasDates = true
			break
		}
	}

	return nil
}

func (d chatDB) GetHandleMap(contactMap map[string]*vcard.Card) (map[int]string, error) {
	handleMap := make(map[int]string)
	handles, err := d.DB.Query("SELECT ROWID, id FROM handle")
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
