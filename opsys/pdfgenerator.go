// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package opsys

import (
	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

//go:generate mockgen -destination=mock_opsys/mock_pdf_generator.go github.com/tagatac/bagoup/opsys PDFGenerator

type (
	// PDFGenerator is a thin wrapper for the wkhtmltopdf.PDFGenerator struct.
	PDFGenerator interface {
		// AddPage adds a new input page to the document.
		// A page is an input HTML page, it can span multiple pages in the output document.
		// It is a Page when read from file or URL or a PageReader when read from memory.
		AddPage(p wkhtmltopdf.PageProvider)
		// Create creates the PDF document and writes is to the output file.
		Create() error
	}
	pdfGenerator struct {
		*wkhtmltopdf.PDFGenerator
	}
)

// NewPDFGenerator sets up a new PDF generator to write to the given file.
func NewPDFGenerator(chatFile afero.File) (PDFGenerator, error) {
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, errors.Wrap(err, "create PDF generator")
	}
	pdfg.SetOutput(chatFile)
	return pdfGenerator{PDFGenerator: pdfg}, nil
}

func (g pdfGenerator) AddPage(p wkhtmltopdf.PageProvider) {
	g.PDFGenerator.AddPage(p)
}

func (g pdfGenerator) Create() error {
	return g.PDFGenerator.Create()
}
