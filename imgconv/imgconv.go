// Copyright (C) 2026  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

// Package imgconv provides an interface ImgConverter for converting HEIC images
// to JPEG format.
package imgconv

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:generate mockgen -destination=mock_imgconv/mock_imgconv.go github.com/tagatac/bagoup/v2/imgconv ImgConverter

type (
	// ImgConverter converts images from HEIC format to JPEG format.
	ImgConverter interface {
		// ConvertHEIC converts the src file to a JPEG image if the src file is an
		// HEIC image, returning the path to the JPEG image. Otherwise the src
		// path is returned.
		ConvertHEIC(src string) (string, error)
	}

	imgConverter struct {
		execCommand func(string, ...string) *exec.Cmd
		tempDir     string
	}
)

func NewImgConverter(tempDir string) ImgConverter {
	return &imgConverter{
		execCommand: exec.Command,
		tempDir:     tempDir,
	}
}

func (i *imgConverter) ConvertHEIC(src string) (string, error) {
	if strings.ToLower(filepath.Ext(src)) != ".heic" {
		return src, nil
	}
	jpgFilename := strings.TrimRight(filepath.Base(src), "HEICheic") + "jpeg"
	dst := filepath.Join(i.tempDir, jpgFilename)
	var stderr bytes.Buffer
	cmd := i.convCmd(src, dst)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return src, fmt.Errorf(
			"convert HEIC file to JPG file %q: %w: %s",
			dst, err, stderr.String(),
		)
	}
	return dst, nil
}
