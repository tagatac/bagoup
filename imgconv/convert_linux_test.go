// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package imgconv

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/tagatac/goheif/heic2jpg/mock_heic2jpg"
	"gotest.tools/v3/assert"
)

func TestConvertHEIC(t *testing.T) {
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
					assert.Equal(t, dst, "testTempDir/testfile.jpeg")
				})
			},
		},
		{
			msg: "conversion error",
			src: "testfile.heic",
			setupMock: func(cMock *mock_heic2jpg.MockConverter) {
				cMock.EXPECT().HEIC2JPG("testfile.heic", gomock.Any()).DoAndReturn(func(_, dst string) error {
					assert.Equal(t, dst, "testTempDir/testfile.jpeg")
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
			i := &imgConverter{
				Converter: cMock,
				tempDir:   "testTempDir",
			}

			converted, err := i.ConvertHEIC(tt.src)
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
