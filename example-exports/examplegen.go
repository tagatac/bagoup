// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/v2/opsys"
	"github.com/tagatac/bagoup/v2/opsys/pdfgen"
)

const _entityName = "Novak Djokovic"

var _version string

type parameters struct {
	isPDF      bool
	wkhtml     bool
	exportPath string
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
		log.Panic(errors.Wrap(err, "get working directory"))
	}
	parentDir := filepath.Dir(wd)
	attachmentPath := filepath.Join(
		parentDir,
		"opsys",
		"testdata",
		"tennisballs.jpeg",
	)

	exportRoot := ""
	if len(os.Args) > 1 {
		exportRoot = os.Args[1]
	}
	runs := []parameters{
		{isPDF: false, exportPath: filepath.Join(exportRoot, "messages-export")},
		{isPDF: true, exportPath: filepath.Join(exportRoot, "messages-export-pdf")},
		{isPDF: true, wkhtml: true, exportPath: filepath.Join(exportRoot, "messages-export-wkhtmltopdf")},
	}
	s := opsys.NewOS(afero.NewOsFs(), os.Stat, _version)
	for _, params := range runs {
		chatPath := filepath.Join(params.exportPath, _entityName)
		if err := s.MkdirAll(chatPath, os.ModePerm); err != nil {
			log.Panic(errors.Wrap(err, "create export directory"))
		}
		chatFilePrefix := filepath.Join(chatPath, "iMessage,-,+3815555555555")
		var of opsys.OutFile
		if params.isPDF {
			chatFile, err := s.Create(chatFilePrefix + ".pdf")
			if err != nil {
				log.Panic(errors.Wrap(err, "create PDF chat file"))
			}
			defer chatFile.Close()
			if params.wkhtml {
				pdfg, err := pdfgen.NewPDFGenerator(chatFile)
				if err != nil {
					log.Panic(errors.Wrap(err, "create PDF generator"))
				}
				of = s.NewWkhtmltopdfFile(_entityName, chatFile, pdfg, false)
			} else {
				of = s.NewWeasyPrintFile(_entityName, chatFile, false)
			}
		} else {
			chatFile, err := s.Create(chatFilePrefix + ".txt")
			if err != nil {
				log.Panic(errors.Wrap(err, "create text chat file"))
			}
			defer chatFile.Close()
			of = s.NewTxtOutFile(chatFile)
		}
		if err := of.WriteMessage(firstMsg); err != nil {
			log.Panic(errors.Wrapf(err, "write message %q", firstMsg))
		}
		if _, err := of.WriteAttachment(attachmentPath); err != nil {
			log.Panic(errors.Wrap(err, "include attachment"))
		}
		for _, msg := range moreMsgs {
			if err := of.WriteMessage(msg); err != nil {
				log.Panic(errors.Wrapf(err, "write message %q", msg))
			}
		}
		if _, err := of.Stage(); err != nil {
			log.Panic(errors.Wrap(err, "stage outfile"))
		}
		if err := of.Flush(); err != nil {
			log.Panic(errors.Wrap(err, "flush outfile"))
		}
	}
}
