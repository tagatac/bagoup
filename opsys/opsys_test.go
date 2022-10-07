// Copyright (C) 2020-2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package opsys

import (
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/opsys/scall/mock_scall"
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
			s := &opSys{Fs: fs}
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
			s := &opSys{osStat: osStat}
			exist, err := s.FileExist("testfile")
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, tt.wantExist, exist)
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
			s := &opSys{execCommand: genFakeExecCommand(tt.swVersOutput, tt.swVersErr)}
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

// Adapted from https://npf.io/2015/06/testing-exec-command/.
func genFakeExecCommand(output, err string) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestRunExecCmd", "--", name}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = []string{
			"BAGOUP_WANT_TEST_RUN_EXEC_CMD=1",
			fmt.Sprintf("BAGOUP_TEST_RUN_EXEC_CMD_OUTPUT=%s", output),
			fmt.Sprintf("BAGOUP_TEST_RUN_EXEC_CMD_ERROR=%s", err),
		}
		return cmd
	}
}

func TestRunExecCmd(t *testing.T) {
	if os.Getenv("BAGOUP_WANT_TEST_RUN_EXEC_CMD") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, os.Getenv("BAGOUP_TEST_RUN_EXEC_CMD_OUTPUT"))
	err := os.Getenv("BAGOUP_TEST_RUN_EXEC_CMD_ERROR")
	if err != "" {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
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
	jelenaCard := &vcard.Card{
		"FN": []*vcard.Field{
			{Value: "Jelena Djokovic"},
		},
		"N": []*vcard.Field{
			{Value: "Djokovic;Jelena;;;"},
		},
		"EMAIL": []*vcard.Field{
			{Value: "info@novakdjokovic.com", Params: vcard.Params{"TYPE": []string{"INTERNET"}}},
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
				"info@novakdjokovic.com": jelenaCard,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.setupFs != nil {
				tt.setupFs(fs)
			}
			s := &opSys{Fs: fs}
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
			assert.Equal(t, tt.clean, sanitizePhone(tt.dirty))
		})
	}
}

func TestReadFile(t *testing.T) {
	t.Run("successful read", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		data := []byte("test file contents\n")
		assert.NilError(t, afero.WriteFile(fs, "testfile", data, os.ModePerm))
		testOS := opSys{Fs: fs}
		contents, err := testOS.ReadFile("testfile")
		assert.NilError(t, err)
		assert.Equal(t, contents, string(data))
	})

	t.Run("read failure", func(t *testing.T) {
		testOS := opSys{Fs: afero.NewMemMapFs()}
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
			s := &opSys{Fs: fs, osStat: stat}
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
				assert.Equal(t, tt.wantTempDir, tempDir)
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
			assert.Equal(t, "", s.tempDir)
		})
	}
}

func TestOpenFilesLimit(t *testing.T) {
	tests := []struct {
		msg        string
		setupMock  func(*mock_scall.MockSyscall)
		wantNewErr string
		wantErr    string
	}{
		{
			msg: "happy",
			setupMock: func(scMock *mock_scall.MockSyscall) {
				gomock.InOrder(
					scMock.EXPECT().Getrlimit(syscall.RLIMIT_NOFILE, gomock.Any()).Do(func(_ int, lim *syscall.Rlimit) {
						lim.Cur = 256
						lim.Max = uint64(math.MaxInt64)
					}),
					scMock.EXPECT().Setrlimit(syscall.RLIMIT_NOFILE, gomock.Any()).Do(func(_ int, lim *syscall.Rlimit) {
						assert.Equal(t, uint64(512), lim.Cur)
						assert.Equal(t, uint64(math.MaxInt64), lim.Max)
					}),
				)
			},
		},
		{
			msg: "error checking limits",
			setupMock: func(scMock *mock_scall.MockSyscall) {
				scMock.EXPECT().Getrlimit(syscall.RLIMIT_NOFILE, gomock.Any()).DoAndReturn(func(_ int, lim *syscall.Rlimit) error {
					lim.Cur = 256
					lim.Max = uint64(math.MaxInt64)
					return errors.New("this is a syscall error")
				})
			},
			wantNewErr: "check file count limit: this is a syscall error",
		},
		{
			msg: "new limit too high",
			setupMock: func(scMock *mock_scall.MockSyscall) {
				scMock.EXPECT().Getrlimit(syscall.RLIMIT_NOFILE, gomock.Any()).Do(func(_ int, lim *syscall.Rlimit) {
					lim.Cur = 256
					lim.Max = 500
				})
			},
			wantErr: "512 exceeds the open fd hard limit of 500 - this can be increased with `sudo ulimit -Hn 512`",
		},
		{
			msg: "error setting limit",
			setupMock: func(scMock *mock_scall.MockSyscall) {
				gomock.InOrder(
					scMock.EXPECT().Getrlimit(syscall.RLIMIT_NOFILE, gomock.Any()).Do(func(_ int, lim *syscall.Rlimit) {
						lim.Cur = 256
						lim.Max = uint64(math.MaxInt64)
					}),
					scMock.EXPECT().Setrlimit(syscall.RLIMIT_NOFILE, gomock.Any()).DoAndReturn(func(_ int, lim *syscall.Rlimit) error {
						assert.Equal(t, uint64(512), lim.Cur)
						assert.Equal(t, uint64(math.MaxInt64), lim.Max)
						return errors.New("this is a syscall error")
					}),
				)
			},
			wantErr: "this is a syscall error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			scMock := mock_scall.NewMockSyscall(ctrl)
			tt.setupMock(scMock)

			s, err := NewOS(nil, nil, nil, scMock)
			if tt.wantNewErr != "" {
				assert.Error(t, err, tt.wantNewErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, 256, s.GetOpenFilesLimit())
			err = s.SetOpenFilesLimit(512)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.Equal(t, 512, s.GetOpenFilesLimit())
			assert.NilError(t, err)
		})
	}
}
