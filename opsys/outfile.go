package opsys

import (
	"errors"
	"fmt"
	"os"

	"github.com/johnfercher/maroto/pkg/consts"
	"github.com/johnfercher/maroto/pkg/pdf"
	"github.com/johnfercher/maroto/pkg/props"
	"github.com/spf13/afero"
)

type OutFile interface {
	Name() string
	WriteMessage(msg string) error
	WriteImage(imgPath string) error
	Close() error
}

type txtFile struct {
	afero.File
}

func (s opSys) NewOutFile(filePath string, isPDF bool) (OutFile, error) {
	if isPDF {
		thisFile := pdfFile{
			Maroto:   pdf.NewMaroto(consts.Portrait, consts.Letter),
			filePath: fmt.Sprintf("%s.pdf", filePath),
		}
		thisFile.AddPage()
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

type pdfFile struct {
	pdf.Maroto
	filePath string
	closed   bool
}

func (f pdfFile) Name() string {
	return f.filePath
}

func (f pdfFile) WriteMessage(msg string) error {
	textProp := props.Text{Extrapolate: true}
	f.Row(4, func() {
		f.Text(msg, textProp)
	})
	return nil
}

func (f pdfFile) WriteImage(imgPath string) error {
	f.Row(40, func() {
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
