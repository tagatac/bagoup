// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package bagoup

import (
	"errors"
	"os"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/v2/chatdb"
	"github.com/tagatac/bagoup/v2/chatdb/mock_chatdb"
	"github.com/tagatac/bagoup/v2/imgconv/mock_imgconv"
	"github.com/tagatac/bagoup/v2/opsys/mock_opsys"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
)

func TestPrepareFileJobs(t *testing.T) {
	pdfMsgs := make([]chatdb.DatedMessageID, 4000)
	for i := range pdfMsgs {
		pdfMsgs[i] = chatdb.DatedMessageID{ID: i, Date: i}
	}

	tests := []struct {
		msg             string
		exportPath      string
		copyAttachments bool
		pdf             bool
		entityName      string
		guids           []string
		messageIDs      []chatdb.DatedMessageID
		setupMocks      func(*mock_opsys.MockOS)
		wantErr         string
		wantJobs        []writeJob
	}{
		{
			msg:        "chat directory creation error",
			exportPath: "messages-export",
			entityName: "friend",
			guids:      []string{"iMessage;-;friend@gmail.com"},
			setupMocks: func(osMock *mock_opsys.MockOS) {
				osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm).Return(errors.New("this is a permissions error"))
			},
			wantErr: `create directory "messages-export/friend": this is a permissions error`,
		},
		{
			msg:             "error creating attachments folder",
			exportPath:      "messages-export",
			copyAttachments: true,
			entityName:      "friend",
			guids:           []string{"iMessage;-;friend@gmail.com"},
			setupMocks: func(osMock *mock_opsys.MockOS) {
				gomock.InOrder(
					osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm),
					osMock.EXPECT().MkdirAll("messages-export/friend/attachments", os.ModePerm).Return(errors.New("this is a permissions error")),
				)
			},
			wantErr: `create directory "messages-export/friend/attachments": this is a permissions error`,
		},
		{
			msg:        "long email address",
			entityName: "friend",
			guids:      []string{"iMessage;-;heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress@gmail.com"},
			setupMocks: func(osMock *mock_opsys.MockOS) {
				osMock.EXPECT().MkdirAll("friend", os.ModePerm)
			},
			wantJobs: []writeJob{
				{entityName: "friend", chatPath: "friend/iMessage;-;heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress.heresareallylongemailaddress@gmail.c.txt", attDir: "friend/attachments"},
			},
		},
		{
			msg:        "multiple PDF chunks",
			exportPath: "messages-export",
			pdf:        true,
			entityName: "friend",
			guids:      []string{"iMessage;-;friend@gmail.com"},
			messageIDs: pdfMsgs,
			setupMocks: func(osMock *mock_opsys.MockOS) {
				osMock.EXPECT().MkdirAll("messages-export/friend", os.ModePerm)
			},
			wantJobs: []writeJob{
				{entityName: "friend", chatPath: "messages-export/friend/iMessage;-;friend@gmail.com.1.pdf", messageIDs: pdfMsgs[:2048], attDir: "messages-export/friend/attachments"},
				{entityName: "friend", chatPath: "messages-export/friend/iMessage;-;friend@gmail.com.2.pdf", messageIDs: pdfMsgs[2048:], attDir: "messages-export/friend/attachments"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			osMock := mock_opsys.NewMockOS(ctrl)
			tt.setupMocks(osMock)

			cfg := configuration{
				Options: Options{
					ExportPath:      tt.exportPath,
					CopyAttachments: tt.copyAttachments,
					OutputPDF:       tt.pdf,
				},
				OS: osMock,
			}
			jobs, err := cfg.prepareFileJobs(tt.entityName, tt.guids, tt.messageIDs)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, jobs, tt.wantJobs, cmp.AllowUnexported(writeJob{}))
		})
	}
}

