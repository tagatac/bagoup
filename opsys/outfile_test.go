// Copyright (C) 2022  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package opsys

import (
	"html/template"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/v2/opsys/pdfgen/mock_pdfgen"
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
	assert.Equal(t, false, embedded)
	embedded, err = roOF.WriteAttachment("tennisballs.jpeg")
	assert.Error(t, err, "write testfile.txt: file handle is read only")
	assert.Equal(t, false, embedded)

	// Stage (no-op) and close the text file
	imgCount, err := rwOF.Flush()
	assert.NilError(t, err)
	assert.Equal(t, imgCount, 0)

	// Check file contents
	contents, err := afero.ReadFile(rwFS, "testfile.txt")
	assert.NilError(t, err)
	assert.Equal(t, string(contents), "test message\n<attached: tennisballs.jpeg>\n")
}

func TestPDFFile(t *testing.T) {
	tests := []struct {
		msg          string
		includePPA   bool
		templatePath string
		setupMock    func(*mock_pdfgen.MockPDFGenerator)
		wantHTML     template.HTML
		wantImgCount int
		wantErr      string
	}{
		{
			msg: "happy",
			setupMock: func(pMock *mock_pdfgen.MockPDFGenerator) {
				gomock.InOrder(
					pMock.EXPECT().AddPage(gomock.Any()),
					pMock.EXPECT().Create(),
				)
			},
			wantHTML: template.HTML(
				`

<!doctype html>
<html>
    <head>
        <title>testfile</title>
        <meta charset="utf-8">

        <style>
            body {
                word-wrap: break-word;
            }
            img {
                max-width: 875px;
                max-height: 1300px;
            }
        </style>

        
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

    </head>
    <body>
        test message<br/>
        <img src="tennisballs.jpeg" alt="tennisballs.jpeg"/><br/>
        <em>&lt;attached: video.mov&gt;</em><br/>
        <em>&lt;attached: signallogo.pluginPayloadAttachment&gt;</em><br/>
        
    </body>
</html>
`,
			),
			wantImgCount: 1,
		},
		{
			msg:        "include plugin payload attachments",
			includePPA: true,
			setupMock: func(pMock *mock_pdfgen.MockPDFGenerator) {
				gomock.InOrder(
					pMock.EXPECT().AddPage(gomock.Any()),
					pMock.EXPECT().Create(),
				)
			},
			wantHTML: template.HTML(
				`

<!doctype html>
<html>
    <head>
        <title>testfile</title>
        <meta charset="utf-8">

        <style>
            body {
                word-wrap: break-word;
            }
            img {
                max-width: 875px;
                max-height: 1300px;
            }
        </style>

        
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

    </head>
    <body>
        test message<br/>
        <img src="tennisballs.jpeg" alt="tennisballs.jpeg"/><br/>
        <em>&lt;attached: video.mov&gt;</em><br/>
        <img src="signallogo.pluginPayloadAttachment" alt="signallogo.pluginPayloadAttachment"/><br/>
        
    </body>
</html>
`,
			),
			wantImgCount: 2,
		},
		{
			msg:          "bad template path",
			templatePath: "invalid template path",
			wantErr:      "parse HTML template: template: pattern matches no files: `invalid template path`",
		},
		{
			msg:          "invalid tamplate",
			templatePath: "testdata/outfile_html_invalid.tmpl",
			wantErr:      `execute HTML template: template: outfile_html_invalid.tmpl:1:2: executing "outfile_html_invalid.tmpl" at <.InvalidReference>: can't evaluate field InvalidReference in type opsys.htmlFileData`,
		},
		{
			msg: "PDF creation error",
			setupMock: func(pMock *mock_pdfgen.MockPDFGenerator) {
				gomock.InOrder(
					pMock.EXPECT().AddPage(gomock.Any()),
					pMock.EXPECT().Create().Return(errors.New("this is a PDF creation error")),
				)
			},
			wantErr: "write out PDF: this is a PDF creation error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			// Create outfile
			chatFile, err := afero.NewMemMapFs().Create("testfile.pdf")
			assert.NilError(t, err)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			pMock := mock_pdfgen.NewMockPDFGenerator(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(pMock)
			}
			of := opSys{}.NewPDFOutFile(chatFile, pMock, tt.includePPA)
			pdf, ok := of.(*pdfFile)
			assert.Equal(t, true, ok)
			if tt.templatePath != "" {
				pdf.templatePath = tt.templatePath
			}

			// Get name
			assert.Equal(t, of.Name(), "testfile.pdf")

			// Write message
			assert.NilError(t, of.WriteMessage("test message\n"))

			// Write attachments
			embedded, err := of.WriteAttachment("tennisballs.jpeg")
			assert.NilError(t, err)
			assert.Equal(t, true, embedded)
			embedded, err = of.WriteAttachment("video.mov")
			assert.NilError(t, err)
			assert.Equal(t, false, embedded)
			embedded, err = of.WriteAttachment("signallogo.pluginPayloadAttachment")
			assert.NilError(t, err)
			assert.Equal(t, tt.includePPA, embedded)

			// Stage the PDF
			imgCount, err := of.Flush()
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, tt.wantImgCount, imgCount)
			assert.Equal(t, pdf.html, tt.wantHTML)
		})
	}
}
