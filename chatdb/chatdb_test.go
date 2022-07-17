// Copyright (C) 2020-2022 David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package chatdb

import (
	"errors"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/tagatac/bagoup/pathtools"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emersion/go-vcard"
	"gotest.tools/v3/assert"
)

func TestGetHandleMap(t *testing.T) {
	tests := []struct {
		msg        string
		contactMap map[string]*vcard.Card
		setupQuery func(*sqlmock.ExpectedQuery)
		wantMap    map[int]string
		wantErr    string
	}{
		{
			msg: "empty contact map",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"ROWID", "id"}).
					AddRow(1, "testhandle1").
					AddRow(2, "testhandle2")
				query.WillReturnRows(rows)
			},
			wantMap: map[int]string{
				1: "testhandle1",
				2: "testhandle2",
			},
		},
		{
			msg: "contact map override",
			contactMap: map[string]*vcard.Card{
				"testhandle1": {
					"N": []*vcard.Field{
						{Value: "contactsurname;contactgiven;;;"},
					},
				},
			},
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"ROWID", "id"}).
					AddRow(1, "testhandle1").
					AddRow(2, "testhandle2")
				query.WillReturnRows(rows)
			},
			wantMap: map[int]string{
				1: "contactgiven",
				2: "testhandle2",
			},
		},
		{
			msg: "DB error",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				query.WillReturnError(errors.New("this is a DB error"))
			},
			wantErr: "get handles from DB: this is a DB error",
		},
		{
			msg: "row scan error",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"ROWID", "id"}).
					AddRow(1, nil).
					AddRow(2, "testhandle2")
				query.WillReturnRows(rows)
			},
			wantErr: "read handle: sql: Scan error on column index 1, name \"id\": converting NULL to string is unsupported",
		},
		{
			msg: "repeated row ID",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"ROWID", "id"}).
					AddRow(1, "testhandle1").
					AddRow(1, "testhandle2")
				query.WillReturnRows(rows)
			},
			wantErr: "multiple handles with the same ID: 1 - handle ID uniqueness assumption violated - open an issue at https://github.com/tagatac/bagoup/issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			db, sMock, err := sqlmock.New()
			assert.NilError(t, err)
			defer db.Close()
			query := sMock.ExpectQuery("SELECT ROWID, id FROM handle")
			tt.setupQuery(query)

			cdb := NewChatDB(db, "Me")
			handleMap, err := cdb.GetHandleMap(tt.contactMap)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, tt.wantMap, handleMap)
		})
	}
}

