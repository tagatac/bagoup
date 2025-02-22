package opsys

import (
	"html/template"
	"testing"

	"github.com/spf13/afero"
	"github.com/tagatac/bagoup/v2/exectest"
	"gotest.tools/v3/assert"
)

func TestWeasyPrintFile(t *testing.T) {
	tests := []struct {
		msg                     string
		includePPA              bool
		includeProblematicPaths bool
		templatePath            string
		wantHTML                template.HTML
		wantImgCount            int
		wantStageErr            string
		weasyErr                string
		weasyExitCode           int
		wantFlushErr            string
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
            @page {
                margin: 0.35in;
                margin-top: 0.3in;
                margin-bottom: 0.3in;
            }
            body {
                font-size: 9.5pt;
                word-wrap: break-word;
            }
            img {
                image-resolution: 120dpi;
                max-width: 7.4in;
                max-height: 11.1in;
            }
        </style>

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
			wantHTML: template.HTML(
				`

<!doctype html>
<html>
    <head>
        <title>testfile</title>
        <meta charset="utf-8">

        <style>
            @page {
                margin: 0.35in;
                margin-top: 0.3in;
                margin-bottom: 0.3in;
            }
            body {
                font-size: 9.5pt;
                word-wrap: break-word;
            }
            img {
                image-resolution: 120dpi;
                max-width: 7.4in;
                max-height: 11.1in;
            }
        </style>

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
			msg:                     "problematic filenames",
			includeProblematicPaths: true,
			wantHTML: template.HTML(
				`

<!doctype html>
<html>
    <head>
        <title>testfile</title>
        <meta charset="utf-8">

        <style>
            @page {
                margin: 0.35in;
                margin-top: 0.3in;
                margin-bottom: 0.3in;
            }
            body {
                font-size: 9.5pt;
                word-wrap: break-word;
            }
            img {
                image-resolution: 120dpi;
                max-width: 7.4in;
                max-height: 11.1in;
            }
        </style>

    </head>
    <body>
        test message<br/>
        <img src="problematic-paths/question%3Fmark.jpeg" alt="question?mark.jpeg"/><br/>
        <img src="problematic-paths/narrow%E2%80%AFno-break%E2%80%AFspace.jpeg" alt="narrow\u202fno-break\u202fspace.jpeg"/><br/>
        
    </body>
</html>
`,
			),
			wantImgCount: 2,
		},
		{
			msg:          "bad template path",
			templatePath: "invalid template path",
			wantStageErr: "parse HTML template: template: pattern matches no files: `invalid template path`",
		},
		{
			msg:          "invalid template",
			templatePath: "testdata/outfile_html_invalid.tmpl",
			wantStageErr: `execute HTML template: template: outfile_html_invalid.tmpl:1:2: executing "outfile_html_invalid.tmpl" at <.InvalidReference>: can't evaluate field InvalidReference in type opsys.htmlFileData`,
		},
		{
			msg: "PDF creation error",
			wantHTML: template.HTML(
				`

<!doctype html>
<html>
    <head>
        <title>testfile</title>
        <meta charset="utf-8">

        <style>
            @page {
                margin: 0.35in;
                margin-top: 0.3in;
                margin-bottom: 0.3in;
            }
            body {
                font-size: 9.5pt;
                word-wrap: break-word;
            }
            img {
                image-resolution: 120dpi;
                max-width: 7.4in;
                max-height: 11.1in;
            }
        </style>

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
			wantImgCount:  1,
			weasyErr:      "this is a PDF creation error",
			weasyExitCode: 1,
			wantFlushErr:  "this is a PDF creation error: exit status 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			// Create outfile
			chatFile, err := afero.NewMemMapFs().Create("testfile.pdf")
			assert.NilError(t, err)
			s := &opSys{execCommand: exectest.GenFakeExecCommand("TestRunExecCmd", "", tt.weasyErr, tt.weasyExitCode)}
			of := s.NewWeasyPrintFile(chatFile, tt.includePPA)
			pdf, ok := of.(*weasyPrintFile)
			assert.Equal(t, ok, true)
			if tt.templatePath != "" {
				pdf.templatePath = tt.templatePath
			}

			// Get name
			assert.Equal(t, of.Name(), "testfile.pdf")

			// Write message
			assert.NilError(t, of.WriteMessage("test message\uFFFC\n"))

			// Write attachments
			if tt.includeProblematicPaths {
				embedded, err := of.WriteAttachment("problematic-paths/question?mark.jpeg")
				assert.NilError(t, err)
				assert.Equal(t, embedded, true)
				embedded, err = of.WriteAttachment("problematic-paths/narrow no-break space.jpeg")
				assert.NilError(t, err)
				assert.Equal(t, embedded, true)
			} else {
				embedded, err := of.WriteAttachment("tennisballs.jpeg")
				assert.NilError(t, err)
				assert.Equal(t, embedded, true)
				embedded, err = of.WriteAttachment("video.mov")
				assert.NilError(t, err)
				assert.Equal(t, embedded, false)
				embedded, err = of.WriteAttachment("signallogo.pluginPayloadAttachment")
				assert.NilError(t, err)
				assert.Equal(t, embedded, tt.includePPA)
			}

			// Stage the PDF
			imgCount, err := of.Stage()
			if tt.wantStageErr != "" {
				assert.Error(t, err, tt.wantStageErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, imgCount, tt.wantImgCount)
			assert.Equal(t, template.HTML(pdf.buf.String()), tt.wantHTML)

			// Flush the PDF
			err = of.Flush()
			if tt.wantFlushErr != "" {
				assert.Error(t, err, tt.wantFlushErr)
				return
			}
			assert.NilError(t, err)
		})
	}
}
