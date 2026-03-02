// Copyright (C) 2023  David Tagatac <david@tagatac.net>
// See cmd/bagoup/main.go for usage terms.

// Package typedstream implements limited decoding of Apple's NSUnarchiver
// typedstream binary format, targeting NSAttributedString blobs stored in the
// attributedBody column of macOS Messages chat.db.
//
// Format reference: https://github.com/dgelessus/python-typedstream
package typedstream

import (
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	tagNew         = byte(0x84) // TAG_NEW: new literal value or new object
	tagNil         = byte(0x85) // TAG_NIL: nil / end of class chain
	tagEndOfObject = byte(0x86) // TAG_END_OF_OBJECT: terminates instance data
	tagInt16       = byte(0x81) // prefix: 2-byte LE integer follows
	tagInt32       = byte(0x82) // prefix: 4-byte LE integer follows
	tagRefBase     = byte(0x92) // first reference index; refs are tagRefBase+N

	headerVersion = 4
	headerMagic   = "streamtyped"
)

// DecodeAttributedBody extracts the plain text from an NSAttributedString
// encoded as a typedstream binary blob (the attributedBody column in chat.db).
func DecodeAttributedBody(data []byte) (string, error) {
	d := &decoder{data: data}
	if err := d.readHeader(); err != nil {
		return "", fmt.Errorf("read typedstream header: %w", err)
	}
	head, err := d.readByte()
	if err != nil {
		return "", fmt.Errorf("read root object tag: %w", err)
	}
	if head != tagNew {
		return "", fmt.Errorf("expected TAG_NEW (0x84) for root object, got 0x%02x", head)
	}
	text, _, err := d.readObject()
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(text, "\n"), nil
}

// decoder holds state for a single typedstream parse session.
// strCache is a shared pool for both class names and type encoding strings,
// indexed by reference bytes (>= tagRefBase).
type decoder struct {
	data     []byte
	pos      int
	strCache []string
}

func (d *decoder) readByte() (byte, error) {
	if d.pos >= len(d.data) {
		return 0, fmt.Errorf("unexpected end of data at position %d", d.pos)
	}
	b := d.data[d.pos]
	d.pos++
	return b, nil
}

func (d *decoder) peekByte() (byte, error) {
	if d.pos >= len(d.data) {
		return 0, fmt.Errorf("unexpected end of data at position %d", d.pos)
	}
	return d.data[d.pos], nil
}

// readTypedUint reads a variable-length unsigned integer.
// Values < 0x81 are stored as a single byte; 0x81 and 0x82 are prefixes for
// 2-byte and 4-byte little-endian values respectively.
func (d *decoder) readTypedUint() (uint64, error) {
	b, err := d.readByte()
	if err != nil {
		return 0, err
	}
	switch b {
	case tagInt16:
		if d.pos+2 > len(d.data) {
			return 0, fmt.Errorf("unexpected end of data reading 2-byte int at pos %d", d.pos)
		}
		v := binary.LittleEndian.Uint16(d.data[d.pos : d.pos+2])
		d.pos += 2
		return uint64(v), nil
	case tagInt32:
		if d.pos+4 > len(d.data) {
			return 0, fmt.Errorf("unexpected end of data reading 4-byte int at pos %d", d.pos)
		}
		v := binary.LittleEndian.Uint32(d.data[d.pos : d.pos+4])
		d.pos += 4
		return uint64(v), nil
	default:
		return uint64(b), nil
	}
}

// readSharedString reads a shared (cached) string from the stream.
// Shared strings are used for class names and type encoding strings.
func (d *decoder) readSharedString() (string, error) {
	head, err := d.readByte()
	if err != nil {
		return "", err
	}
	return d.readSharedStringWithHead(head)
}

