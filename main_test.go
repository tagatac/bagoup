// Copyright (C) 2020-2022 David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package main

import (
	"errors"
	"os"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/golang/mock/gomock"
	"github.com/tagatac/bagoup/chatdb"
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
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				osMock.EXPECT().FileAccess("~/Library/Messages/chat.db").Return(errors.New("this is a permissions error"))
			},
			wantErr: `test DB file "~/Library/Messages/chat.db" - FIX: https://github.com/tagatac/bagoup/blob/master/README.md#protected-file-access: this is a permissions error`,
		},
		{
			msg:  "running on Windows",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
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
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
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
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
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
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
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
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
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
			msg:  "error getting handle map",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist("messages-export"),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
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

func TestExportChats(t *testing.T) {
	tests := []struct {
		msg             string
		separateChats   bool
		pdf             bool
		copyAttachments bool
		setupMocks      func(*mock_chatdb.MockChatDB, *mock_opsys.MockOS, []*mock_opsys.MockOutFile)
		wantCount       int
		wantErr         string
	}{
		{
			msg: "two chats for one display name, one for another",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths().Return(map[int][]string{
						100: {"attachmentpath"},
					}, nil),
					dbMock.EXPECT().GetChats(nil).Return([]chatdb.EntityChats{
						{
							Name: "testdisplayname",
							Chats: []chatdb.Chat{
								{
									ID:   1,
									GUID: "testguid",
								},
								{
									ID:   2,
									GUID: "testguid2",
								},
							},
						},
						{
							Name: "testdisplayname2",
							Chats: []chatdb.Chat{
								{
									ID:   3,
									GUID: "testguid3",
								},
							},
						},
					}, nil),
					dbMock.EXPECT().GetMessageIDs(1),
					dbMock.EXPECT().GetMessageIDs(2),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/testdisplayname/testguid;;;testguid2", false, false).Return(ofMocks[0], nil),
					ofMocks[0].EXPECT().Close().Times(2),
					dbMock.EXPECT().GetMessageIDs(3),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname2", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/testdisplayname2/testguid3", false, false).Return(ofMocks[1], nil),
					ofMocks[1].EXPECT().Close().Times(2),
				)
			},
			wantCount: 0,
		},
		{
			msg:           "separate chats",
			separateChats: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths().Return(map[int][]string{
						100: {"attachmentpath"},
					}, nil),
					dbMock.EXPECT().GetChats(nil).Return([]chatdb.EntityChats{
						{
							Name: "testdisplayname",
							Chats: []chatdb.Chat{
								{
									ID:   1,
									GUID: "testguid",
								},
								{
									ID:   2,
									GUID: "testguid2",
								},
							},
						},
						{
							Name: "testdisplayname2",
							Chats: []chatdb.Chat{
								{
									ID:   3,
									GUID: "testguid3",
								},
							},
						},
					}, nil),
					dbMock.EXPECT().GetMessageIDs(1),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/testdisplayname/testguid", false, false).Return(ofMocks[0], nil),
					ofMocks[0].EXPECT().Close().Times(2),
					dbMock.EXPECT().GetMessageIDs(2),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/testdisplayname/testguid2", false, false).Return(ofMocks[1], nil),
					ofMocks[1].EXPECT().Close().Times(2),
					dbMock.EXPECT().GetMessageIDs(3),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname2", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/testdisplayname2/testguid3", false, false).Return(ofMocks[2], nil),
					ofMocks[2].EXPECT().Close().Times(2),
				)
			},
			wantCount: 0,
		},
		{
			msg: "pdf",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths().Return(map[int][]string{
						100: {"attachmentpath"},
					}, nil),
					osMock.EXPECT().FileAccess("attachmentpath"),
					dbMock.EXPECT().GetChats(nil).Return([]chatdb.EntityChats{
						{
							Name: "testdisplayname",
							Chats: []chatdb.Chat{
								{
									ID:   1,
									GUID: "testguid",
								},
								{
									ID:   2,
									GUID: "testguid2",
								},
							},
						},
						{
							Name: "testdisplayname2",
							Chats: []chatdb.Chat{
								{
									ID:   3,
									GUID: "testguid3",
								},
							},
						},
					}, nil),
					dbMock.EXPECT().GetMessageIDs(1),
					dbMock.EXPECT().GetMessageIDs(2),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/testdisplayname/testguid;;;testguid2", true, false).Return(ofMocks[0], nil),
					ofMocks[0].EXPECT().Close().Times(2),
					dbMock.EXPECT().GetMessageIDs(3),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname2", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/testdisplayname2/testguid3", true, false).Return(ofMocks[1], nil),
					ofMocks[1].EXPECT().Close().Times(2),
				)
			},
			wantCount: 0,
		},
		{
			msg: "pdf without attachments",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths().Return(map[int][]string{
						100: {},
					}, nil),
					dbMock.EXPECT().GetChats(nil).Return([]chatdb.EntityChats{
						{
							Name: "testdisplayname",
							Chats: []chatdb.Chat{
								{
									ID:   1,
									GUID: "testguid",
								},
								{
									ID:   2,
									GUID: "testguid2",
								},
							},
						},
						{
							Name: "testdisplayname2",
							Chats: []chatdb.Chat{
								{
									ID:   3,
									GUID: "testguid3",
								},
							},
						},
					}, nil),
					dbMock.EXPECT().GetMessageIDs(1),
					dbMock.EXPECT().GetMessageIDs(2),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/testdisplayname/testguid;;;testguid2", true, false).Return(ofMocks[0], nil),
					ofMocks[0].EXPECT().Close().Times(2),
					dbMock.EXPECT().GetMessageIDs(3),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname2", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/testdisplayname2/testguid3", true, false).Return(ofMocks[1], nil),
					ofMocks[1].EXPECT().Close().Times(2),
				)
			},
			wantCount: 0,
		},
		{
			msg:             "copy attachments",
			copyAttachments: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths().Return(map[int][]string{
						100: {"attachmentpath"},
					}, nil),
					osMock.EXPECT().FileAccess("attachmentpath"),
					dbMock.EXPECT().GetChats(nil).Return([]chatdb.EntityChats{
						{
							Name: "testdisplayname",
							Chats: []chatdb.Chat{
								{
									ID:   1,
									GUID: "testguid",
								},
								{
									ID:   2,
									GUID: "testguid2",
								},
							},
						},
						{
							Name: "testdisplayname2",
							Chats: []chatdb.Chat{
								{
									ID:   3,
									GUID: "testguid3",
								},
							},
						},
					}, nil),
					dbMock.EXPECT().GetMessageIDs(1),
					dbMock.EXPECT().GetMessageIDs(2),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/testdisplayname/testguid;;;testguid2", false, false).Return(ofMocks[0], nil),
					osMock.EXPECT().Mkdir("messages-export/testdisplayname/attachments", os.ModePerm),
					ofMocks[0].EXPECT().Close().Times(2),
					dbMock.EXPECT().GetMessageIDs(3),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname2", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/testdisplayname2/testguid3", false, false).Return(ofMocks[1], nil),
					osMock.EXPECT().Mkdir("messages-export/testdisplayname2/attachments", os.ModePerm),
					ofMocks[1].EXPECT().Close().Times(2),
				)
			},
			wantCount: 0,
		},
		{
			msg: "error getting attachment paths",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				dbMock.EXPECT().GetAttachmentPaths().Return(nil, errors.New("this is a DB error"))
			},
			wantCount: 0,
			wantErr:   "get attachment paths: this is a DB error",
		},
		{
			msg: "pdf export - no access to attachment path",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths().Return(map[int][]string{
						100: {"attachmentpath"},
					}, nil),
					osMock.EXPECT().FileAccess("attachmentpath").Return(errors.New("this is a permissions error")),
				)
			},
			wantErr: "access to attachments - FIX: https://github.com/tagatac/bagoup/blob/master/README.md#protected-file-access: this is a permissions error",
		},
		{
			msg: "GetMessageIDs error",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths().Return(map[int][]string{
						100: {"attachmentpath"},
					}, nil),
					dbMock.EXPECT().GetChats(nil).Return([]chatdb.EntityChats{
						{
							Name: "testdisplayname",
							Chats: []chatdb.Chat{
								{
									ID:   1,
									GUID: "testguid",
								},
								{
									ID:   2,
									GUID: "testguid2",
								},
							},
						},
						{
							Name: "testdisplayname2",
							Chats: []chatdb.Chat{
								{
									ID:   3,
									GUID: "testguid3",
								},
							},
						},
					}, nil),
					dbMock.EXPECT().GetMessageIDs(1).Return(nil, errors.New("this is a DB error")),
				)
			},
			wantErr: "get message IDs for chat ID 1: this is a DB error",
		},
		{
			msg: "writeFile error",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths().Return(map[int][]string{
						100: {"attachmentpath"},
					}, nil),
					dbMock.EXPECT().GetChats(nil).Return([]chatdb.EntityChats{
						{
							Name: "testdisplayname",
							Chats: []chatdb.Chat{
								{
									ID:   1,
									GUID: "testguid",
								},
								{
									ID:   2,
									GUID: "testguid2",
								},
							},
						},
						{
							Name: "testdisplayname2",
							Chats: []chatdb.Chat{
								{
									ID:   3,
									GUID: "testguid3",
								},
							},
						},
					}, nil),
					dbMock.EXPECT().GetMessageIDs(1),
					dbMock.EXPECT().GetMessageIDs(2),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname", os.ModePerm).Return(errors.New("this is a permissions error")),
				)
			},
			wantErr: `create directory "messages-export/testdisplayname": this is a permissions error`,
		},
		{
			msg:           "separate chats - writeFile error",
			separateChats: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths().Return(map[int][]string{
						100: {"attachmentpath"},
					}, nil),
					dbMock.EXPECT().GetChats(nil).Return([]chatdb.EntityChats{
						{
							Name: "testdisplayname",
							Chats: []chatdb.Chat{
								{
									ID:   1,
									GUID: "testguid",
								},
								{
									ID:   2,
									GUID: "testguid2",
								},
							},
						},
						{
							Name: "testdisplayname2",
							Chats: []chatdb.Chat{
								{
									ID:   3,
									GUID: "testguid3",
								},
							},
						},
					}, nil),
					dbMock.EXPECT().GetMessageIDs(1),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname", os.ModePerm).Return(errors.New("this is a permissions error")),
				)
			},
			wantErr: `create directory "messages-export/testdisplayname": this is a permissions error`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			dbMock := mock_chatdb.NewMockChatDB(ctrl)
			osMock := mock_opsys.NewMockOS(ctrl)
			ofMocks := []*mock_opsys.MockOutFile{
				mock_opsys.NewMockOutFile(ctrl),
				mock_opsys.NewMockOutFile(ctrl),
				mock_opsys.NewMockOutFile(ctrl),
			}
			tt.setupMocks(dbMock, osMock, ofMocks)

			cfg := configuration{
				Options: options{
					ExportPath:      "messages-export",
					SeparateChats:   tt.separateChats,
					OutputPDF:       tt.pdf,
					CopyAttachments: tt.copyAttachments,
				},
				OS:     osMock,
				ChatDB: dbMock,
			}
			count, err := cfg.exportChats(nil)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, tt.wantCount, count)
		})
	}
}

