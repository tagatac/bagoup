package main

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestSanitizePhone(t *testing.T) {
	tests := []struct {
		msg   string
		dirty string
		clean string
	}{
		{
			msg:   "already clean",
			dirty: "+14155555555",
			clean: "+14155555555",
		},
		{
			msg:   "dirty",
			dirty: "+1 (415) 555-5555",
			clean: "+14155555555",
		},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			assert.Equal(t, tt.clean, sanitizePhone(tt.dirty))
		})
	}
}
