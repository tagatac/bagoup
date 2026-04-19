package chatdb

import (
	_ "embed"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gotest.tools/v3/assert"
)

//go:embed testdata/nsstring.bin
var _attributedBodyNSString []byte

//go:embed testdata/venmo.bin
var _attributedBodyVenmo []byte

//go:embed testdata/google.bin
var _attributedBodyGoogle []byte

func TestDecode(t *testing.T) {
	handleMap := map[int]string{
		10: "testhandle1",
	}

	tests := []struct {
		msg         string
		setupQuery  func(*sqlmock.ExpectedQuery)
		wantMessage string
		wantValid   bool
		wantErr     string
	}{
		{
			msg: "typical iMessage",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, string(_attributedBodyNSString), "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			wantMessage: "[2019-10-04 18:26:31] testhandle1: no, should i?\n",
			wantValid:   true,
		},
		{
			msg: "2FA code",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, string(_attributedBodyVenmo), "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			wantMessage: "[2019-10-04 18:26:31] testhandle1: Venmo here! NEVER share this code via call/text. ONLY YOU should enter the code. BEWARE: If someone asks for the code, it's a scam. Code: 975002\n",
			wantValid:   true,
		},
		{
			msg: "Google 2FA code",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, string(_attributedBodyGoogle), "2023-12-17 21:27:07")
				query.WillReturnRows(rows)
			},
			wantMessage: "[2023-12-17 21:27:07] testhandle1: G-913121 is your Google verification code.\n",
			wantValid:   true,
		},
		{
			msg: "failure creating unarchiver",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, "", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			wantMessage: "[2019-10-04 18:26:31] testhandle1: \n",
		},
		{
			msg: "failure to decode all",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, "\x04\x0bstreamtyped\x62\x84\x85", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			wantMessage: "[2019-10-04 18:26:31] testhandle1: \n",
		},
		{
			msg: "empty stream",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, "\x04\x0bstreamtyped\x62", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			wantMessage: "[2019-10-04 18:26:31] testhandle1: \n",
		},
		{
			msg: "wrong top-level value type",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, "\x04\x0bstreamtyped\x62\x84\x01i\x01", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			wantMessage: "[2019-10-04 18:26:31] testhandle1: \n",
		},
		{
			msg: "no contents in the first group",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, "\x04\x0bstreamtyped\x62\x84\x01@\x84\x84\x84\x01Z\x00\x85\x86", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			wantMessage: "[2019-10-04 18:26:31] testhandle1: \n",
		},
		{
			msg: "no string in the contents",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, "\x04\x0bstreamtyped\x62\x84\x01@\x84\x84\x84\x01Z\x00\x85\x84\x01i\x01\x86", "2019-10-04 18:26:31")
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
			cdb := &chatDB{
				DB:             db,
				selfHandle:     "Me",
				dateDivisor:    _modernVersionDateDivisor,
				cmJoinHasDates: true,
			}

			message, ok, err := cdb.GetMessage(42, handleMap)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, ok, tt.wantValid)
			assert.Equal(t, message, tt.wantMessage)
		})
	}
}
