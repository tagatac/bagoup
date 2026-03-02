// Copyright (C) 2023  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

package typedstream_test

import (
	"encoding/binary"
	"testing"

	"github.com/tagatac/bagoup/v2/chatdb/typedstream"
	"gotest.tools/v3/assert"
)

// --- Binary construction helpers ---
// These helpers build valid typedstream binary blobs for use in tests.
// The shared string cache state during a full parse of buildAttributedString:
//   [0]: "NSMutableAttributedString"
//   [1]: "NSAttributedString"
//   [2]: "NSObject"
//   [3]: "@"
//   [4]: "NSMutableString"
//   [5]: "NSString"
//   [6]: "+"

const (
	bTagNew         = byte(0x84)
	bTagNil         = byte(0x85)
	bTagEndOfObject = byte(0x86)
	bTagInt16       = byte(0x81)
	bTagRefBase     = byte(0x92)
)

func bTypedUint(v uint64) []byte {
	if v < 0x81 {
		return []byte{byte(v)}
	}
	if v <= 0xffff {
		b := make([]byte, 3)
		b[0] = bTagInt16
		binary.LittleEndian.PutUint16(b[1:], uint16(v))
		return b
	}
	b := make([]byte, 5)
	b[0] = 0x82
	binary.LittleEndian.PutUint32(b[1:], uint32(v))
	return b
}

func bNewSharedStr(s string) []byte {
	result := []byte{bTagNew}
	result = append(result, bTypedUint(uint64(len(s)))...)
	result = append(result, []byte(s)...)
	return result
}

func bHeader() []byte {
	var buf []byte
	buf = append(buf, 0x04) // version
	buf = append(buf, 0x0b) // magic string length = 11
	buf = append(buf, []byte("streamtyped")...)
	buf = append(buf, bTypedUint(1000)...) // system version → 0x81 0xe8 0x03
	return buf
}

// buildAttributedString assembles a minimal but structurally correct
// NSMutableAttributedString typedstream blob that encodes the given text.
// The embedded NSMutableString contains only the plain text; no attribute
// ranges are included. The parser returns early upon finding the '+' string,
// so the trailing TAG_END_OF_OBJECT for the outer object is present but never read.
func buildAttributedString(text string) []byte {
	var buf []byte
	buf = append(buf, bHeader()...)
	buf = append(buf, bTagNew) // root object TAG_NEW

	// Class chain for NSMutableAttributedString (→ NSAttributedString → NSObject)
	buf = append(buf, bTagNew)                                        // class entry head
	buf = append(buf, bNewSharedStr("NSMutableAttributedString")...)  // strCache[0]
	buf = append(buf, bTypedUint(0)...)                               // version
	buf = append(buf, bTagNew)                                        // class entry head
	buf = append(buf, bNewSharedStr("NSAttributedString")...)         // strCache[1]
	buf = append(buf, bTypedUint(0)...)                               // version
	buf = append(buf, bTagNew)                                        // class entry head
	buf = append(buf, bNewSharedStr("NSObject")...)                   // strCache[2]
	buf = append(buf, bTypedUint(0)...)                               // version
	buf = append(buf, bTagNil)                                        // end class chain

	// Instance data group 1: type "@" → embedded NSMutableString object
	buf = append(buf, bNewSharedStr("@")...) // type encoding; strCache[3]
	buf = append(buf, bTagNew)               // TAG_NEW for the NSMutableString object

	// Class chain for NSMutableString (→ NSString → NSObject [cached as class ref 2])
	buf = append(buf, bTagNew)                              // class entry head
	buf = append(buf, bNewSharedStr("NSMutableString")...)  // strCache[4]
	buf = append(buf, bTypedUint(0)...)                     // version
	buf = append(buf, bTagNew)                              // class entry head
	buf = append(buf, bNewSharedStr("NSString")...)         // strCache[5]
	buf = append(buf, bTypedUint(0)...)                     // version
	buf = append(buf, bTagRefBase+2)                        // class object ref to NSObject (class index 2)

	// NSMutableString instance data: type "+" → the plain text
	buf = append(buf, bNewSharedStr("+")...)            // type encoding; strCache[6]
	buf = append(buf, bTypedUint(uint64(len(text)))...) // string length
	buf = append(buf, []byte(text)...)                  // string bytes
	buf = append(buf, bTagEndOfObject)                  // end NSMutableString

	// Outer NSMutableAttributedString TAG_END_OF_OBJECT
	// (not reached due to early return, but included for binary correctness)
	buf = append(buf, bTagEndOfObject)
	return buf
}

// --- Tests ---

func TestDecodeAttributedBody(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    string
		wantErr string
	}{
		{
			name: "plain ASCII text",
			data: buildAttributedString("Hello, world!"),
			want: "Hello, world!",
		},
		{
			name: "empty string",
			data: buildAttributedString(""),
			want: "",
		},
		{
			name: "unicode text",
			data: buildAttributedString("こんにちは"),
			want: "こんにちは",
		},
		{
			name: "long text requiring 2-byte length",
			data: buildAttributedString(string(make([]byte, 200))),
			want: string(make([]byte, 200)),
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: "read typedstream header: read version: unexpected end of data at position 0",
		},
		{
			name:    "wrong version byte",
			data:    []byte{0x03},
			wantErr: "read typedstream header: unsupported version 3 (expected 4)",
		},
		{
			name:    "wrong magic string",
			data:    append(append([]byte{0x04, 0x0b}, []byte("streamTYPED")...), bTypedUint(1000)...),
			wantErr: `read typedstream header: invalid magic string "streamTYPED"`,
		},
		{
			name:    "missing root TAG_NEW",
			data:    append(bHeader(), 0x00),
			wantErr: "expected TAG_NEW (0x84) for root object, got 0x00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := typedstream.DecodeAttributedBody(tt.data)
			if tt.wantErr != "" {
				assert.Error(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestSplitEncodings(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"@", []string{"@"}},
		{"@i", []string{"@", "i"}},
		{"+", []string{"+"}},
		{"{NSRange=QQ}", []string{"{NSRange=QQ}"}},
		{"{CGPoint=dd}i", []string{"{CGPoint=dd}", "i"}},
		{"@{NSRange=QQ}", []string{"@", "{NSRange=QQ}"}},
		{"iqi", []string{"i", "q", "i"}},
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := typedstream.SplitEncodings(tt.input)
			assert.DeepEqual(t, got, tt.want)
		})
	}
}
