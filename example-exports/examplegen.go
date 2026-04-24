// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/v2/imgconv"
	"github.com/tagatac/bagoup/v2/opsys"
	"github.com/tagatac/bagoup/v2/opsys/pdfgen"
)

const _entityName = "Novak Djokovic"

var _version string

type parameters struct {
	isPDF          bool
	wkhtml         bool
	exportPath     string
	attachmentPath string
}

func main() {
	firstMsg := "[2020-03-01 15:34:05] Me: Want to play tennis?\n"
	moreMsgs := []string{
		"[2020-03-01 15:34:41] Novak: I can't today. I'm still at the Dubai Open\n",
		"[2020-03-01 15:34:43] Novak: https://dubaidutyfreetennischampionships.com/\n",
		"[2020-03-01 15:34:53] Me: Ah, okay. When are you back in SF?\n",
		"[2020-03-01 15:35:23] Novak: Possibly next month. I'll let you know\n",
		"[2020-03-01 15:35:50] Me: 👍\n",
	}
	wd, err := os.Getwd()
	if err != nil {
		panic(fmt.Errorf("get working directory: %w", err))
	}
	parentDir := filepath.Dir(wd)
	attachmentPath := filepath.Join(
		parentDir,
		"opsys",
		"testdata",
		"tennisballs.heic",
	)

	exportRoot := ""
	if len(os.Args) > 1 {
		exportRoot = os.Args[1]
	}
	s := opsys.NewOS(afero.NewOsFs(), os.Stat, _version)
	tempDir, err := s.GetTempDir()
	if err != nil {
		panic(fmt.Errorf("get temporary directory: %w", err))
	}
	defer s.RmTempDir()
	ic := imgconv.NewImgConverter(tempDir)
	convertedImagePath, err := ic.ConvertHEIC(attachmentPath)
	runs := []parameters{
		{
			exportPath:     filepath.Join(exportRoot, "messages-export"),
			attachmentPath: attachmentPath,
		},
		{
			isPDF:          true,
			exportPath:     filepath.Join(exportRoot, "messages-export-pdf"),
			attachmentPath: convertedImagePath,
		},
		{
			isPDF:          true,
			wkhtml:         true,
			exportPath:     filepath.Join(exportRoot, "messages-export-wkhtmltopdf"),
			attachmentPath: convertedImagePath,
		},
	}
	if err != nil {
		panic(fmt.Errorf("convert HEIC image: %w", err))
	}
	for _, params := range runs {
		chatPath := filepath.Join(params.exportPath, _entityName)
		if err := s.MkdirAll(chatPath, os.ModePerm); err != nil {
			panic(fmt.Errorf("create export directory: %w", err))
		}
		chatFilePrefix := filepath.Join(chatPath, "iMessage,-,+3815555555555")
		var of opsys.OutFile
		var cf afero.File
		if params.isPDF {
			of, cf = createPDFFile(chatFilePrefix, s, params.wkhtml)
		} else {
			of, cf = createTxtFile(chatFilePrefix, s)
		}
		defer cf.Close()
		if err := of.WriteMessage(firstMsg); err != nil {
			panic(fmt.Errorf("write message %q: %w", firstMsg, err))
		}
		if _, err := of.WriteAttachment(params.attachmentPath); err != nil {
			panic(fmt.Errorf("include attachment: %w", err))
		}
		for _, msg := range moreMsgs {
			if err := of.WriteMessage(msg); err != nil {
				panic(fmt.Errorf("write message %q: %w", msg, err))
			}
		}
		if _, err := of.Stage(); err != nil {
			panic(fmt.Errorf("stage outfile: %w", err))
		}
		if err := of.Flush(); err != nil {
			panic(fmt.Errorf("flush outfile: %w", err))
		}
	}
	err = s.RmTempDir()
	if err != nil {
		panic(fmt.Errorf("remove temporary directory: %w", err))
	}
}

func createPDFFile(
	chatFilePrefix string,
	s opsys.OS,
	wkhtml bool,
) (opsys.OutFile, afero.File) {
	cf, err := s.Create(chatFilePrefix + ".pdf")
	if err != nil {
		panic(fmt.Errorf("create PDF chat file: %w", err))
	}
	if wkhtml {
		pdfg, err := pdfgen.NewPDFGenerator(cf)
		if err != nil {
			panic(fmt.Errorf("create PDF generator: %w", err))
		}
		return s.NewWkhtmltopdfFile(_entityName, cf, pdfg, false), cf
	}
	return s.NewWeasyPrintFile(_entityName, cf, false), cf
}

func createTxtFile(
	chatFilePrefix string,
	s opsys.OS,
) (opsys.OutFile, afero.File) {
	cf, err := s.Create(chatFilePrefix + ".txt")
	if err != nil {
		panic(fmt.Errorf("create text chat file: %w", err))
	}
	return s.NewTxtOutFile(cf), cf
}
