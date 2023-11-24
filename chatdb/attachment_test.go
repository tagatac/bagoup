// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package chatdb

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/pkg/errors"
	"github.com/tagatac/bagoup/pathtools"
	"gotest.tools/v3/assert"
)

func TestGetAttachmentPaths(t *testing.T) {
	ptools, err := pathtools.NewPathTools()
	assert.NilError(t, err)
	tests := []struct {
		msg          string
		setupMock    func(sqlmock.Sqlmock)
		wantAttPaths map[int][]Attachment
		wantErr      string
	}{
		{
			msg: "two attachments to one message, one to another",
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id", "attachment_id"}).
					AddRow(1, 1).
					AddRow(1, 2).
					AddRow(2, 3)
				sMock.ExpectQuery("SELECT message_id, attachment_id FROM message_attachment_join").WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_name"}).
					AddRow("attachment1.jpeg", "image/jpeg", "attachment1.jpeg")
				sMock.ExpectQuery(`SELECT filename, mime_type, transfer_name FROM attachment WHERE ROWID\=1`).WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_name"}).
					AddRow("~/attachment2.heic", "image/heic", "attachment2.heic")
				sMock.ExpectQuery(`SELECT filename, mime_type, transfer_name FROM attachment WHERE ROWID\=2`).WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_name"}).
					AddRow("/var/folder/attachment3.mp4", "video/mp4", "attachment3.mp4")
				sMock.ExpectQuery(`SELECT filename, mime_type, transfer_name FROM attachment WHERE ROWID\=3`).WillReturnRows(rows)
			},
			wantAttPaths: map[int][]Attachment{
				1: {
					{
						ID:           1,
						Filename:     "attachment1.jpeg",
						MIMEType:     "image/jpeg",
						TransferName: "attachment1.jpeg",
					},
					{
						ID:           2,
						Filename:     ptools.ReplaceTilde("~/attachment2.heic"),
						MIMEType:     "image/heic",
						TransferName: "attachment2.heic",
					},
				},
				2: {
					{
						ID:           3,
						Filename:     "/var/folder/0/attachment3.mp4",
						MIMEType:     "video/mp4",
						TransferName: "attachment3.mp4",
					},
				},
			},
		},
		{
			msg: "undefined MIME type",
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id", "attachment_id"}).
					AddRow(1, 1).
					AddRow(1, 2)
				sMock.ExpectQuery("SELECT message_id, attachment_id FROM message_attachment_join").WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_name"}).
					AddRow("attachment1.jpeg", "image/jpeg", "attachment1.jpeg")
				sMock.ExpectQuery(`SELECT filename, mime_type, transfer_name FROM attachment WHERE ROWID\=1`).WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_name"}).
					AddRow("~/attachment2.heic", nil, "attachment2.heic")
				sMock.ExpectQuery(`SELECT filename, mime_type, transfer_name FROM attachment WHERE ROWID\=2`).WillReturnRows(rows)
			},
			wantAttPaths: map[int][]Attachment{
				1: {
					{
						ID:           1,
						Filename:     "attachment1.jpeg",
						MIMEType:     "image/jpeg",
						TransferName: "attachment1.jpeg",
					},
					{
						ID:           2,
						Filename:     ptools.ReplaceTilde("~/attachment2.heic"),
						MIMEType:     "application/octet-stream",
						TransferName: "attachment2.heic",
					},
				},
			},
		},
		{
			msg: "join table query error",
			setupMock: func(sMock sqlmock.Sqlmock) {
				sMock.ExpectQuery("SELECT message_id, attachment_id FROM message_attachment_join").WillReturnError(errors.New("this is a DB error"))
			},
			wantErr: "scan message_attachment_join table: this is a DB error",
		},
		{
			msg: "join table row scan error",
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id", "attachment_id"}).
					AddRow(1, nil)
				sMock.ExpectQuery("SELECT message_id, attachment_id FROM message_attachment_join").WillReturnRows(rows)
			},
			wantErr: `read data from message_attachment_join table: sql: Scan error on column index 1, name "attachment_id": converting NULL to int is unsupported`,
		},
		{
			msg: "attachments table query error",
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id", "attachment_id"}).
					AddRow(1, 1).
					AddRow(1, 2).
					AddRow(2, 3)
				sMock.ExpectQuery("SELECT message_id, attachment_id FROM message_attachment_join").WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_state"}).
					AddRow("attachment1.jpeg", "image/jpeg", 5)
				sMock.ExpectQuery(`SELECT filename, mime_type, transfer_name FROM attachment WHERE ROWID\=1`).WillReturnRows(rows)
				sMock.ExpectQuery(`SELECT filename, mime_type, transfer_name FROM attachment WHERE ROWID\=2`).WillReturnError(errors.New("this is a DB error"))
			},
			wantErr: "get path for attachment 2 to message 1: query attachment table for ID 2: this is a DB error",
		},
		{
			msg: "attachments table row scan error",
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id", "attachment_id"}).
					AddRow(1, 1).
					AddRow(1, 2)
				sMock.ExpectQuery("SELECT message_id, attachment_id FROM message_attachment_join").WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_name"}).
					AddRow("attachment1.jpeg", "image/jpeg", "attachment1.jpeg")
				sMock.ExpectQuery(`SELECT filename, mime_type, transfer_name FROM attachment WHERE ROWID\=1`).WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_name"}).
					RowError(1, errors.New("this is a row error"))
				sMock.ExpectQuery(`SELECT filename, mime_type, transfer_name FROM attachment WHERE ROWID\=2`).WillReturnRows(rows)
			},
			wantErr: `get path for attachment 2 to message 1: read data for attachment ID 2: sql: Rows are closed`,
		},
		{
			msg: "duplicate attachment ID",
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id", "attachment_id"}).
					AddRow(1, 1).
					AddRow(1, 2).
					AddRow(2, 3)
				sMock.ExpectQuery("SELECT message_id, attachment_id FROM message_attachment_join").WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_state"}).
					AddRow("attachment1.jpeg", "image/jpeg", 5)
				sMock.ExpectQuery(`SELECT filename, mime_type, transfer_name FROM attachment WHERE ROWID\=1`).WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_state"}).
					AddRow("~/attachment2.heic", "image/heic", 5).
					AddRow("attachment2.mov", "video/mov", 5)
				sMock.ExpectQuery(`SELECT filename, mime_type, transfer_name FROM attachment WHERE ROWID\=2`).WillReturnRows(rows)
			},
			wantErr: "get path for attachment 2 to message 1: multiple attachments with the same ID: 2 - attachment ID uniqueness assumption violated - open an issue at https://github.com/tagatac/bagoup/issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			db, sMock, err := sqlmock.New()
			assert.NilError(t, err)
			defer db.Close()
			tt.setupMock(sMock)
			cdb := &chatDB{DB: db}

			attPaths, err := cdb.GetAttachmentPaths(ptools)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, tt.wantAttPaths, attPaths)
		})
	}
}
