package imgconv

import (
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

//go:generate mockgen -destination=mock_imgconv/mock_imgconv.go github.com/tagatac/bagoup/v2/imgconv ImgConverter

type ImgConverter interface {
	// ConvertHEIC converts the src file to a JPEG image if the src file is an
	// HEIC image, returning the path to the JPEG image. Otherwise the src
	// path is returned.
	ConvertHEIC(src string) (string, error)
}

func (i *imgConverter) ConvertHEIC(src string) (string, error) {
	if strings.ToLower(filepath.Ext(src)) != ".heic" {
		return src, nil
	}
	jpgFilename := strings.TrimRight(filepath.Base(src), "HEICheic") + "jpeg"
	dst := filepath.Join(i.tempDir, jpgFilename)
	if err := i.convert(src, dst); err != nil {
		return src, errors.Wrapf(err, "convert HEIC file to JPG file %q", dst)
	}
	return dst, nil
}
