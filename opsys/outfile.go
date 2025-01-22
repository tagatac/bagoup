// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package opsys

import (
	"embed"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
)

//go:embed templates/* all:testdata/*
var _embedFS embed.FS

//go:generate mockgen -destination=mock_opsys/mock_outfile.go github.com/tagatac/bagoup/v2/opsys OutFile

// Outfile represents single messages export file, either text or PDF.
type OutFile interface {
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
