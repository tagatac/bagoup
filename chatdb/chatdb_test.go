// Copyright (C) 2020 David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package chatdb

import (
	"errors"
	"testing"

	"github.com/Masterminds/semver"

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
				assert.ErrorContains(t, err, tt.wantErr)
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
		wantChats  []Chat
		wantErr    string
	}{
		{
			msg: "empty contact map",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"ROWID", "guid", "chat_identifier", "display_name"}).
					AddRow(1, "testguid1", "testchatname1", "testdisplayname1").
					AddRow(2, "testguid2", "testchatname2", "")
				query.WillReturnRows(rows)
			},
			wantChats: []Chat{
				{
					ID:          1,
					GUID:        "testguid1",
					DisplayName: "testdisplayname1",
				},
				{
					ID:          2,
					GUID:        "testguid2",
					DisplayName: "testchatname2",
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
					AddRow(2, "testguid2", "testchatname2", "")
				query.WillReturnRows(rows)
			},
			wantChats: []Chat{
				{
					ID:          1,
					GUID:        "testguid1",
					DisplayName: "testdisplayname1",
				},
				{
					ID:          2,
					GUID:        "testguid2",
					DisplayName: "Contactgiven Contactsurname",
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
				assert.ErrorContains(t, err, tt.wantErr)
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
		wantIDs    []int
		wantErr    string
	}{
		{
			msg: "success",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"message_id"}).
					AddRow(192).
					AddRow(168)
				query.WillReturnRows(rows)
			},
			wantIDs: []int{192, 168},
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
				rows := sqlmock.NewRows([]string{"message_id"}).
					AddRow(192).
					AddRow(nil)
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
			query := sMock.ExpectQuery("SELECT message_id FROM chat_message_join WHERE chat_id=42")
			tt.setupQuery(query)
			cdb := &chatDB{DB: db}

			ids, err := cdb.GetMessageIDs(42)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
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
			wantErr: "read data for message ID 42: sql: Scan error on column index 1, name \"handle_id\": converting NULL to int is unsupported",
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
				assert.ErrorContains(t, err, tt.wantErr)
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
