package opsys

import (
	"embed"
	"fmt"
	"image/jpeg"
	"math"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/maroto/pkg/consts"
	"github.com/tagatac/maroto/pkg/pdf"
	"github.com/tagatac/maroto/pkg/props"
)

//go:embed fonts/*
var _embedFS embed.FS

type (
	OutFile interface {
		Name() string
		WriteMessage(msg string) error
		WriteImage(imgPath string) error
		Close() error
	}

	txtFile struct {
		afero.File
	}

	pdfFile struct {
		pdf.Maroto
		filePath   string
		pageWidth  float64
		pageHeight float64
		closed     bool
	}
)

func (s opSys) NewOutFile(filePath string, isPDF bool) (OutFile, error) {
	if isPDF {
		m := pdf.NewMaroto(consts.Portrait, consts.Letter)
		font, err := _embedFS.ReadFile("fonts/seguiemj.ttf")
		if err != nil {
			return nil, errors.Wrap(err, "read font file")
		}
		m.AddUTF8FontFromBytes("SegoeUIColorEmoji", "", font)
		m.AddPage()
		pageWidth, pageHeight := m.GetPageSize()
		leftMargin, topMargin, rightMargin, bottomMargin := m.GetPageMargins()
		pageWidth = pageWidth - leftMargin - rightMargin
		pageHeight = pageHeight - topMargin - bottomMargin
		thisFile := pdfFile{
			Maroto:     m,
			filePath:   fmt.Sprintf("%s.pdf", filePath),
			pageWidth:  pageWidth,
			pageHeight: pageHeight,
		}
		return &thisFile, nil
	}
	chatFile, err := s.OpenFile(fmt.Sprintf("%s.txt", filePath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	return txtFile{chatFile}, err
}

func (f txtFile) WriteMessage(msg string) error {
	_, err := f.WriteString(msg)
	return err
}

func (f txtFile) WriteImage(imgPath string) error {
	return errors.New("illegal attempt to write image to text file - open an issue at https://github.com/tagatac/bagoup/issues")
}

func (f *pdfFile) Name() string {
	return f.filePath
}

func (f *pdfFile) WriteMessage(msg string) error {
	textProp := props.Text{Extrapolate: true}
	f.Row(4, func() {
		f.Text(msg, textProp)
	})
	return nil
}

func (f *pdfFile) WriteImage(imgPath string) error {
	reader, err := os.Open(imgPath)
	if err != nil {
		return errors.Wrap(err, "open image file to get dimensions")
	}
	imgCfg, err := jpeg.DecodeConfig(reader)
	if err != nil {
		return errors.Wrap(err, "decode config from JPEG to get dimensions")
	}
	rowHeight := math.Min(f.pageHeight-1, float64(imgCfg.Height))
	f.Row(rowHeight, func() {
		f.FileImage(imgPath)
	})
	return nil
}

func (f *pdfFile) Close() error {
	if f.closed {
		return nil
	}
	f.closed = true
	return f.OutputFileAndClose(f.filePath)
}
