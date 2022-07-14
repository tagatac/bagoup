// Copyright (C) 2022 David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package opsys

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"golang.org/x/net/html"
)

//go:embed templates/* testdata/*
var _embedFS embed.FS

var _unhandledAttachmentTypes []string = []string{".mov"}
var _errFileClosed error = errors.New("file already closed")

//go:generate mockgen -destination=mock_opsys/mock_outfile.go github.com/tagatac/bagoup/opsys OutFile

type (
	// Outfile represents single messages export file, either text or PDF.
	OutFile interface {
		// Name returns the filepath of the Outfile.
		Name() string
		// WriteMessage adds the given message to the Outfile.
		WriteMessage(msg string) error
		// WriteAttachment embeds the given attachment in the Outfile, or adds a
		// reference to it if embedding is not possible (e.g. if the Outfile is
		// plain text, or the attachment is a movie).
		WriteAttachment(attPath string) error
		// Close closes the outfile, writing it to disk. Writes
		Close() error
	}

	txtFile struct {
		afero.File
		closed bool
	}

	pdfFile struct {
		*wkhtmltopdf.PDFGenerator
		filePath                 string
		contents                 htmlFileData
		closed                   bool
		unhandledAttachmentTypes []string
	}

	htmlFileData struct {
		Title string
		Lines []htmlFileLine
	}
	htmlFileLine struct {
		Element template.HTML
	}
)

func (s opSys) NewOutFile(filePath string, isPDF, includePPA bool) (OutFile, error) {
	if isPDF {
		pdfg, err := wkhtmltopdf.NewPDFGenerator()
		if err != nil {
			return nil, errors.Wrap(err, "create PDF generator")
		}
		unhandledAttachmentTypes := _unhandledAttachmentTypes
		if !includePPA {
			unhandledAttachmentTypes = append(unhandledAttachmentTypes, ".pluginpayloadattachment")
		}
		thisFile := pdfFile{
			PDFGenerator: pdfg,
			filePath:     fmt.Sprintf("%s.pdf", filePath),
			contents: htmlFileData{
				Title: filepath.Base(filePath),
				Lines: []htmlFileLine{},
			},
			unhandledAttachmentTypes: unhandledAttachmentTypes,
		}
		return &thisFile, nil
	}
	chatFile, err := s.OpenFile(fmt.Sprintf("%s.txt", filePath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	return &txtFile{File: chatFile}, err
}

func (f txtFile) WriteMessage(msg string) error {
	if f.closed {
		return _errFileClosed
	}
	_, err := f.WriteString(msg)
	return err
}

func (f txtFile) WriteAttachment(attPath string) error {
	if f.closed {
		return _errFileClosed
	}
	return f.WriteMessage(fmt.Sprintf("<attached: %s>\n", filepath.Base(attPath)))
}

func (f *txtFile) Close() error {
	f.closed = true
	return f.File.Close()
}

func (f *pdfFile) Name() string {
	return f.filePath
}

func (f *pdfFile) WriteMessage(msg string) error {
	if f.closed {
		return _errFileClosed
	}
	htmlMsg := template.HTML(strings.ReplaceAll(html.EscapeString(msg), "\n", "<br/>"))
	f.contents.Lines = append(f.contents.Lines, htmlFileLine{Element: htmlMsg})
	return nil
}

func (f *pdfFile) WriteAttachment(attPath string) error {
	if f.closed {
		return _errFileClosed
	}
	att := template.HTML(fmt.Sprintf("<img src=%q alt=%s/><br/>", attPath, filepath.Base(attPath)))
	ext := strings.ToLower(filepath.Ext(attPath))
	for _, t := range f.unhandledAttachmentTypes {
		if ext == t {
			att = template.HTML(fmt.Sprintf("<em>&lt;attached: %s&gt;</em><br/>", filepath.Base(attPath)))
			break
		}
	}
	f.contents.Lines = append(f.contents.Lines, htmlFileLine{Element: att})
	return nil
}

func (f *pdfFile) Close() error {
	if f.closed {
		return nil
	}
	f.closed = true
	tmpl, err := template.ParseFS(_embedFS, "templates/outfile_html.tmpl")
	if err != nil {
		return errors.Wrap(err, "parse HTML template")
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, f.contents); err != nil {
		return errors.Wrap(err, "execute HTML template")
	}
	page := wkhtmltopdf.NewPageReader(bytes.NewReader(buf.Bytes()))
	page.EnableLocalFileAccess.Set(true)
	f.AddPage(page)
	if err := f.Create(); err != nil {
		return errors.Wrap(err, "create PDF in internal buffer")
	}
	return errors.Wrap(f.WriteFile(f.filePath), "write buffer contents to file")
}
