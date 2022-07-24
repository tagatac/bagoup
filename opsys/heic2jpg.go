// Copyright 2019 github.com/jdeng <jackdeng@gmail.com> as
// https://github.com/jdeng/goheif/blob/master/heic2jpg/main.go
// Copyright 2022 David Tagatac <david@tagatac.net> as
// https://github.com/tagatac/bagoup/blob/main/opsys/heic2jpg.go

// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package opsys

import (
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func (s *opSys) HEIC2JPG(src string) (string, error) {
	if strings.ToLower(filepath.Ext(src)) != ".heic" {
		return src, nil
	}
	tempDir, err := s.getTempDir()
	if err != nil {
		return src, err
	}
	jpgFilename := strings.TrimRight(filepath.Base(src), "HEICheic") + "jpeg"
	dst := filepath.Join(tempDir, jpgFilename)
	if err := s.Converter.HEIC2JPG(src, dst); err != nil {
		return src, errors.Wrapf(err, "convert HEIC file to JPG file %q", dst)
	}
	return dst, nil
}
