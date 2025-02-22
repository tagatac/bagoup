package opsys

import (
	"bytes"
	"os/exec"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

type weasyPrintFile struct {
	*pdfFile
	execCommand func(string, ...string) *exec.Cmd
}

func (s *opSys) NewWeasyPrintFile(chatFile afero.File, includePPA bool) OutFile {
	return &weasyPrintFile{
		pdfFile:     newPDFFile(chatFile, includePPA, "templates/weasyprint_html.tmpl"),
		execCommand: s.execCommand,
	}
}

func (f *weasyPrintFile) Flush() error {
	cmd := f.execCommand("weasyprint", "-", "-")
	cmd.Stdin = &f.buf
	cmd.Stdout = f.File
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, stderr.String())
	}
	return nil
}