func TestWriteChunk(t *testing.T) {
	fileSys := afero.NewMemMapFs()
	chatFile, err := fileSys.Create("testfile")
	assert.NilError(t, err)

	// attachmentPaths shared by all write-path tests.
	attPaths := map[int][]chatdb.Attachment{
		2: {
			{Filename: "attachment1.heic", MIMEType: "image/heic", TransferName: "att1transfer.heic"},
			{Filename: "attachment2.jpeg", MIMEType: "image/jpeg", TransferName: "att2transfer.jpeg"},
			{Filename: "", MIMEType: "image/png", TransferName: "att3transfer.png"},
		},
	}
	baseJob := writeJob{
		entityName: "friend",
		chatPath:   "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt",
		messageIDs: []chatdb.DatedMessageID{{ID: 1, Date: 1}, {ID: 2, Date: 2}},
		attDir:     "messages-export/friend/attachments",
	}
	pdfJob := writeJob{
		entityName: "friend",
		chatPath:   "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf",
		messageIDs: []chatdb.DatedMessageID{{ID: 1, Date: 1}, {ID: 2, Date: 2}},
		attDir:     "messages-export/friend/attachments",
	}

	tests := []struct {
		msg             string
		job             writeJob
		pdf             bool
		wkhtml          bool
		copyAttachments bool
		preservePaths   bool
		setupMocks      func(*mock_chatdb.MockChatDB, *mock_opsys.MockOS, *mock_imgconv.MockImgConverter, *mock_opsys.MockOutFile)
		wantInvalid     int
		wantJPGs        int
		wantEmbedded    int
		wantConv        int
		wantConvFail    int
		wantErr         string
	}{
		{
			msg: "text export",
			job: baseJob,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
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
			job: pdfJob,
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, icMock *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg").Return(true, nil),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment2.jpeg").Return("attachment2.jpeg", nil),
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
			job:    pdfJob,
			pdf:    true,
			wkhtml: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, icMock *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWkhtmltopdfFile("friend", chatFile, gomock.Any(), false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg").Return(true, nil),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment2.jpeg").Return("attachment2.jpeg", nil),
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
			job: pdfJob,
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, icMock *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg").Return(true, nil),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment2.jpeg").Return("attachment2.jpeg", nil),
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
			job:             baseJob,
			copyAttachments: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
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
			job:             baseJob,
			copyAttachments: true,
			preservePaths:   true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
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
			job:             pdfJob,
			pdf:             true,
			copyAttachments: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, icMock *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment1.heic", "messages-export/friend/attachments", true).Return("messages-export/friend/attachments/attachment1.heic", nil),
					icMock.EXPECT().ConvertHEIC("messages-export/friend/attachments/attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg").Return(true, nil),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					osMock.EXPECT().CopyFile("attachment2.jpeg", "messages-export/friend/attachments", true).Return("messages-export/friend/attachments/attachment2.jpeg", nil),
					icMock.EXPECT().ConvertHEIC("messages-export/friend/attachments/attachment2.jpeg").Return("messages-export/friend/attachments/attachment2.jpeg", nil),
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
			msg: "pdf chat file creation error",
			job: pdfJob,
			pdf: true,
			setupMocks: func(_ *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_imgconv.MockImgConverter, _ *mock_opsys.MockOutFile) {
				osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(nil, errors.New("this is a permissions error"))
			},
			wantErr: `create file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf": this is a permissions error`,
		},
		{
			msg: "chat file creation error",
			job: baseJob,
			setupMocks: func(_ *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_imgconv.MockImgConverter, _ *mock_opsys.MockOutFile) {
				osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt").Return(nil, errors.New("this is a permissions error"))
			},
			wantErr: `create file "messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.txt": this is a permissions error`,
		},
		{
			msg: "GetMessage error",
			job: baseJob,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
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
			job: baseJob,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
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
			job: pdfJob,
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, icMock *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment2.jpeg").Return("attachment2.jpeg", nil),
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
			job: pdfJob,
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, icMock *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment2.jpeg").Return("attachment2.jpeg", nil),
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
			job: pdfJob,
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, icMock *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment2.jpeg").Return("attachment2.jpeg", nil),
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
			job: pdfJob,
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, icMock *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment1.heic").Return("attachment1.jpeg", nil),
					ofMock.EXPECT().WriteAttachment("attachment1.jpeg"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment2.jpeg").Return("attachment2.jpeg", nil),
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
			job: baseJob,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
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
			job: baseJob,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
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
			job: baseJob,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
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
			job:             baseJob,
			copyAttachments: true,
			preservePaths:   true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
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
			job:             baseJob,
			copyAttachments: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
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
			job: pdfJob,
			pdf: true,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, icMock *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
					osMock.EXPECT().Create("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf").Return(chatFile, nil),
					osMock.EXPECT().NewWeasyPrintFile("friend", chatFile, false).Return(ofMock),
					dbMock.EXPECT().GetMessage(1, nil).Return("message1", true, nil),
					ofMock.EXPECT().WriteMessage("message1"),
					dbMock.EXPECT().GetMessage(2, nil).Return("message2", true, nil),
					ofMock.EXPECT().WriteMessage("message2"),
					osMock.EXPECT().FileExist("attachment1.heic").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment1.heic").Return("attachment1.heic", errors.New("this is a goheif error")),
					ofMock.EXPECT().Name().Return("messages-export/friend/iMessage;-;friend@gmail.com;;;iMessage;-;friend@hotmail.com.pdf"),
					ofMock.EXPECT().WriteAttachment("attachment1.heic"),
					osMock.EXPECT().FileExist("attachment2.jpeg").Return(true, nil),
					icMock.EXPECT().ConvertHEIC("attachment2.jpeg").Return("attachment2.jpeg", nil),
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
			job: baseJob,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
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
			job: baseJob,
			setupMocks: func(dbMock *mock_chatdb.MockChatDB, osMock *mock_opsys.MockOS, _ *mock_imgconv.MockImgConverter, ofMock *mock_opsys.MockOutFile) {
				gomock.InOrder(
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
			icMock := mock_imgconv.NewMockImgConverter(ctrl)
			ofMock := mock_opsys.NewMockOutFile(ctrl)
			tt.setupMocks(dbMock, osMock, icMock, ofMock)

			cfg := configuration{
				Options: Options{
					ExportPath:      "messages-export",
					OutputPDF:       tt.pdf,
					UseWkhtmltopdf:  tt.wkhtml,
					CopyAttachments: tt.copyAttachments,
					PreservePaths:   tt.preservePaths,
				},
				OS:              osMock,
				ChatDB:          dbMock,
				ImgConverter:    icMock,
				macOSVersion:    semver.MustParse("12.4"),
				attachmentPaths: attPaths,
			}
			c := newCounts()
			err := cfg.writeChunk(tt.job, c)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, c.messages, 2-tt.wantInvalid)
			assert.Equal(t, c.messagesInvalid, tt.wantInvalid)
			assert.Equal(t, c.attachments["image/jpeg"], tt.wantJPGs)
			assert.Equal(t, c.attachmentsEmbedded["image/jpeg"], tt.wantEmbedded)
			assert.Equal(t, c.conversions, tt.wantConv)
			assert.Equal(t, c.conversionsFailed, tt.wantConvFail)
		})
	}
}

