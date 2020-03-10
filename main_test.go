// Copyright (C) 2020 David Tagatac <david@tagatac.net>
// See the COPYING and LICENSE files for full usage terms.

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/chatdb"
	"github.com/tagatac/bagoup/mocks/mock_chatdb"
	"gotest.tools/v3/assert"
)

// Adapted from https://stackoverflow.com/a/33404435/5403337.
func TestExitOnError(t *testing.T) {
	if os.Getenv("BAGOUP_TEST_EXIT") == "1" {
		var err error
		errStr := os.Getenv("BAGOUP_EXIT_ERROR")
		if errStr != "" {
			err = errors.New(errStr)
		}
		exitOnError("here's a context string", err)
		return
	}

	tests := []struct {
		msg          string
		wantErr      string
		wantExitCode int
	}{
		{
			msg: "no error",
		},
		{
			msg:          "error",
			wantErr:      "this is an error",
			wantExitCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestExitOnError")
			cmd.Env = []string{
				"BAGOUP_TEST_EXIT=1",
				fmt.Sprintf("BAGOUP_EXIT_ERROR=%s", tt.wantErr),
			}
			err := cmd.Run()
			if tt.wantExitCode == 0 {
				assert.NilError(t, err)
				return
			}
			e, ok := err.(*exec.ExitError)
			assert.Assert(t, ok)
			assert.Equal(t, tt.wantExitCode, e.ExitCode())
		})
	}
}

func TestGetMacOSVersion(t *testing.T) {
	tests := []struct {
		msg          string
		swVersOutput string
		swVersErr    string
		wantVersion  *semver.Version
		wantErr      string
	}{
		{
			msg:          "catalina",
			swVersOutput: "10.15.3\n",
			wantVersion:  semver.MustParse("10.15.3"),
		},
		{
			msg:       "sw_vers error",
			swVersErr: "beep boop beep\n",
			wantErr:   "call sw_vers: exit status 1",
		},
		{
			msg:          "bad version",
			swVersOutput: "asdf",
			wantErr:      "parse semantic version \"asdf\": Invalid Semantic Version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			v, err := getMacOSVersion(genFakeExecCommand(tt.swVersOutput, tt.swVersErr))
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Assert(t, v.Equal(tt.wantVersion), "Expected: %s\nGot: %s\n", tt.wantVersion, v)
		})
	}
}

// Adapted from https://npf.io/2015/06/testing-exec-command/.
func genFakeExecCommand(output, err string) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestRunExecCmd", "--", name}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{
			"BAGOUP_WANT_TEST_RUN_EXEC_CMD=1",
			fmt.Sprintf("BAGOUP_TEST_RUN_EXEC_CMD_OUTPUT=%s", output),
			fmt.Sprintf("BAGOUP_TEST_RUN_EXEC_CMD_ERROR=%s", err),
		}
		return cmd
	}
}

