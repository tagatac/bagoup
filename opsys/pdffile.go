package opsys

import (
	"bytes"
	"fmt"
	"html/template"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"golang.org/x/net/html"
)

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

type (
	pdfFile struct {
		afero.File
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

func newPDFFile(chatFile afero.File, includePPA bool, templatePath string) *pdfFile {
	chatFilename := chatFile.Name()
	title := strings.TrimSuffix(filepath.Base(chatFilename), filepath.Ext(chatFilename))
	embeddableImageTypes := _embeddableImageTypes
	if includePPA {
		embeddableImageTypes = append(embeddableImageTypes, ".pluginpayloadattachment")
	}
	return &pdfFile{
		File: chatFile,
		contents: htmlFileData{
			Title: title,
			Lines: []htmlFileLine{},
		},
		embeddableImageTypes: embeddableImageTypes,
		templatePath:         templatePath,
	}
}

func (f *pdfFile) WriteMessage(msg string) error {
	msg = strings.ReplaceAll(html.EscapeString(msg), "\n", "<br/>")
	// Remove object replacement characters (U+FFFC) from the message. These
	// characters are used by the chat database to represent attachments, but
	// they are not valid in HTML. https://en.wiktionary.org/wiki/%EF%BF%BC
	msg = strings.ReplaceAll(msg, "\uFFFC", "")
	f.contents.Lines = append(f.contents.Lines, htmlFileLine{Element: template.HTML(msg)})
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
	return strings.Count(f.buf.String(), "<img"), nil
}
