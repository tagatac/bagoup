package chatdb

import (
	"fmt"

	"github.com/pkg/errors"
	ts "github.com/tagatac/typedstream-go"
)

func (d *chatDB) decodeTypedStream(s string) (string, error) {
	u, err := ts.NewUnarchiverFromData([]byte(s))
	if err != nil {
		return "", errors.Wrap(err, "create unarchiver")
	}
	groups, err := u.DecodeAll()
	if err != nil {
		return "", errors.Wrap(err, "decode all")
	}
	// The top-level group has one value: the NSMutableAttributedString object.
	if len(groups) == 0 || len(groups[0].Values) == 0 {
		return "", fmt.Errorf("empty stream")
	}
	obj, ok := groups[0].Values[0].(*ts.GenericArchivedObject)
	if !ok {
		return "", fmt.Errorf("unexpected top-level type %T", groups[0].Values[0])
	}
	// The first content group holds the NSMutableString / NSString.
	if len(obj.Contents) == 0 || len(obj.Contents[0].Values) == 0 {
		return "", fmt.Errorf("no string content")
	}
	switch s := obj.Contents[0].Values[0].(type) {
	case *ts.NSMutableString:
		return s.Value, nil
	case *ts.NSString:
		return s.Value, nil
	default:
		return "", fmt.Errorf("unexpected string type %T", obj.Contents[0].Values[0])
	}
}
