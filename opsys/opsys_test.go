// Copyright (C) 2020  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package opsys

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/v2/exectest"
	"github.com/tagatac/bagoup/v2/opsys/scall/mock_scall"
	"gotest.tools/v3/assert"
)

func TestFileAccess(t *testing.T) {
	tests := []struct {
		msg     string
		setupFS func(afero.Fs)
		wantErr string
	}{
		{
			msg: "have access",
			setupFS: func(fs afero.Fs) {
				_, err := fs.Create("testfile")
				assert.NilError(t, err)
			},
		},
		//Uncomment this when https://github.com/spf13/afero/issues/150 is resolved.
		// {
		// 	msg: "no access",
		// 	setupFS: func(fs afero.Fs) {
		// 		_, err := fs.Create("testfile")
		// 		assert.NilError(t, err)
		// 		err = fs.Chmod("testfile", 0000)
		// 		assert.NilError(t, err)
		// 	},
		// 	wantErr: `open file "testfile": permissions`,
		// },
		{
			msg:     "file doesn't exist",
			wantErr: "open testfile: file does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.setupFS != nil {
				tt.setupFS(fs)
			}
			s := NewOS(fs, nil, "")
			err := s.FileAccess("testfile")
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
		})
	}
}

func TestFileExist(t *testing.T) {
	tests := []struct {
		msg       string
		err       error
		wantExist bool
		wantErr   string
	}{
		{
			msg:       "file exists",
			wantExist: true,
		},
		{
			msg: "file doesn't exist",
			err: os.ErrNotExist,
		},
		{
			msg:     "stat error",
			err:     errors.New("this is a stat error"),
			wantErr: `check existence of file "testfile": this is a stat error`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			osStat := func(string) (os.FileInfo, error) {
				return nil, tt.err
			}
			s := NewOS(nil, osStat, "")
			exist, err := s.FileExist("testfile")
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, exist, tt.wantExist)
		})
	}
}

func TestGetMacOSVersion(t *testing.T) {
	tests := []struct {
		msg          string
		swVersOutput string
		swVersErr    string
		wantVersion  *semver.Version
		wantErr      string
	}{
		{
			msg:          "catalina",
			swVersOutput: "10.15.3\n",
			wantVersion:  semver.MustParse("10.15.3"),
		},
		{
			msg:       "sw_vers error",
			swVersErr: "beep boop beep\n",
			wantErr:   "call sw_vers: exit status 1",
		},
		{
			msg:          "bad version",
			swVersOutput: "asdf",
			wantErr:      `parse semantic version "asdf": Invalid Semantic Version`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			exitCode := 0
			if tt.swVersErr != "" {
				exitCode = 1
			}
			s := &opSys{execCommand: exectest.GenFakeExecCommand("TestRunExecCmd", tt.swVersOutput, tt.swVersErr, exitCode)}
			v, err := s.GetMacOSVersion()
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Assert(t, v.Equal(tt.wantVersion), "Expected: %s\nGot: %s\n", tt.wantVersion, v)
		})
	}
}