func TestGetChats(t *testing.T) {
	tests := []struct {
		msg        string
		contactMap map[string]*vcard.Card
		setupQuery func(*sqlmock.ExpectedQuery)
		wantChats  []EntityChats
		wantErr    string
	}{
		{
			msg: "empty contact map",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"ROWID", "guid", "chat_identifier", "display_name"}).
					AddRow(1, "testguid1", "testchatname1", "testdisplayname1").
					AddRow(2, "testguid2", "testchatname2", "").
					AddRow(3, "testguid3", "testchatname2", "")
				query.WillReturnRows(rows)
			},
			wantChats: []EntityChats{
				{
					Name: "testchatname2",
					Chats: []Chat{
						{
							ID:   2,
							GUID: "testguid2",
						},
						{
							ID:   3,
							GUID: "testguid3",
						},
					},
				},
				{
					Name: "testdisplayname1",
					Chats: []Chat{
						{
							ID:   1,
							GUID: "testguid1",
						},
					},
				},
			},
		},
		{
			msg: "contact map override",
			contactMap: map[string]*vcard.Card{
				"testchatname2": {
					"FN": []*vcard.Field{
						{Value: "Contactgiven Contactsurname", Params: vcard.Params{"TYPE": []string{"pref"}}},
					},
				},
			},
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"ROWID", "guid", "chat_identifier", "display_name"}).
					AddRow(1, "testguid1", "testchatname1", "testdisplayname1").
					AddRow(2, "testguid2", "testchatname2", "testdisplayname2").
					AddRow(3, "testguid3", "testchatname2", "testdisplayname2")
				query.WillReturnRows(rows)
			},
			wantChats: []EntityChats{
				{
					Name: "Contactgiven Contactsurname",
					Chats: []Chat{
						{
							ID:   2,
							GUID: "testguid2",
						},
						{
							ID:   3,
							GUID: "testguid3",
						},
					},
				},
				{
					Name: "testdisplayname1",
					Chats: []Chat{
						{
							ID:   1,
							GUID: "testguid1",
						},
					},
				},
			},
		},
		{
			msg: "DB error",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				query.WillReturnError(errors.New("this is a DB error"))
			},
			wantErr: "query chats table: this is a DB error",
		},
		{
			msg: "row scan error",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"ROWID", "guid", "chat_identifier", "display_name"}).
					AddRow(1, "testguid1", "testchatname1", "testdisplayname1").
					AddRow(2, "testguid2", "testchatname2", nil)
				query.WillReturnRows(rows)
			},
			wantErr: "read chat: sql: Scan error on column index 3, name \"display_name\": converting NULL to string is unsupported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			db, sMock, err := sqlmock.New()
			assert.NilError(t, err)
			defer db.Close()
			query := sMock.ExpectQuery(`SELECT ROWID, guid, chat_identifier, COALESCE\(display_name, ''\) FROM chat`)
			tt.setupQuery(query)
			cdb := NewChatDB(db, "Me")

			chats, err := cdb.GetChats(tt.contactMap)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, tt.wantChats, chats)
		})
	}
}

func TestGetMessageIDs(t *testing.T) {
	tests := []struct {
		msg        string
		setupQuery func(*sqlmock.ExpectedQuery)
		wantIDs    []DatedMessageID
		wantErr    string
	}{
		{
			msg: "success",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"message_id", "message_date"}).
					AddRow(192, 593720716622331392).
					AddRow(168, 601412272470654464)
				query.WillReturnRows(rows)
			},
			wantIDs: []DatedMessageID{
				{192, 593720716},
				{168, 601412272},
			},
		},
		{
			msg: "DB error",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				query.WillReturnError(errors.New("this is a DB error"))
			},
			wantErr: "query chat_message_join table for chat ID 42: this is a DB error",
		},
		{
			msg: "row scan error",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"message_id", "message_date"}).
					AddRow(192, 593720716622331392).
					AddRow(nil, 601412272470654464)
				query.WillReturnRows(rows)
			},
			wantErr: "read message ID for chat ID 42: sql: Scan error on column index 0, name \"message_id\": converting NULL to int is unsupported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			db, sMock, err := sqlmock.New()
			assert.NilError(t, err)
			defer db.Close()
			query := sMock.ExpectQuery("SELECT message_id, message_date FROM chat_message_join WHERE chat_id=42")
			tt.setupQuery(query)
			cdb := &chatDB{DB: db}

			ids, err := cdb.GetMessageIDs(42)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, tt.wantIDs, ids)
		})
	}
}