// readSharedStringWithHead continues reading a shared string when the leading
// discriminator byte has already been consumed by the caller.
func (d *decoder) readSharedStringWithHead(head byte) (string, error) {
	switch {
	case head == tagNew:
		length, err := d.readTypedUint()
		if err != nil {
			return "", fmt.Errorf("read shared string length: %w", err)
		}
		if d.pos+int(length) > len(d.data) {
			return "", fmt.Errorf("data too short for shared string of length %d at pos %d", length, d.pos)
		}
		s := string(d.data[d.pos : d.pos+int(length)])
		d.pos += int(length)
		d.strCache = append(d.strCache, s)
		return s, nil

	case head == tagNil:
		return "", nil

	case head >= tagRefBase:
		idx := int(head - tagRefBase)
		if idx >= len(d.strCache) {
			return "", fmt.Errorf("shared string reference %d out of range (cache size %d)", idx, len(d.strCache))
		}
		return d.strCache[idx], nil

	case head == tagInt16:
		if d.pos+2 > len(d.data) {
			return "", fmt.Errorf("data too short for 2-byte string reference at pos %d", d.pos)
		}
		raw := int(binary.LittleEndian.Uint16(d.data[d.pos : d.pos+2]))
		d.pos += 2
		idx := raw - int(tagRefBase)
		if idx < 0 || idx >= len(d.strCache) {
			return "", fmt.Errorf("2-byte string reference %d out of range (cache size %d)", idx, len(d.strCache))
		}
		return d.strCache[idx], nil

	default:
		return "", fmt.Errorf("unexpected byte 0x%02x reading shared string at pos %d", head, d.pos-1)
	}
}

// readHeader validates the typedstream header and advances past it.
// Expected layout: [version=4] [0x0b] "streamtyped" [typed-uint system-version]
func (d *decoder) readHeader() error {
	ver, err := d.readByte()
	if err != nil {
		return fmt.Errorf("read version: %w", err)
	}
	if ver != headerVersion {
		return fmt.Errorf("unsupported version %d (expected %d)", ver, headerVersion)
	}
	mlen, err := d.readByte()
	if err != nil {
		return fmt.Errorf("read magic length: %w", err)
	}
	if int(mlen) != len(headerMagic) {
		return fmt.Errorf("unexpected magic length %d", mlen)
	}
	if d.pos+int(mlen) > len(d.data) {
		return fmt.Errorf("data too short for magic string")
	}
	if string(d.data[d.pos:d.pos+int(mlen)]) != headerMagic {
		return fmt.Errorf("invalid magic string %q", string(d.data[d.pos:d.pos+int(mlen)]))
	}
	d.pos += int(mlen)
	_, err = d.readTypedUint() // system version; value not needed
	return err
}

// readClassChain reads and discards the class inheritance chain following a
// TAG_NEW object marker. Each entry is: [head byte] [class name shared string]
// [version typed-uint]. The chain ends with TAG_NIL (0x85) or a reference byte
// (>= 0x92) pointing to a previously-seen class object.
func (d *decoder) readClassChain() error {
	for {
		head, err := d.readByte()
		if err != nil {
			return fmt.Errorf("read class chain head: %w", err)
		}
		switch {
		case head == tagNil:
			return nil
		case head >= tagRefBase:
			// Reference to a previously-seen class object; the chain ends here
			// because the referenced class already includes its superclass chain.
			return nil
		case head == tagNew:
			if _, err := d.readSharedString(); err != nil {
				return fmt.Errorf("read class name: %w", err)
			}
			if _, err := d.readTypedUint(); err != nil {
				return fmt.Errorf("read class version: %w", err)
			}
		default:
			return fmt.Errorf("unexpected byte 0x%02x in class chain at pos %d", head, d.pos-1)
		}
	}
}

// readObject reads an object's class chain and instance data.
// TAG_NEW has already been consumed by the caller.
// Returns the first plain-text string found within this object or any nested @-typed objects.
func (d *decoder) readObject() (string, bool, error) {
	if err := d.readClassChain(); err != nil {
		return "", false, fmt.Errorf("class chain: %w", err)
	}
	return d.readInstanceData()
}

// readInstanceData reads type-prefixed value groups until TAG_END_OF_OBJECT (0x86).
// Returns as soon as a plain-text string ('+' type) is found anywhere in the hierarchy.
func (d *decoder) readInstanceData() (string, bool, error) {
	for {
		next, err := d.peekByte()
		if err != nil {
			return "", false, err
		}
		if next == tagEndOfObject {
			d.pos++
			return "", false, nil
		}
		enc, err := d.readSharedString()
		if err != nil {
			return "", false, fmt.Errorf("read type encoding: %w", err)
		}
		if enc == "" {
			continue
		}
		text, found, err := d.readValuesByEncoding(enc)
		if err != nil {
			return "", false, fmt.Errorf("read values for encoding %q: %w", enc, err)
		}
		if found {
			return text, true, nil
		}
	}
}

