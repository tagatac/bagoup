// Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package main

import (
	"errors"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/golang/mock/gomock"
	"github.com/tagatac/bagoup/chatdb/mock_chatdb"
	"github.com/tagatac/bagoup/opsys/mock_opsys"
	"gotest.tools/v3/assert"
)

func TestBagoup(t *testing.T) {
	defaultOpts := options{
		DBPath:     "~/Library/Messages/chat.db",
		ExportPath: "messages-export",
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
			msg:  "default options",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist("messages-export"),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					dbMock.EXPECT().Init(semver.MustParse("12.4")),
					dbMock.EXPECT().GetHandleMap(nil),
					dbMock.EXPECT().GetAttachmentPaths(),
					dbMock.EXPECT().GetChats(nil),
					osMock.EXPECT().RmTempDir().Times(2),
				)
			},
		},
		{
			msg:  "missing chatDB read permissions",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, _ *mock_chatdb.MockChatDB) {
				osMock.EXPECT().FileAccess("~/Library/Messages/chat.db").Return(errors.New("this is a permissions error"))
			},
			wantErr: `test DB file "~/Library/Messages/chat.db" - FIX: https://github.com/tagatac/bagoup/blob/master/README.md#protected-file-access: this is a permissions error`,
		},
		{
			msg:  "running on Windows",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, _ *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist("messages-export"),
					osMock.EXPECT().GetMacOSVersion().Return(nil, errors.New("this is an exec error")),
				)
			},
			wantErr: "get Mac OS version - FIX: specify the Mac OS version from which chat.db was copied with the --mac-os-version option: this is an exec error",
		},
		{
			msg:  "export path exists",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, _ *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist("messages-export").Return(true, nil),
				)
			},
			wantErr: `export folder "messages-export" already exists - FIX: move it or specify a different export path with the --export-path option`,
		},
		{
			msg:  "error checking export path",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, _ *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist("messages-export").Return(false, errors.New("this is a stat error")),
				)
			},
			wantErr: `check export path "messages-export": this is a stat error`,
		},
		{
			msg: "chat.db version specified",
			opts: options{
				DBPath:       "~/Library/Messages/chat.db",
				ExportPath:   "messages-export",
				MacOSVersion: &tenDotTwelve,
				SelfHandle:   "Me",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist("messages-export"),
					dbMock.EXPECT().Init(semver.MustParse("10.12")),
					dbMock.EXPECT().GetHandleMap(nil),
					dbMock.EXPECT().GetAttachmentPaths(),
					dbMock.EXPECT().GetChats(nil),
					osMock.EXPECT().RmTempDir().Times(2),
				)
			},
		},
		{
			msg: "invalid chat.db version specified",
			opts: options{
				DBPath:       "~/Library/Messages/chat.db",
				ExportPath:   "messages-export",
				MacOSVersion: &tenDotTenDotTenDotTen,
				SelfHandle:   "Me",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, _ *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist("messages-export"),
				)
			},
			wantErr: `parse Mac OS version "10.10.10.10": Invalid Semantic Version`,
		},
		{
			msg: "contacts file specified",
			opts: options{
				DBPath:       "~/Library/Messages/chat.db",
				ExportPath:   "messages-export",
				ContactsPath: &contactsPath,
				SelfHandle:   "Me",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist("messages-export"),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					osMock.EXPECT().GetContactMap("contacts.vcf"),
					dbMock.EXPECT().Init(semver.MustParse("12.4")),
					dbMock.EXPECT().GetHandleMap(nil),
					dbMock.EXPECT().GetAttachmentPaths(),
					dbMock.EXPECT().GetChats(nil),
					osMock.EXPECT().RmTempDir().Times(2),
				)
			},
		},
		{
			msg: "error getting contact map",
			opts: options{
				DBPath:       "~/Library/Messages/chat.db",
				ExportPath:   "messages-export",
				ContactsPath: &contactsPath,
				SelfHandle:   "Me",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, _ *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist("messages-export"),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					osMock.EXPECT().GetContactMap("contacts.vcf").Return(nil, errors.New("this is an os error")),
				)
			},
			wantErr: `get contacts from vcard file "contacts.vcf": this is an os error`,
		},
		{
			msg:  "error initializing chat DB",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist("messages-export"),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					dbMock.EXPECT().Init(semver.MustParse("12.4")).Return(errors.New("this is a DB error")),
				)
			},
			wantErr: "initialize the database for reading on Mac OS version 12.4.0: this is a DB error",
		},
		{
			msg:  "error getting handle map",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist("messages-export"),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					dbMock.EXPECT().Init(semver.MustParse("12.4")),
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
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist("messages-export"),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					dbMock.EXPECT().Init(semver.MustParse("12.4")),
					dbMock.EXPECT().GetHandleMap(nil),
					dbMock.EXPECT().GetAttachmentPaths(),
					dbMock.EXPECT().GetChats(nil).Return(nil, errors.New("this is a DB error")),
					osMock.EXPECT().RmTempDir(),
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

			cfg := configuration{
				Options: tt.opts,
				OS:      osMock,
				ChatDB:  dbMock,
			}
			err := cfg.bagoup()
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
		})
	}
}
