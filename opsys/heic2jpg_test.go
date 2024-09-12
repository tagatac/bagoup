// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package opsys

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/goheif/heic2jpg/mock_heic2jpg"
	"gotest.tools/v3/assert"
)

func TestHEIC2JPG(t *testing.T) {
	fs := afero.NewMemMapFs()
	tempDir := afero.GetTempDir(fs, "")
	dstRegex := regexp.MustCompile(fmt.Sprintf(`%sbagoup\d*/testfile.jpeg`, tempDir))

	tests := []struct {
		msg       string
		src       string
		setupMock func(cMock *mock_heic2jpg.MockConverter)
		roFS      bool
		wantName  string
		wantErr   string
	}{
		{
			msg:      "non HEIC file",
			src:      "testfile.png",
			wantName: "testfile.png",
		},
		{
			msg: "HEIC file",
			src: "testfile.heic",
			setupMock: func(cMock *mock_heic2jpg.MockConverter) {
				cMock.EXPECT().HEIC2JPG("testfile.heic", gomock.Any()).Do(func(_, dst string) {
					assert.Assert(t, dstRegex.MatchString(dst), "unexpected destination file %q", dst)
				})
			},
		},
		{
			msg:     "error getting temp dir",
			src:     "testfile.heic",
			roFS:    true,
			wantErr: `create temporary directory "": operation not permitted`,
		},
		{
			msg: "conversion error",
			src: "testfile.heic",
			setupMock: func(cMock *mock_heic2jpg.MockConverter) {
				cMock.EXPECT().HEIC2JPG("testfile.heic", gomock.Any()).DoAndReturn(func(_, dst string) error {
					assert.Assert(t, dstRegex.MatchString(dst), "unexpected destination file %q", dst)
					return errors.New("this is a conversion error")
				})
			},
			wantErr: "convert HEIC file to JPG file ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			cMock := mock_heic2jpg.NewMockConverter(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(cMock)
			}
			s := &opSys{
				Fs:        fs,
				Converter: cMock,
			}
			if tt.roFS {
				s.Fs = afero.NewReadOnlyFs(fs)
			}

			converted, err := s.HEIC2JPG(tt.src)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			if tt.wantName != "" {
				assert.Equal(t, converted, tt.wantName)
			}
		})
	}
}
