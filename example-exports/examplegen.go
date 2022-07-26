package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/opsys"
)

type parameters struct {
	isPDF      bool
	exportPath string
}

func main() {
	firstMsg := "[2020-03-01 15:34:05] Me: Want to play tennis?\n"
	moreMsgs := []string{
		"[2020-03-01 15:34:41] Novak: I can't today. I'm still at the Dubai Open\n",
		"[2020-03-01 15:34:53] Me: Ah, okay. When are you back in SF?\n",
		"[2020-03-01 15:35:23] Novak: Possibly next month. I'll let you know\n",
		"[2020-03-01 15:35:50] Me: 👍\n",
	}
	wd, err := os.Getwd()
	if err != nil {
		log.Panic(errors.Wrap(err, "get working directory"))
	}
	parentDir := filepath.Dir(wd)
	attachmentPath := filepath.Join(parentDir, "opsys/testdata/tennisballs.jpeg")

	runs := []parameters{
		{isPDF: false, exportPath: "messages-export"},
		{isPDF: true, exportPath: "messages-export-pdf"},
	}
	for _, params := range runs {
		s, err := opsys.NewOS(afero.NewOsFs(), os.Stat, exec.Command)
		if err != nil {
			log.Panic(errors.Wrap(err, "instantiate OS"))
		}
		chatPath := filepath.Join(params.exportPath, "Novak Djokovic")
		if err := s.MkdirAll(chatPath, os.ModePerm); err != nil {
			log.Panic(errors.Wrap(err, "create export directory"))
		}
		of, err := s.NewOutFile(filepath.Join(chatPath, "iMessage;-;+3815555555555"), params.isPDF, false)
		if err != nil {
			log.Panic(errors.Wrap(err, "create outfile"))
		}
		defer of.Close()
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
		if err := of.Close(); err != nil {
			log.Panic(errors.Wrap(err, "close outfile"))
		}
	}
}
