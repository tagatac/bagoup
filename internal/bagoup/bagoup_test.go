// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package bagoup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/v2/chatdb/mock_chatdb"
	"github.com/tagatac/bagoup/v2/opsys/mock_opsys"
	"github.com/tagatac/bagoup/v2/pathtools/mock_pathtools"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
)

func TestBagoup(t *testing.T) {
	wd, err := os.Getwd()
	assert.NilError(t, err)
	defaultOpts := Options{
		DBPath:          "~/Library/Messages/chat.db",
		ExportPath:      "messages-export",
		SelfHandle:      "Me",
		AttachmentsPath: "/",
		Timezone:        "Local",
	}
	exportPathAbs := filepath.Join(wd, "messages-export")
	logDirAbs := filepath.Join(exportPathAbs, ".bagoup")
	logFileAbs := filepath.Join(logDirAbs, "out.log")
	tildeexpansionAbs := filepath.Join(exportPathAbs, PreservedPathDir, PreservedPathTildeExpansionFile)
	tenDotTwelve := "10.12"
	tenDotTenDotTenDotTen := "10.10.10.10"
	contactsPath := "contacts.vcf"
	devnull, err := os.Open(os.DevNull)
	assert.NilError(t, err)

	tests := []struct {
		msg        string
		opts       Options
		setupMocks func(*mock_opsys.MockOS, *mock_chatdb.MockChatDB, *mock_pathtools.MockPathTools)
		wantCfgErr string
		wantErr    string
	}{
		{
			msg:  "default options",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, ptMock *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					dbMock.EXPECT().Init(semver.MustParse("12.4"), time.Local),
					dbMock.EXPECT().GetHandleMap(nil),
					dbMock.EXPECT().GetAttachmentPaths(ptMock),
					dbMock.EXPECT().GetChats(nil),
					osMock.EXPECT().RmTempDir(),
				)
			},
		},
		{
			msg: "relative attachments path",
			opts: Options{
				DBPath:          "~/Library/Messages/chat.db",
				ExportPath:      "messages-export",
				SelfHandle:      "Me",
				AttachmentsPath: "testrelativepath",
				Timezone:        "Local",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, ptMock *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().ReadFile("testrelativepath/.tildeexpansion"),
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					dbMock.EXPECT().Init(semver.MustParse("12.4"), time.Local),
					dbMock.EXPECT().GetHandleMap(nil),
					dbMock.EXPECT().GetAttachmentPaths(ptMock),
					dbMock.EXPECT().GetChats(nil),
					osMock.EXPECT().RmTempDir(),
				)
			},
		},
		{
			msg: "invalid timezone",
			opts: Options{
				DBPath:          "~/Library/Messages/chat.db",
				ExportPath:      "messages-export",
				SelfHandle:      "Me",
				AttachmentsPath: "/",
				Timezone:        "NotATimezone",
			},
			wantCfgErr: `load timezone "NotATimezone": unknown time zone NotATimezone`,
		},
		{
			msg: "error reading tilde expansion file",
			opts: Options{
				DBPath:          "~/Library/Messages/chat.db",
				ExportPath:      "messages-export",
				SelfHandle:      "Me",
				AttachmentsPath: "testrelativepath",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, ptMock *mock_pathtools.MockPathTools) {
				osMock.EXPECT().ReadFile("testrelativepath/.tildeexpansion").Return("", errors.New("this is a file permissions error"))
			},
			wantCfgErr: `read tilde expansion file "testrelativepath/.tildeexpansion" - POSSIBLE FIX: create a file .tildeexpansion with the expanded home directory from the previous run and place it at the root of the preserved-paths copied attachments directory (usually "bagoup-attachments"): this is a file permissions error`,
		},
		{
			msg:  "missing chatDB read permissions",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, _ *mock_chatdb.MockChatDB, _ *mock_pathtools.MockPathTools) {
				osMock.EXPECT().FileAccess("~/Library/Messages/chat.db").Return(errors.New("this is a permissions error"))
			},
			wantErr: `test DB file "~/Library/Messages/chat.db" - FIX: https://github.com/tagatac/bagoup/blob/master/README.md#protected-file-access: this is a permissions error`,
		},
		{
			msg:  "running on Windows",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, _ *mock_chatdb.MockChatDB, _ *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					osMock.EXPECT().GetMacOSVersion().Return(nil, errors.New("this is an exec error")),
				)
			},
			wantErr: "get macOS version - FIX: specify the macOS version from which chat.db was copied with the --mac-os-version option: this is an exec error",
		},
		{
			msg:  "export path exists",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, _ *mock_chatdb.MockChatDB, _ *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs).Return(true, nil),
				)
			},
			wantErr: fmt.Sprintf(`export folder %q already exists - FIX: move it or specify a different export path with the --export-path option`, exportPathAbs),
		},
		{
			msg:  "error checking export path",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, _ *mock_chatdb.MockChatDB, _ *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs).Return(false, errors.New("this is a stat error")),
				)
			},
			wantErr: fmt.Sprintf(`check export path %q: this is a stat error`, exportPathAbs),
		},
		{
			msg:  "error creating log directory",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, _ *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm).Return(errors.New("this is a permissions error")),
				)
			},
			wantErr: "make log directory: this is a permissions error",
		},
		{
			msg:  "error creating log file",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, _ *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, errors.New("this is a permissions error")),
				)
			},
			wantErr: "create log file: this is a permissions error",
		},
		{
			msg: "chat.db version specified",
			opts: Options{
				DBPath:          "~/Library/Messages/chat.db",
				ExportPath:      "messages-export",
				MacOSVersion:    &tenDotTwelve,
				SelfHandle:      "Me",
				AttachmentsPath: "/",
				Timezone:        "Local",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, ptMock *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					dbMock.EXPECT().Init(semver.MustParse("10.12"), time.Local),
					dbMock.EXPECT().GetHandleMap(nil),
					dbMock.EXPECT().GetAttachmentPaths(ptMock),
					dbMock.EXPECT().GetChats(nil),
					osMock.EXPECT().RmTempDir(),
				)
			},
		},
		{
			msg: "invalid chat.db version specified",
			opts: Options{
				DBPath:          "~/Library/Messages/chat.db",
				ExportPath:      "messages-export",
				MacOSVersion:    &tenDotTenDotTenDotTen,
				SelfHandle:      "Me",
				AttachmentsPath: "/",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, _ *mock_chatdb.MockChatDB, _ *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
				)
			},
			wantErr: `parse macOS version "10.10.10.10": invalid semantic version`,
		},
		{
			msg: "contacts file specified",
			opts: Options{
				DBPath:          "~/Library/Messages/chat.db",
				ExportPath:      "messages-export",
				ContactsPath:    &contactsPath,
				SelfHandle:      "Me",
				AttachmentsPath: "/",
				Timezone:        "Local",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, ptMock *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					osMock.EXPECT().GetContactMap("contacts.vcf"),
					dbMock.EXPECT().Init(semver.MustParse("12.4"), time.Local),
					dbMock.EXPECT().GetHandleMap(nil),
					dbMock.EXPECT().GetAttachmentPaths(ptMock),
					dbMock.EXPECT().GetChats(nil),
					osMock.EXPECT().RmTempDir(),
				)
			},
		},
		{
			msg: "error getting contact map",
			opts: Options{
				DBPath:          "~/Library/Messages/chat.db",
				ExportPath:      "messages-export",
				ContactsPath:    &contactsPath,
				SelfHandle:      "Me",
				AttachmentsPath: "/",
				Timezone:        "Local",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, _ *mock_chatdb.MockChatDB, _ *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					osMock.EXPECT().GetContactMap("contacts.vcf").Return(nil, errors.New("this is an os error")),
				)
			},
			wantErr: `get contacts from vcard file "contacts.vcf": this is an os error`,
		},
		{
			msg:  "error initializing chat DB",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, _ *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					dbMock.EXPECT().Init(semver.MustParse("12.4"), time.Local).Return(errors.New("this is a DB error")),
				)
			},
			wantErr: "initialize the database for reading on macOS version 12.4.0: this is a DB error",
		},
		{
			msg:  "error getting handle map",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, _ *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					dbMock.EXPECT().Init(semver.MustParse("12.4"), time.Local),
					dbMock.EXPECT().GetHandleMap(nil).Return(nil, errors.New("this is a DB error")),
				)
			},
			wantErr: "get handle map: this is a DB error",
		},
		{
			msg: "pdf output",
			opts: Options{
				DBPath:          "~/Library/Messages/chat.db",
				ExportPath:      "messages-export",
				SelfHandle:      "Me",
				OutputPDF:       true,
				AttachmentsPath: "/",
				Timezone:        "Local",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, ptMock *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					dbMock.EXPECT().Init(semver.MustParse("12.4"), time.Local),
					dbMock.EXPECT().GetHandleMap(nil),
					osMock.EXPECT().GetTempDir(),
					dbMock.EXPECT().GetAttachmentPaths(ptMock),
					dbMock.EXPECT().GetChats(nil),
					osMock.EXPECT().RmTempDir().Times(2),
				)
			},
		},
		{
			msg: "error getting temp dir",
			opts: Options{
				DBPath:          "~/Library/Messages/chat.db",
				ExportPath:      "messages-export",
				SelfHandle:      "Me",
				OutputPDF:       true,
				AttachmentsPath: "/",
				Timezone:        "Local",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, ptMock *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					dbMock.EXPECT().Init(semver.MustParse("12.4"), time.Local),
					dbMock.EXPECT().GetHandleMap(nil),
					osMock.EXPECT().GetTempDir().Return("", errors.New("this is a tempdir error")),
				)
			},
			wantErr: "get temporary directory: this is a tempdir error",
		},
		{
			msg:  "export chats error",
			opts: defaultOpts,
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, ptMock *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					dbMock.EXPECT().Init(semver.MustParse("12.4"), time.Local),
					dbMock.EXPECT().GetHandleMap(nil),
					dbMock.EXPECT().GetAttachmentPaths(ptMock),
					dbMock.EXPECT().GetChats(nil).Return(nil, errors.New("this is a DB error")),
				)
			},
			wantErr: "export chats: get chats: this is a DB error",
		},
		{
			msg: "copy attachments with preserved paths",
			opts: Options{
				DBPath:          "~/Library/Messages/chat.db",
				ExportPath:      "messages-export",
				SelfHandle:      "Me",
				AttachmentsPath: "/",
				CopyAttachments: true,
				PreservePaths:   true,
				Timezone:        "Local",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, ptMock *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					dbMock.EXPECT().Init(semver.MustParse("12.4"), time.Local),
					dbMock.EXPECT().GetHandleMap(nil),
					dbMock.EXPECT().GetAttachmentPaths(ptMock),
					dbMock.EXPECT().GetChats(nil),
					ptMock.EXPECT().GetHomeDir(),
					osMock.EXPECT().Create(tildeexpansionAbs).Return(afero.NewMemMapFs().Create("dummy")),
					osMock.EXPECT().RmTempDir(),
				)
			},
		},
		{
			msg: "error creating tilde expansion file",
			opts: Options{
				DBPath:          "~/Library/Messages/chat.db",
				ExportPath:      "messages-export",
				SelfHandle:      "Me",
				AttachmentsPath: "/",
				CopyAttachments: true,
				PreservePaths:   true,
				Timezone:        "Local",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, ptMock *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					dbMock.EXPECT().Init(semver.MustParse("12.4"), time.Local),
					dbMock.EXPECT().GetHandleMap(nil),
					dbMock.EXPECT().GetAttachmentPaths(ptMock),
					dbMock.EXPECT().GetChats(nil),
					ptMock.EXPECT().GetHomeDir(),
					osMock.EXPECT().Create(tildeexpansionAbs).Return(nil, errors.New("this is a permissions error")),
				)
			},
			wantErr: "write out tilde expansion file: this is a permissions error",
		},
		{
			msg: "error writing to tilde expansion file",
			opts: Options{
				DBPath:          "~/Library/Messages/chat.db",
				ExportPath:      "messages-export",
				SelfHandle:      "Me",
				AttachmentsPath: "/",
				CopyAttachments: true,
				PreservePaths:   true,
				Timezone:        "Local",
			},
			setupMocks: func(osMock *mock_opsys.MockOS, dbMock *mock_chatdb.MockChatDB, ptMock *mock_pathtools.MockPathTools) {
				rwfs := afero.NewMemMapFs()
				_, err := rwfs.Create("dummy")
				assert.NilError(t, err)
				rofs := afero.NewReadOnlyFs(rwfs)
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					osMock.EXPECT().GetMacOSVersion().Return(semver.MustParse("12.4"), nil),
					dbMock.EXPECT().Init(semver.MustParse("12.4"), time.Local),
					dbMock.EXPECT().GetHandleMap(nil),
					dbMock.EXPECT().GetAttachmentPaths(ptMock),
					dbMock.EXPECT().GetChats(nil),
					ptMock.EXPECT().GetHomeDir(),
					osMock.EXPECT().Create(tildeexpansionAbs).Return(rofs.Open("dummy")),
				)
			},
			wantErr: "write out tilde expansion file: write dummy: file handle is read only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			osMock := mock_opsys.NewMockOS(ctrl)
			dbMock := mock_chatdb.NewMockChatDB(ctrl)
			ptMock := mock_pathtools.NewMockPathTools(ctrl)
			if tt.setupMocks != nil {
				tt.setupMocks(osMock, dbMock, ptMock)
			}

			cfg, err := NewConfiguration(
				tt.opts,
				osMock,
				dbMock,
				ptMock,
				logDirAbs,
				time.Now(),
				"",
			)
			if tt.wantCfgErr != "" {
				assert.Error(t, err, tt.wantCfgErr)
				return
			}
			assert.NilError(t, err)
			cfg.(*configuration).PathTools = ptMock
			cfg.(*configuration).counts.attachments["image/jpeg"] = 1
			attPathIn := tt.opts.AttachmentsPath
			err = cfg.Run()
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			if !strings.HasPrefix(attPathIn, "/") {
				assert.Equal(t, cfg.(*configuration).Options.AttachmentsPath, filepath.Join(wd, attPathIn))
			}
		})
	}
}
