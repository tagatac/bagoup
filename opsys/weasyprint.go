package opsys

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/spf13/afero"
)

type weasyPrintFile struct {
	*pdfFile
	execCommand func(string, ...string) *exec.Cmd
}

func (s *opSys) NewWeasyPrintFile(entityName string, chatFile afero.File, includePPA bool) OutFile {
	return &weasyPrintFile{
		pdfFile:     newPDFFile(chatFile, includePPA, "templates/weasyprint_html.tmpl", entityName, s.bagoupVersion),
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
		return fmt.Errorf("%s: %w", stderr.String(), err)
	}
	return nil
}