func TestGetContactMap(t *testing.T) {
	tagCard := &vcard.Card{
		"VERSION": []*vcard.Field{
			{Value: "3.0"},
		},
		"FN": []*vcard.Field{
			{Value: "David Tagatac"},
		},
		"N": []*vcard.Field{
			{Value: "Tagatac;David;;;"},
		},
		"TEL": []*vcard.Field{
			{Value: "+1415555555", Params: vcard.Params{"TYPE": []string{"CELL"}}},
		},
		"EMAIL": []*vcard.Field{
			{Value: "david@tagatac.net", Params: vcard.Params{"TYPE": []string{"INTERNET"}}},
		},
		"CATEGORIES": []*vcard.Field{
			{Value: "myContacts"},
		},
	}
	noleCard := &vcard.Card{
		"VERSION": []*vcard.Field{
			{Value: "3.0"},
		},
		"FN": []*vcard.Field{
			{Value: "Novak Djokovic"},
		},
		"N": []*vcard.Field{
			{Value: "Djokovic;Novak;;;"},
		},
		"TEL": []*vcard.Field{
			{Value: "+3815555555", Params: vcard.Params{"TYPE": []string{"CELL"}}},
		},
		"EMAIL": []*vcard.Field{
			{Value: "info@novakdjokovic.com", Params: vcard.Params{"TYPE": []string{"INTERNET"}}},
		},
		"CATEGORIES": []*vcard.Field{
			{Value: "myContacts"},
		},
	}
	combinedCard := &vcard.Card{
		"FN": []*vcard.Field{
			{Value: "Novak Djokovic and Jelena Djokovic"},
		},
		"N": []*vcard.Field{
			{Value: ";Novak or Jelena;;;"},
		},
	}

	tests := []struct {
		msg     string
		setupFs func(afero.Fs)
		wantMap map[string]*vcard.Card
		wantErr string
	}{
		{
			msg: "two contacts",
			setupFs: func(fs afero.Fs) {
				afero.WriteFile(fs, "contacts.vcf", []byte(
					`BEGIN:VCARD
VERSION:3.0
FN:David Tagatac
N:Tagatac;David;;;
TEL;TYPE=CELL:+1415555555
EMAIL;TYPE=INTERNET:david@tagatac.net
CATEGORIES:myContacts
END:VCARD
BEGIN:VCARD
VERSION:3.0
FN:Novak Djokovic
N:Djokovic;Novak;;;
TEL;TYPE=CELL:+3815555555
EMAIL;TYPE=INTERNET:info@novakdjokovic.com
CATEGORIES:myContacts
END:VCARD
`), 0644)
			},
			wantMap: map[string]*vcard.Card{
				"+1415555555":            tagCard,
				"david@tagatac.net":      tagCard,
				"+3815555555":            noleCard,
				"info@novakdjokovic.com": noleCard,
			},
		},
		{
			msg:     "no contacts file",
			wantErr: `open contacts.vcf: file does not exist`,
		},
		{
			msg: "bad vcard file",
			setupFs: func(fs afero.Fs) {
				afero.WriteFile(fs, "contacts.vcf", []byte("BEGIN::VCARD\n"), 0644)
			},
			wantErr: "decode vcard: vcard: invalid BEGIN value",
		},
		{
			msg: "shared email address",
			setupFs: func(fs afero.Fs) {
				afero.WriteFile(fs, "contacts.vcf", []byte(
					`BEGIN:VCARD
VERSION:3.0
FN:Novak Djokovic
N:Djokovic;Novak;;;
TEL;TYPE=CELL:+3815555555
EMAIL;TYPE=INTERNET:info@novakdjokovic.com
CATEGORIES:myContacts
END:VCARD
BEGIN:VCARD
FN:Jelena Djokovic
N:Djokovic;Jelena;;;
EMAIL;TYPE=INTERNET:info@novakdjokovic.com
END:VCARD
`), 0644)
			},
			wantMap: map[string]*vcard.Card{
				"+3815555555":            noleCard,
				"info@novakdjokovic.com": combinedCard,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.setupFs != nil {
				tt.setupFs(fs)
			}
			s := NewOS(fs, nil, "")
			contactMap, err := s.GetContactMap("contacts.vcf")
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, tt.wantMap, contactMap)
		})
	}
}

func TestSanitizePhone(t *testing.T) {
	tests := []struct {
		msg   string
		dirty string
		clean string
	}{
		{
			msg:   "already clean",
			dirty: "+14155555555",
			clean: "+14155555555",
		},
		{
			msg:   "dirty",
			dirty: "+1 (415) 555-5555",
			clean: "+14155555555",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			assert.Equal(t, sanitizePhone(tt.dirty), tt.clean)
		})
	}
}

