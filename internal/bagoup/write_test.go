// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package bagoup

import (
	"fmt"
	"os"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/v2/chatdb"
	"github.com/tagatac/bagoup/v2/chatdb/mock_chatdb"
	"github.com/tagatac/bagoup/v2/opsys/mock_opsys"
	"gotest.tools/v3/assert"
)

func TestWriteFile(t *testing.T) {
	fileSys := afero.NewMemMapFs()
	chatFile, err := fileSys.Create("testfile")
	assert.NilError(t, err)

	tests := []struct {
		msg             string
		pdf             bool
		wkhtml          bool
		copyAttachments bool
		preservePaths   bool
		setupMocks      func(*mock_chatdb.MockChatDB, *mock_opsys.MockOS, *mock_opsys.MockOutFile)
		wantInvalid     int
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
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment1.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Stage(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
					ofMock.EXPECT().Flush(),
				)
			},
			wantJPGs: 1,
		},
		{
			msg: "WeasyPrint pdf export",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg").Return(true, nil),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2.jpeg").Return("attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg").Return(true, nil),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Stage(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
					ofMock.EXPECT().Flush(),
				)
			},
			wantJPGs:     2,
			wantEmbedded: 2,
			wantConv:     1,
		},
		{
			msg:    "wkhtmltopdf export",
			pdf:    true,
			wkhtml: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWkhtmltopdfFile("friend", chatFile, gomock.Any(), false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg").Return(true, nil),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2.jpeg").Return("attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg").Return(true, nil),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Stage(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
					ofMock.EXPECT().Flush(),
				)
			},
			wantJPGs:     2,
			wantEmbedded: 2,
			wantConv:     1,
		},
		{
			msg: "pdf export needs open files limit increase",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg").Return(true, nil),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2.jpeg").Return("attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg").Return(true, nil),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Stage().Return(500, nil),
					osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
					osMock.EXPECT().SetOpenFilesLimit(1000),
					ofMock.EXPECT().Flush(),
				)
			},
			wantJPGs:     2,
			wantEmbedded: 2,
			wantConv:     1,
		},
		{
			msg:             "copy attachments",
			copyAttachments: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().MkdirAll("messages-export/friend/attachments", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment1.heic", "messages-export/friend/attachments", true).Return("messages-export/friend/attachments/attachment1.heic", nil),
					ofMock.EXPECT().WriteAttachment("messages-export/friend/attachments/attachment1.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment2.jpeg", "messages-export/friend/attachments", true).Return("messages-export/friend/attachments/attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("messages-export/friend/attachments/attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Stage(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
					ofMock.EXPECT().Flush(),
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
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().MkdirAll("messages-export/bagoup-attachments", os.ModePerm),
					osMock.EXPECT().CopyFile("attachment1.heic", "messages-export/bagoup-attachments", false).Return("messages-export/bagoup-attachments/attachment1.heic", nil),
					ofMock.EXPECT().WriteAttachment("messages-export/bagoup-attachments/attachment1.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().MkdirAll("messages-export/bagoup-attachments", os.ModePerm),
					osMock.EXPECT().CopyFile("attachment2.jpeg", "messages-export/bagoup-attachments", false).Return("messages-export/bagoup-attachments/attachment2-1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("messages-export/bagoup-attachments/attachment2-1.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Stage(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
					ofMock.EXPECT().Flush(),
				)
			},
			wantJPGs: 1,
		},
		{
			msg:             "copy attachments and pdf export",
			copyAttachments: true,
			pdf:             true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().MkdirAll("messages-export/friend/attachments", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment1.heic", "messages-export/friend/attachments", true).Return("messages-export/friend/attachments/attachment1.heic", nil),
					osMock.EXPECT().HEIC2JPG("messages-export/friend/attachments/attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg").Return(true, nil),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment2.jpeg", "messages-export/friend/attachments", true).Return("messages-export/friend/attachments/attachment2.jpeg", nil),
					osMock.EXPECT().HEIC2JPG("messages-export/friend/attachments/attachment2.jpeg").Return("messages-export/friend/attachments/attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("messages-export/friend/attachments/attachment2.jpeg").Return(true, nil),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Stage(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
					ofMock.EXPECT().Flush(),
				)
			},
			wantJPGs:     2,
			wantEmbedded: 2,
			wantConv:     1,
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
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("", false, errors.New("this is a DB error")),
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
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2").Return(errors.New("this is an outfile error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
				)
			},
			wantErr: `write message "message2" to file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt": this is an outfile error`,
		},
		{
			msg: "Staging error",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2.jpeg").Return("attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Stage().Return(0, errors.New("this is a staging error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
				)
			},
			wantErr: `stage chat file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf" for writing: this is a staging error`,
		},
		{
			msg: "get open files limit fails",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2.jpeg").Return("attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Stage().Return(500, nil),
					osMock.EXPECT().GetOpenFilesLimit().Return(0, errors.New("this is a ulimit error")),
				)
			},
			wantErr: `this is a ulimit error`,
		},
		{
			msg: "open files limit increase fails",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2.jpeg").Return("attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Stage().Return(500, nil),
					osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
					osMock.EXPECT().SetOpenFilesLimit(1000).Return(errors.New("this is a syscall error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
				)
			},
			wantErr: `chat file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf" - increase the open file limit from 256 to 1000 to support 500 embedded images: this is a syscall error`,
		},
		{
			msg: "Flush error",
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2.jpeg").Return("attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Stage(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
					ofMock.EXPECT().Flush().Return(errors.New("this is a flush error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
				)
			},
			wantErr: `flush chat file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf" to disk: this is a flush error`,
		},
		{
			msg: "attachment file does not exist",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(false, nil),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					ofMock.EXPECT().ReferenceAttachment("att1transfer.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Stage(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
					ofMock.EXPECT().Flush(),
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
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
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
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
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
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
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
					osMock.EXPECT().MkdirAll("messages-export/friend/attachments", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment1.heic", "messages-export/friend/attachments", true).Return("messages-export/friend/attachments/attachment1.heic", nil),
					ofMock.EXPECT().WriteAttachment("messages-export/friend/attachments/attachment1.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment2.jpeg", "messages-export/friend/attachments", true).Return("", errors.New("this is a permissions error")),
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
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment1.heic").Return("attachment1.heic", errors.New("this is a goheif error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().WriteAttachment("attachment1.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().HEIC2JPG("attachment2.jpeg").Return("attachment2.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg").Return(true, nil),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Stage(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
					ofMock.EXPECT().Flush(),
				)
			},
			wantJPGs:     1,
			wantEmbedded: 1,
			wantConvFail: 1,
		},
		{
			msg: "WriteAttachment error",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
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
		{
			msg: "1 message invalid",
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(chatFile, nil),
					osMock.EXPECT().NewTxtOutFile(chatFile).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("", false, nil),
					ofMock.EXPECT().WriteMessage(""),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment1.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					ofMock.EXPECT().WriteAttachment("attachment2.jpeg"),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt"),
					ofMock.EXPECT().ReferenceAttachment("att3transfer.png"),
					ofMock.EXPECT().Stage(),
					osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
					ofMock.EXPECT().Flush(),
				)
			},
			wantInvalid: 1,
			wantJPGs:    1,
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
					UseWkhtmltopdf:  tt.wkhtml,
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
			assert.Equal(t, cfg.counts.messages, 2-tt.wantInvalid)
			assert.Equal(t, cfg.counts.messagesInvalid, tt.wantInvalid)
			assert.Equal(t, cfg.counts.attachments["image/jpeg"], tt.wantJPGs)
			assert.Equal(t, cfg.counts.attachmentsEmbedded["image/jpeg"], tt.wantEmbedded)
			assert.Equal(t, cfg.counts.conversions, tt.wantConv)
			assert.Equal(t, cfg.counts.conversionsFailed, tt.wantConvFail)
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
			ofMock.EXPECT().Stage(),
			osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
			ofMock.EXPECT().Flush(),
		)

		cfg := configuration{OS: osMock}
		cfg.writeFile(
			"friend",
			[]string{"iMessage;-;heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress@gmail.com"},
			nil,
		)
	})

	t.Run("multiple PDF files", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		dbMock := mock_chatdb.NewMockChatDB(ctrl)
		osMock := mock_opsys.NewMockOS(ctrl)
		chatFile1, err := fileSys.Create("testfile1")
		assert.NilError(t, err)
		ofMock1 := mock_opsys.NewMockOutFile(ctrl)
		chatFile2, err := fileSys.Create("testfile2")
		assert.NilError(t, err)
		ofMock2 := mock_opsys.NewMockOutFile(ctrl)

		msgs := []chatdb.DatedMessageID{}
		mockCalls := []*gomock.Call{
			osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
			osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com.1.pdf").Return(chatFile1, nil),
			osMock.EXPECT().NewWeasyPrintFile("friend", chatFile1, false).Return(ofMock1),
		}
		for i := 0; i < 2048; i++ {
			msgs = append(msgs, chatdb.DatedMessageID{ID: i, Date: i})
			msg := fmt.Sprintf("message%d", i)
			mockCalls = append(
				mockCalls,
				dbMock.EXPECT().GetMessage(i, nil).Return(msg, true, nil),
				ofMock1.EXPECT().WriteMessage(msg),
			)
		}
		mockCalls = append(
			mockCalls,
			ofMock1.EXPECT().Stage(),
			osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
			ofMock1.EXPECT().Flush(),
			osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com.2.pdf").Return(chatFile2, nil),
			osMock.EXPECT().NewWeasyPrintFile("friend", chatFile2, false).Return(ofMock2),
		)
		for i := 2048; i < 4000; i++ {
			msgs = append(msgs, chatdb.DatedMessageID{ID: i, Date: i})
			msg := fmt.Sprintf("message%d", i)
			mockCalls = append(
				mockCalls,
				dbMock.EXPECT().GetMessage(i, nil).Return(msg, true, nil),
				ofMock2.EXPECT().WriteMessage(msg),
			)
		}
		mockCalls = append(
			mockCalls,
			ofMock2.EXPECT().Stage(),
			osMock.EXPECT().GetOpenFilesLimit().Return(256, nil),
			ofMock2.EXPECT().Flush(),
		)
		gomock.InOrder(mockCalls...)

		cnts := counts{
			attachments:         map[string]int{},
			attachmentsCopied:   map[string]int{},
			attachmentsEmbedded: map[string]int{},
		}
		cfg := configuration{
			Options: Options{
				ExportPath: "messages-export",
				OutputPDF:  true,
			},
			OS:              osMock,
			ChatDB:          dbMock,
			attachmentPaths: map[int][]chatdb.Attachment{},
			counts:          cnts,
		}
		err = cfg.writeFile(
			"friend",
			[]string{"iMessage;-;friend@gmail.com"},
			msgs,
		)
		assert.NilError(t, err)
		assert.Equal(t, cfg.counts.messages, 4000)
		assert.Equal(t, cfg.counts.messagesInvalid, 0)
		assert.Equal(t, len(cfg.counts.attachments), 0)
		assert.Equal(t, len(cfg.counts.attachmentsEmbedded), 0)
		assert.Equal(t, cfg.counts.conversions, 0)
		assert.Equal(t, cfg.counts.conversionsFailed, 0)
	})
}
