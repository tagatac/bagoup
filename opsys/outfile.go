// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package opsys

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/v2/opsys/pdfgen"
	"golang.org/x/net/html"
)

//go:embed templates/* all:testdata/**
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

//go:generate mockgen -destination=mock_opsys/mock_outfile.go github.com/tagatac/bagoup/v2/opsys OutFile

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
		// Stage prepares the OutFile for flushing to disk, and returns the number
		// of images embedded in the OutFile.
		Stage() (int, error)
		// Flush flushes the contents of an OutFile to disk.
		Flush() error
	}
)

type txtFile struct {
	afero.File
}

func (opSys) NewTxtOutFile(chatFile afero.File) OutFile {
	return txtFile{File: chatFile}
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

func (f txtFile) Flush() error {
	return nil
}

type (
	pdfFile struct {
		afero.File
		pdfgen.PDFGenerator
		contents             htmlFileData
		embeddableImageTypes []string
		templatePath         string
		buf                  bytes.Buffer
	}

	htmlFileData struct {
		Title string
		Lines []htmlFileLine
	}
	htmlFileLine struct {
		Element template.HTML
	}
)

func (opSys) NewPDFOutFile(chatFile afero.File, pdfg pdfgen.PDFGenerator, includePPA bool) OutFile {
	chatFilename := chatFile.Name()
	title := strings.TrimSuffix(filepath.Base(chatFilename), filepath.Ext(chatFilename))
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
		templatePath:         "templates/outfile_html.tmpl",
	}
	return &thisFile
}

func (f *pdfFile) WriteMessage(msg string) error {
	htmlMsg := template.HTML(strings.ReplaceAll(html.EscapeString(msg), "\n", "<br/>"))
	f.contents.Lines = append(f.contents.Lines, htmlFileLine{Element: htmlMsg})
	return nil
}

func (f *pdfFile) WriteAttachment(attPath string) (bool, error) {
	embedded := false
	var att template.HTML
	ext := strings.ToLower(filepath.Ext(attPath))
	for _, t := range f.embeddableImageTypes {
		if ext == t {
			embedded = true
			att = template.HTML(fmt.Sprintf("<img src=%q alt=%q/><br/>", urlEscapeFilePath(attPath), filepath.Base(attPath)))
			break
		}
	}
	if !embedded {
		return false, f.ReferenceAttachment(filepath.Base(attPath))
	}
	f.contents.Lines = append(f.contents.Lines, htmlFileLine{Element: att})
	return true, nil
}

// urlEscapeFilePath escapes the path parts of a file path, so that it can be
// used as a URL in wkhtmltopdf. Notably, it does not escape the separator.
// See https://github.com/wkhtmltopdf/wkhtmltopdf/issues/4406 for more info.
func urlEscapeFilePath(path string) string {
	parts := strings.Split(path, string(filepath.Separator))
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, string(filepath.Separator))
}

func (f *pdfFile) ReferenceAttachment(filename string) error {
	att := template.HTML(fmt.Sprintf("<em>&lt;attached: %s&gt;</em><br/>", filename))
	f.contents.Lines = append(f.contents.Lines, htmlFileLine{Element: att})
	return nil
}

func (f *pdfFile) Stage() (int, error) {
	tmpl, err := template.ParseFS(_embedFS, f.templatePath)
	if err != nil {
		return 0, errors.Wrap(err, "parse HTML template")
	}
	if err := tmpl.Execute(&f.buf, f.contents); err != nil {
		return 0, errors.Wrap(err, "execute HTML template")
	}
	htmlStr := template.HTML(f.buf.String())
	return strings.Count(string(htmlStr), "<img"), nil
}

func (f *pdfFile) Flush() error {
	page := wkhtmltopdf.NewPageReader(bytes.NewReader(f.buf.Bytes()))
	page.EnableLocalFileAccess.Set(true)
	f.PDFGenerator.AddPage(page)
	return f.PDFGenerator.Create()
}
