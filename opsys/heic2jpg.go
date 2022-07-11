package opsys

import (
	"image/jpeg"
	"io"
	"log"
	"path/filepath"
	"strings"

	"github.com/adrium/goheif"
	"github.com/pkg/errors"
)

// Skip Writer for exif writing
type writerSkipper struct {
	w           io.Writer
	bytesToSkip int
}

func (s *opSys) HEIC2JPG(src string) (string, error) {
	if strings.ToLower(filepath.Ext(src)) != ".heic" {
		return src, nil
	}
	tempDir, err := s.getTempDir()
	if err != nil {
		return "", err
	}
	jpgFileName := strings.TrimRight(filepath.Base(src), "HEICheic") + "jpeg"
	dst := filepath.Join(tempDir, jpgFileName)
	fin, err := s.Open(src)
	if err != nil {
		return "", errors.Wrapf(err, "open HEIC file %q", src)
	}
	defer fin.Close()
	exif, err := goheif.ExtractExif(fin)
	if err != nil {
		log.Printf("WARN: failed to get EXIF data from file %q: %s\n", fin.Name(), err)
	}
	img, err := goheif.Decode(fin)
	if err != nil {
		return "", errors.Wrap(err, "decode HEIC image")
	}
	fout, err := s.Create(dst)
	if err != nil {
		return "", errors.Wrapf(err, "create JPG file %q", dst)
	}
	defer fout.Close()
	w, err := newWriterExif(fout, exif)
	if err != nil {
		return "", errors.Wrap(err, "create writer with EXIF")
	}
	err = jpeg.Encode(w, img, nil)
	if err != nil {
		return "", errors.Wrap(err, "encode JPG image")
	}
	return dst, nil
}

func (w *writerSkipper) Write(data []byte) (int, error) {
	if w.bytesToSkip <= 0 {
		return w.w.Write(data)
	}

	if dataLen := len(data); dataLen < w.bytesToSkip {
		w.bytesToSkip -= dataLen
		return dataLen, nil
	}

	if n, err := w.w.Write(data[w.bytesToSkip:]); err == nil {
		n += w.bytesToSkip
		w.bytesToSkip = 0
		return n, nil
	} else {
		return n, err
	}
}

func newWriterExif(w io.Writer, exif []byte) (io.Writer, error) {
	writer := &writerSkipper{w, 2}
	soi := []byte{0xff, 0xd8}
	if _, err := w.Write(soi); err != nil {
		return nil, err
	}

	if exif != nil {
		app1Marker := 0xe1
		markerlen := 2 + len(exif)
		marker := []byte{0xff, uint8(app1Marker), uint8(markerlen >> 8), uint8(markerlen & 0xff)}
		if _, err := w.Write(marker); err != nil {
			return nil, err
		}

		if _, err := w.Write(exif); err != nil {
			return nil, err
		}
	}

	return writer, nil
}
