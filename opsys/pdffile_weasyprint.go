package opsys

import (
	"os/exec"

	"github.com/spf13/afero"
)

type weasyprintFile struct{ *pdfFile }

func (opSys) NewWeasyPrintOutFile(chatFile afero.File, includePPA bool) OutFile {
	return &weasyprintFile{pdfFile: newPDFFile(chatFile, includePPA, "templates/weasyprint_html.tmpl")}
}

func (f *weasyprintFile) Flush() error {
	cmd := exec.Command("weasyprint", "-", "-")
	cmd.Stdin = &f.buf
	cmd.Stdout = f.File
	return cmd.Run()
}
