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
	"github.com/google/go-cmp/cmp"
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
					osMock.EXPECT().RmTempDir(),
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
		{
			msg: "start profiling error",
			opts: Options{
				DBPath:          "~/Library/Messages/chat.db",
				ExportPath:      "messages-export",
				SelfHandle:      "Me",
				AttachmentsPath: "/",
				Timezone:        "Local",
				Profiling:       profiling{Trace: "trace.out"},
			},
			setupMocks: func(osMock *mock_opsys.MockOS, _ *mock_chatdb.MockChatDB, _ *mock_pathtools.MockPathTools) {
				gomock.InOrder(
					osMock.EXPECT().FileAccess("~/Library/Messages/chat.db"),
					osMock.EXPECT().FileExist(exportPathAbs),
					osMock.EXPECT().MkdirAll(logDirAbs, os.ModePerm),
					osMock.EXPECT().Create(logFileAbs).Return(devnull, nil),
					osMock.EXPECT().Create("trace.out").Return(nil, errors.New("perm error")),
				)
			},
			wantErr: "create trace file: perm error",
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

func TestMergeCounts(t *testing.T) {
	tests := []struct {
		msg        string
		base       *counts
		incoming   *counts
		wantCounts counts
	}{
		{
			msg:  "merge into empty",
			base: newCounts(),
			incoming: &counts{
				files:               2,
				chats:               3,
				messages:            10,
				messagesInvalid:     1,
				attachmentsMissing:  2,
				conversions:         4,
				conversionsFailed:   1,
				attachments:         map[string]int{"image/jpeg": 5},
				attachmentsCopied:   map[string]int{"image/jpeg": 3},
				attachmentsEmbedded: map[string]int{"image/jpeg": 2},
			},
			wantCounts: counts{
				files:               2,
				chats:               3,
				messages:            10,
				messagesInvalid:     1,
				attachmentsMissing:  2,
				conversions:         4,
				conversionsFailed:   1,
				attachments:         map[string]int{"image/jpeg": 5},
				attachmentsCopied:   map[string]int{"image/jpeg": 3},
				attachmentsEmbedded: map[string]int{"image/jpeg": 2},
			},
		},
		{
			msg: "accumulate existing map keys",
			base: &counts{
				files:               1,
				messages:            5,
				attachments:         map[string]int{"image/jpeg": 3},
				attachmentsCopied:   map[string]int{"image/jpeg": 1},
				attachmentsEmbedded: map[string]int{"image/jpeg": 1},
			},
			incoming: &counts{
				files:               2,
				messages:            7,
				attachments:         map[string]int{"image/jpeg": 4},
				attachmentsCopied:   map[string]int{"image/jpeg": 2},
				attachmentsEmbedded: map[string]int{"image/jpeg": 2},
			},
			wantCounts: counts{
				files:               3,
				messages:            12,
				attachments:         map[string]int{"image/jpeg": 7},
				attachmentsCopied:   map[string]int{"image/jpeg": 3},
				attachmentsEmbedded: map[string]int{"image/jpeg": 3},
			},
		},
		{
			msg: "add new map keys",
			base: &counts{
				attachments:         map[string]int{"image/jpeg": 1},
				attachmentsCopied:   map[string]int{},
				attachmentsEmbedded: map[string]int{},
			},
			incoming: &counts{
				attachments:         map[string]int{"image/heic": 2},
				attachmentsCopied:   map[string]int{"image/heic": 1},
				attachmentsEmbedded: map[string]int{"image/heic": 1},
			},
			wantCounts: counts{
				attachments:         map[string]int{"image/jpeg": 1, "image/heic": 2},
				attachmentsCopied:   map[string]int{"image/heic": 1},
				attachmentsEmbedded: map[string]int{"image/heic": 1},
			},
		},
		{
			msg:        "merge zero counts",
			base:       &counts{files: 5, messages: 10, attachments: map[string]int{"image/jpeg": 3}, attachmentsCopied: map[string]int{}, attachmentsEmbedded: map[string]int{}},
			incoming:   newCounts(),
			wantCounts: counts{files: 5, messages: 10, attachments: map[string]int{"image/jpeg": 3}, attachmentsCopied: map[string]int{}, attachmentsEmbedded: map[string]int{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			cfg := &configuration{counts: tt.base}
			cfg.mergeCounts(tt.incoming)
			assert.DeepEqual(t, *cfg.counts, tt.wantCounts, cmp.AllowUnexported(counts{}))
		})
	}
}

func TestStartProfiling(t *testing.T) {
	traceFile, err := os.CreateTemp(t.TempDir(), "trace*.out")
	assert.NilError(t, err)
	memFile, err := os.CreateTemp(t.TempDir(), "mem*.prof")
	assert.NilError(t, err)

	tests := []struct {
		msg        string
		setupCfg   func(*configuration)
		setupMocks func(*mock_opsys.MockOS)
		callStop   bool
		wantErr    string
	}{
		{
			msg:        "no profiling",
			setupCfg:   func(*configuration) {},
			setupMocks: func(*mock_opsys.MockOS) {},
			callStop:   true,
		},
		{
			msg: "trace file creation error",
			setupCfg: func(cfg *configuration) {
				cfg.Options.Profiling.Trace = "trace.out"
			},
			setupMocks: func(osMock *mock_opsys.MockOS) {
				osMock.EXPECT().Create("trace.out").Return(nil, errors.New("perm error"))
			},
			wantErr: "create trace file: perm error",
		},
		{
			msg: "cpu profile file creation error",
			setupCfg: func(cfg *configuration) {
				cfg.Options.Profiling.CPUProfile = "cpu.prof"
			},
			setupMocks: func(osMock *mock_opsys.MockOS) {
				osMock.EXPECT().Create("cpu.prof").Return(nil, errors.New("perm error"))
			},
			wantErr: "create CPU profile: perm error",
		},
		{
			msg: "trace + mem profile success",
			setupCfg: func(cfg *configuration) {
				cfg.Options.Profiling.Trace = "trace.out"
				cfg.Options.Profiling.MemProfile = "mem.prof"
			},
			setupMocks: func(osMock *mock_opsys.MockOS) {
				gomock.InOrder(
					osMock.EXPECT().Create("trace.out").Return(traceFile, nil),
					osMock.EXPECT().Create("mem.prof").Return(memFile, nil),
				)
			},
			callStop: true,
		},
		{
			msg: "mem profile creation error in stop",
			setupCfg: func(cfg *configuration) {
				cfg.Options.Profiling.MemProfile = "mem.prof"
			},
			setupMocks: func(osMock *mock_opsys.MockOS) {
				osMock.EXPECT().Create("mem.prof").Return(nil, errors.New("perm error"))
			},
			callStop: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			osMock := mock_opsys.NewMockOS(ctrl)
			tt.setupMocks(osMock)

			cfg := &configuration{OS: osMock}
			tt.setupCfg(cfg)

			stop, err := cfg.startProfiling()
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			if tt.callStop {
				stop()
			}
		})
	}
}
