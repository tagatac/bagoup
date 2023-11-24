// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package chatdb

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emersion/go-vcard"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
)

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
