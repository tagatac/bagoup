package chatdb

import (
	"bytes"
	"strings"

	"github.com/pkg/errors"
)

func (d *chatDB) decodeTypedStream(s string) (string, error) {
	cmd := d.execCommand("typedstream-decode")
	cmd.Stdin = bytes.NewReader([]byte(s))
	decodedBodyBytes, err := cmd.Output()
	if err != nil {
		return "", errors.Wrap(err, "decode attributedBody - POSSIBLE FIX: Add typedstream-decode to your system path (installed with bagoup)")
	}
	decodedBody := string(decodedBodyBytes)
	decodedBody = _TypedStreamAttributeRE.ReplaceAllString(decodedBody, "")
	decodedBody = _TypedStreamMultilineAttributeRE.ReplaceAllString(decodedBody, "")
	return strings.TrimSuffix(decodedBody, "\n"), nil
}