func TestRunExecCmd(t *testing.T) {
	if os.Getenv("BAGOUP_WANT_TEST_RUN_EXEC_CMD") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, os.Getenv("BAGOUP_TEST_RUN_EXEC_CMD_OUTPUT"))
	err := os.Getenv("BAGOUP_TEST_RUN_EXEC_CMD_ERROR")
	if err != "" {
		fmt.Fprintf(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestGetContactMap(t *testing.T) {
	tagCard := &vcard.Card{
		"VERSION": []*vcard.Field{
			{Value: "3.0"},
		},
		"FN": []*vcard.Field{
			{Value: "David Tagatac"},
		},
		"N": []*vcard.Field{
			{Value: "Tagatac;David;;;"},
		},
		"TEL": []*vcard.Field{
			{Value: "+1415555555", Params: vcard.Params{"TYPE": []string{"CELL"}}},
		},
		"EMAIL": []*vcard.Field{
			{Value: "david@tagatac.net", Params: vcard.Params{"TYPE": []string{"INTERNET"}}},
		},
		"CATEGORIES": []*vcard.Field{
			{Value: "myContacts"},
		},
	}
	noleCard := &vcard.Card{
		"VERSION": []*vcard.Field{
			{Value: "3.0"},
		},
		"FN": []*vcard.Field{
			{Value: "Novak Djokovic"},
		},
		"N": []*vcard.Field{
			{Value: "Djokovic;Novak;;;"},
		},
		"TEL": []*vcard.Field{
			{Value: "+3815555555", Params: vcard.Params{"TYPE": []string{"CELL"}}},
		},
		"EMAIL": []*vcard.Field{
			{Value: "info@novakdjokovic.com", Params: vcard.Params{"TYPE": []string{"INTERNET"}}},
		},
		"CATEGORIES": []*vcard.Field{
			{Value: "myContacts"},
		},
	}
	jelenaCard := &vcard.Card{
		"FN": []*vcard.Field{
			{Value: "Jelena Djokovic"},
		},
		"N": []*vcard.Field{
			{Value: "Djokovic;Jelena;;;"},
		},
		"EMAIL": []*vcard.Field{
			{Value: "info@novakdjokovic.com", Params: vcard.Params{"TYPE": []string{"INTERNET"}}},
		},
	}

	tests := []struct {
		msg     string
		setupFs func(afero.Fs)
		wantMap map[string]*vcard.Card
		wantErr string
	}{
		{
			msg: "two contacts",
			setupFs: func(fs afero.Fs) {
				afero.WriteFile(fs, "contacts.vcf", []byte(
					`BEGIN:VCARD
VERSION:3.0
FN:David Tagatac
N:Tagatac;David;;;
TEL;TYPE=CELL:+1415555555
EMAIL;TYPE=INTERNET:david@tagatac.net
CATEGORIES:myContacts
END:VCARD
BEGIN:VCARD
VERSION:3.0
FN:Novak Djokovic
N:Djokovic;Novak;;;
TEL;TYPE=CELL:+3815555555
EMAIL;TYPE=INTERNET:info@novakdjokovic.com
CATEGORIES:myContacts
END:VCARD
`), 0644)
			},
			wantMap: map[string]*vcard.Card{
				"+1415555555":            tagCard,
				"david@tagatac.net":      tagCard,
				"+3815555555":            noleCard,
				"info@novakdjokovic.com": noleCard,
			},
		},
		{
			msg:     "no contacts file",
			wantMap: nil,
		},
		{
			msg: "bad vcard file",
			setupFs: func(fs afero.Fs) {
				afero.WriteFile(fs, "contacts.vcf", []byte("BEGIN::VCARD\n"), 0644)
			},
			wantErr: "vcard: invalid BEGIN value",
		},
		{
			msg: "shared email address",
			setupFs: func(fs afero.Fs) {
				afero.WriteFile(fs, "contacts.vcf", []byte(
					`BEGIN:VCARD
VERSION:3.0
FN:Novak Djokovic
N:Djokovic;Novak;;;
TEL;TYPE=CELL:+3815555555
EMAIL;TYPE=INTERNET:info@novakdjokovic.com
CATEGORIES:myContacts
END:VCARD
BEGIN:VCARD
FN:Jelena Djokovic
N:Djokovic;Jelena;;;
EMAIL;TYPE=INTERNET:info@novakdjokovic.com
END:VCARD
`), 0644)
			},
			wantMap: map[string]*vcard.Card{
				"+3815555555":            noleCard,
				"info@novakdjokovic.com": jelenaCard,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.setupFs != nil {
				tt.setupFs(fs)
			}

			contactMap, err := getContactMap("contacts.vcf", fs)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, tt.wantMap, contactMap)
		})
	}
}

func TestSanitizePhone(t *testing.T) {
	tests := []struct {
		msg   string
		dirty string
		clean string
	}{
		{
			msg:   "already clean",
			dirty: "+14155555555",
			clean: "+14155555555",
		},
		{
			msg:   "dirty",
			dirty: "+1 (415) 555-5555",
			clean: "+14155555555",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			assert.Equal(t, tt.clean, sanitizePhone(tt.dirty))
		})
	}
}

