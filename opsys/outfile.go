package opsys

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"golang.org/x/net/html"
)

// Copied from https://github.com/wkhtmltopdf/wkhtmltopdf/issues/2913#issuecomment-1011269370
const _twemojiScript string = `
<style>
img.emoji {
    height: 1em;
    width: 1em;
    margin: 0 .05em 0 .1em;
    vertical-align: -0.1em;
}
</style>
<script src="https://twemoji.maxcdn.com/v/latest/twemoji.min.js"></script>
<script>window.onload = function () { twemoji.parse(document.body);}</script>
`
const _imgMaxDimensions string = `
<style>
img {
    max-width:900px;
    max-height:1300px;
}
</style>
`

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
		*wkhtmltopdf.PDFGenerator
		filePath string
		contents string
		closed   bool
	}
)

func (s opSys) NewOutFile(filePath string, isPDF bool) (OutFile, error) {
	if isPDF {
		pdfg, err := wkhtmltopdf.NewPDFGenerator()
		if err != nil {
			return nil, errors.Wrap(err, "create PDF generator")
		}
		contents := fmt.Sprintf(`<!doctype html>
<html>
	<head>
		<title>%s</title>
		<meta charset="UTF-8">
		%s
		%s
	</head>
	<body>
`,
			path.Base(filePath),
			_twemojiScript,
			_imgMaxDimensions,
		)
		thisFile := pdfFile{
			PDFGenerator: pdfg,
			filePath:     fmt.Sprintf("%s.pdf", filePath),
			contents:     contents,
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
	msg = strings.ReplaceAll(html.EscapeString(msg), "\n", "<br/>")
	f.contents = f.contents + "\t\t" + msg + "\n"
	return nil
}

func (f *pdfFile) WriteImage(imgPath string) error {
	f.contents = f.contents + fmt.Sprintf("\t\t<img src=%q/><br/>\n", imgPath)
	return nil
}

func (f *pdfFile) Close() error {
	if f.closed {
		return nil
	}
	f.closed = true
	contents := f.contents + "\n\t</body>\n</html>"
	page := wkhtmltopdf.NewPageReader(strings.NewReader(contents))
	page.EnableLocalFileAccess.Set(true)
	f.AddPage(page)
	if err := f.Create(); err != nil {
		return errors.Wrap(err, "create PDF in internal buffer")
	}
	return errors.Wrap(f.WriteFile(f.filePath), "write buffer contents to file")
}
