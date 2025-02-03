// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package opsys

import (
	"testing"

	"github.com/spf13/afero"
	"gotest.tools/v3/assert"
)

func TestTxtFile(t *testing.T) {
	rwFS := afero.NewMemMapFs()

	// Create OutFile in read/write filesystem
	rwOS := &opSys{Fs: rwFS}
	rwFile, err := rwOS.Create("testfile.txt")
	assert.NilError(t, err)
	defer rwFile.Close()
	rwOF := opSys{}.NewTxtOutFile(rwFile)
	assert.NilError(t, err)

	// Create OutFile in read-only filesystem
	roOS := &opSys{Fs: afero.NewReadOnlyFs(rwFS)}
	roFile, err := roOS.Open("testfile.txt")
	assert.NilError(t, err)
	roOF := &txtFile{File: roFile}

	// Get name
	assert.Equal(t, rwOF.Name(), "testfile.txt")

	// Write message
	assert.NilError(t, rwOF.WriteMessage("test message\n"))
	assert.Error(t, roOF.WriteMessage("test message\n"), "write testfile.txt: file handle is read only")

	// Write attachment
	embedded, err := rwOF.WriteAttachment("tennisballs.jpeg")
	assert.NilError(t, err)
	assert.Equal(t, embedded, false)
	embedded, err = roOF.WriteAttachment("tennisballs.jpeg")
	assert.Error(t, err, "write testfile.txt: file handle is read only")
	assert.Equal(t, embedded, false)

	// Stage (no-op) and close the text file
	imgCount, err := rwOF.Stage()
	assert.NilError(t, err)
	assert.Equal(t, imgCount, 0)
	err = rwOF.Flush()
	assert.NilError(t, err)

	// Check file contents
	contents, err := afero.ReadFile(rwFS, "testfile.txt")
	assert.NilError(t, err)
	assert.Equal(t, string(contents), "test message\n<attached: tennisballs.jpeg>\n")
}
