// Copyright (C) 2020-2022 David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package opsys

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Masterminds/semver"
	"github.com/emersion/go-vcard"
	"github.com/spf13/afero"
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
			s := NewOS(fs, nil, nil)
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
			s := NewOS(nil, osStat, nil)
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
			s := NewOS(nil, nil, genFakeExecCommand(tt.swVersOutput, tt.swVersErr))
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
			s := NewOS(fs, nil, nil)
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

func TestCopyFile(t *testing.T) {
	textBytes, err := _embedFS.ReadFile("testdata/text.txt")
	assert.NilError(t, err)
	jpegBytes, err := _embedFS.ReadFile("testdata/tennisballs.jpeg")
	assert.NilError(t, err)

	tests := []struct {
		msg       string
		setupFS   func(afero.Fs)
		roFS      bool
		wantBytes []byte
		wantErr   string
	}{
		{
			msg: "text file",
			setupFS: func(fs afero.Fs) {
				assert.NilError(t, afero.WriteFile(fs, "testfile", textBytes, os.ModePerm))
				assert.NilError(t, fs.Mkdir("destinationdir", os.ModePerm))
			},
			wantBytes: textBytes,
		},
		{
			msg: "jpeg file",
			setupFS: func(fs afero.Fs) {
				assert.NilError(t, afero.WriteFile(fs, "testfile", jpegBytes, os.ModePerm))
				assert.NilError(t, fs.Mkdir("destinationdir", os.ModePerm))
			},
			wantBytes: jpegBytes,
		},
		{
			msg:     "source file does not exist",
			wantErr: "open testfile: file does not exist",
		},
		{
			msg: "read only filesystem",
			setupFS: func(fs afero.Fs) {
				assert.NilError(t, afero.WriteFile(fs, "testfile", textBytes, os.ModePerm))
				assert.NilError(t, fs.Mkdir("destinationdir", os.ModePerm))
			},
			roFS:    true,
			wantErr: `create destination file "destinationdir/testfile": operation not permitted`,
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
			s := NewOS(fs, nil, nil)
			err := s.CopyFile("testfile", "destinationdir")
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			newBytes, err := afero.ReadFile(fs, "destinationdir/testfile")
			assert.NilError(t, err)
			assert.DeepEqual(t, newBytes, tt.wantBytes)
		})
	}
}

func TestTempDir(t *testing.T) {
	tests := []struct {
		msg         string
		prevTempDir string
		setupFS     func(afero.Fs)
	}{
		{
			msg:         "temp dir already exists",
			prevTempDir: filepath.Join(os.TempDir(), "bagoup-12345"),
			setupFS: func(fs afero.Fs) {
				err := fs.Mkdir(filepath.Join(os.TempDir(), "bagoup-12345"), os.ModePerm)
				assert.NilError(t, err)
			},
		},
		{
			msg: "temp dir doesn't exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			rwFS := afero.NewOsFs()
			rwOS := &opSys{
				Fs:      rwFS,
				tempDir: tt.prevTempDir,
			}
			if tt.setupFS != nil {
				tt.setupFS(rwFS)
			}

			tempDir, err := rwOS.getTempDir()
			assert.NilError(t, err)
			prefix := filepath.Join(os.TempDir(), "bagoup")
			assert.Assert(t, strings.HasPrefix(tempDir, prefix), "temp dir %q does not start with %q", tempDir, prefix)
			fmt.Println(tempDir)
			isDir, err := afero.IsDir(rwFS, tempDir)
			assert.NilError(t, err)
			assert.Assert(t, isDir, `temp dir %q has not been created`, tempDir)

			roOS := &opSys{
				Fs:      afero.NewReadOnlyFs(rwFS),
				tempDir: rwOS.tempDir,
			}
			assert.Error(t, roOS.RmTempDir(), fmt.Sprintf("remove temporary directory %q: operation not permitted", roOS.tempDir))

			assert.NilError(t, rwOS.RmTempDir())
			assert.Equal(t, rwOS.tempDir, "")
			assert.NilError(t, rwOS.RmTempDir())
		})
	}
}
