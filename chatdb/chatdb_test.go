// Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package chatdb

import (
	"errors"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/tagatac/bagoup/exectest"
	"github.com/tagatac/bagoup/pathtools"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emersion/go-vcard"
	"gotest.tools/v3/assert"
)

func TestInit(t *testing.T) {
	tests := []struct {
		msg              string
		macOSVersion     *semver.Version
		setupQuery       func(*sqlmock.ExpectedQuery)
		wantDivisor      int
		wantJoinHasDates bool
		wantErr          string
	}{
		{
			msg:          "modern version",
			macOSVersion: semver.MustParse("12.5"),
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"cid", "name", "type", "notnull", "dflt_value", "pk"}).
					AddRow(1, "chat_id", "INTEGER", 0, nil, 0).
					AddRow(2, "message_id", "INTEGER", 0, nil, 0).
					AddRow(3, "message_date", "INTEGER", 0, 0, 0)
				query.WillReturnRows(rows)
			},
			wantDivisor:      _modernVersionDateDivisor,
			wantJoinHasDates: true,
		},
		{
			msg:          "older version",
			macOSVersion: semver.MustParse("10.11"),
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"cid", "name", "type", "notnull", "dflt_value", "pk"}).
					AddRow(1, "chat_id", "INTEGER", 0, nil, 0).
					AddRow(2, "message_id", "INTEGER", 0, nil, 0)
				query.WillReturnRows(rows)
			},
			wantDivisor:      1,
			wantJoinHasDates: false,
		},
		{
			msg:          "PRAGMA querry error",
			macOSVersion: semver.MustParse("12.5"),
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				query.WillReturnError(errors.New("this is a database error"))
			},
			wantErr: "get chat_message_join table info: this is a database error",
		},
		{
			msg:          "DB scan error",
			macOSVersion: semver.MustParse("12.5"),
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"cid", "name", "type", "notnull", "dflt_value", "pk"}).
					AddRow("one", "chat_id", "INTEGER", 0, nil, 0)
				query.WillReturnRows(rows)
			},
			wantErr: `read chat_message_join column info: sql: Scan error on column index 0, name "cid": converting driver.Value type string ("one") to a int: invalid syntax`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			db, sMock, err := sqlmock.New()
			assert.NilError(t, err)
			defer db.Close()
			query := sMock.ExpectQuery(`PRAGMA table_info\(chat_message_join\)`)
			tt.setupQuery(query)

			cdb := &chatDB{DB: db}
			err = cdb.Init(tt.macOSVersion)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, tt.wantDivisor, cdb.dateDivisor)
			assert.Equal(t, tt.wantJoinHasDates, cdb.cmJoinHasDates)
		})
	}
}

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
		msg       string
		legacyDB  bool
		setupMock func(sqlmock.Sqlmock)
		wantIDs   []DatedMessageID
		wantErr   string
	}{
		{
			msg: "success",
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id", "message_date"}).
					AddRow(192, 593720716622331392).
					AddRow(168, 601412272470654464)
				sMock.ExpectQuery("SELECT message_id, message_date FROM chat_message_join WHERE chat_id=42").WillReturnRows(rows)
			},
			wantIDs: []DatedMessageID{
				{192, 593720716622331392},
				{168, 601412272470654464},
			},
		},
		{
			msg:      "success legacy",
			legacyDB: true,
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id"}).
					AddRow(192).
					AddRow(168)
				sMock.ExpectQuery("SELECT message_id FROM chat_message_join WHERE chat_id=42").WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"date"}).
					AddRow(593720716622331392)
				sMock.ExpectQuery("SELECT date FROM message WHERE ROWID=192").WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"date"}).
					AddRow(601412272470654464)
				sMock.ExpectQuery("SELECT date FROM message WHERE ROWID=168").WillReturnRows(rows)
			},
			wantIDs: []DatedMessageID{
				{192, 593720716622331392},
				{168, 601412272470654464},
			},
		},
		{
			msg: "DB error",
			setupMock: func(sMock sqlmock.Sqlmock) {
				sMock.ExpectQuery("SELECT message_id, message_date FROM chat_message_join WHERE chat_id=42").WillReturnError(errors.New("this is a DB error"))
			},
			wantErr: "query chat_message_join table for chat ID 42: this is a DB error",
		},
		{
			msg: "row scan error",
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id", "message_date"}).
					AddRow(192, 593720716622331392).
					AddRow(nil, 601412272470654464)
				sMock.ExpectQuery("SELECT message_id, message_date FROM chat_message_join WHERE chat_id=42").WillReturnRows(rows)
			},
			wantErr: "read message ID for chat ID 42: sql: Scan error on column index 0, name \"message_id\": converting NULL to int is unsupported",
		},
		{
			msg:      "legacy join query error",
			legacyDB: true,
			setupMock: func(sMock sqlmock.Sqlmock) {
				sMock.ExpectQuery("SELECT message_id FROM chat_message_join WHERE chat_id=42").WillReturnError(errors.New("this is a DB error"))
			},
			wantErr: "query chat_message_join table for chat ID 42: this is a DB error",
		},
		{
			msg:      "legacy join scan error",
			legacyDB: true,
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id"}).
					AddRow("one")
				sMock.ExpectQuery("SELECT message_id FROM chat_message_join WHERE chat_id=42").WillReturnRows(rows)
			},
			wantErr: `read message ID for chat ID 42: sql: Scan error on column index 0, name "message_id": converting driver.Value type string ("one") to a int: invalid syntax`,
		},
		{
			msg:      "legacy message table query error",
			legacyDB: true,
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id"}).
					AddRow(192).
					AddRow(168)
				sMock.ExpectQuery("SELECT message_id FROM chat_message_join WHERE chat_id=42").WillReturnRows(rows)
				sMock.ExpectQuery("SELECT date FROM message WHERE ROWID=192").WillReturnError(errors.New("this is a DB error"))
			},
			wantErr: "query message table for ID 192: this is a DB error",
		},
		{
			msg:      "legacy message table scan error",
			legacyDB: true,
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id"}).
					AddRow(192).
					AddRow(168)
				sMock.ExpectQuery("SELECT message_id FROM chat_message_join WHERE chat_id=42").WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"date"}).RowError(1, nil)
				sMock.ExpectQuery("SELECT date FROM message WHERE ROWID=192").WillReturnRows(rows)
			},
			wantErr: "read date for message ID 192: sql: Rows are closed",
		},
		{
			msg:      "legacy message table duplicate message ID",
			legacyDB: true,
			setupMock: func(sMock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"message_id"}).
					AddRow(192).
					AddRow(168)
				sMock.ExpectQuery("SELECT message_id FROM chat_message_join WHERE chat_id=42").WillReturnRows(rows)
				rows = sqlmock.NewRows([]string{"date"}).
					AddRow(593720716622331392).
					AddRow(601412272470654464)
				sMock.ExpectQuery("SELECT date FROM message WHERE ROWID=192").WillReturnRows(rows)
			},
			wantErr: "multiple messages with the same ID: 192 - message ID uniqueness assumption violated - open an issue at https://github.com/tagatac/bagoup/issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			db, sMock, err := sqlmock.New()
			assert.NilError(t, err)
			defer db.Close()
			tt.setupMock(sMock)
			cdb := &chatDB{
				DB:             db,
				cmJoinHasDates: !tt.legacyDB,
			}

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
		ptsOutput   string
		ptsErr      string
		wantMessage string
		wantValid   bool
		wantErr     string
	}{
		{
			msg: "message to me",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, "message text", "", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			wantMessage: "[2019-10-04 18:26:31] testhandle1: message text\n",
			wantValid:   true,
		},
		{
			msg: "message from me",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(1, 10, "message text", "", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			wantMessage: "[2019-10-04 18:26:31] Me: message text\n",
			wantValid:   true,
		},
		{
			msg: "message encoded in attributedBody",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, "", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			ptsOutput: `type b'@': object of class NSMutableAttributedString v0, extends NSAttributedString v0, extends NSObject v0:
	super object: <NSObject>
	type b'@': NSMutableString('message text')
	group:
		type b'i': 1
		type b'I': 28`,
			wantMessage: "[2019-10-04 18:26:31] testhandle1: message text\n",
			wantValid:   true,
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
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, nil, "message text", "", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			wantErr: `read data for message ID 42: sql: Scan error on column index 1, name "handle_id": converting NULL to int is unsupported`,
		},
		{
			msg: "duplicate message ID",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, "message text", "", "2019-10-04 18:26:31").
					AddRow(1, 10, "response message text", "", "2019-10-04 18:26:54")
				query.WillReturnRows(rows)
			},
			wantErr: "multiple messages with the same ID: 42 - message ID uniqueness assumption violated - open an issue at https://github.com/tagatac/bagoup/issues",
		},
		{
			msg: "error decoding attributedBody",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, "", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			ptsErr:      "this is a pytypedstream error",
			wantMessage: "[2019-10-04 18:26:31] testhandle1: \n",
		},
		{
			msg: "decoded attributedBody doesn't match regexp",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, "", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			ptsOutput:   "this is a bad decoding",
			wantMessage: "[2019-10-04 18:26:31] testhandle1: \n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			db, sMock, err := sqlmock.New()
			assert.NilError(t, err)
			defer db.Close()
			query := sMock.ExpectQuery(`SELECT is_from_me, handle_id, text, attributedBody, DATETIME\(\(date\/1000000000\) \+ STRFTIME\('%s', '2001\-01\-01 00\:00\:00'\), 'unixepoch', 'localtime'\) FROM message WHERE ROWID\=42`)
			tt.setupQuery(query)
			cdb := &chatDB{
				DB:             db,
				selfHandle:     "Me",
				dateDivisor:    _modernVersionDateDivisor,
				cmJoinHasDates: true,
				execCommand:    exectest.GenFakeExecCommand(tt.ptsOutput, tt.ptsErr),
			}

			message, ok, err := cdb.GetMessage(42, handleMap)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, tt.wantValid, ok)
			assert.Equal(t, tt.wantMessage, message)
		})
	}
}

func TestRunExecCmd(t *testing.T) { exectest.RunExecCmd() }

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
