// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package opsys

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/v2/exectest"
	"gotest.tools/v3/assert"
)

func TestConvertHEIC(t *testing.T) {
	fs := afero.NewMemMapFs()
	tempDir := afero.GetTempDir(fs, "")
	dstRegex := regexp.MustCompile(fmt.Sprintf(`%sbagoup\d*/testfile.jpeg`, tempDir))

	tests := []struct {
		msg      string
		src      string
		roFS     bool
		sipsErr  string
		wantName string
		wantErr  string
	}{
		{
			msg:      "non HEIC file",
			src:      "testfile.png",
			wantName: "testfile.png",
		},
		{
			msg: "HEIC file",
			src: "testfile.heic",
		},
		{
			msg:     "error getting temp dir",
			src:     "testfile.heic",
			roFS:    true,
			wantErr: `create temporary directory "": operation not permitted`,
		},
		{
			msg:     "conversion error",
			src:     "testfile.heic",
			sipsErr: "this is a sips error",
			wantErr: "convert HEIC file to JPG file ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			exitCode := 0
			if tt.wantErr != "" {
				exitCode = 1
			}
			s := &opSys{
				Fs:          fs,
				execCommand: exectest.GenFakeExecCommand("TestRunExecCmd", "", tt.sipsErr, exitCode),
			}
			if tt.roFS {
				s.Fs = afero.NewReadOnlyFs(fs)
			}

			converted, err := s.ConvertHEIC(tt.src)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			if tt.wantName == "" {
				assert.Assert(t, dstRegex.MatchString(converted), "unexpected destination file %q", converted)
			} else {
				assert.Equal(t, converted, tt.wantName)
			}
		})
	}
}
