// Copyright (C) 2022 David Tagatac <david@tagatac.net>
// See main.go for usage terms.

package opsys

import (
	"html/template"
	"os"
	"testing"

	"github.com/spf13/afero"
	"gotest.tools/v3/assert"
)

func TestTxtFile(t *testing.T) {
	rwFS := afero.NewMemMapFs()

	// Create OutFile in read/write filesystem
	rwOS := NewOS(rwFS, nil, nil)
	var err error
	rwOF, err := rwOS.NewOutFile("testfile", false, false)
	assert.NilError(t, err)

	// Create OutFile in read-only filesystem
	roOS := NewOS(afero.NewReadOnlyFs(rwFS), nil, nil)
	_, err = roOS.NewOutFile("testfile", false, false)
	assert.Error(t, err, `open file "testfile.txt": operation not permitted`)
	roFile, err := roOS.OpenFile("testfile.txt", os.O_RDONLY, 0444)
	assert.NilError(t, err)
	roOF := &txtFile{File: roFile}

	// Get name
	assert.Equal(t, rwOF.Name(), "testfile.txt")

	// Write message
	assert.NilError(t, rwOF.WriteMessage("test message\n"))
	assert.Error(t, roOF.WriteMessage("test message\n"), "write testfile.txt: file handle is read only")

	// Write attachment
	assert.NilError(t, rwOF.WriteAttachment("tennisballs.jpeg"))
	assert.Error(t, roOF.WriteAttachment("tennisballs.jpeg"), "write testfile.txt: file handle is read only")

	// Close
	assert.NilError(t, rwOF.Close())
	assert.NilError(t, rwOF.Close())

	// Check file contents
	contents, err := afero.ReadFile(rwFS, "testfile.txt")
	assert.NilError(t, err)
	assert.Equal(t, string(contents), "test message\n<attached: tennisballs.jpeg>\n")

	// Write after closing
	assert.Error(t, rwOF.WriteMessage("test message after closing\n"), "File is closed")
	assert.Error(t, rwOF.WriteMessage("attachment"), "File is closed")
}

func TestPDFFile(t *testing.T) {
	tests := []struct {
		msg        string
		includePPA bool
		wantHTML   template.HTML
		wantErr    string
	}{
		{
			msg: "happy",
			wantHTML: template.HTML(
				`

<!doctype html>
<html>
    <head>
        <title>testfile</title>
        <meta charset="utf-8">

        
        <style>
            img.emoji {
                height: 1em;
                width: 1em;
                margin: 0 .05em 0 .1em;
                vertical-align: -0.1em;
            }
        </style>
        <script src="https://twemoji.maxcdn.com/v/latest/twemoji.min.js"></script>
        <script>window.onload = function () { twemoji.parse(document.body); }</script>

        <style>
            img {
                max-width: 875px;
                max-height: 1300px;
            }
        </style>
    </head>
    <body>
        test message<br/>
        <img src="tennisballs.jpeg" alt=tennisballs.jpeg/><br/>
        <em>&lt;attached: video.mov&gt;</em><br/>
        <em>&lt;attached: signallogo.pluginPayloadAttachment&gt;</em><br/>
        
    </body>
</html>
`,
			),
		},
		{
			msg:        "include plugin payload attachments",
			includePPA: true,
			wantHTML: template.HTML(
				`

<!doctype html>
<html>
    <head>
        <title>testfile</title>
        <meta charset="utf-8">

        
        <style>
            img.emoji {
                height: 1em;
                width: 1em;
                margin: 0 .05em 0 .1em;
                vertical-align: -0.1em;
            }
        </style>
        <script src="https://twemoji.maxcdn.com/v/latest/twemoji.min.js"></script>
        <script>window.onload = function () { twemoji.parse(document.body); }</script>

        <style>
            img {
                max-width: 875px;
                max-height: 1300px;
            }
        </style>
    </head>
    <body>
        test message<br/>
        <img src="tennisballs.jpeg" alt=tennisballs.jpeg/><br/>
        <em>&lt;attached: video.mov&gt;</em><br/>
        <img src="signallogo.pluginPayloadAttachment" alt=signallogo.pluginPayloadAttachment/><br/>
        
    </body>
</html>
`,
			),
			wantErr: "write out PDF: Loading page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			// Create OutFile in read/write filesystem
			of, err := NewOS(afero.NewMemMapFs(), nil, nil).NewOutFile("testfile", true, tt.includePPA)
			assert.NilError(t, err)

			// Create OutFile in read-only filesystem
			_, err = NewOS(afero.NewReadOnlyFs(afero.NewMemMapFs()), nil, nil).NewOutFile("testfile", true, tt.includePPA)
			assert.Error(t, err, `open file "testfile.pdf": operation not permitted`)

			// Get name
			assert.Equal(t, of.Name(), "testfile.pdf")

			// Write message
			assert.NilError(t, of.WriteMessage("test message\n"))

			// Write attachments
			assert.NilError(t, of.WriteAttachment("tennisballs.jpeg"))
			assert.NilError(t, of.WriteAttachment("video.mov"))
			assert.NilError(t, of.WriteAttachment("signallogo.pluginPayloadAttachment"))

			if tt.wantErr != "" {
				assert.ErrorContains(t, of.Close(), tt.wantErr)
			}
			assert.NilError(t, of.Close())
			assert.NilError(t, of.Close())

			// Check HTML
			pdf := of.(*pdfFile)
			assert.Equal(t, pdf.html, tt.wantHTML)

			// Write after closing
			assert.Error(t, of.WriteMessage("test message after closing\n"), _errFileClosed.Error())
			assert.Error(t, of.WriteAttachment("attachment"), _errFileClosed.Error())
		})
	}
}
