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
	"path/filepath"
	"sort"
	"strings"

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
		// writing to a chat file.
		GetMessage(messageID int, handleMap map[int]string) (string, error)
		// GetImagePaths returns a list of attachment filepaths associated with
		// each message ID.
		GetAttachmentPaths(ptools pathtools.PathTools) (map[int][]Attachment, error)
	}

	// EntityChats represents all of the chats with a given entity (associated
	// with the same vCard, phone number, or email address). In the case of group
	// chats, this struct will only contain a single Chat.
	EntityChats struct {
		Name  string
		Chats []Chat
	}

	// Chat represents a row from the chat table.
	Chat struct {
		ID   int
		GUID string
	}

	// DatedMessageID pairs a message ID and its date, in the legacy date format.
	DatedMessageID struct {
		ID   int
		Date int
	}

	// Attachment represents a row from the attachment table.
	Attachment struct {
		ID           int
		Filename     string
		MIMEType     string
		TransferName string
	}

	chatDB struct {
		*sql.DB
		selfHandle     string
		dateDivisor    int
		cmJoinHasDates bool
	}
)

// NewChatDB returns a ChatDB interface using the given DB. Init must be called
// on it before use.
func NewChatDB(db *sql.DB, selfHandle string) ChatDB {
	return &chatDB{
		DB:         db,
		selfHandle: selfHandle,
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

func (d chatDB) GetChats(contactMap map[string]*vcard.Card) ([]EntityChats, error) {
	chatRows, err := d.DB.Query("SELECT ROWID, guid, chat_identifier, COALESCE(display_name, '') FROM chat")
	if err != nil {
		return nil, errors.Wrap(err, "query chats table")
	}
	defer chatRows.Close()
	contactChats := map[*vcard.Card]EntityChats{}
	addressChats := map[string]EntityChats{}
	for chatRows.Next() {
		var id int
		var guid, name, displayName string
		if err := chatRows.Scan(&id, &guid, &name, &displayName); err != nil {
			return nil, errors.Wrap(err, "read chat")
		}
		if displayName == "" {
			displayName = name
		}
		chat := Chat{
			ID:   id,
			GUID: guid,
		}
		if card, ok := contactMap[name]; ok {
			addContactChat(card, displayName, chat, contactChats)
		} else {
			addAddressChat(name, displayName, chat, addressChats)
		}
	}
	chats := []EntityChats{}
	for _, entityChats := range contactChats {
		chats = append(chats, entityChats)
	}
	for _, entityChats := range addressChats {
		chats = append(chats, entityChats)
	}
	sort.SliceStable(chats, func(i, j int) bool { return chats[i].Name < chats[j].Name })
	return chats, nil
}

func addContactChat(card *vcard.Card, displayName string, chat Chat, contactChats map[*vcard.Card]EntityChats) {
	if entityChats, ok := contactChats[card]; !ok {
		contactName := card.PreferredValue(vcard.FieldFormattedName)
		if contactName != "" {
			displayName = contactName
		}
		contactChats[card] = EntityChats{
			Name:  displayName,
			Chats: []Chat{chat},
		}
	} else {
		entityChats.Chats = append(entityChats.Chats, chat)
		contactChats[card] = entityChats
	}
}

func addAddressChat(address, displayName string, chat Chat, addressChats map[string]EntityChats) {
	if entityChats, ok := addressChats[address]; !ok {
		// We don't have contact info, and this is a new address.
		addressChats[address] = EntityChats{
			Name:  displayName,
			Chats: []Chat{chat},
		}
	} else {
		// We don't have contact info, and we have seen this address before.
		entityChats.Chats = append(entityChats.Chats, chat)
		addressChats[address] = entityChats
	}
}

func (d chatDB) GetMessageIDs(chatID int) ([]DatedMessageID, error) {
	if !d.cmJoinHasDates {
		return d.getMessageIDsLegacy(chatID)
	}
	rows, err := d.DB.Query(fmt.Sprintf("SELECT message_id, message_date FROM chat_message_join WHERE chat_id=%d", chatID))
	if err != nil {
		return nil, errors.Wrapf(err, "query chat_message_join table for chat ID %d", chatID)
	}
	defer rows.Close()
	msgIDs := []DatedMessageID{}
	for rows.Next() {
		var id int
		var date int
		if err := rows.Scan(&id, &date); err != nil {
			return nil, errors.Wrapf(err, "read message ID for chat ID %d", chatID)
		}
		msgIDs = append(msgIDs, DatedMessageID{id, date})
	}
	return msgIDs, nil
}

// Older chat.db files do not have the chat_message_join.message_date column, so
// we need to also query the message table in this case to get dates.
func (d chatDB) getMessageIDsLegacy(chatID int) ([]DatedMessageID, error) {
	rows, err := d.DB.Query(fmt.Sprintf("SELECT message_id FROM chat_message_join WHERE chat_id=%d", chatID))
	if err != nil {
		return nil, errors.Wrapf(err, "query chat_message_join table for chat ID %d", chatID)
	}
	defer rows.Close()
	msgIDs := []DatedMessageID{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, errors.Wrapf(err, "read message ID for chat ID %d", chatID)
		}
		msgIDs = append(msgIDs, DatedMessageID{ID: id})
	}

	for i, msgID := range msgIDs {
		messages, err := d.DB.Query(fmt.Sprintf("SELECT date FROM message WHERE ROWID=%d", msgID.ID))
		if err != nil {
			return nil, errors.Wrapf(err, "query message table for ID %d", msgID.ID)
		}
		defer messages.Close()
		messages.Next()
		var date int
		if err := messages.Scan(&date); err != nil {
			return nil, errors.Wrapf(err, "read date for message ID %d", msgID.ID)
		}
		if messages.Next() {
			return nil, fmt.Errorf("multiple messages with the same ID: %d - message ID uniqueness assumption violated - %s", msgID.ID, _githubIssueMsg)
		}
		msgIDs[i].Date = date
	}

	return msgIDs, nil
}

func (d *chatDB) GetMessage(messageID int, handleMap map[int]string) (string, error) {
	datetimeFormula := fmt.Sprintf("(date/%d) + STRFTIME('%%s', '2001-01-01 00:00:00'), 'unixepoch', 'localtime'", d.dateDivisor)
	messages, err := d.DB.Query(fmt.Sprintf("SELECT is_from_me, handle_id, text, DATETIME(%s) FROM message WHERE ROWID=%d", datetimeFormula, messageID))
	if err != nil {
		return "", errors.Wrapf(err, "query message table for ID %d", messageID)
	}
	defer messages.Close()
	messages.Next()
	var fromMe, handleID int
	var text sql.NullString
	var date string
	if err := messages.Scan(&fromMe, &handleID, &text, &date); err != nil {
		return "", errors.Wrapf(err, "read data for message ID %d", messageID)
	}
	if messages.Next() {
		return "", fmt.Errorf("multiple messages with the same ID: %d - message ID uniqueness assumption violated - %s", messageID, _githubIssueMsg)
	}
	handle := handleMap[handleID]
	if fromMe == 1 {
		handle = d.selfHandle
	}
	return fmt.Sprintf("[%s] %s: %s\n", date, handle, text.String), nil
}

func (d *chatDB) GetAttachmentPaths(ptools pathtools.PathTools) (map[int][]Attachment, error) {
	attachmentJoins, err := d.DB.Query("SELECT message_id, attachment_id FROM message_attachment_join")
	if err != nil {
		return nil, errors.Wrapf(err, "scan message_attachment_join table")
	}
	defer attachmentJoins.Close()

	atts := map[int][]Attachment{}
	for attachmentJoins.Next() {
		var msgID, attID int
		if err := attachmentJoins.Scan(&msgID, &attID); err != nil {
			return atts, errors.Wrap(err, "read data from message_attachment_join table")
		}
		att, err := d.getAttachmentPath(attID, ptools)
		if err != nil {
			return atts, errors.Wrapf(err, "get path for attachment %d to message %d", attID, msgID)
		}
		atts[msgID] = append(atts[msgID], att)
	}
	return atts, nil
}

func (d *chatDB) getAttachmentPath(attachmentID int, ptools pathtools.PathTools) (Attachment, error) {
	attachments, err := d.DB.Query(fmt.Sprintf("SELECT filename, mime_type, transfer_name FROM attachment WHERE ROWID=%d", attachmentID))
	if err != nil {
		return Attachment{}, errors.Wrapf(err, "query attachment table for ID %d", attachmentID)
	}
	defer attachments.Close()
	attachments.Next()
	var filenameOrNull, mimeTypeOrNull, transferNameOrNull sql.NullString
	if err := attachments.Scan(&filenameOrNull, &mimeTypeOrNull, &transferNameOrNull); err != nil {
		return Attachment{}, errors.Wrapf(err, "read data for attachment ID %d", attachmentID)
	}
	if attachments.Next() {
		return Attachment{}, fmt.Errorf("multiple attachments with the same ID: %d - attachment ID uniqueness assumption violated - %s", attachmentID, _githubIssueMsg)
	}
	filename := filenameOrNull.String
	filename = ptools.ReplaceTilde(filename)
	if strings.HasPrefix(filename, "/var") {
		filename = filepath.Join(filepath.Dir(filename), "0", filepath.Base(filename))
	}
	mimeType := "application/octet-stream"
	if mimeTypeOrNull.Valid {
		mimeType = mimeTypeOrNull.String
	}
	transferName := "(unknown attachment)"
	if transferNameOrNull.Valid {
		transferName = transferNameOrNull.String
	}
	return Attachment{ID: attachmentID, Filename: filename, MIMEType: mimeType, TransferName: transferName}, nil
}
