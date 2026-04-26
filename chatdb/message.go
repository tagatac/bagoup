// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package chatdb

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// appleEpochUnixSec is the Unix timestamp of Apple's reference date 2001-01-01 00:00:00 UTC.
const appleEpochUnixSec int64 = 978307200

// DatedMessageID pairs a message ID and its date, in the legacy date format.
type DatedMessageID struct {
	ID   int
	Date int
}

func (d chatDB) GetMessageIDs(chatID int) ([]DatedMessageID, error) {
	if !d.cmJoinHasDates {
		return d.getMessageIDsLegacy(chatID)
	}
	rows, err := d.DB.Query(fmt.Sprintf("SELECT message_id, message_date FROM chat_message_join WHERE chat_id=%d", chatID))
	if err != nil {
		return nil, fmt.Errorf("query chat_message_join table for chat ID %d: %w", chatID, err)
	}
	defer rows.Close()
	msgIDs := []DatedMessageID{}
	for rows.Next() {
		var id int
		var date int
		if err := rows.Scan(&id, &date); err != nil {
			return nil, fmt.Errorf("read message ID for chat ID %d: %w", chatID, err)
		}
		// We can't trust that all of the dates in the chat_message_join table have
		// been converted (see https://github.com/tagatac/bagoup/issues/40).
		if date < 1_000*_modernVersionDateDivisor {
			date *= _modernVersionDateDivisor
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
		return nil, fmt.Errorf("query chat_message_join table for chat ID %d: %w", chatID, err)
	}
	defer rows.Close()
	msgIDs := []DatedMessageID{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read message ID for chat ID %d: %w", chatID, err)
		}
		msgIDs = append(msgIDs, DatedMessageID{ID: id})
	}

	for i, msgID := range msgIDs {
		messages, err := d.DB.Query(fmt.Sprintf("SELECT date FROM message WHERE ROWID=%d", msgID.ID))
		if err != nil {
			return nil, fmt.Errorf("query message table for ID %d: %w", msgID.ID, err)
		}
		defer messages.Close()
		messages.Next()
		var date int
		if err := messages.Scan(&date); err != nil {
			return nil, fmt.Errorf("read date for message ID %d: %w", msgID.ID, err)
		}
		if messages.Next() {
			return nil, fmt.Errorf("multiple messages with the same ID: %d - message ID uniqueness assumption violated - %s", msgID.ID, _githubIssueMsg)
		}
		msgIDs[i].Date = date
	}

	return msgIDs, nil
}

func (d *chatDB) GetMessage(messageID int, handleMap map[int]string) (string, bool, error) {
	messages, err := d.DB.Query(fmt.Sprintf("SELECT is_from_me, handle_id, text, attributedBody, date FROM message WHERE ROWID=%d", messageID))
	if err != nil {
		return "", false, fmt.Errorf("query message table for ID %d: %w", messageID, err)
	}
	defer messages.Close()
	messages.Next()
	var fromMe, handleID int
	var text, attributedBody sql.NullString
	var rawDate int64
	if err := messages.Scan(&fromMe, &handleID, &text, &attributedBody, &rawDate); err != nil {
		return "", false, fmt.Errorf("read data for message ID %d: %w", messageID, err)
	}
	if messages.Next() {
		return "", false, fmt.Errorf("multiple messages with the same ID: %d - message ID uniqueness assumption violated - %s", messageID, _githubIssueMsg)
	}
	unixSec := rawDate/int64(d.dateDivisor) + appleEpochUnixSec
	date := time.Unix(unixSec, 0).In(d.loc).Format(time.DateTime)
	handle := handleMap[handleID]
	if fromMe == 1 {
		handle = d.selfHandle
	}
	var msg string
	valid := true
	if text.Valid {
		msg = text.String
	} else if attributedBody.Valid {
		msg, err = d.decodeTypedStream(attributedBody.String)
		if err != nil {
			valid = false
			slog.Warn("failed to get plain text for message",
				"messageID", messageID,
				"err", fmt.Errorf("decode typedstream: %w", err),
			)
		}
	} else {
		valid = false
		slog.Warn("no valid text or attributedBody for message", "messageID", messageID)
	}
	return fmt.Sprintf("[%s] %s: %s\n", date, handle, msg), valid, nil
}
