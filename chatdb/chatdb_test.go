// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package chatdb

import (
	"errors"
	"testing"

	"github.com/Masterminds/semver"

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
