package opsys

import (
	"os/exec"

	"github.com/spf13/afero"
)

type weasyprintFile struct {
	*pdfFile
	execCommand func(string, ...string) *exec.Cmd
}

func (s *opSys) NewWeasyPrintFile(chatFile afero.File, includePPA bool) OutFile {
	return &weasyprintFile{
		pdfFile:     newPDFFile(chatFile, includePPA, "templates/weasyprint_html.tmpl"),
		execCommand: s.execCommand,
	}
}

func (f *weasyprintFile) Flush() error {
	cmd := f.execCommand("weasyprint", "-", "-")
	cmd.Stdin = &f.buf
	cmd.Stdout = f.File
	return cmd.Run()
}
