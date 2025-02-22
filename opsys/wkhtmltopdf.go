package opsys

import (
	"bytes"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/v2/opsys/pdfgen"
)

type wkhtmltopdfFile struct {
	*pdfFile
	pdfgen.PDFGenerator
}

func (s *opSys) NewWkhtmltopdfFile(entityName string, chatFile afero.File, pdfg pdfgen.PDFGenerator, includePPA bool) OutFile {
	return &wkhtmltopdfFile{
		pdfFile:      newPDFFile(chatFile, includePPA, "templates/wkhtmltopdf_html.tmpl", entityName, s.bagoupVersion),
		PDFGenerator: pdfg,
	}
}

func (f *wkhtmltopdfFile) Flush() error {
	page := wkhtmltopdf.NewPageReader(bytes.NewReader(f.buf.Bytes()))
	page.EnableLocalFileAccess.Set(true)
	f.PDFGenerator.AddPage(page)
	return f.PDFGenerator.Create()
}
