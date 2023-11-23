package chatdb

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var (
	_TypedStreamAttributeRE          = regexp.MustCompile(`(\{\n)? {4}"__kIM[[:alpha:]]+" = ([^\n]+);\n\}?`)
	_TypedStreamMultilineAttributeRE = regexp.MustCompile(`(\{\n)? {4}"__kIM[[:alpha:]]+" = {5}\{\n( {8}[[:alpha:]]+ = \d+;\n)+ {4}\};\n\}?`)
)

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

func (d *chatDB) GetMessage(messageID int, handleMap map[int]string) (string, bool, error) {
	datetimeFormula := fmt.Sprintf("(date/%d) + STRFTIME('%%s', '2001-01-01 00:00:00'), 'unixepoch', 'localtime'", d.dateDivisor)
	messages, err := d.DB.Query(fmt.Sprintf("SELECT is_from_me, handle_id, text, attributedBody, DATETIME(%s) FROM message WHERE ROWID=%d", datetimeFormula, messageID))
	if err != nil {
		return "", false, errors.Wrapf(err, "query message table for ID %d", messageID)
	}
	defer messages.Close()
	messages.Next()
	var fromMe, handleID int
	var text, attributedBody sql.NullString
	var date string
	if err := messages.Scan(&fromMe, &handleID, &text, &attributedBody, &date); err != nil {
		return "", false, errors.Wrapf(err, "read data for message ID %d", messageID)
	}
	if messages.Next() {
		return "", false, fmt.Errorf("multiple messages with the same ID: %d - message ID uniqueness assumption violated - %s", messageID, _githubIssueMsg)
	}
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
			log.Printf("WARN: get plain text for message %d: %s", messageID, err)
		}
	} else {
		valid = false
		log.Printf("WARN: no valid text or attributedBody for message %d", messageID)
	}
	return fmt.Sprintf("[%s] %s: %s\n", date, handle, msg), valid, nil
}

func (d *chatDB) decodeTypedStream(s string) (string, error) {
	cmd := d.execCommand("typedstream-decode")
	cmd.Stdin = bytes.NewReader([]byte(s))
	decodedBodyBytes, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "decode attributedBody - POSSIBLE FIX: Add typedstream-decode to your system path (installed with bagoup)")
	}
	decodedBody := string(decodedBodyBytes)
	decodedBody = _TypedStreamAttributeRE.ReplaceAllString(decodedBody, "")
	decodedBody = _TypedStreamMultilineAttributeRE.ReplaceAllString(decodedBody, "")
	return strings.TrimSuffix(decodedBody, "\n"), nil
}