func TestWriteFile(t *testing.T) {
	tests := []struct {
		msg             string
		pdf             bool
		copyAttachments bool
		setupMocks      func(*mock_chatdb.MockChatDB, *mock_opsys.MockOS, *mock_opsys.MockOutFile)
		wantCount       int
		wantErr         string
	}{
		{
			msg: "text export",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com", false, false).Return(ofMock, nil),
					dbMock.EXPECT().GetMessage(1, nil, semver.MustParse("12.4")).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil, semver.MustParse("12.4")).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment1"),
					osMock.EXPECT().FileExist("attachment2").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment2"),
					ofMock.EXPECT().Close().Times(2),
				)
			},
			wantCount: 2,
		},
		{
			msg: "pdf export",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com", true, false).Return(ofMock, nil),
					dbMock.EXPECT().GetMessage(1, nil, semver.MustParse("12.4")).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil, semver.MustParse("12.4")).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1").Return("attachment1", nil),
					ofMock.EXPECT().WriteAttachment("attachment1"),
					osMock.EXPECT().FileExist("attachment2").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2").Return("attachment2", nil),
					ofMock.EXPECT().WriteAttachment("attachment2"),
					ofMock.EXPECT().Close().Times(2),
				)
			},
			wantCount: 2,
		},
		{
			msg:             "copy attachments",
			copyAttachments: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com", false, false).Return(ofMock, nil),
					osMock.EXPECT().Mkdir("messages-export/friend/attachments", os.ModePerm),
					dbMock.EXPECT().GetMessage(1, nil, semver.MustParse("12.4")).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil, semver.MustParse("12.4")).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment1", "messages-export/friend/attachments"),
					ofMock.EXPECT().WriteAttachment("attachment1"),
					osMock.EXPECT().FileExist("attachment2").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment2", "messages-export/friend/attachments"),
					ofMock.EXPECT().WriteAttachment("attachment2"),
					ofMock.EXPECT().Close().Times(2),
				)
			},
			wantCount: 2,
		},
		{
			msg: "NewOutFile error",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com", false, false).Return(nil, errors.New("this is a permissions error")),
				)
			},
			wantErr: `open/create file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com": this is a permissions error`,
		},
		{
			msg:             "error creating attachments folder",
			copyAttachments: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com", false, false).Return(ofMock, nil),
					osMock.EXPECT().Mkdir("messages-export/friend/attachments", os.ModePerm).Return(errors.New("this is a permissions error")),
					ofMock.EXPECT().Close(),
				)
			},
			wantErr: `create directory "messages-export/friend/attachments": this is a permissions error`,
		},
		{
			msg: "GetMessage error",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com", false, false).Return(ofMock, nil),
					dbMock.EXPECT().GetMessage(1, nil, semver.MustParse("12.4")).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil, semver.MustParse("12.4")).Return("message2", nil).Return("", errors.New("this is a DB error")),
					ofMock.EXPECT().Close(),
				)
			},
			wantCount: 1,
			wantErr:   "get message with ID 2: this is a DB error",
		},
		{
			msg: "WriteMessage error",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com", false, false).Return(ofMock, nil),
					dbMock.EXPECT().GetMessage(1, nil, semver.MustParse("12.4")).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil, semver.MustParse("12.4")).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2").Return(errors.New("this is an outfile error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					ofMock.EXPECT().Close(),
				)
			},
			wantCount: 1,
			wantErr:   `write message "message2" to file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt": this is an outfile error`,
		},
		{
			msg: "attachment file does not exist",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com", false, false).Return(ofMock, nil),
					dbMock.EXPECT().GetMessage(1, nil, semver.MustParse("12.4")).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil, semver.MustParse("12.4")).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1").Return(false, nil),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					osMock.EXPECT().FileExist("attachment2").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment2"),
					ofMock.EXPECT().Close().Times(2),
				)
			},
			wantCount: 2,
		},
		{
			msg: "file existence check fails",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com", false, false).Return(ofMock, nil),
					dbMock.EXPECT().GetMessage(1, nil, semver.MustParse("12.4")).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil, semver.MustParse("12.4")).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1").Return(false, errors.New("this is a permissions error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					ofMock.EXPECT().Close(),
				)
			},
			wantCount: 2,
			wantErr:   `chat file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt" - message 2: check existence of file "attachment1" - POSSIBLE FIX: https://github.com/tagatac/bagoup/blob/master/README.md#protected-file-access: this is a permissions error`,
		},
		{
			msg:             "CopyFile error",
			copyAttachments: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com", false, false).Return(ofMock, nil),
					osMock.EXPECT().Mkdir("messages-export/friend/attachments", os.ModePerm),
					dbMock.EXPECT().GetMessage(1, nil, semver.MustParse("12.4")).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil, semver.MustParse("12.4")).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment1", "messages-export/friend/attachments"),
					ofMock.EXPECT().WriteAttachment("attachment1"),
					osMock.EXPECT().FileExist("attachment2").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment2", "messages-export/friend/attachments").Return(errors.New("this is a permissions error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					ofMock.EXPECT().Close(),
				)
			},
			wantCount: 1,
			wantErr:   `chat file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt" - message 2: copy attachment "attachment2" to "messages-export/friend/attachments": this is a permissions error`,
		},
		{
			msg: "HEIC conversion error",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com", true, false).Return(ofMock, nil),
					dbMock.EXPECT().GetMessage(1, nil, semver.MustParse("12.4")).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil, semver.MustParse("12.4")).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1").Return("attachment1", nil),
					ofMock.EXPECT().WriteAttachment("attachment1"),
					osMock.EXPECT().FileExist("attachment2").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2").Return("attachment2", nil).Return("", errors.New("this is a goheif error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().Close(),
				)
			},
			wantCount: 1,
			wantErr:   `chat file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf" - message 2: convert HEIC file "attachment2" to JPG: this is a goheif error`,
		},
		{
			msg: "WriteAttachment error",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().NewOutFile("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com", false, false).Return(ofMock, nil),
					dbMock.EXPECT().GetMessage(1, nil, semver.MustParse("12.4")).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil, semver.MustParse("12.4")).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment1"),
					osMock.EXPECT().FileExist("attachment2").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment2").Return(errors.New("this is an outfile error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					ofMock.EXPECT().Close(),
				)
			},
			wantCount: 1,
			wantErr:   `chat file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt" - message 2: include attachment "attachment2": this is an outfile error`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			dbMock := mock_chatdb.NewMockChatDB(ctrl)
			osMock := mock_opsys.NewMockOS(ctrl)
			ofMock := mock_opsys.NewMockOutFile(ctrl)
			tt.setupMocks(dbMock, osMock, ofMock)

			cfg := configuration{
				Options: options{
					ExportPath:      "messages-export",
					OutputPDF:       tt.pdf,
					CopyAttachments: tt.copyAttachments,
				},
				OS:           osMock,
				ChatDB:       dbMock,
				MacOSVersion: semver.MustParse("12.4"),
				AttachmentPaths: map[int][]string{
					2: {"attachment1", "attachment2"},
				},
			}
			count, err := cfg.writeFile(
				"friend",
				[]string{"iMessage;-;friend@gmail.com", "iMessage;-;friend@hotmail.com"},
				[]chatdb.DatedMessageID{
					{ID: 2, Date: 2},
					{ID: 1, Date: 1},
				},
			)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, tt.wantCount, count)
		})
	}
}