func TestGetMessage(t *testing.T) {
	handleMap := map[int]string{
		10: "testhandle1",
	}

	tests := []struct {
		msg         string
		setupQuery  func(*sqlmock.ExpectedQuery)
		wantMessage string
		wantErr     string
	}{
		{
			msg: "message to me",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "date"}).
					AddRow(0, 10, "message text", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			wantMessage: "[2019-10-04 18:26:31] testhandle1: message text\n",
		},
		{
			msg: "message from me",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "date"}).
					AddRow(1, 10, "message text", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			wantMessage: "[2019-10-04 18:26:31] Me: message text\n",
		},
		{
			msg: "DB error",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				query.WillReturnError(errors.New("this is a DB error"))
			},
			wantErr: "query message table for ID 42: this is a DB error",
		},
		{
			msg: "row scan error",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "date"}).
					AddRow(0, nil, "message text", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			wantErr: `read data for message ID 42: sql: Scan error on column index 1, name "handle_id": converting NULL to int is unsupported`,
		},
		{
			msg: "duplicate message ID",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "date"}).
					AddRow(0, 10, "message text", "2019-10-04 18:26:31").
					AddRow(1, 10, "response message text", "2019-10-04 18:26:54")
				query.WillReturnRows(rows)
			},
			wantErr: "multiple messages with the same ID: 42 - message ID uniqeness assumption violated - open an issue at https://github.com/tagatac/bagoup/issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			db, sMock, err := sqlmock.New()
			assert.NilError(t, err)
			defer db.Close()
			query := sMock.ExpectQuery(`SELECT is_from_me, handle_id, COALESCE\(text, ''\), DATETIME\(\(date\/1000000000\) \+ STRFTIME\('%s', '2001\-01\-01 00\:00\:00'\), 'unixepoch', 'localtime'\) FROM message WHERE ROWID\=42`)
			tt.setupQuery(query)
			cdb := &chatDB{DB: db, selfHandle: "Me"}

			message, err := cdb.GetMessage(42, handleMap, nil)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, tt.wantMessage, message)
		})
	}
}

func TestGetDatetimeFormula(t *testing.T) {
	tests := []struct {
		msg         string
		v           *semver.Version
		prevFormula string
		wantFormula string
	}{
		{
			msg:         "catalina",
			v:           semver.MustParse("10.15.3"),
			wantFormula: _datetimeFormula,
		},
		{
			msg:         "missing version",
			wantFormula: _datetimeFormula,
		},
		{
			msg:         "high sierra",
			v:           semver.MustParse("10.13"),
			wantFormula: _datetimeFormula,
		},
		{
			msg:         "sierra",
			v:           semver.MustParse("10.12.6"),
			wantFormula: _datetimeFormulaLegacy,
		},
		{
			msg:         "previously saved formula",
			prevFormula: "previous formula",
			wantFormula: "previous formula",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			cdb := &chatDB{datetimeFormula: tt.prevFormula}
			assert.Equal(t, tt.wantFormula, cdb.getDatetimeFormula(tt.v))
		})
	}
}

