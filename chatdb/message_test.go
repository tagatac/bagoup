// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package chatdb

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/pkg/errors"
	"github.com/tagatac/bagoup/v2/exectest"
	"gotest.tools/v3/assert"
)

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
					AddRow(168, 601412272)
				sMock.ExpectQuery("SELECT message_id, message_date FROM chat_message_join WHERE chat_id=42").WillReturnRows(rows)
			},
			wantIDs: []DatedMessageID{
				{192, 593720716622331392},
				{168, 601412272000000000},
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
			msg: "2FA code encoded in attributedBody",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, "", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			ptsOutput: `Venmo here! NEVER share this code via call/text. ONLY YOU should enter the code. BEWARE: If someone asks for the code, it's a scam. Code: {
    "__kIMMessagePartAttributeName" = 0;
}555555{
    "__kIMDataDetectedAttributeName" = {length = 553, bytes = 0x62706c69 73743030 d4010203 04050607 ... 00000000 00000193 };
    "__kIMMessagePartAttributeName" = 0;
    "__kIMOneTimeCodeAttributeName" =     {
        code = 555555;
        displayCode = 555555;
    };
}
`,
			wantMessage: "[2019-10-04 18:26:31] testhandle1: Venmo here! NEVER share this code via call/text. ONLY YOU should enter the code. BEWARE: If someone asks for the code, it's a scam. Code: 555555\n",
			wantValid:   true,
		},
		{
			msg: "Google 2FA code encoded in attributedBody",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, "", "2023-12-17 21:27:07")
				query.WillReturnRows(rows)
			},
			ptsOutput: `G-123456{
    "__kIMDataDetectedAttributeName" = {length = 537, bytes = 0x62706c69 73743030 d4010203 04050607 ... 00000000 00000185 };
    "__kIMMessagePartAttributeName" = 0;
    "__kIMOneTimeCodeAttributeName" =     {
        code = 123456;
        displayCode = "G-123456";
    };
} is your Google verification code.{
    "__kIMMessagePartAttributeName" = 0;
}
`,
			wantMessage: "[2023-12-17 21:27:07] testhandle1: G-123456 is your Google verification code.\n",
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
			ptsErr:      "this is a typedstream-decode error",
			wantMessage: "[2019-10-04 18:26:31] testhandle1: \n",
		},
		{
			msg: "no valid text or attributedBody",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, nil, "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
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
			exitCode := 0
			if tt.ptsErr != "" {
				exitCode = 1
			}
			cdb := &chatDB{
				DB:             db,
				selfHandle:     "Me",
				dateDivisor:    _modernVersionDateDivisor,
				cmJoinHasDates: true,
				execCommand:    exectest.GenFakeExecCommand("TestRunExecCmd", tt.ptsOutput, tt.ptsErr, exitCode),
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
