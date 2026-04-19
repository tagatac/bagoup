package chatdb

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/tagatac/bagoup/v2/exectest"
	"gotest.tools/v3/assert"
)

func TestDecode(t *testing.T) {
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
	}{
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
			msg: "error decoding attributedBody",
			setupQuery: func(query *sqlmock.ExpectedQuery) {
				rows := sqlmock.NewRows([]string{"is_from_me", "handle_id", "text", "attributedBody", "date"}).
					AddRow(0, 10, nil, "", "2019-10-04 18:26:31")
				query.WillReturnRows(rows)
			},
			ptsErr:      "this is a typedstream-decode error",
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
			assert.NilError(t, err)
			assert.Equal(t, ok, tt.wantValid)
			assert.Equal(t, message, tt.wantMessage)
		})
	}
}

func TestRunExecCmd(t *testing.T) { exectest.RunExecCmd() }
