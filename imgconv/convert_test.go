// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package imgconv

import (
	"testing"

	"github.com/tagatac/bagoup/v2/exectest"
	"gotest.tools/v3/assert"
)

func TestConvertHEIC(t *testing.T) {
	tests := []struct {
		msg      string
		src      string
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
			msg:      "HEIC file",
			src:      "testfile.heic",
			wantName: "testTempDir/5e71bbb03bd4bb80_testfile.jpeg",
		},
		{
			msg:     "conversion error",
			src:     "testfile.heic",
			sipsErr: "this is a conversion error",
			wantErr: "convert HEIC file to JPG file \"testTempDir/5e71bbb03bd4bb80_testfile.jpeg\": exit status 1: this is a conversion error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			exitCode := 0
			if tt.wantErr != "" {
				exitCode = 1
			}
			i := &imgConverter{
				execCommand: exectest.GenFakeExecCommand("TestRunExecCmd", "", tt.sipsErr, exitCode),
				tempDir:     "testTempDir",
			}

			converted, err := i.ConvertHEIC(tt.src)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, converted, tt.wantName)
		})
	}
}

func TestRunExecCmd(t *testing.T) { exectest.RunExecCmd() }