func TestGetAttachmentPaths(t *testing.T) {
	tests := []struct {
		msg          string
		setupMock    func(sqlmock.Sqlmock)
		wantAttPaths map[int][]Attachment
		wantMissing  int
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
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_state"}).
					AddRow("attachment1.jpeg", "image/jpeg", 5)
				sMock.ExpectQuery(`SELECT COALESCE\(filename, ''\), COALESCE\(mime_type, 'application/octet-stream'\), transfer_state FROM attachment WHERE ROWID\=1`).WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_state"}).
					AddRow("~/attachment2.heic", "image/heic", 5)
				sMock.ExpectQuery(`SELECT COALESCE\(filename, ''\), COALESCE\(mime_type, 'application/octet-stream'\), transfer_state FROM attachment WHERE ROWID\=2`).WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_state"}).
					AddRow("/var/folder/attachment3.mp4", "video/mp4", 5)
				sMock.ExpectQuery(`SELECT COALESCE\(filename, ''\), COALESCE\(mime_type, 'application/octet-stream'\), transfer_state FROM attachment WHERE ROWID\=3`).WillReturnRows(rows)
			},
			wantAttPaths: map[int][]Attachment{
				1: {
					{
						Filename:      "attachment1.jpeg",
						MIMEType:      "image/jpeg",
						TransferState: 5,
					},
					{
						Filename:      pathtools.MustReplaceTilde("~/attachment2.heic"),
						MIMEType:      "image/heic",
						TransferState: 5,
					},
				},
				2: {
					{
						Filename:      "/var/folder/0/attachment3.mp4",
						MIMEType:      "video/mp4",
						TransferState: 5,
					},
				},
			},
		},
		{
			msg: "invalid transfer state",
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id", "attachment_id"}).
					AddRow(1, 1)
				sMock.ExpectQuery("SELECT message_id, attachment_id FROM message_attachment_join").WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_state"}).
					AddRow("attachment1.jpeg", "image/jpeg", 0)
				sMock.ExpectQuery(`SELECT COALESCE\(filename, ''\), COALESCE\(mime_type, 'application/octet-stream'\), transfer_state FROM attachment WHERE ROWID\=1`).WillReturnRows(rows)
			},
			wantAttPaths: map[int][]Attachment{
				1: {
					{
						Filename:      "attachment1.jpeg",
						MIMEType:      "image/jpeg",
						TransferState: 0,
					},
				},
			},
		},
		{
			msg: "missing attachment",
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id", "attachment_id"}).
					AddRow(1, 1)
				sMock.ExpectQuery("SELECT message_id, attachment_id FROM message_attachment_join").WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_state"}).
					AddRow("", "image/jpeg", 5)
				sMock.ExpectQuery(`SELECT COALESCE\(filename, ''\), COALESCE\(mime_type, 'application/octet-stream'\), transfer_state FROM attachment WHERE ROWID\=1`).WillReturnRows(rows)
			},
			wantAttPaths: map[int][]Attachment{},
			wantMissing:  1,
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
				sMock.ExpectQuery(`SELECT COALESCE\(filename, ''\), COALESCE\(mime_type, 'application/octet-stream'\), transfer_state FROM attachment WHERE ROWID\=1`).WillReturnRows(rows)
				sMock.ExpectQuery(`SELECT COALESCE\(filename, ''\), COALESCE\(mime_type, 'application/octet-stream'\), transfer_state FROM attachment WHERE ROWID\=2`).WillReturnError(errors.New("this is a DB error"))
			},
			wantErr: "get path for attachment 2 to message 1: query attachment table for ID 2: this is a DB error",
		},
		{
			msg: "attachments table row scan error",
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id", "attachment_id"}).
					AddRow(1, 1).
					AddRow(1, 2).
					AddRow(2, 3)
				sMock.ExpectQuery("SELECT message_id, attachment_id FROM message_attachment_join").WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_state"}).
					AddRow("attachment1.jpeg", "image/jpeg", 5)
				sMock.ExpectQuery(`SELECT COALESCE\(filename, ''\), COALESCE\(mime_type, 'application/octet-stream'\), transfer_state FROM attachment WHERE ROWID\=1`).WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_state"}).
					AddRow("~/attachment2.heic", nil, 5)
				sMock.ExpectQuery(`SELECT COALESCE\(filename, ''\), COALESCE\(mime_type, 'application/octet-stream'\), transfer_state FROM attachment WHERE ROWID\=2`).WillReturnRows(rows)
			},
			wantErr: `get path for attachment 2 to message 1: read data for attachment ID 2: sql: Scan error on column index 1, name "mime_type": converting NULL to string is unsupported`,
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
				sMock.ExpectQuery(`SELECT COALESCE\(filename, ''\), COALESCE\(mime_type, 'application/octet-stream'\), transfer_state FROM attachment WHERE ROWID\=1`).WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"filename", "mime_type", "transfer_state"}).
					AddRow("~/attachment2.heic", "image/heic", 5).
					AddRow("attachment2.mov", "video/mov", 5)
				sMock.ExpectQuery(`SELECT COALESCE\(filename, ''\), COALESCE\(mime_type, 'application/octet-stream'\), transfer_state FROM attachment WHERE ROWID\=2`).WillReturnRows(rows)
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

			attPaths, missing, err := cdb.GetAttachmentPaths()
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, tt.wantAttPaths, attPaths)
			assert.Equal(t, tt.wantMissing, missing)
		})
	}
}
