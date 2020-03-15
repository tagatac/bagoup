// Copyright (C) 2020 David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package main

import (
	"errors"
	"os"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/chatdb"
	"github.com/tagatac/bagoup/chatdb/mock_chatdb"
	"github.com/tagatac/bagoup/opsys"
	"github.com/tagatac/bagoup/opsys/mock_opsys"
	"gotest.tools/v3/assert"
)

func TestBagoup(t *testing.T) {
	defaultOpts := options{
		DBPath:     "~/Library/Messages/chat.db",
		ExportPath: "backup",
		SelfHandle: "Me",
	}
	tenDotTwelve := "10.12"
	tenDotTenDotTenDotTen := "10.10.10.10"
	contactsPath := "contacts.vcf"

	tests := []struct {
		msg        string
		opts       options
		setupMocks func(*mock_opsys.MockOS, *mock_chatdb.MockChatDB)
		wantErr    string
	}{
		{
			msg:  "default options running on Mac OS",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().Open("~/Library/Messages/chat.db").Return(&os.File{}, nil),
					osMock.EXPECT().FileExist("backup").Return(false, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("10.15"), nil),
					dbMock.EXPECT().GetHandleMap(nil).Return(nil, nil),
					dbMock.EXPECT().GetChats(nil).Return(nil, nil),
				)
			},
		},
		{
			msg:  "default options running on Mac OS",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				osMock.EXPECT().Open("~/Library/Messages/chat.db").Return(nil, errors.New("this is a permissions error"))
			},
			wantErr: `test DB file "~/Library/Messages/chat.db" - FIX: https://github.com/tagatac/bagoup/blob/master/README.md#chatdb-access: this is a permissions error`,
		},
		{
			msg:  "default options running on Windows",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().Open("~/Library/Messages/chat.db").Return(&os.File{}, nil),
					osMock.EXPECT().FileExist("backup").Return(false, nil),
					osMock.EXPECT().GetMacOSVersion().Return(nil, errors.New("this is an exec error")),
				)
			},
			wantErr: "get Mac OS version - FIX: specify the Mac OS version from which chat.db was copied with the --mac-os-version option: this is an exec error",
		},
		{
			msg:  "export path exists",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().Open("~/Library/Messages/chat.db").Return(&os.File{}, nil),
					osMock.EXPECT().FileExist("backup").Return(true, nil),
				)
			},
			wantErr: `export folder "backup" already exists - FIX: move it or specify a different export path with the --export-path option`,
		},
		{
			msg:  "error checking export path",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().Open("~/Library/Messages/chat.db").Return(&os.File{}, nil),
					osMock.EXPECT().FileExist("backup").Return(false, errors.New("this is a stat error")),
				)
			},
			wantErr: `check export path "backup": this is a stat error`,
		},
		{
			msg: "chat.db version specified",
			opts: options{
				DBPath:       "~/Library/Messages/chat.db",
				ExportPath:   "backup",
				MacOSVersion: &tenDotTwelve,
				SelfHandle:   "Me",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().Open("~/Library/Messages/chat.db").Return(&os.File{}, nil),
					osMock.EXPECT().FileExist("backup").Return(false, nil),
					dbMock.EXPECT().GetHandleMap(nil).Return(nil, nil),
					dbMock.EXPECT().GetChats(nil).Return(nil, nil),
				)
			},
		},
		{
			msg: "invalid chat.db version specified",
			opts: options{
				DBPath:       "~/Library/Messages/chat.db",
				ExportPath:   "backup",
				MacOSVersion: &tenDotTenDotTenDotTen,
				SelfHandle:   "Me",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().Open("~/Library/Messages/chat.db").Return(&os.File{}, nil),
					osMock.EXPECT().FileExist("backup").Return(false, nil),
				)
			},
			wantErr: `parse Mac OS version "10.10.10.10": Invalid Semantic Version`,
		},
		{
			msg: "contacts file specified",
			opts: options{
				DBPath:       "~/Library/Messages/chat.db",
				ContactsPath: &contactsPath,
				ExportPath:   "backup",
				SelfHandle:   "Me",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().Open("~/Library/Messages/chat.db").Return(&os.File{}, nil),
					osMock.EXPECT().FileExist("backup").Return(false, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("10.15"), nil),
					osMock.EXPECT().GetContactMap("contacts.vcf").Return(nil, nil),
					dbMock.EXPECT().GetHandleMap(nil).Return(nil, nil),
					dbMock.EXPECT().GetChats(nil).Return(nil, nil),
				)
			},
		},
		{
			msg: "error getting contact map",
			opts: options{
				DBPath:       "~/Library/Messages/chat.db",
				ContactsPath: &contactsPath,
				ExportPath:   "backup",
				SelfHandle:   "Me",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().Open("~/Library/Messages/chat.db").Return(&os.File{}, nil),
					osMock.EXPECT().FileExist("backup").Return(false, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("10.15"), nil),
					osMock.EXPECT().GetContactMap("contacts.vcf").Return(nil, errors.New("this is an os error")),
				)
			},
			wantErr: `get contacts from vcard file "contacts.vcf": this is an os error`,
		},
		{
			msg:  "error getting handle map",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().Open("~/Library/Messages/chat.db").Return(&os.File{}, nil),
					osMock.EXPECT().FileExist("backup").Return(false, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("10.15"), nil),
					dbMock.EXPECT().GetHandleMap(nil).Return(nil, errors.New("this is a DB error")),
				)
			},
			wantErr: "get handle map: this is a DB error",
		},
		{
			msg:  "export chats error",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().Open("~/Library/Messages/chat.db").Return(&os.File{}, nil),
					osMock.EXPECT().FileExist("backup").Return(false, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("10.15"), nil),
					dbMock.EXPECT().GetHandleMap(nil).Return(nil, nil),
					dbMock.EXPECT().GetChats(nil).Return(nil, errors.New("this is a DB error")),
				)
			},
			wantErr: "export chats: get chats: this is a DB error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			osMock := mock_opsys.NewMockOS(ctrl)
			dbMock := mock_chatdb.NewMockChatDB(ctrl)
			tt.setupMocks(osMock, dbMock)

			err := bagoup(tt.opts, osMock, dbMock)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
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
			setupMock: func(dbMock *mock_chatdb.MockChatDB) {
				dbMock.EXPECT().GetChats(nil).Return([]chatdb.Chat{
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
				dbMock.EXPECT().GetMessageIDs(1).Return([]int{100, 200}, nil)
				dbMock.EXPECT().GetMessage(100, nil, nil).Return("message100\n", nil)
				dbMock.EXPECT().GetMessage(200, nil, nil).Return("message200\n", nil)
				dbMock.EXPECT().GetMessageIDs(2).Return([]int{300, 400}, nil)
				dbMock.EXPECT().GetMessage(300, nil, nil).Return("message300\n", nil)
				dbMock.EXPECT().GetMessage(400, nil, nil).Return("message400\n", nil)
				dbMock.EXPECT().GetMessageIDs(3).Return([]int{500, 600}, nil)
				dbMock.EXPECT().GetMessage(500, nil, nil).Return("message500\n", nil)
				dbMock.EXPECT().GetMessage(600, nil, nil).Return("message600\n", nil)
			},
			wantFiles: map[string]string{
				"backup/testdisplayname/testguid.txt":   "message100\nmessage200\n",
				"backup/testdisplayname/testguid2.txt":  "message300\nmessage400\n",
				"backup/testdisplayname2/testguid3.txt": "message500\nmessage600\n",
			},
		},
		{
			msg: "GetChats error",
			setupMock: func(dbMock *mock_chatdb.MockChatDB) {
				dbMock.EXPECT().GetChats(nil).Return(nil, errors.New("this is a DB error"))
			},
			wantErr: "get chats: this is a DB error",
		},
		{
			msg: "directory creation error",
			setupMock: func(dbMock *mock_chatdb.MockChatDB) {
				dbMock.EXPECT().GetChats(nil).Return([]chatdb.Chat{
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
			setupMock: func(dbMock *mock_chatdb.MockChatDB) {
				dbMock.EXPECT().GetChats(nil).Return([]chatdb.Chat{
					{
						ID:          1,
						GUID:        "testguid",
						DisplayName: "testdisplayname",
					},
				}, nil)
				dbMock.EXPECT().GetMessageIDs(1).Return(nil, errors.New("this is a DB error"))
			},
			wantErr: "get message IDs for chat ID 1: this is a DB error",
		},
		{
			msg: "GetMessage error",
			setupMock: func(dbMock *mock_chatdb.MockChatDB) {
				dbMock.EXPECT().GetChats(nil).Return([]chatdb.Chat{
					{
						ID:          1,
						GUID:        "testguid",
						DisplayName: "testdisplayname",
					},
				}, nil)
				dbMock.EXPECT().GetMessageIDs(1).Return([]int{100, 200}, nil)
				dbMock.EXPECT().GetMessage(100, nil, nil).Return("message100\n", nil)
				dbMock.EXPECT().GetMessage(200, nil, nil).Return("", errors.New("this is a DB error"))
			},
			wantErr: "get message with ID 200: this is a DB error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			dbMock := mock_chatdb.NewMockChatDB(ctrl)
			tt.setupMock(dbMock)
			fs := afero.NewMemMapFs()
			if tt.roFs {
				fs = afero.NewReadOnlyFs(fs)
			}
			s := opsys.NewOS(fs, nil, nil)

			err := exportChats(s, dbMock, "backup", nil, nil, nil)
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