func TestExportChats(t *testing.T) {
	tests := []struct {
		msg       string
		setupMock func(*mock_chatdb.MockChatDB)
		roFs      bool
		wantFiles map[string]string
		wantErr   string
	}{
		{
			msg: "two chats for one display name, one for another",
			setupMock: func(dMock *mock_chatdb.MockChatDB) {
				dMock.EXPECT().GetChats().Return([]chatdb.Chat{
					{
						ID:          1,
						GUID:        "testguid",
						DisplayName: "testdisplayname",
					},
					{
						ID:          2,
						GUID:        "testguid2",
						DisplayName: "testdisplayname",
					},
					{
						ID:          3,
						GUID:        "testguid3",
						DisplayName: "testdisplayname2",
					},
				}, nil)
				dMock.EXPECT().GetMessageIDs(1).Return([]int{100, 200}, nil)
				dMock.EXPECT().GetMessage(100).Return("message100\n", nil)
				dMock.EXPECT().GetMessage(200).Return("message200\n", nil)
				dMock.EXPECT().GetMessageIDs(2).Return([]int{300, 400}, nil)
				dMock.EXPECT().GetMessage(300).Return("message300\n", nil)
				dMock.EXPECT().GetMessage(400).Return("message400\n", nil)
				dMock.EXPECT().GetMessageIDs(3).Return([]int{500, 600}, nil)
				dMock.EXPECT().GetMessage(500).Return("message500\n", nil)
				dMock.EXPECT().GetMessage(600).Return("message600\n", nil)
			},
			wantFiles: map[string]string{
				"backup/testdisplayname/testguid.txt":   "message100\nmessage200\n",
				"backup/testdisplayname/testguid2.txt":  "message300\nmessage400\n",
				"backup/testdisplayname2/testguid3.txt": "message500\nmessage600\n",
			},
		},
		{
			msg: "GetChats error",
			setupMock: func(dMock *mock_chatdb.MockChatDB) {
				dMock.EXPECT().GetChats().Return(nil, errors.New("this is a DB error"))
			},
			wantErr: "get chats: this is a DB error",
		},
		{
			msg: "directory creation error",
			setupMock: func(dMock *mock_chatdb.MockChatDB) {
				dMock.EXPECT().GetChats().Return([]chatdb.Chat{
					{
						ID:          1,
						GUID:        "testguid",
						DisplayName: "testdisplayname",
					},
				}, nil)
			},
			roFs:    true,
			wantErr: "create directory \"backup/testdisplayname\": operation not permitted",
		},
		{
			msg: "GetMessageIDs error",
			setupMock: func(dMock *mock_chatdb.MockChatDB) {
				dMock.EXPECT().GetChats().Return([]chatdb.Chat{
					{
						ID:          1,
						GUID:        "testguid",
						DisplayName: "testdisplayname",
					},
				}, nil)
				dMock.EXPECT().GetMessageIDs(1).Return(nil, errors.New("this is a DB error"))
			},
			wantErr: "get message IDs for chat ID 1: this is a DB error",
		},
		{
			msg: "GetMessage error",
			setupMock: func(dMock *mock_chatdb.MockChatDB) {
				dMock.EXPECT().GetChats().Return([]chatdb.Chat{
					{
						ID:          1,
						GUID:        "testguid",
						DisplayName: "testdisplayname",
					},
				}, nil)
				dMock.EXPECT().GetMessageIDs(1).Return([]int{100, 200}, nil)
				dMock.EXPECT().GetMessage(100).Return("message100\n", nil)
				dMock.EXPECT().GetMessage(200).Return("", errors.New("this is a DB error"))
			},
			wantErr: "get message with ID 200: this is a DB error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			dMock := mock_chatdb.NewMockChatDB(ctrl)
			tt.setupMock(dMock)
			fs := afero.NewMemMapFs()
			if tt.roFs {
				fs = afero.NewReadOnlyFs(fs)
			}

			err := exportChats(dMock, _exportFolder, fs)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			for filename, expected := range tt.wantFiles {
				actual, err := afero.ReadFile(fs, filename)
				assert.NilError(t, err)
				assert.Equal(t, expected, string(actual))
			}
		})
	}
}
