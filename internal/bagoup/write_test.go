// Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package bagoup

import (
	"os"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/chatdb"
	"github.com/tagatac/bagoup/chatdb/mock_chatdb"
	"github.com/tagatac/bagoup/opsys/mock_opsys"
	"gotest.tools/v3/assert"
)

func TestWriteFile(t *testing.T) {
	chatFile, err := afero.NewMemMapFs().Create("testfile")
	assert.NilError(t, err)

	tests := []struct {
		msg             string
		pdf             bool
		copyAttachments bool
		preservePaths   bool
		setupMocks      func(*mock_chatdb.MockChatDB, *mock_opsys.MockOS, *mock_opsys.MockOutFile)
		wantJPGs        int
		wantEmbedded    int
		wantConv        int
		wantConvFail    int
		wantErr         string
	}{
		{
			msg: "text export",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment1.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
				)
			},
			wantJPGs: 1,
		},
		{
			msg: "pdf export",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewPDFOutFile(chatFile, gomock.Any(), false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg").Return(true, nil),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2.jpeg").Return("attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
				)
			},
			wantJPGs:     2,
			wantEmbedded: 1,
			wantConv:     1,
		},
		{
			msg: "pdf export needs open files limit increase",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewPDFOutFile(chatFile, gomock.Any(), false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2.jpeg").Return("attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Flush().Return(500, nil),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
					osMock.EXPECT().SetOpenFilesLimit(1000),
				)
			},
			wantJPGs: 2,
			wantConv: 1,
		},
		{
			msg:             "copy attachments",
			copyAttachments: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					osMock.EXPECT().MkdirAll("messages-export/friend/attachments", os.ModePerm),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment1.heic", "messages-export/friend/attachments", true),
					ofMock.EXPECT().WriteAttachment("attachment1.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment2.jpeg", "messages-export/friend/attachments", true),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
				)
			},
			wantJPGs: 1,
		},
		{
			msg:             "copy attachments preserving paths",
			copyAttachments: true,
			preservePaths:   true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().MkdirAll("messages-export/bagoup-attachments", os.ModePerm),
					osMock.EXPECT().CopyFile("attachment1.heic", "messages-export/bagoup-attachments", false),
					ofMock.EXPECT().WriteAttachment("attachment1.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().MkdirAll("messages-export/bagoup-attachments", os.ModePerm),
					osMock.EXPECT().CopyFile("attachment2.jpeg", "messages-export/bagoup-attachments", false),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
				)
			},
			wantJPGs: 1,
		},
		{
			msg: "chat directory creation error",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm).Return(errors.New("this is a permissions error")),
				)
			},
			wantErr: `create directory "messages-export/friend": this is a permissions error`,
		},
		{
			msg: "pdf chat file creation error",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(nil, errors.New("this is a permissions error")),
				)
			},
			wantErr: `create file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf": this is a permissions error`,
		},
		{
			msg: "chat file creation error",
			setupMocks: func(_ *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(nil, errors.New("this is a permissions error")),
				)
			},
			wantErr: `create file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt": this is a permissions error`,
		},
		{
			msg:             "error creating attachments folder",
			copyAttachments: true,
			setupMocks: func(_ *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					osMock.EXPECT().MkdirAll("messages-export/friend/attachments", os.ModePerm).Return(errors.New("this is a permissions error")),
				)
			},
			wantErr: `create directory "messages-export/friend/attachments": this is a permissions error`,
		},
		{
			msg: "GetMessage error",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil).Return("", errors.New("this is a DB error")),
				)
			},
			wantErr: "get message with ID 2: this is a DB error",
		},
		{
			msg: "WriteMessage error",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2").Return(errors.New("this is an outfile error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
				)
			},
			wantErr: `write message "message2" to file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt": this is an outfile error`,
		},
		{
			msg: "Flush error",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewPDFOutFile(chatFile, gomock.Any(), false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2.jpeg").Return("attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Flush().Return(0, errors.New("this is a staging error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
				)
			},
			wantErr: `flush chat file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf" to disk: this is a staging error`,
		},
		{
			msg: "open files limit increase fails",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewPDFOutFile(chatFile, gomock.Any(), false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2.jpeg").Return("attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Flush().Return(500, nil),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
					osMock.EXPECT().SetOpenFilesLimit(1000).Return(errors.New("this is a syscall error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
				)
			},
			wantErr: `chat file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf" - increase the open file limit from 256 to 1000 to support 500 embedded images: this is a syscall error`,
		},
		{
			msg: "attachment file does not exist",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(false, nil),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					ofMock.EXPECT().ReferenceAttachment("att1transfer.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
				)
			},
			wantJPGs: 1,
		},
		{
			msg: "error referencing attachment",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(false, nil),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					ofMock.EXPECT().ReferenceAttachment("att1transfer.heic").Return(errors.New("this is a permissions error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
				)
			},
			wantErr: `chat file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf" - message 2: reference attachment "att1transfer.heic": this is a permissions error`,
		},
		{
			msg: "file existence check fails",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(false, errors.New("this is a permissions error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
				)
			},
			wantErr: `chat file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt" - message 2: check existence of file "attachment1.heic" - POSSIBLE FIX: https://github.com/tagatac/bagoup/blob/master/README.md#protected-file-access: this is a permissions error`,
		},
		{
			msg:             "error creating preserved path",
			copyAttachments: true,
			preservePaths:   true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().MkdirAll("messages-export/bagoup-attachments", os.ModePerm).Return(errors.New("this is a permissions error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
				)
			},
			wantErr: `chat file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt" - message 2: create directory "messages-export/bagoup-attachments": this is a permissions error`,
		},
		{
			msg:             "CopyFile error",
			copyAttachments: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					osMock.EXPECT().MkdirAll("messages-export/friend/attachments", os.ModePerm),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment1.heic", "messages-export/friend/attachments", true),
					ofMock.EXPECT().WriteAttachment("attachment1.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment2.jpeg", "messages-export/friend/attachments", true).Return(errors.New("this is a permissions error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
				)
			},
			wantErr: `chat file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt" - message 2: copy attachment "attachment2.jpeg" to "messages-export/friend/attachments": this is a permissions error`,
		},
		{
			msg: "HEIC conversion error",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewPDFOutFile(chatFile, gomock.Any(), false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1.heic").Return("attachment1.heic", errors.New("this is a goheif error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().WriteAttachment("attachment1.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2.jpeg").Return("attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Flush(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256),
				)
			},
			wantJPGs:     1,
			wantConvFail: 1,
		},
		{
			msg: "WriteAttachment error",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment1.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg").Return(false, errors.New("this is an outfile error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
				)
			},
			wantErr: `chat file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt" - message 2: include attachment "attachment2.jpeg": this is an outfile error`,
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

			cnts := counts{
				attachments:         map[string]int{},
				attachmentsCopied:   map[string]int{},
				attachmentsEmbedded: map[string]int{},
			}
			cfg := configuration{
				Options: Options{
					ExportPath:      "messages-export",
					OutputPDF:       tt.pdf,
					CopyAttachments: tt.copyAttachments,
					PreservePaths:   tt.preservePaths,
				},
				OS:           osMock,
				ChatDB:       dbMock,
				macOSVersion: semver.MustParse("12.4"),
				attachmentPaths: map[int][]chatdb.Attachment{
					2: {
						chatdb.Attachment{Filename: "attachment1.heic", MIMEType: "image/heic", TransferName: "att1transfer.heic"},
						chatdb.Attachment{Filename: "attachment2.jpeg", MIMEType: "image/jpeg", TransferName: "att2transfer.jpeg"},
						chatdb.Attachment{Filename: "", MIMEType: "image/png", TransferName: "att3transfer.png"},
					},
				},
				counts: cnts,
			}
			err := cfg.writeFile(
				"friend. ",
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
			assert.Equal(t, 2, cfg.counts.messages)
			assert.Equal(t, tt.wantJPGs, cfg.counts.attachments["image/jpeg"])
			assert.Equal(t, tt.wantEmbedded, cfg.counts.attachmentsEmbedded["image/jpeg"])
			assert.Equal(t, tt.wantConv, cfg.counts.conversions)
			assert.Equal(t, tt.wantConvFail, cfg.counts.conversionsFailed)
		})
	}

	t.Run("long email address", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		osMock := mock_opsys.NewMockOS(ctrl)
		ofMock := mock_opsys.NewMockOutFile(ctrl)
		gomock.InOrder(
			osMock.EXPECT().MkdirAll("friend", os.ModePerm),
			osMock.EXPECT().Create("friend/iMessage;-;heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress@gmail.c.txt").Return(chatFile, nil),
			osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
			ofMock.EXPECT().Flush(),
			osMock.EXPECT().GetOpenFilesLimit().Return(256),
		)

		cfg := configuration{OS: osMock}
		cfg.writeFile(
			"friend",
			[]string{"iMessage;-;heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress@gmail.com"},
			nil,
		)
	})
}
