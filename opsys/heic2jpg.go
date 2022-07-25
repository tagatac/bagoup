// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package opsys

import (
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// HEIC2JPG is a thin wrapper on goheif.Converter.HEIC2JPG that creates a
// temporary directory for the converted image.
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