func TestReadFile(t *testing.T) {
	t.Run("successful read", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		data := []byte("test file contents\n")
		assert.NilError(t, afero.WriteFile(fs, "testfile", data, os.ModePerm))
		testOS := NewOS(fs, nil, "")
		contents, err := testOS.ReadFile("testfile")
		assert.NilError(t, err)
		assert.Equal(t, contents, string(data))
	})

	t.Run("read failure", func(t *testing.T) {
		testOS := NewOS(afero.NewMemMapFs(), nil, "")
		_, err := testOS.ReadFile("nonexistentfile")
		assert.Error(t, err, "open nonexistentfile: file does not exist")
	})
}

func TestCopyFile(t *testing.T) {
	textBytes, err := _embedFS.ReadFile("testdata/text.txt")
	assert.NilError(t, err)
	jpegBytes, err := _embedFS.ReadFile("testdata/tennisballs.jpeg")
	assert.NilError(t, err)

	tests := []struct {
		msg         string
		setupFS     func(afero.Fs)
		roFS        bool
		statErr     bool
		unique      bool
		wantDstPath string
		wantFile    string
		wantBytes   []byte
		wantErr     string
	}{
		{
			msg: "text file",
			setupFS: func(fs afero.Fs) {
				assert.NilError(t, afero.WriteFile(fs, "testfile.txt", textBytes, os.ModePerm))
				assert.NilError(t, fs.Mkdir("destinationdir", os.ModePerm))
			},
			wantDstPath: "destinationdir/testfile.txt",
			wantFile:    "destinationdir/testfile.txt",
			wantBytes:   textBytes,
		},
		{
			msg: "jpeg file",
			setupFS: func(fs afero.Fs) {
				assert.NilError(t, afero.WriteFile(fs, "testfile.txt", jpegBytes, os.ModePerm))
				assert.NilError(t, fs.Mkdir("destinationdir", os.ModePerm))
			},
			wantDstPath: "destinationdir/testfile.txt",
			wantFile:    "destinationdir/testfile.txt",
			wantBytes:   jpegBytes,
		},
		{
			msg: "two files already exist - unique wanted",
			setupFS: func(fs afero.Fs) {
				assert.NilError(t, afero.WriteFile(fs, "testfile.txt", textBytes, os.ModePerm))
				assert.NilError(t, fs.Mkdir("destinationdir", os.ModePerm))
				f, err := fs.Create("destinationdir/testfile.txt")
				assert.NilError(t, err)
				f.Close()
				f, err = fs.Create("destinationdir/testfile-1.txt")
				assert.NilError(t, err)
				f.Close()
			},
			unique:      true,
			wantDstPath: "destinationdir/testfile-2.txt",
			wantFile:    "destinationdir/testfile-2.txt",
			wantBytes:   textBytes,
		},
		{
			msg: "file already exists - unique not wanted",
			setupFS: func(fs afero.Fs) {
				assert.NilError(t, afero.WriteFile(fs, "destinationdir/testfile.txt", textBytes, os.ModePerm))
			},
			wantDstPath: "destinationdir/testfile.txt",
			wantFile:    "destinationdir/testfile.txt",
			wantBytes:   textBytes,
		},
		{
			msg: "folder disguised as a file",
			setupFS: func(fs afero.Fs) {
				assert.NilError(t, afero.WriteFile(fs, "testfile.txt/realfile.txt", textBytes, os.ModePerm))
				assert.NilError(t, fs.Mkdir("destinationdir", os.ModePerm))
			},
			wantDstPath: "destinationdir/testfile.txt",
			wantFile:    "destinationdir/testfile.txt/realfile.txt",
			wantBytes:   textBytes,
		},
		{
			msg:     "error checking for duplicate files",
			statErr: true,
			wantErr: `check existence of file "destinationdir/testfile.txt": this is a stat error`,
		},
		{
			msg:     "source file does not exist",
			wantErr: "open testfile.txt: file does not exist",
		},
		{
			msg: "read only filesystem",
			setupFS: func(fs afero.Fs) {
				assert.NilError(t, afero.WriteFile(fs, "testfile.txt", textBytes, os.ModePerm))
				assert.NilError(t, fs.Mkdir("destinationdir", os.ModePerm))
			},
			roFS:    true,
			wantErr: "operation not permitted",
		},
		{
			msg: "copy directory - read only filesystem",
			setupFS: func(fs afero.Fs) {
				assert.NilError(t, afero.WriteFile(fs, "testfile.txt/realfile.txt", textBytes, os.ModePerm))
				assert.NilError(t, fs.Mkdir("destinationdir", os.ModePerm))
			},
			roFS:    true,
			wantErr: "operation not permitted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.setupFS != nil {
				tt.setupFS(fs)
			}
			if tt.roFS {
				fs = afero.NewReadOnlyFs(fs)
			}
			stat := fs.Stat
			if tt.statErr {
				stat = func(name string) (os.FileInfo, error) {
					return nil, errors.New("this is a stat error")
				}
			}
			s := NewOS(fs, stat, "")
			dstPath, err := s.CopyFile("testfile.txt", "destinationdir", tt.unique)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, dstPath, tt.wantDstPath)
			newBytes, err := afero.ReadFile(fs, tt.wantFile)
			assert.NilError(t, err)
			assert.DeepEqual(t, newBytes, tt.wantBytes)
		})
	}
}

