// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package chatdb

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/tagatac/bagoup/v2/pathtools"
)

// Attachment represents a row from the attachment table.
type Attachment struct {
	ID           int
	Filename     string
	Filepath     string
	MIMEType     string
	TransferName string
}

func (d *chatDB) GetAttachmentPaths(ptools pathtools.PathTools) (map[int][]Attachment, error) {
	attachmentJoins, err := d.DB.Query("SELECT message_id, attachment_id FROM message_attachment_join")
	if err != nil {
		return nil, fmt.Errorf("scan message_attachment_join table: %w", err)
	}
	defer attachmentJoins.Close()

	atts := map[int][]Attachment{}
	for attachmentJoins.Next() {
		var msgID, attID int
		if err := attachmentJoins.Scan(&msgID, &attID); err != nil {
			return atts, fmt.Errorf("read data from message_attachment_join table: %w", err)
		}
		att, err := d.getAttachmentPath(attID, ptools)
		if err != nil {
			return atts, fmt.Errorf("get path for attachment %d to message %d: %w", attID, msgID, err)
		}
		atts[msgID] = append(atts[msgID], att)
	}
	return atts, nil
}

func (d *chatDB) getAttachmentPath(attachmentID int, ptools pathtools.PathTools) (Attachment, error) {
	attachments, err := d.DB.Query(fmt.Sprintf("SELECT filename, mime_type, transfer_name FROM attachment WHERE ROWID=%d", attachmentID))
	if err != nil {
		return Attachment{}, fmt.Errorf("query attachment table for ID %d: %w", attachmentID, err)
	}
	defer attachments.Close()
	attachments.Next()
	var filenameOrNull, mimeTypeOrNull, transferNameOrNull sql.NullString
	if err := attachments.Scan(&filenameOrNull, &mimeTypeOrNull, &transferNameOrNull); err != nil {
		return Attachment{}, fmt.Errorf("read data for attachment ID %d: %w", attachmentID, err)
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
