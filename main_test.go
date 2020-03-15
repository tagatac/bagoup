// Copyright (C) 2020 David Tagatac <david@tagatac.net>
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
					osMock.EXPECT().FileExist("backup").Return(false, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("10.15"), nil),
					dbMock.EXPECT().GetHandleMap(nil).Return(nil, nil),
					osMock.EXPECT().ExportChats(dbMock, "backup", nil, nil, semver.MustParse("10.15")).Return(nil),
				)
			},
		},
		{
			msg:  "default options running on Windows",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().FileExist("backup").Return(false, nil),
					osMock.EXPECT().GetMacOSVersion().Return(nil, errors.New("this is an exec error")),
				)
			},
			wantErr: "get Mac OS version - specify the Mac OS version from which chat.db was copied with the --mac-os-version option: this is an exec error",
		},
		{
			msg:  "export path exists",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				osMock.EXPECT().FileExist("backup").Return(true, nil)
			},
			wantErr: `export folder "backup" already exists - move it or specify a different export path`,
		},
		{
			msg:  "error checking export path",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				osMock.EXPECT().FileExist("backup").Return(false, errors.New("this is a stat error"))
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
					osMock.EXPECT().FileExist("backup").Return(false, nil),
					dbMock.EXPECT().GetHandleMap(nil).Return(nil, nil),
					osMock.EXPECT().ExportChats(dbMock, "backup", nil, nil, semver.MustParse("10.12")).Return(nil),
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
				osMock.EXPECT().FileExist("backup").Return(false, nil)
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
					osMock.EXPECT().FileExist("backup").Return(false, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("10.15"), nil),
					osMock.EXPECT().GetContactMap("contacts.vcf").Return(nil, nil),
					dbMock.EXPECT().GetHandleMap(nil).Return(nil, nil),
					osMock.EXPECT().ExportChats(dbMock, "backup", nil, nil, semver.MustParse("10.15")).Return(nil),
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
					osMock.EXPECT().FileExist("backup").Return(false, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("10.15"), nil),
					dbMock.EXPECT().GetHandleMap(nil).Return(nil, errors.New("this is a DB error")),
				)
			},
			wantErr: "get handle map: this is a DB error",
		},
		{
			msg:  "default options running on Mac OS",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().FileExist("backup").Return(false, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("10.15"), nil),
					dbMock.EXPECT().GetHandleMap(nil).Return(nil, nil),
					osMock.EXPECT().ExportChats(dbMock, "backup", nil, nil, semver.MustParse("10.15")).Return(errors.New("this is a file write error")),
				)
			},
			wantErr: "export chats: this is a file write error",
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