func TestGetTempDir(t *testing.T) {
	tests := []struct {
		msg         string
		prevTempDir string
		roFS        bool
		wantTempDir string
		wantErr     string
	}{
		{
			msg:         "temp dir already exists",
			prevTempDir: "/var/bagoup12345",
			wantTempDir: "/var/bagoup12345",
		},
		{
			msg:         "successful creation",
			prevTempDir: "",
		},
		{
			msg:     "creation fails",
			roFS:    true,
			wantErr: "create temporary directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.roFS {
				fs = afero.NewReadOnlyFs(fs)
			}
			s := &opSys{
				Fs:      fs,
				tempDir: tt.prevTempDir,
			}

			tempDir, err := s.getTempDir()
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			if tt.wantTempDir != "" {
				assert.Equal(t, tempDir, tt.wantTempDir)
				return
			}
			prefix := filepath.Join(afero.GetTempDir(fs, ""), "bagoup")
			assert.Assert(t, strings.HasPrefix(tempDir, prefix), "temporary directory %q does not have prefix %q", tempDir, prefix)
		})
	}
}

func TestRmTempDir(t *testing.T) {
	tests := []struct {
		msg         string
		prevTempDir string
		roFS        bool
		wantErr     string
	}{
		{
			msg: "no temp dir set",
		},
		{
			msg:         "successful removal",
			prevTempDir: "/var/bagoup12345",
		},
		{
			msg:         "removal fails",
			prevTempDir: "/var/bagoup12345",
			roFS:        true,
			wantErr:     `remove temporary directory "/var/bagoup12345": operation not permitted`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.roFS {
				fs = afero.NewReadOnlyFs(fs)
			}
			s := &opSys{
				Fs:      fs,
				tempDir: tt.prevTempDir,
			}

			err := s.RmTempDir()
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, s.tempDir, "")
		})
	}
}

