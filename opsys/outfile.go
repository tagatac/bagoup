// Copyright (C) 2022  David Tagatac <david@tagatac.net>
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

// Embeddable image types copied from
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Basics_of_HTTP/MIME_types/Common_types
// but many types are untested. If you find one that doesn't work or one that
// should work but is not embedded, please open an issue.
var _embeddableImageTypes []string = []string{
	".avif",
	".bmp",
	".gif",
	".ico",
	".jpeg",
	".jpg",
	".png",
	".svg",
	".tif",
	".tiff",
	".webp",
}
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
		// plain text, or the attachment is a movie). The return value lets the
		// caller know whether the file was embedded or not.
		WriteAttachment(attPath string) (bool, error)
		// ReferenceAttachment adds a reference to the given filename in the Outfile.
		ReferenceAttachment(filename string) error
		// Stage prepares an OutFile for writing and closing, and returns the
		// number of images to be embedded in the OutFile.
		Stage() (int, error)
		// Close closes the OutFile, writing it to disk. Writes
		Close() error
	}
)

func (s opSys) NewOutFile(filePath string, isPDF, includePPA bool) (OutFile, error) {
	title := filepath.Base(filePath)
	if isPDF {
		filePath = fmt.Sprintf("%s.pdf", filePath)
		chatFile, err := s.Fs.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, errors.Wrapf(err, "open file %q", filePath)
		}
		pdfg, err := NewPDFGenerator(chatFile)
		if err != nil {
			return nil, errors.Wrap(err, "create PDF generator")
		}
		embeddableImageTypes := _embeddableImageTypes
		if includePPA {
			embeddableImageTypes = append(embeddableImageTypes, ".pluginpayloadattachment")
		}
		thisFile := pdfFile{
			File:         chatFile,
			PDFGenerator: pdfg,
			contents: htmlFileData{
				Title: title,
				Lines: []htmlFileLine{},
			},
			embeddableImageTypes: embeddableImageTypes,
		}
		return &thisFile, nil
	}
	filePath = fmt.Sprintf("%s.txt", filePath)
	chatFile, err := s.Fs.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	return &txtFile{File: chatFile}, errors.Wrapf(err, "open file %q", filePath)
}

type txtFile struct {
	afero.File
}

func (f txtFile) WriteMessage(msg string) error {
	_, err := f.File.WriteString(msg)
	return err
}

func (f txtFile) WriteAttachment(attPath string) (bool, error) {
	return false, f.ReferenceAttachment(filepath.Base(attPath))
}

func (f txtFile) ReferenceAttachment(filename string) error {
	return f.WriteMessage(fmt.Sprintf("<attached: %s>\n", filename))
}

func (f txtFile) Stage() (int, error) {
	return 0, nil
}

type (
	pdfFile struct {
		afero.File
		PDFGenerator
		contents             htmlFileData
		closed               bool
		embeddableImageTypes []string
		html                 template.HTML
	}

	htmlFileData struct {
		Title string
		Lines []htmlFileLine
	}
	htmlFileLine struct {
		Element template.HTML
	}
)

func (f *pdfFile) WriteMessage(msg string) error {
	if f.closed {
		return _errFileClosed
	}
	htmlMsg := template.HTML(strings.ReplaceAll(html.EscapeString(msg), "\n", "<br/>"))
	f.contents.Lines = append(f.contents.Lines, htmlFileLine{Element: htmlMsg})
	return nil
}

func (f *pdfFile) WriteAttachment(attPath string) (bool, error) {
	if f.closed {
		return false, _errFileClosed
	}
	embedded := false
	var att template.HTML
	ext := strings.ToLower(filepath.Ext(attPath))
	for _, t := range f.embeddableImageTypes {
		if ext == t {
			embedded = true
			// whktmltopdf doesn't handle URLs quite right:
			// https://github.com/wkhtmltopdf/wkhtmltopdf/issues/4406
			attPath = strings.ReplaceAll(attPath, "?", "%3F")
			att = template.HTML(fmt.Sprintf("<img src=%q alt=%q/><br/>", attPath, filepath.Base(attPath)))
			break
		}
	}
	if !embedded {
		return false, f.ReferenceAttachment(filepath.Base(attPath))
	}
	f.contents.Lines = append(f.contents.Lines, htmlFileLine{Element: att})
	return true, nil
}

func (f *pdfFile) ReferenceAttachment(filename string) error {
	if f.closed {
		return _errFileClosed
	}
	att := template.HTML(fmt.Sprintf("<em>&lt;attached: %s&gt;</em><br/>", filename))
	f.contents.Lines = append(f.contents.Lines, htmlFileLine{Element: att})
	return nil
}

func (f *pdfFile) Stage() (int, error) {
	if f.closed {
		return 0, _errFileClosed
	}
	tmpl, err := template.ParseFS(_embedFS, "templates/outfile_html.tmpl")
	if err != nil {
		return 0, errors.Wrap(err, "parse HTML template")
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, f.contents); err != nil {
		return 0, errors.Wrap(err, "execute HTML template")
	}
	f.html = template.HTML(buf.String())
	page := wkhtmltopdf.NewPageReader(bytes.NewReader(buf.Bytes()))
	page.EnableLocalFileAccess.Set(true)
	f.PDFGenerator.AddPage(page)
	return strings.Count(string(f.html), "<img"), nil
}

func (f *pdfFile) Close() error {
	if f.closed {
		return nil
	}
	defer f.File.Close()
	f.closed = true
	if err := f.PDFGenerator.Create(); err != nil {
		return errors.Wrap(err, "write out PDF")
	}
	return errors.Wrap(f.File.Close(), "close PDF")
}
