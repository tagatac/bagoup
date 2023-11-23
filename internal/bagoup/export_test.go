// Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package bagoup

import (
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/chatdb"
	"github.com/tagatac/bagoup/chatdb/mock_chatdb"
	"github.com/tagatac/bagoup/opsys/mock_opsys"
	"gotest.tools/v3/assert"
)

func TestExportChats(t *testing.T) {
	chatFile, err := afero.NewMemMapFs().Create("testfile")
	assert.NilError(t, err)

	tests := []struct {
		msg             string
		separateChats   bool
		pdf             bool
		copyAttachments bool
		entities        []string
		setupMocks      func(*mock_chatdb.MockChatDB, *mock_opsys.MockOS, []*mock_opsys.MockOutFile)
		wantErr         string
	}{
		{
			msg: "two chats for one display name, one for another",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths(nil).Return(map[int][]chatdb.Attachment{
						100: {{Filename: "attachmentpath"}},
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
					osMock.EXPECT().Create("messages-export/testdisplayname/testguid;;;testguid2.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMocks[0]),
					ofMocks[0].EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
					dbMock.EXPECT().GetMessageIDs(3),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname2", os.ModePerm),
					osMock.EXPECT().Create("messages-export/testdisplayname2/testguid3.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMocks[1]),
					ofMocks[1].EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
				)
			},
		},
		{
			msg:      "filter one entity",
			entities: []string{"testdisplayname"},
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths(nil).Return(map[int][]chatdb.Attachment{
						100: {{Filename: "attachmentpath"}},
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
					osMock.EXPECT().Create("messages-export/testdisplayname/testguid;;;testguid2.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMocks[0]),
					ofMocks[0].EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
				)
			},
		},
		{
			msg:      "specify both entities, so don't filter any",
			entities: []string{"testdisplayname", "testdisplayname2"},
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths(nil).Return(map[int][]chatdb.Attachment{
						100: {{Filename: "attachmentpath"}},
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
					osMock.EXPECT().Create("messages-export/testdisplayname/testguid;;;testguid2.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMocks[0]),
					ofMocks[0].EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
					dbMock.EXPECT().GetMessageIDs(3),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname2", os.ModePerm),
					osMock.EXPECT().Create("messages-export/testdisplayname2/testguid3.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMocks[1]),
					ofMocks[1].EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
				)
			},
		},
		{
			msg:           "separate chats",
			separateChats: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths(nil).Return(map[int][]chatdb.Attachment{
						100: {{Filename: "attachmentpath"}},
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
					}, nil),
					dbMock.EXPECT().GetMessageIDs(1),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname", os.ModePerm),
					osMock.EXPECT().Create("messages-export/testdisplayname/testguid.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMocks[0]),
					ofMocks[0].EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
					dbMock.EXPECT().GetMessageIDs(2),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname", os.ModePerm),
					osMock.EXPECT().Create("messages-export/testdisplayname/testguid2.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMocks[1]),
					ofMocks[1].EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
				)
			},
		},
		{
			msg: "pdf",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths(nil).Return(map[int][]chatdb.Attachment{
						100: {{Filename: "attachmentpath"}},
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
							},
						},
					}, nil),
					dbMock.EXPECT().GetMessageIDs(1),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname", os.ModePerm),
					osMock.EXPECT().Create("messages-export/testdisplayname/testguid.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewPDFOutFile(chatFile, gomock.Any(), false).Return(ofMocks[0]),
					ofMocks[0].EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
				)
			},
		},
		{
			msg: "pdf without attachments",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths(nil).Return(map[int][]chatdb.Attachment{
						100: {},
						200: {{}},
					}, nil),
					dbMock.EXPECT().GetChats(nil).Return([]chatdb.EntityChats{
						{
							Name: "testdisplayname",
							Chats: []chatdb.Chat{
								{
									ID:   1,
									GUID: "testguid",
								},
							},
						},
					}, nil),
					dbMock.EXPECT().GetMessageIDs(1),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname", os.ModePerm),
					osMock.EXPECT().Create("messages-export/testdisplayname/testguid.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewPDFOutFile(chatFile, gomock.Any(), false).Return(ofMocks[0]),
					ofMocks[0].EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
				)
			},
		},
		{
			msg:             "copy attachments",
			copyAttachments: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMocks []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths(nil).Return(map[int][]chatdb.Attachment{
						100: {{Filename: "attachmentpath"}},
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
							},
						},
					}, nil),
					dbMock.EXPECT().GetMessageIDs(1),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname", os.ModePerm),
					osMock.EXPECT().MkdirAll("messages-export/testdisplayname/attachments", os.ModePerm),
					osMock.EXPECT().Create("messages-export/testdisplayname/testguid.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMocks[0]),
					ofMocks[0].EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
				)
			},
		},
		{
			msg: "error getting attachment paths",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, _ *mock_opsys.MockOS, _ []*mock_opsys.MockOutFile) {
				dbMock.EXPECT().GetAttachmentPaths(nil).Return(nil, errors.New("this is a DB error"))
			},
			wantErr: "get attachment paths: this is a DB error",
		},
		{
			msg: "pdf export - no access to attachment path",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths(nil).Return(map[int][]chatdb.Attachment{
						100: {{Filename: "attachmentpath"}},
					}, nil),
					osMock.EXPECT().FileAccess("attachmentpath").Return(errors.New("this is a permissions error")),
				)
			},
			wantErr: "access to attachments - FIX: https://github.com/tagatac/bagoup/blob/master/README.md#protected-file-access: this is a permissions error",
		},
		{
			msg: "GetMessageIDs error",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, _ *mock_opsys.MockOS, _ []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths(nil),
					dbMock.EXPECT().GetChats(nil).Return([]chatdb.EntityChats{
						{
							Name: "testdisplayname",
							Chats: []chatdb.Chat{
								{
									ID:   1,
									GUID: "testguid",
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
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths(nil),
					dbMock.EXPECT().GetChats(nil).Return([]chatdb.EntityChats{
						{
							Name: "testdisplayname",
							Chats: []chatdb.Chat{
								{
									ID:   1,
									GUID: "testguid",
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
		{
			msg:           "separate chats - writeFile error",
			separateChats: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ []*mock_opsys.MockOutFile) {
				gomock.InOrder(
					dbMock.EXPECT().GetAttachmentPaths(nil),
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

			cnts := counts{
				attachments:         map[string]int{},
				attachmentsEmbedded: map[string]int{},
			}
			cfg := configuration{
				Options: Options{
					ExportPath:      "messages-export",
					SeparateChats:   tt.separateChats,
					OutputPDF:       tt.pdf,
					CopyAttachments: tt.copyAttachments,
					Entities:        tt.entities,
				},
				OS:     osMock,
				ChatDB: dbMock,
				counts: cnts,
			}
			err = cfg.exportChats(nil)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
		})
	}
}