// readValuesByEncoding reads all values described by a (possibly multi-type)
// Objective-C type encoding string and returns the first string found.
func (d *decoder) readValuesByEncoding(enc string) (string, bool, error) {
	for _, part := range SplitEncodings(enc) {
		text, found, err := d.readValue(part)
		if err != nil {
			return "", false, fmt.Errorf("read value for type %q: %w", part, err)
		}
		if found {
			return text, true, nil
		}
	}
	return "", false, nil
}

// readValue reads a single value of the given Objective-C type encoding.
// Returns (text, true, nil) when a '+'-typed string is found.
func (d *decoder) readValue(enc string) (string, bool, error) {
	if len(enc) == 0 {
		return "", false, nil
	}
	switch enc[0] {
	case '+':
		// NSString data: typed-uint length followed by UTF-8 bytes.
		length, err := d.readTypedUint()
		if err != nil {
			return "", false, fmt.Errorf("read string length: %w", err)
		}
		if d.pos+int(length) > len(d.data) {
			return "", false, fmt.Errorf("data too short for string of length %d at pos %d", length, d.pos)
		}
		s := string(d.data[d.pos : d.pos+int(length)])
		d.pos += int(length)
		return s, true, nil

	case '@':
		// Objective-C object: TAG_NEW starts a new object, TAG_NIL is nil.
		next, err := d.readByte()
		if err != nil {
			return "", false, err
		}
		if next == tagNil {
			return "", false, nil
		}
		if next != tagNew {
			return "", false, fmt.Errorf("expected TAG_NEW or TAG_NIL for '@' type, got 0x%02x at pos %d", next, d.pos-1)
		}
		return d.readObject()

	case '#':
		// Class object: encoded as a shared string.
		if _, err := d.readSharedString(); err != nil {
			return "", false, fmt.Errorf("read class '#' value: %w", err)
		}
		return "", false, nil

	case 'c', 'C', 'B':
		return d.skipBytes(1)
	case 's', 'S':
		return d.skipBytes(2)
	case 'i', 'I', 'l', 'L', 'f':
		return d.skipBytes(4)
	case 'q', 'Q', 'd':
		return d.skipBytes(8)

	case '{':
		return d.readStruct(enc)

	default:
		return "", false, fmt.Errorf("unsupported type encoding %q at pos %d", enc, d.pos)
	}
}

func (d *decoder) skipBytes(n int) (string, bool, error) {
	if d.pos+n > len(d.data) {
		return "", false, fmt.Errorf("unexpected end of data skipping %d bytes at pos %d", n, d.pos)
	}
	d.pos += n
	return "", false, nil
}

// readStruct reads a struct-encoded value: "{name=type1type2...}".
func (d *decoder) readStruct(enc string) (string, bool, error) {
	eqIdx := strings.Index(enc, "=")
	if eqIdx < 0 || enc[len(enc)-1] != '}' {
		return "", false, fmt.Errorf("malformed struct encoding %q", enc)
	}
	return d.readValuesByEncoding(enc[eqIdx+1 : len(enc)-1])
}

// SplitEncodings splits an Objective-C type encoding string into individual
// type tokens. For example "@i" → ["@","i"] and "{NSRange=QQ}" → ["{NSRange=QQ}"].
func SplitEncodings(enc string) []string {
	if enc == "" {
		return nil
	}
	var parts []string
	i := 0
	for i < len(enc) {
		switch enc[i] {
		case '{', '(', '[':
			open := enc[i]
			close := closingBracket(open)
			depth := 0
			j := i
			for j < len(enc) {
				if enc[j] == open {
					depth++
				} else if enc[j] == close {
					depth--
					if depth == 0 {
						j++
						break
					}
				}
				j++
			}
			parts = append(parts, enc[i:j])
			i = j
		default:
			parts = append(parts, string(enc[i]))
			i++
		}
	}
	return parts
}

func closingBracket(open byte) byte {
	switch open {
	case '{':
		return '}'
	case '(':
		return ')'
	case '[':
		return ']'
	default:
		return 0
	}
}
