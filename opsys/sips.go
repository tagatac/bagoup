// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package opsys

import (
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func (s *opSys) ConvertHEIC(src string) (string, error) {
	if strings.ToLower(filepath.Ext(src)) != ".heic" {
		return src, nil
	}
	tempDir, err := s.getTempDir()
	if err != nil {
		return src, err
	}
	jpgFilename := strings.TrimRight(filepath.Base(src), "HEICheic") + "jpeg"
	dst := filepath.Join(tempDir, jpgFilename)
	if err := s.sipsConvert(src, dst); err != nil {
		return src, errors.Wrapf(err, "convert HEIC file to JPG file %q", dst)
	}
	return dst, nil
}

func (s opSys) sipsConvert(src, dst string) error {
	cmd := s.execCommand(
		"sips",
		"--setProperty", "format", "jpeg",
		"--setProperty", "formatOptions", "best",
		"--out", dst,
		src,
	)
	return cmd.Run()
}