func TestGetOpenFilesLimit(t *testing.T) {
	tests := []struct {
		msg              string
		currentSoftLimit int
		currentHardLimit uint64
		setupMocks       func(*mock_scall.MockSyscall)
		ulimitOutput     string
		ulimitErr        string
		ulimitExitCode   int
		wantSoftLimit    int
		wantHardLimit    uint64
		wantErr          string
	}{
		{
			msg:              "limit already known",
			currentSoftLimit: 256,
			wantSoftLimit:    256,
		},
		{
			msg:          "happy",
			ulimitOutput: "256\n",
			setupMocks: func(scMock *mock_scall.MockSyscall) {
				scMock.EXPECT().Getrlimit(syscall.RLIMIT_NOFILE, gomock.Any()).Do(func(_ int, lim *syscall.Rlimit) {
					lim.Cur = 256
					lim.Max = math.MaxInt64
				})
			},
			wantSoftLimit: 256,
			wantHardLimit: math.MaxInt64,
		},
		{
			msg:            "ulimit error",
			ulimitErr:      "this is a ulimit error",
			ulimitExitCode: 1,
			wantErr:        "call ulimit: exit status 1",
		},
		{
			msg:          "ulimit bad output",
			ulimitOutput: "asdf\n",
			wantErr:      `parse open files soft limit: strconv.Atoi: parsing "asdf": invalid syntax`,
		},
		{
			msg:          "syscall error",
			ulimitOutput: "256\n",
			setupMocks: func(scMock *mock_scall.MockSyscall) {
				scMock.EXPECT().Getrlimit(syscall.RLIMIT_NOFILE, gomock.Any()).Return(errors.New("this is a syscall error"))
			},
			wantErr: "get open files hard limit: this is a syscall error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			scMock := mock_scall.NewMockSyscall(ctrl)
			if tt.setupMocks != nil {
				tt.setupMocks(scMock)
			}

			s := &opSys{
				execCommand:        exectest.GenFakeExecCommand("TestRunExecCmd", tt.ulimitOutput, tt.ulimitErr, tt.ulimitExitCode),
				Syscall:            scMock,
				openFilesLimitSoft: tt.currentSoftLimit,
				openFilesLimitHard: tt.currentHardLimit,
			}
			limit, err := s.GetOpenFilesLimit()
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, limit, tt.wantSoftLimit)
			assert.Equal(t, s.openFilesLimitSoft, tt.wantSoftLimit)
			assert.Equal(t, s.openFilesLimitHard, tt.wantHardLimit)
		})
	}
}

func TestSetOpenFilesLimit(t *testing.T) {
	tests := []struct {
		msg              string
		currentHardLimit uint64
		setupMock        func(*mock_scall.MockSyscall)
		wantErr          string
	}{
		{
			msg: "happy",
			setupMock: func(scMock *mock_scall.MockSyscall) {
				scMock.EXPECT().Setrlimit(syscall.RLIMIT_NOFILE, gomock.Any()).Do(func(_ int, lim *syscall.Rlimit) {
					assert.Equal(t, lim.Cur, uint64(512))
					assert.Equal(t, lim.Max, uint64(math.MaxInt64))
				})
			},
		},
		{
			msg:              "new limit too high",
			currentHardLimit: 500,
			wantErr:          "512 exceeds the open fd hard limit of 500 - this can be increased with `sudo ulimit -Hn 512`",
		},
		{
			msg: "error setting limit",
			setupMock: func(scMock *mock_scall.MockSyscall) {
				scMock.EXPECT().Setrlimit(syscall.RLIMIT_NOFILE, gomock.Any()).DoAndReturn(func(_ int, lim *syscall.Rlimit) error {
					assert.Equal(t, lim.Cur, uint64(512))
					assert.Equal(t, lim.Max, uint64(math.MaxInt64))
					return errors.New("this is a syscall error")
				})
			},
			wantErr: "this is a syscall error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			scMock := mock_scall.NewMockSyscall(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(scMock)
			}

			s := &opSys{
				Syscall:            scMock,
				openFilesLimitSoft: 256,
				openFilesLimitHard: math.MaxInt64,
			}
			if tt.currentHardLimit != 0 {
				s.openFilesLimitHard = tt.currentHardLimit
			}
			err := s.SetOpenFilesLimit(512)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.Equal(t, s.openFilesLimitSoft, 512)
			assert.NilError(t, err)
		})
	}
}

func TestRunExecCmd(t *testing.T) { exectest.RunExecCmd() }
