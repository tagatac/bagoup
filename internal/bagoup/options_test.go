package bagoup_test

import (
	"testing"

	"github.com/tagatac/bagoup/v2/internal/bagoup"
	"gotest.tools/v3/assert"
)

func TestValidateOptions(t *testing.T) {
	tests := []struct {
		msg     string
		opts    bagoup.Options
		wantErr string
	}{
		{
			msg: "no errors",
			opts: bagoup.Options{
				AttachmentsPath: "/",
			},
			wantErr: "",
		},
		{
			msg: "use wkhtmltopdf without PDF output",
			opts: bagoup.Options{
				OutputPDF:       false,
				UseWkhtmltopdf:  true,
				AttachmentsPath: "/",
			},
			wantErr: "the --wkhtml flag requires the --pdf flag",
		},
		{
			msg: "include plugin payload attachments without PDF output",
			opts: bagoup.Options{
				OutputPDF:       false,
				IncludePPA:      true,
				AttachmentsPath: "/",
			},
			wantErr: "the --include-ppa flag requires the --pdf flag",
		},
		{
			msg: "preserve paths without copying attachments",
			opts: bagoup.Options{
				CopyAttachments: false,
				PreservePaths:   true,
				AttachmentsPath: "/",
			},
			wantErr: "the --preserve-paths flag requires the --copy-attachments flag",
		},
		{
			msg: "custom attachments path without using attachments",
			opts: bagoup.Options{
				CopyAttachments: false,
				OutputPDF:       false,
				AttachmentsPath: "testpath",
			},
			wantErr: "the --attachments-path flag requires a flag that uses those attachments: --copy-attachments or --pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			err := bagoup.ValidateOptions(tt.opts)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
		})
	}
}
